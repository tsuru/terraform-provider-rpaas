// Copyright 2021 tsuru authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package rpaas

import (
	"context"
	"os"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/sirupsen/logrus"
	"github.com/tsuru/tsuru/cmd"
	"istio.io/pkg/log"

	"github.com/tsuru/rpaas-operator/pkg/rpaas/client"
	rpaas_client "github.com/tsuru/rpaas-operator/pkg/rpaas/client"
)

func Provider() *schema.Provider {
	p := &schema.Provider{
		Schema: map[string]*schema.Schema{
			"host": {
				Type:        schema.TypeString,
				Description: "Target to tsuru API",
				Optional:    true,
			},
			"token": {
				Type:        schema.TypeString,
				Description: "Token to authenticate on tsuru API (optional)",
				Optional:    true,
			},
			"skip_cert_verification": {
				Type:        schema.TypeBool,
				Description: "Disable certificate verification",
				Default:     false,
				Optional:    true,
				DefaultFunc: schema.EnvDefaultFunc("TSURU_SKIP_CERT_VERIFICATION", nil),
			},
		},
		ResourcesMap: map[string]*schema.Resource{
			"rpaas_autoscale": resourceRpaasAutoscale(),
			"rpaas_block":     resourceRpaasBlock(),
			"rpaas_route":     resourceRpaasRoute(),
		},
	}
	p.ConfigureContextFunc = func(ctx context.Context, d *schema.ResourceData) (interface{}, diag.Diagnostics) {
		return providerConfigure(ctx, d, p.TerraformVersion)
	}
	return p
}

type rpaasProvider struct {
	RpaasClient rpaas_client.Client
	Log         *logrus.Logger
}

func providerConfigure(ctx context.Context, d *schema.ResourceData, terraformVersion string) (interface{}, diag.Diagnostics) {
	logger := logrus.New()
	file, err := os.OpenFile("/tmp/rpaas-provider.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err == nil {
		logger.Out = file
	} else {
		log.Info("Failed to log to file, using default stderr")
	}

	host := d.Get("host").(string)
	if host == "" {
		target, err := cmd.GetTarget()
		if err != nil {
			return nil, diag.FromErr(err)
		}
		if target == "" {
			return nil, diag.Errorf("Tsuru target is empty")
		}
	} else {
		os.Setenv("TSURU_TARGET", host)
	}

	token := d.Get("token").(string)
	if token == "" {
		t, err := cmd.ReadToken()
		if err != nil {
			return nil, diag.FromErr(err)
		}
		if t == "" {
			return nil, diag.Errorf("Tsuru token is empty")
		}
		token = t
	}

	cli, err := rpaas_client.NewClientThroughTsuruWithOptions(
		host,
		token,
		"unset",
		rpaas_client.ClientOptions{
			Timeout:            10 * time.Second,
			InsecureSkipVerify: d.Get("skip_cert_verification").(bool),
		},
	)
	if err != nil {
		return nil, diag.FromErr(err)
	}

	p := &rpaasProvider{
		Log:         logger,
		RpaasClient: cli,
	}

	return p, nil
}

func rpaasRetry(ctx context.Context, d *schema.ResourceData, retryFunc func() error) error {
	return resource.RetryContext(ctx, d.Timeout(schema.TimeoutCreate), func() *resource.RetryError {
		err := retryFunc()

		if err == nil {
			return nil
		}

		if errUnexpected, ok := err.(*client.ErrUnexpectedStatusCode); ok {
			if strings.Contains(errUnexpected.Body, "event locked") {
				return resource.RetryableError(err)
			}
		}

		return resource.NonRetryableError(err)
	})
}
