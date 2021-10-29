// Copyright 2021 tsuru authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package rpaas

import (
	"context"
	"fmt"
	"log"
	"strings"

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
				Description: "Certificate issuer name",
			},
			"dns_names": {
				Type: schema.TypeList,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Required:    true,
				MinItems:    1,
				Description: "A list of DNS names to be associated with the certificate in Subject Alternative Names extension",
			},
		},
	}
}

func resourceRpaasCertManagerUpsert(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	provider := meta.(*rpaasProvider)

	serviceName := d.Get("service_name").(string)
	rpaasClient, err := provider.RpaasClient.SetService(serviceName)
	if err != nil {
		return diag.Errorf("Unable to create client for service %s: %v", serviceName, err)
	}

	instance, issuer, dnsNames := d.Get("instance").(string), d.Get("issuer").(string), asSliceOfStrings(d.Get("dns_names"))
	err = rpaasRetry(ctx, d, func() error {
		log.Printf("[DEBUG] Creating Cert Manager certificate request: {service: %s, instance: %s, issuer: %v, DNSes: %s}", serviceName, instance, issuer, strings.Join(dnsNames, ", "))

		return rpaasClient.UpdateCertManager(ctx, rpaas_client.UpdateCertManagerArgs{
			Instance: instance,
			CertManager: types.CertManager{
				Issuer:   issuer,
				DNSNames: dnsNames,
			},
		})
	})
	if err != nil {
		return diag.Errorf("could not create/update Cert Manager request: %v", err)
	}

	id := fmt.Sprintf("%s %s %s", serviceName, instance, issuer)
	log.Printf("[DEBUG] Cert Manager certificate request created/updated successfully, stored in ID: %s", id)
	d.SetId(id)

	return nil
}

func resourceRpaasCertManagerRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	provider := meta.(*rpaasProvider)

	serviceName := d.Get("service_name").(string)
	rpaasClient, err := provider.RpaasClient.SetService(serviceName)
	if err != nil {
		return diag.Errorf("Unable to create client for service %s: %v", serviceName, err)
	}

	instance := d.Get("instance").(string)
	requests, err := rpaasClient.ListCertManagerRequests(ctx, instance)
	if err != nil {
		return diag.Errorf("could not list Cert Manager requests: %v", err)
	}

	issuer := d.Get("issuer").(string)
	request, found := findCertManagerRequestByIssuer(requests, issuer)
	if !found {
		log.Printf("[DEBUG] Removing resource (ID: %s) from state as it's not found on RPaaS", d.Id())
		d.SetId("")
		return nil
	}

	d.Set("dns_names", request.DNSNames)

	return nil
}

func resourceRpaasCertManagerDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	provider := meta.(*rpaasProvider)

	serviceName := d.Get("service_name").(string)
	rpaasClient, err := provider.RpaasClient.SetService(serviceName)
	if err != nil {
		return diag.Errorf("Unable to create client for service %s: %v", serviceName, err)
	}

	instance, issuer := d.Get("instance").(string), d.Get("issuer").(string)
	err = rpaasRetry(ctx, d, func() error {
		log.Printf("[DEBUG] Removing Cert Manager certificate request: {Service: %s, Instance: %s, Issuer: %s}", serviceName, instance, issuer)
		return rpaasClient.DeleteCertManager(ctx, instance, issuer)
	})
	if err != nil {
		return diag.Errorf("cannot remove Cert Manager request: %v", err)
	}

	return nil
}

func findCertManagerRequestByIssuer(reqs []types.CertManager, issuer string) (*types.CertManager, bool) {
	for _, r := range reqs {
		if r.Issuer == issuer {
			return &r, true
		}
	}

	return nil, false
}

func asSliceOfStrings(data interface{}) []string {
	var values []string
	for _, item := range data.([]interface{}) {
		values = append(values, item.(string))
	}

	return values
}
