// Copyright 2021 tsuru authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package provider

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"

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
			"certificate_name": {
				Type:        schema.TypeString,
				ForceNew:    true,
				Optional:    true,
				Description: "Certificate Name, required on new version of RPaaS API",
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

	instance := d.Get("instance").(string)
	issuer := d.Get("issuer").(string)
	certificateName := d.Get("certificate_name").(string)

	dnsNames := asSliceOfStrings(d.Get("dns_names"))

	tflog.Info(ctx, "Create rpaas_cert_manager", map[string]interface{}{
		"certificate_name": certificateName,
		"service":          serviceName,
		"instance":         instance,
		"issuer":           issuer,
		"dnsNames":         dnsNames,
	})

	err = rpaasRetry(ctx, d.Timeout(schema.TimeoutCreate), func() (*http.Response, error) {
		// UpdateCertManager is really an upsert
		return nil, rpaasClient.UpdateCertManager(ctx, rpaas_client.UpdateCertManagerArgs{
			Instance: instance,
			CertManager: types.CertManager{
				Name:     certificateName,
				Issuer:   issuer,
				DNSNames: dnsNames,
			},
		})
	})

	if err != nil {
		return diag.Errorf("could not create Cert Manager request: %v", err)
	}

	var id string
	if certificateName == "" {
		id = fmt.Sprintf("%s::%s::%s", serviceName, instance, issuer)
	} else {
		id = fmt.Sprintf("%s::%s::%s::%s", serviceName, instance, issuer, certificateName)
	}

	d.SetId(id)
	return resourceRpaasCertManagerRead(ctx, d, meta)
}

func resourceRpaasCertManagerRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	provider := meta.(*rpaasProvider)

	serviceName, instance, issuer, certificateName, err := parseCertManagerID(d.Id())
	if err != nil {
		return diag.Errorf("Unable to parse CertManager ID: %v", err)
	}
	var id string
	if certificateName == "" {
		id = fmt.Sprintf("%s::%s::%s", serviceName, instance, issuer)
	} else {
		id = fmt.Sprintf("%s::%s::%s::%s", serviceName, instance, issuer, certificateName)
	}
	d.SetId(id)
	d.Set("service_name", serviceName)
	d.Set("instance", instance)
	d.Set("issuer", issuer)
	d.Set("certificate_name", certificateName)

	rpaasClient, err := provider.RpaasClient.SetService(serviceName)
	if err != nil {
		return diag.Errorf("Unable to create client for service %s: %v", serviceName, err)
	}

	var requests []types.CertManager

	err = rpaasRetry(ctx, d.Timeout(schema.TimeoutRead), func() (*http.Response, error) {
		r, nerr := rpaasClient.ListCertManagerRequests(ctx, instance)
		if nerr != nil {
			return nil, nerr
		}

		requests = r
		return nil, nil
	})

	if err != nil {
		return diag.Errorf("could not list Cert Manager requests: %v", err)
	}

	request, found := findCertManagerRequestByIssuerAndName(requests, issuer, certificateName)
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

	serviceName, instance, issuer, certificateName, err := parseCertManagerID(d.Id())
	if err != nil {
		return diag.Errorf("Unable to parse CertManager ID: %v", err)
	}
	rpaasClient, err := provider.RpaasClient.SetService(serviceName)
	if err != nil {
		return diag.Errorf("Unable to create client for service %s: %v", serviceName, err)
	}

	dnsNames := asSliceOfStrings(d.Get("dns_names"))

	tflog.Info(ctx, "Update rpaas_cert_manager", map[string]interface{}{
		"certificate_name": certificateName,
		"service":          serviceName,
		"instance":         instance,
		"issuer":           issuer,
		"dnsNames":         dnsNames,
	})

	err = rpaasRetry(ctx, d.Timeout(schema.TimeoutUpdate), func() (*http.Response, error) {
		return nil, rpaasClient.UpdateCertManager(ctx, rpaas_client.UpdateCertManagerArgs{
			Instance: instance,
			CertManager: types.CertManager{
				Name:     certificateName,
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
	serviceName, instance, issuer, certificateName, err := parseCertManagerID(d.Id())
	if err != nil {
		return diag.Errorf("Unable to parse CertManager ID: %v", err)
	}

	provider := meta.(*rpaasProvider)

	rpaasClient, err := provider.RpaasClient.SetService(serviceName)
	if err != nil {
		return diag.Errorf("Unable to create client for service %s: %v", serviceName, err)
	}

	tflog.Info(ctx, "Delete rpaas_cert_manager", map[string]interface{}{
		"certificate_name": certificateName,
		"service":          serviceName,
		"instance":         instance,
		"issuer":           issuer,
	})

	err = rpaasRetry(ctx, d.Timeout(schema.TimeoutDelete), func() (*http.Response, error) {
		log.Printf("[DEBUG] Removing Cert Manager certificate request: {Service: %s, Instance: %s, Issuer: %s}", serviceName, instance, issuer)

		if certificateName != "" {
			return nil, rpaasClient.DeleteCertManagerByName(ctx, instance, certificateName)
		}
		return nil, rpaasClient.DeleteCertManagerByIssuer(ctx, instance, issuer)
	})

	if err != nil {
		return diag.Errorf("cannot remove Cert Manager request: %v", err)
	}

	return nil
}

func findCertManagerRequestByIssuerAndName(reqs []types.CertManager, issuer, name string) (*types.CertManager, bool) {
	for _, r := range reqs {
		if r.Issuer == issuer {
			if name != "" && r.Name != name {
				continue
			}
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

func parseCertManagerID(id string) (serviceName, instance, issuer, name string, err error) {
	splitID := strings.Split(id, "::")

	if len(splitID) > 4 || len(splitID) < 3 {
		serviceName, instance, issuer, err = parseCertManagerID_legacyV0(id)
		if err != nil {
			err = fmt.Errorf("Could not parse id %q. Format should be \"service::instance::issuer::certificateName\"", id)
		}
		return
	}

	serviceName = splitID[0]
	instance = splitID[1]
	issuer = splitID[2]
	if len(splitID) == 4 {
		name = splitID[3] // blank on older versions
	}
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
