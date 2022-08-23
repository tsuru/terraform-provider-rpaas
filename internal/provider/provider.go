// Copyright 2021 tsuru authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package provider

import (
	"context"
	"fmt"
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
			"rpaas_autoscale":    resourceRpaasAutoscale(),
			"rpaas_block":        resourceRpaasBlock(),
			"rpaas_route":        resourceRpaasRoute(),
			"rpaas_certificate":  resourceRpaasCertificate(),
			"rpaas_cert_manager": resourceRpaasCertManager(),
			"rpaas_acl":          resourceRpaasACL(),
			"rpaas_file":         resourceRpaasFile(),
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

	var cli rpaas_client.Client
	rpaasClientOptions := rpaas_client.ClientOptions{
		Timeout:            10 * time.Minute,
		InsecureSkipVerify: d.Get("skip_cert_verification").(bool),
	}

	providerSkipTsuruPassthrough := os.Getenv("PROVIDER_SKIP_TSURU_PASSTHROUGH")
	if providerSkipTsuruPassthrough == "true" {
		cli, err = rpaas_client.NewClientWithOptions(os.Getenv("RPAAS_TARGET"), "", "", rpaasClientOptions)
	} else {
		cli, err = rpaas_client.NewClientThroughTsuruWithOptions(host, token, "unset", rpaasClientOptions)
	}
	if err != nil {
		return nil, diag.Errorf("Could not start Rpaas Client: %v", err)
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

func parseRpaasInstanceID(id string) (serviceName, instance string, err error) {
	parts := strings.Split(id, "::")
	if len(parts) != 2 {
		serviceName, instance, err = parseRpaasInstanceID_legacyV0(id)
		if err != nil {
			err = fmt.Errorf("Could not parse id %q. Format should be service::instance", id)
		}
		return
	}

	return parts[0], parts[1], nil
}

func parseRpaasInstanceID_legacyV0(id string) (serviceName, instance string, err error) {
	parts := strings.Split(id, "/")
	if len(parts) != 2 {
		return "", "", fmt.Errorf("Legacy ID cound not be parsed. Legacy format: service/instance")
	}

	return parts[0], parts[1], nil
}
