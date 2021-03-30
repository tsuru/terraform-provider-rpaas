// Copyright 2021 tsuru authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package rpaas

import (
	"context"
	"os"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/sirupsen/logrus"
	rpaas_client "github.com/tsuru/rpaas-operator/pkg/rpaas/client"
	"istio.io/pkg/log"
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
		},
		ResourcesMap: map[string]*schema.Resource{
			"rpaas_autoscale": resourceRpaasAutoscale(),
		},
	}
	p.ConfigureContextFunc = func(ctx context.Context, d *schema.ResourceData) (interface{}, diag.Diagnostics) {
		return providerConfigure(ctx, d, p.TerraformVersion)
	}
	return p
}

type rpaasProvider struct {
	RPaaSClient *rpaas_client.Client
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
	// userAgent := fmt.Sprintf("HashiCorp/1.0 Terraform/%s", terraformVersion)

	// cfg := &tsuru.Configuration{
	// 	DefaultHeader: map[string]string{},
	// 	UserAgent:     userAgent,
	// }

	// host := d.Get("host").(string)
	// if host != "" {
	// 	cfg.BasePath = host
	// }
	// token := d.Get("token").(string)
	// if token != "" {
	// 	cfg.DefaultHeader["Authorization"] = token
	// }

	// client, err := client.ClientFromEnvironment(cfg)
	// if err != nil {
	// 	return nil, diag.FromErr(err)
	// }

	return &rpaasProvider{
		Log: logger,
	}, nil
}
