// Copyright 2021 tsuru authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package rpaas

import (
	"context"
	"fmt"
	"log"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	rpaas_client "github.com/tsuru/rpaas-operator/pkg/rpaas/client"
	"github.com/tsuru/rpaas-operator/pkg/rpaas/client/types"
)

func resourceRpaasCertManager() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceRpaasCertManagerUpsert,
		ReadContext:   resourceRpaasCertManagerRead,
		UpdateContext: resourceRpaasCertManagerUpsert,
		DeleteContext: resourceRpaasCertManagerDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			"instance": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "RPaaS Instance Name",
			},
			"service_name": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "RPaaS Service Name",
			},
			"issuer": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "Issuer of certificate",
			},
			"dns_names": {
				Type:     schema.TypeList,
				Required: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Description: "DNS Names content",
			},
		},
	}
}

func resourceRpaasCertManagerUpsert(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	provider := meta.(*rpaasProvider)

	instance := d.Get("instance").(string)
	serviceName := d.Get("service_name").(string)

	rpaasClient, err := provider.RpaasClient.SetService(serviceName)
	if err != nil {
		return diag.Errorf("Unable to create client for service %s: %v", serviceName, err)
	}

	args := rpaas_client.UpdateCertManagerArgs{
		Instance: instance,
		CertManager: types.CertManager{
			Issuer:   d.Get("issuer").(string),
			DNSNames: parseDNSNames(d.Get("dns_names")),
		},
	}
	log.Printf("[DEBUG] creating certificate for instance %s, issuer name %s", args.Instance, args.CertManager.Issuer)

	err = rpaasRetry(ctx, d, func() error {
		return rpaasClient.UpdateCertManager(ctx, args)
	})

	if err != nil {
		return diag.Errorf("Unable to create/update cert-manager, issuer %s for instance %s: %v", args.CertManager.Issuer, instance, err)
	}

	d.SetId(fmt.Sprintf("%s %s", serviceName, instance))
	return resourceRpaasCertManagerRead(ctx, d, meta)
}

func resourceRpaasCertManagerRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	provider := meta.(*rpaasProvider)

	instance := d.Get("instance").(string)
	serviceName := d.Get("service_name").(string)
	rpaasClient, err := provider.RpaasClient.SetService(serviceName)
	if err != nil {
		return diag.Errorf("Unable to create client for service %s: %v", serviceName, err)
	}

	info, err := rpaasClient.Info(ctx, rpaas_client.InfoArgs{
		Instance: instance,
	})

	if err != nil {
		return diag.Errorf("Unable to read rpaas instance %s: %v", instance, err)
	}

	for _, certificate := range info.Certificates {
		if certificate.Name == "cert-manager" {
			return nil
		}
	}

	d.SetId("")

	return nil
}

func resourceRpaasCertManagerDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	provider := meta.(*rpaasProvider)

	instance := d.Get("instance").(string)
	serviceName := d.Get("service_name").(string)
	rpaasClient, err := provider.RpaasClient.SetService(serviceName)
	if err != nil {
		return diag.Errorf("Unable to create client for service %s: %v", serviceName, err)
	}

	err = rpaasRetry(ctx, d, func() error {
		return rpaasClient.DeleteCertManager(ctx, instance)
	})

	if err != nil {
		return diag.Errorf("Unable to remove cert-manager for instance %s: %v", instance, err)
	}
	return nil
}

func parseDNSNames(data interface{}) []string {
	values := []string{}

	for _, item := range data.([]interface{}) {
		values = append(values, item.(string))
	}

	return values
}
