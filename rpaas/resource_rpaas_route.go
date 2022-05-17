// Copyright 2021 tsuru authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package rpaas

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	rpaas_client "github.com/tsuru/rpaas-operator/pkg/rpaas/client"
)

func resourceRpaasRoute() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceRpaasRouteCreate,
		ReadContext:   resourceRpaasRouteRead,
		UpdateContext: resourceRpaasRouteCreate,
		DeleteContext: resourceRpaasRouteDelete,
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
			"path": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "Path for this route",
			},
			"destination": {
				Type:         schema.TypeString,
				Optional:     true,
				ExactlyOneOf: []string{"destination", "content"},
				Description:  "Custom Nginx upstream destination",
			},
			"content": {
				Type:         schema.TypeString,
				Optional:     true,
				ExactlyOneOf: []string{"destination", "content"},
				Description:  "Custom Nginx configuration content",
			},
			"https_only": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "Only on https",
			},
		},
	}
}

func resourceRpaasRouteCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	provider := meta.(*rpaasProvider)

	serviceName := d.Get("service_name").(string)
	rpaasClient, err := provider.RpaasClient.SetService(serviceName)
	if err != nil {
		return diag.Errorf("Unable to create client for service %s: %v", serviceName, err)
	}

	instance := d.Get("instance").(string)
	path := d.Get("path").(string)
	httpsOnly := d.Get("https_only").(bool)
	args := rpaas_client.UpdateRouteArgs{
		Instance:  instance,
		Path:      path,
		HTTPSOnly: httpsOnly,
	}

	if content, ok := d.GetOk("content"); ok {
		args.Content = content.(string)
	}

	if destination, ok := d.GetOk("destination"); ok {
		args.Destination = destination.(string)
	}

	err = rpaasRetry(ctx, d, func() error {
		return rpaasClient.UpdateRoute(ctx, args)
	})
	if err != nil {
		return diag.Errorf("Unable to create/update route %s for instance %s: %v", path, instance, err)
	}

	d.SetId(fmt.Sprintf("%s::%s::%s", serviceName, instance, path)) // using :: instead of / because of Path
	return resourceRpaasRouteRead(ctx, d, meta)
}

func resourceRpaasRouteRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	provider := meta.(*rpaasProvider)

	serviceName, instance, path, err := parseRpaasRouteID(d.Id())
	if err != nil {
		return diag.Errorf("Unable to parse Route ID: %v", err)
	}

	d.Set("instance", instance)
	d.Set("service_name", serviceName)
	d.Set("path", path)

	rpaasClient, err := provider.RpaasClient.SetService(serviceName)
	if err != nil {
		return diag.Errorf("Unable to create client for service %s: %v", serviceName, err)
	}

	routes, err := rpaasClient.ListRoutes(ctx, rpaas_client.ListRoutesArgs{Instance: instance})
	if err != nil {
		return diag.Errorf("Unable to get block %s for instance %s: %v", path, instance, err)
	}

	if path == "" {
		// auto-fix old buggy ID
		if len(routes) > 1 {
			return diag.Errorf("This resource was created with a old buggy version of the provider. There are multiple routes and it is not possible to figure out which one should be used. You must resolve it manually")
		}
		if len(routes) == 1 {
			path = routes[0].Path
			d.SetId(fmt.Sprintf("%s::%s::%s", serviceName, instance, path))
		}
	}

	for _, b := range routes {
		if b.Path == path {
			d.Set("path", b.Path)
			d.Set("https_only", b.HTTPSOnly)
			d.Set("destination", b.Destination)
			d.Set("content", b.Content)
			return nil
		}
	}

	d.SetId("")
	return nil
}

func resourceRpaasRouteDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	provider := meta.(*rpaasProvider)

	instance := d.Get("instance").(string)
	serviceName := d.Get("service_name").(string)
	path := d.Get("path").(string)
	rpaasClient, err := provider.RpaasClient.SetService(serviceName)
	if err != nil {
		return diag.Errorf("Unable to create client for service %s: %v", serviceName, err)
	}

	err = rpaasRetry(ctx, d, func() error {
		return rpaasClient.DeleteRoute(ctx, rpaas_client.DeleteRouteArgs{
			Instance: instance,
			Path:     path,
		})
	})

	if err != nil {
		return diag.Errorf("Unable to remove route for instance %s: %v", instance, err)
	}
	return nil
}

func parseRpaasRouteID(id string) (serviceName, instance, path string, err error) {
	splitID := strings.Split(id, "::")

	if len(splitID) == 1 {
		// handle old buggy format: "service/instance"
		splitID = strings.Split(id, "/")
		if len(splitID) != 2 {
			err = fmt.Errorf("Resource ID could not be parsed. Format should be \"service::instance::path\", got %q", id)
			return
		}
		splitID = append(splitID, "") // setting path as empty string
	}

	if len(splitID) != 3 {
		err = fmt.Errorf("Resource ID could not be parsed. Format should be \"service::instance::path\", got %q", id)
		return
	}

	serviceName = splitID[0]
	instance = splitID[1]
	path = splitID[2]
	return
}
