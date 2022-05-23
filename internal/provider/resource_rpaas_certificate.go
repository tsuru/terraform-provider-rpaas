// Copyright 2021 tsuru authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package provider

import (
	"context"
	"fmt"
	"log"
	"strings"

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

	d.SetId(fmt.Sprintf("%s::%s::%s", serviceName, instance, args.Name))
	return resourceRpaasCertificateRead(ctx, d, meta)
}

func resourceRpaasCertificateRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	provider := meta.(*rpaasProvider)

	serviceName, instance, name, err := parseRpaasCertificateID(d.Id())
	if err != nil {
		return diag.Errorf("Unable to parse Certificate ID: %v", err)
	}

	d.SetId(fmt.Sprintf("%s::%s::%s", serviceName, instance, name))
	d.Set("service_name", serviceName)
	d.Set("instance", instance)
	d.Set("name", name)

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

func parseRpaasCertificateID(id string) (serviceName, instance, certName string, err error) {
	splitID := strings.Split(id, "::")

	if len(splitID) != 3 {
		serviceName, instance, certName, err = parseRpaasCertificateID_legacyV0(id)
		if err != nil {
			err = fmt.Errorf("Could not parse id %q. Format should be \"service::instance::certName\"", id)
		}
		return
	}

	serviceName = splitID[0]
	instance = splitID[1]
	certName = splitID[2]
	return
}

func parseRpaasCertificateID_legacyV0(id string) (serviceName, instance, certName string, err error) {
	splitID := strings.Split(id, " ")

	if len(splitID) != 3 {
		err = fmt.Errorf("Resource ID could not be parsed. Legacy format: \"service instance certName\"")
		return
	}

	serviceName = splitID[0]
	instance = splitID[1]
	certName = splitID[2]
	return
}
