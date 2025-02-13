// Copyright 2021 tsuru authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package provider

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	rpaas_client "github.com/tsuru/rpaas-operator/pkg/rpaas/client"
	rpaastypes "github.com/tsuru/rpaas-operator/pkg/rpaas/client/types"
)

var validBlocks = []string{"root", "http", "server", "lua-server", "lua-worker"}

func resourceRpaasBlock() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceRpaasBlockCreate,
		ReadContext:   resourceRpaasBlockRead,
		UpdateContext: resourceRpaasBlockUpdate,
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
			"server_name": {
				Type:        schema.TypeString,
				Optional:    true,
				ForceNew:    true,
				Description: "Optional parameter used to match the server name in the block. If not provided, it will apply to all servers.",
			},
			"extend": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "Extend is a flag to indicate if the block should be appended to the default configuration, only valid when specify a server_name.",
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
	serverName := d.Get("server_name").(string)
	blockName := d.Get("name").(string)
	content := d.Get("content").(string)
	extend := d.Get("extend").(bool)

	rpaasClient, err := provider.RpaasClient.SetService(serviceName)
	if err != nil {
		return diag.Errorf("Unable to create client for service %s: %v", serviceName, err)
	}

	tflog.Info(ctx, "Create block", map[string]interface{}{
		"service":    serviceName,
		"instance":   instance,
		"serverName": serverName,
		"name":       blockName,
		"extend":     extend,
	})

	err = rpaasRetry(ctx, d.Timeout(schema.TimeoutCreate), func() (*http.Response, error) {
		return nil, rpaasClient.UpdateBlock(ctx, rpaas_client.UpdateBlockArgs{
			Instance:   instance,
			Name:       blockName,
			ServerName: serverName,
			Content:    content,
			Extend:     extend,
		})
	})

	if err != nil {
		return diag.Errorf("Unable to create/update block %s for instance %s: %v", blockName, instance, err)
	}

	if serverName == "" {
		d.SetId(fmt.Sprintf("%s::%s::%s", serviceName, instance, blockName))
	} else {
		d.SetId(fmt.Sprintf("%s::%s::%s::%s", serviceName, instance, serverName, blockName))
	}
	return resourceRpaasBlockRead(ctx, d, meta)
}

func resourceRpaasBlockUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	provider := meta.(*rpaasProvider)

	serviceName, instance, serverName, blockName, err := parseRpaasBlockID(d.Id())
	if err != nil {
		return diag.Errorf("Unable to parse Block ID: %v", err)
	}

	rpaasClient, err := provider.RpaasClient.SetService(serviceName)
	if err != nil {
		return diag.Errorf("Unable to create client for service %s: %v", serviceName, err)
	}

	content := d.Get("content").(string)
	extend := d.Get("extend").(bool)
	tflog.Info(ctx, "Update block", map[string]interface{}{
		"id":         d.Id(),
		"service":    serviceName,
		"instance":   instance,
		"serverName": serverName,
		"name":       blockName,
	})

	err = rpaasRetry(ctx, d.Timeout(schema.TimeoutUpdate), func() (*http.Response, error) {
		return nil, rpaasClient.UpdateBlock(ctx, rpaas_client.UpdateBlockArgs{
			Instance:   instance,
			Name:       blockName,
			ServerName: serverName,
			Content:    content,
			Extend:     extend,
		})
	})

	if err != nil {
		return diag.Errorf("Unable to update block %s for instance %s: %v", blockName, instance, err)
	}

	return resourceRpaasBlockRead(ctx, d, meta)
}

func resourceRpaasBlockRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	provider := meta.(*rpaasProvider)

	serviceName, instance, serverName, blockName, err := parseRpaasBlockID(d.Id())
	if err != nil {
		return diag.Errorf("Unable to parse Block ID: %v", err)
	}

	d.SetId(fmt.Sprintf("%s::%s::%s", serviceName, instance, blockName))
	d.Set("instance", instance)
	d.Set("service_name", serviceName)
	d.Set("server_name", serverName)

	rpaasClient, err := provider.RpaasClient.SetService(serviceName)
	if err != nil {
		return diag.Errorf("Unable to create client for service %s: %v", serviceName, err)
	}

	var blocks []rpaastypes.Block

	err = rpaasRetry(ctx, d.Timeout(schema.TimeoutRead), func() (*http.Response, error) {
		bs, nerr := rpaasClient.ListBlocks(ctx, rpaas_client.ListBlocksArgs{Instance: instance})
		if nerr != nil {
			return nil, nerr
		}

		blocks = bs
		return nil, nil
	})

	if err != nil {
		return diag.Errorf("Unable to get block %s for instance %s: %v", blockName, instance, err)
	}

	// auto-fix old buggy ID
	if blockName == "" {
		blockName = d.Get("name").(string) // defaults to config's value, if present
	}
	if blockName == "" && len(blocks) > 1 {
		return diag.Errorf("This resource was created with a old buggy version of the provider. There are multiple blocks and it is not possible to figure out which one should be used. You must resolve it manually")
	} else if blockName == "" && len(blocks) == 1 {
		blockName = blocks[0].Name
	}

	for _, b := range blocks {
		if b.ServerName != serverName || b.Name != blockName {
			continue
		}

		d.Set("name", b.Name)
		d.Set("content", b.Content)
		d.Set("extend", b.Extend)

		if serverName == "" {
			d.SetId(fmt.Sprintf("%s::%s::%s", serviceName, instance, blockName))
		} else {
			d.SetId(fmt.Sprintf("%s::%s::%s::%s", serviceName, instance, serverName, blockName))
		}
		return nil
	}

	// no match
	d.SetId("")
	return nil
}

func resourceRpaasBlockDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	provider := meta.(*rpaasProvider)

	serviceName, instance, serverName, blockName, err := parseRpaasBlockID(d.Id())
	if err != nil {
		return diag.Errorf("Unable to parse Block ID: %v", err)
	}

	rpaasClient, err := provider.RpaasClient.SetService(serviceName)
	if err != nil {
		return diag.Errorf("Unable to create client for service %s: %v", serviceName, err)
	}

	tflog.Info(ctx, "Delete block", map[string]interface{}{
		"id":         d.Id(),
		"service":    serviceName,
		"instance":   instance,
		"serverName": serverName,
		"name":       blockName,
	})

	err = rpaasRetry(ctx, d.Timeout(schema.TimeoutDelete), func() (*http.Response, error) {
		return nil, rpaasClient.DeleteBlock(ctx, rpaas_client.DeleteBlockArgs{
			Instance:   instance,
			Name:       blockName,
			ServerName: serverName,
		})
	})

	if err != nil {
		return diag.Errorf("Unable to remove block for instance %s: %v", instance, err)
	}

	return nil
}

func parseRpaasBlockID(id string) (serviceName, instance, serverName, blockName string, err error) {
	splitID := strings.Split(id, "::")

	if len(splitID) == 3 {
		serviceName = splitID[0]
		instance = splitID[1]
		blockName = splitID[2]
	} else if len(splitID) == 4 {
		serviceName = splitID[0]
		instance = splitID[1]
		serverName = splitID[2]
		blockName = splitID[3]
	} else {
		serviceName, instance, blockName, err = parseRpaasBlockID_legacyV0(id)
	}

	return
}

func parseRpaasBlockID_legacyV0(id string) (serviceName, instance, blockName string, err error) {
	splitID := strings.Split(id, "/")

	if len(splitID) != 2 {
		err = fmt.Errorf("Could not parse id %q. Format should be \"service::instance::blockName\" or \"service::instance::serverName::blockName\"", id)
		return
	}

	serviceName = splitID[0]
	instance = splitID[1]
	return
}
