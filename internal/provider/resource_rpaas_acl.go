package provider

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceRpaasACL() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceRpaasACLCreate,
		ReadContext:   resourceRpaasACLRead,
		DeleteContext: resourceRpaasACLDelete,
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
			"host": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
				// TODO: domain validation
				Description: "Hostname of desired destination",
			},
			"port": {
				Type:        schema.TypeInt,
				Required:    true,
				ForceNew:    true,
				Description: "Number of port",
			},
		},
	}
}

func resourceRpaasACLCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	provider := meta.(*rpaasProvider)

	instance := d.Get("instance").(string)
	serviceName := d.Get("service_name").(string)
	host := d.Get("host").(string)
	port := d.Get("port").(int)

	rpaasClient, err := provider.RpaasClient.SetService(serviceName)
	if err != nil {
		return diag.Errorf("Unable to create client for service %s: %v", serviceName, err)
	}

	err = rpaasRetry(ctx, d, func() error {
		return rpaasClient.AddAccessControlList(ctx, instance, host, port)
	})

	if err != nil {
		return diag.Errorf("Unable to create ACL for instance %s: %v", instance, err)
	}

	d.SetId(fmt.Sprintf("%s::%s::%s::%d", serviceName, instance, host, port))
	return resourceRpaasACLRead(ctx, d, meta)
}

func resourceRpaasACLRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	id := d.Id()

	serviceName, instance, host, port, err := parseACLID(id)
	if err != nil {
		return diag.Errorf("Unable to parse ACL ID: %v", err)
	}
	d.SetId(fmt.Sprintf("%s::%s::%s::%d", serviceName, instance, host, port))

	provider := meta.(*rpaasProvider)
	rpaasClient, err := provider.RpaasClient.SetService(serviceName)
	if err != nil {
		return diag.Errorf("Unable to create client for service %s: %v", serviceName, err)
	}

	acls, err := rpaasClient.ListAccessControlList(ctx, instance)
	if err != nil {
		return diag.Errorf("Unable to list ACL for instance %s: %v", instance, err)
	}

	for _, acl := range acls {
		if acl.Host == host && acl.Port == port {
			d.Set("service_name", serviceName)
			d.Set("instance", instance)
			d.Set("host", acl.Host)
			d.Set("port", acl.Port)
			return nil
		}
	}

	d.SetId("")

	return nil
}

func resourceRpaasACLDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	id := d.Id()
	serviceName, instance, host, port, err := parseACLID(id)

	if err != nil {
		return diag.Errorf("Unable to parse ACL ID: %v", err)
	}

	provider := meta.(*rpaasProvider)
	rpaasClient, err := provider.RpaasClient.SetService(serviceName)
	if err != nil {
		return diag.Errorf("Unable to create client for service %s: %v", serviceName, err)
	}

	err = rpaasRetry(ctx, d, func() error {
		return rpaasClient.RemoveAccessControlList(ctx, instance, host, port)
	})

	if err != nil {
		return diag.Errorf("Unable to delete ACL for instance %s: %v", instance, err)
	}

	return nil
}

func parseACLID(id string) (serviceName string, instance string, host string, port int, err error) {
	splitID := strings.Split(id, "::")

	if len(splitID) != 4 {
		serviceName, instance, host, port, err = parseACLID_legacyV0(id)
		if err != nil {
			err = fmt.Errorf("Could not parse id %q. Format should be \"service::instance::host::port\"", id)
		}
		return
	}

	serviceName = splitID[0]
	instance = splitID[1]
	host = splitID[2]
	if port, err = strconv.Atoi(splitID[3]); err != nil {
		err = fmt.Errorf("Resource id %q has a wrong format. Format should be \"service::instance::host::port\" (port must be integer).", id)
	}
	return
}

func parseACLID_legacyV0(id string) (serviceName string, instance string, host string, port int, err error) {
	parts0 := strings.Split(id, " ")
	if len(parts0) != 2 {
		return "", "", "", 0, fmt.Errorf("invalid ACL ID. Legacy format: \"service/instance host:port\".")
	}

	parts1 := strings.Split(parts0[0], "/")
	if len(parts1) != 2 {
		return "", "", "", 0, fmt.Errorf("invalid ACL ID. Legacy format: \"service/instance host:port\".")
	}

	parts2 := strings.Split(parts0[1], ":")
	if len(parts2) != 2 {
		return "", "", "", 0, fmt.Errorf("invalid ACL ID. Legacy format: \"service/instance host:port\".")
	}

	port, _ = strconv.Atoi(parts2[1])

	return parts1[0], parts1[1], parts2[0], port, nil
}
