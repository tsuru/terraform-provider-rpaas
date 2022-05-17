// Copyright 2021 tsuru authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package rpaas

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	rpaas_client "github.com/tsuru/rpaas-operator/pkg/rpaas/client"
)

var validBlocks = []string{"root", "http", "server", "lua-server", "lua-worker"}

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
				ForceNew: true,
				ValidateDiagFunc: func(value interface{}, path cty.Path) diag.Diagnostics {
					v := value.(string)

					for _, b := range validBlocks {
						if b == v {
							return nil
						}
					}
					return diag.Errorf("Unexpected block name value %q. Allowed values: %v", v, validBlocks)
				},
				Description: fmt.Sprintf("Name of the block that will receive the custom configuration content. Allowed values: %v", validBlocks),
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

	d.SetId(fmt.Sprintf("%s/%s/%s", serviceName, instance, blockName))
	return nil
}

func resourceRpaasBlockRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	provider := meta.(*rpaasProvider)

	serviceName, instance, blockName, err := parseRpaasBlockID(d.Id())
	d.Set("instance", instance)
	d.Set("service_name", serviceName)
	d.Set("name", blockName)

	rpaasClient, err := provider.RpaasClient.SetService(serviceName)
	if err != nil {
		return diag.Errorf("Unable to create client for service %s: %v", serviceName, err)
	}

	blocks, err := rpaasClient.ListBlocks(ctx, rpaas_client.ListBlocksArgs{Instance: instance})
	if err != nil {
		return diag.Errorf("Unable to get block %s for instance %s: %v", blockName, instance, err)
	}

	if blockName == "" {
		// auto-fix old buggy ID
		if len(blocks) > 1 {
			return diag.Errorf("This resource was created with a old buggy version of the provider. There are multiple blocks and it is not possible to figure out which one should be used. You must resolve it manually")
		}
		b := blocks[0]
		d.Set("name", b.Name)
		d.Set("content", b.Content)
		d.SetId(fmt.Sprintf("%s/%s/%s", serviceName, instance, b.Name))
		return nil
	}

	for _, b := range blocks {
		if b.Name == blockName {
			d.Set("name", b.Name)
			d.Set("content", b.Content)
			return nil
		}
	}

	d.SetId("")
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

func parseRpaasBlockID(id string) (serviceName, instance, blockName string, err error) {
	splitID := strings.Split(id, "/")

	// support old bugging ID
	if len(splitID) < 2 || len(splitID) > 3 {
		err = errors.New("Resource ID could not be parsed. Format should be \"service/instance/block\"")
		return
	}

	serviceName = splitID[0]
	instance = splitID[1]
	if len(splitID) == 3 {
		blockName = splitID[2]
	}

	return
}
