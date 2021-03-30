// Copyright 2021 tsuru authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package rpaas

import (
	"context"
	"fmt"
	"time"

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
	instance := d.Get("instance").(string)
	serviceName := d.Get("service_name").(string)
	path := d.Get("path").(string)
	httpsOnly := false
	if v, ok := d.GetOk("force_https"); ok {
		httpsOnly = v.(bool)
	}

	cli, err := rpaas_client.NewClientThroughTsuruWithOptions("", "", serviceName, rpaas_client.ClientOptions{
		Timeout: 10 * time.Second,
	})
	if err != nil {
		return diag.Errorf("Unable to create client: %v", err)
	}

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

	err = cli.UpdateRoute(ctx, args)
	if err != nil {
		return diag.Errorf("Unable to create/update route %s for instance %s: %v", path, instance, err)
	}

	d.SetId(fmt.Sprintf("%s/%s", serviceName, instance))
	return nil
}

func resourceRpaasRouteRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	instance := d.Get("instance").(string)
	serviceName := d.Get("service_name").(string)
	path := d.Get("path").(string)

	cli, err := rpaas_client.NewClientThroughTsuruWithOptions("", "", serviceName, rpaas_client.ClientOptions{
		Timeout: 10 * time.Second,
	})
	if err != nil {
		return diag.Errorf("Unable to create client: %v", err)
	}

	routes, err := cli.ListRoutes(ctx, rpaas_client.ListRoutesArgs{Instance: instance})
	if err != nil {
		return diag.Errorf("Unable to get block %s for instance %s: %v", path, instance, err)
	}

	for _, b := range routes {
		if b.Path != path {
			continue
		}
		d.Set("path", b.Path)
		d.Set("https_only", b.HTTPSOnly)
		if b.Destination != "" {
			d.Set("destination", b.Destination)
		}
		if b.Content != "" {
			d.Set("content", b.Content)
		}
		return nil
	}

	return nil
}

func resourceRpaasRouteDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	instance := d.Get("instance").(string)
	serviceName := d.Get("service_name").(string)
	path := d.Get("path").(string)

	cli, err := rpaas_client.NewClientThroughTsuruWithOptions("", "", serviceName, rpaas_client.ClientOptions{
		Timeout: 10 * time.Second,
	})
	if err != nil {
		return diag.Errorf("Unable to create client: %v", err)
	}

	err = cli.DeleteRoute(ctx, rpaas_client.DeleteRouteArgs{
		Instance: instance,
		Path:     path,
	})

	if err != nil {
		return diag.Errorf("Unable to remove route for instance %s: %v", instance, err)
	}
	return nil
}
