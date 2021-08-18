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
)

func resourceRpaasCertificate() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceRpaasCertificateUpsert,
		ReadContext:   resourceRpaasCertificateRead,
		UpdateContext: resourceRpaasCertificateUpsert,
		DeleteContext: resourceRpaasCertificateDelete,
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
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "Name of certificate",
			},
			"certificate": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Certificate content",
			},
			"key": {
				Type:        schema.TypeString,
				Required:    true,
				Sensitive:   true,
				Description: "Key content",
			},
		},
	}
}

func resourceRpaasCertificateUpsert(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	provider := meta.(*rpaasProvider)

	instance := d.Get("instance").(string)
	serviceName := d.Get("service_name").(string)

	rpaasClient, err := provider.RpaasClient.SetService(serviceName)
	if err != nil {
		return diag.Errorf("Unable to create client for service %s: %v", serviceName, err)
	}

	args := rpaas_client.UpdateCertificateArgs{
		Instance:    instance,
		Name:        d.Get("name").(string),
		Certificate: d.Get("certificate").(string),
		Key:         d.Get("key").(string),
	}
	log.Printf("[DEBUG] creating certificate for instance %s, certificate name %s", args.Instance, args.Certificate)

	err = rpaasRetry(ctx, d, func() error {
		return rpaasClient.UpdateCertificate(ctx, args)
	})

	if err != nil {
		return diag.Errorf("Unable to create/update certificate %s for instance %s: %v", args.Certificate, instance, err)
	}

	d.SetId(fmt.Sprintf("%s %s %s", serviceName, instance, args.Name))
	return resourceRpaasCertificateRead(ctx, d, meta)
}

func resourceRpaasCertificateRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	provider := meta.(*rpaasProvider)

	instance := d.Get("instance").(string)
	serviceName := d.Get("service_name").(string)
	name := d.Get("name").(string)
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
		if certificate.Name == name {
			return nil
		}
	}

	d.SetId("")

	return nil
}

func resourceRpaasCertificateDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	provider := meta.(*rpaasProvider)

	instance := d.Get("instance").(string)
	serviceName := d.Get("service_name").(string)
	name := d.Get("name").(string)
	rpaasClient, err := provider.RpaasClient.SetService(serviceName)
	if err != nil {
		return diag.Errorf("Unable to create client for service %s: %v", serviceName, err)
	}

	err = rpaasRetry(ctx, d, func() error {
		return rpaasClient.DeleteCertificate(ctx, rpaas_client.DeleteCertificateArgs{
			Instance: instance,
			Name:     name,
		})
	})

	if err != nil {
		return diag.Errorf("Unable to remove certificate for instance %s: %v", instance, err)
	}
	return nil
}
