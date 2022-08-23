// Copyright 2021 tsuru authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package provider

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	rpaas_client "github.com/tsuru/rpaas-operator/pkg/rpaas/client"
	"github.com/tsuru/rpaas-operator/pkg/rpaas/client/types"
)

func resourceRpaasCertManager() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceRpaasCertManagerCreate,
		ReadContext:   resourceRpaasCertManagerRead,
		UpdateContext: resourceRpaasCertManagerUpdate,
		DeleteContext: resourceRpaasCertManagerDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(40 * time.Minute),
			Update: schema.DefaultTimeout(80 * time.Minute),
			Delete: schema.DefaultTimeout(40 * time.Minute),
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

func resourceRpaasCertManagerCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	provider := meta.(*rpaasProvider)

	serviceName := d.Get("service_name").(string)
	rpaasClient, err := provider.RpaasClient.SetService(serviceName)
	if err != nil {
		return diag.Errorf("Unable to create client for service %s: %v", serviceName, err)
	}

	instance, issuer, dnsNames := d.Get("instance").(string), d.Get("issuer").(string), asSliceOfStrings(d.Get("dns_names"))

	tflog.Info(ctx, "Create rpaas_cert_manager", map[string]interface{}{
		"service":  serviceName,
		"instance": instance,
		"issuer":   issuer,
		"dnsNames": dnsNames,
	})

	err = rpaasRetry(ctx, d, func() error {
		// UpdateCertManager is really an upsert
		return rpaasClient.UpdateCertManager(ctx, rpaas_client.UpdateCertManagerArgs{
			Instance: instance,
			CertManager: types.CertManager{
				Issuer:   issuer,
				DNSNames: dnsNames,
			},
		})
	})
	if err != nil {
		return diag.Errorf("could not create Cert Manager request: %v", err)
	}

	d.SetId(fmt.Sprintf("%s::%s::%s", serviceName, instance, issuer))
	return resourceRpaasCertManagerRead(ctx, d, meta)
}

func resourceRpaasCertManagerRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	provider := meta.(*rpaasProvider)

	serviceName, instance, issuer, err := parseCertManagerID(d.Id())
	if err != nil {
		return diag.Errorf("Unable to parse CertManager ID: %v", err)
	}

	d.SetId(fmt.Sprintf("%s::%s::%s", serviceName, instance, issuer))
	d.Set("service_name", serviceName)
	d.Set("instance", instance)
	d.Set("issuer", issuer)

	rpaasClient, err := provider.RpaasClient.SetService(serviceName)
	if err != nil {
		return diag.Errorf("Unable to create client for service %s: %v", serviceName, err)
	}

	requests, err := rpaasClient.ListCertManagerRequests(ctx, instance)
	if err != nil {
		return diag.Errorf("could not list Cert Manager requests: %v", err)
	}

	request, found := findCertManagerRequestByIssuer(requests, issuer)
	if !found {
		log.Printf("[DEBUG] Removing resource (ID: %s) from state as it's not found on RPaaS", d.Id())
		d.SetId("")
		return nil
	}

	d.Set("dns_names", request.DNSNames)

	return nil
}

func resourceRpaasCertManagerUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	provider := meta.(*rpaasProvider)

	serviceName, instance, issuer, err := parseCertManagerID(d.Id())
	if err != nil {
		return diag.Errorf("Unable to parse CertManager ID: %v", err)
	}
	rpaasClient, err := provider.RpaasClient.SetService(serviceName)
	if err != nil {
		return diag.Errorf("Unable to create client for service %s: %v", serviceName, err)
	}

	dnsNames := asSliceOfStrings(d.Get("dns_names"))

	tflog.Info(ctx, "Update rpaas_cert_manager", map[string]interface{}{
		"service":  serviceName,
		"instance": instance,
		"issuer":   issuer,
		"dnsNames": dnsNames,
	})

	err = rpaasRetry(ctx, d, func() error {
		return rpaasClient.UpdateCertManager(ctx, rpaas_client.UpdateCertManagerArgs{
			Instance: instance,
			CertManager: types.CertManager{
				Issuer:   issuer,
				DNSNames: dnsNames,
			},
		})
	})
	if err != nil {
		return diag.Errorf("could not update Cert Manager request: %v", err)
	}

	return resourceRpaasCertManagerRead(ctx, d, meta)
}

func resourceRpaasCertManagerDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	provider := meta.(*rpaasProvider)

	serviceName := d.Get("service_name").(string)
	rpaasClient, err := provider.RpaasClient.SetService(serviceName)
	if err != nil {
		return diag.Errorf("Unable to create client for service %s: %v", serviceName, err)
	}

	instance, issuer := d.Get("instance").(string), d.Get("issuer").(string)

	tflog.Info(ctx, "Delete rpaas_cert_manager", map[string]interface{}{
		"service":  serviceName,
		"instance": instance,
		"issuer":   issuer,
	})

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

func parseCertManagerID(id string) (serviceName, instance, issuer string, err error) {
	splitID := strings.Split(id, "::")

	if len(splitID) != 3 {
		serviceName, instance, issuer, err = parseCertManagerID_legacyV0(id)
		if err != nil {
			err = fmt.Errorf("Could not parse id %q. Format should be \"service::instance::issuer\"", id)
		}
		return
	}

	serviceName = splitID[0]
	instance = splitID[1]
	issuer = splitID[2]
	return
}

func parseCertManagerID_legacyV0(id string) (serviceName, instance, issuer string, err error) {
	splitID := strings.Split(id, " ")
	if len(splitID) != 3 {
		err = fmt.Errorf("Legacy ID cound not be parsed. Legacy format: \"service instance issuer\"")
		return
	}

	serviceName = splitID[0]
	instance = splitID[1]
	issuer = splitID[2]
	return
}
