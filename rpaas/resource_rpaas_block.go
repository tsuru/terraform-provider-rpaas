// Copyright 2021 tsuru authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package rpaas

import (
	"context"
	"fmt"

	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	rpaas_client "github.com/tsuru/rpaas-operator/pkg/rpaas/client"
)

func resourceRpaasBlock() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceRpaasBlockCreate,
		ReadContext:   resourceRpaasBlockRead,
		UpdateContext: resourceRpaasBlockCreate,
		DeleteContext: resourceRpaasBlockDelete,
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
				Type:     schema.TypeString,
				Required: true,
				ValidateDiagFunc: func(value interface{}, path cty.Path) diag.Diagnostics {
					v := value.(string)
					validBlocks := []string{"root", "http", "server", "lua-server", "lua-worker"}
					for _, b := range validBlocks {
						if b == v {
							return nil
						}
					}
					return diag.Errorf("Unexpected block name value: %s", v)
				},
				Description: "Name of the block that will receive the custom configuration content",
			},
			"content": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Custom Nginx configuration",
			},
		},
	}
}

func resourceRpaasBlockCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	provider := meta.(*rpaasProvider)

	instance := d.Get("instance").(string)
	serviceName := d.Get("service_name").(string)
	blockName := d.Get("name").(string)
	content := d.Get("content").(string)
	rpaasClient, err := provider.RpaasClient.SetService(serviceName)
	if err != nil {
		return diag.Errorf("Unable to create client for service %s: %v", serviceName, err)
	}

	args := rpaas_client.UpdateBlockArgs{
		Instance: instance,
		Name:     blockName,
		Content:  content,
	}

	err = rpaasRetry(ctx, d, func() error {
		return rpaasClient.UpdateBlock(ctx, args)
	})

	if err != nil {
		return diag.Errorf("Unable to create/update block %s for instance %s: %v", blockName, instance, err)
	}

	d.SetId(fmt.Sprintf("%s/%s", serviceName, instance))
	return nil
}

func resourceRpaasBlockRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	provider := meta.(*rpaasProvider)

	instance := d.Get("instance").(string)
	serviceName := d.Get("service_name").(string)
	blockName := d.Get("name").(string)
	rpaasClient, err := provider.RpaasClient.SetService(serviceName)
	if err != nil {
		return diag.Errorf("Unable to create client for service %s: %v", serviceName, err)
	}

	blocks, err := rpaasClient.ListBlocks(ctx, rpaas_client.ListBlocksArgs{Instance: instance})
	if err != nil {
		return diag.Errorf("Unable to get block %s for instance %s: %v", blockName, instance, err)
	}

	for _, b := range blocks {
		if b.Name != blockName {
			continue
		}
		d.Set("name", b.Name)
		d.Set("content", b.Content)
		return nil
	}

	return nil
}

func resourceRpaasBlockDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	provider := meta.(*rpaasProvider)

	instance := d.Get("instance").(string)
	serviceName := d.Get("service_name").(string)
	blockName := d.Get("name").(string)
	rpaasClient, err := provider.RpaasClient.SetService(serviceName)
	if err != nil {
		return diag.Errorf("Unable to create client for service %s: %v", serviceName, err)
	}

	err = rpaasRetry(ctx, d, func() error {
		return rpaasClient.DeleteBlock(ctx, rpaas_client.DeleteBlockArgs{
			Instance: instance,
			Name:     blockName,
		})
	})

	if err != nil {
		return diag.Errorf("Unable to remove block for instance %s: %v", instance, err)
	}
	return nil
}
