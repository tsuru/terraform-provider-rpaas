// Copyright 2025 tsuru authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package provider

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/tsuru/rpaas-operator/api/v1alpha1"
	"github.com/tsuru/rpaas-operator/pkg/rpaas/client"
	"github.com/tsuru/rpaas-operator/pkg/rpaas/client/types"
)

func resourceRpaasUpstream() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceRpaasUpstreamCreate,
		ReadContext:   resourceRpaasUpstreamRead,
		UpdateContext: resourceRpaasUpstreamUpdate,
		DeleteContext: resourceRpaasUpstreamDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		CustomizeDiff: func(ctx context.Context, diff *schema.ResourceDiff, meta interface{}) error {
			loadBalance := diff.Get("load_balance").(string)
			loadBalanceHashKey := diff.Get("load_balance_hash_key").(string)

			if loadBalance == "chash" && loadBalanceHashKey == "" {
				return fmt.Errorf("load_balance_hash_key is required when load_balance is set to 'chash'")
			}

			return nil
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
			"app": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "Primary app name",
			},
			"canary": {
				Type:        schema.TypeList,
				Optional:    true,
				Description: "Canary app names that participate in traffic distribution",
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"load_balance": {
				Type:         schema.TypeString,
				Optional:     true,
				Computed:     true,
				Description:  "Load balancing algorithm to be used for this upstream. Defaults to 'round_robin' if not specified",
				ValidateFunc: validation.StringInSlice([]string{"round_robin", "chash", "ewma"}, false),
			},
			"load_balance_hash_key": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Nginx variable for consistent hashing when load_balance is 'chash'",
			},
			"traffic_shaping_policy": {
				Type:        schema.TypeList,
				Optional:    true,
				MaxItems:    1,
				Description: "Traffic shaping policy for this upstream",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"weight": {
							Type:        schema.TypeInt,
							Optional:    true,
							Description: "Weight for weight-based routing",
						},
						"weight_total": {
							Type:        schema.TypeInt,
							Optional:    true,
							Description: "Total weight for weight-based routing",
						},
						"header": {
							Type:        schema.TypeString,
							Optional:    true,
							Description: "Header on which to redirect requests to this backend",
						},
						"header_value": {
							Type:          schema.TypeString,
							Optional:      true,
							ConflictsWith: []string{"traffic_shaping_policy.0.header_pattern"},
							Description:   "Header value on which to redirect requests to this backend",
						},
						"header_pattern": {
							Type:          schema.TypeString,
							Optional:      true,
							ConflictsWith: []string{"traffic_shaping_policy.0.header_value"},
							Description:   "Header value match pattern, support exact, regex",
						},
						"cookie": {
							Type:        schema.TypeString,
							Optional:    true,
							Description: "Cookie on which to redirect requests to this backend",
						},
					},
				},
			},
		},
	}
}

func resourceRpaasUpstreamCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	provider := meta.(*rpaasProvider)

	instance := d.Get("instance").(string)
	serviceName := d.Get("service_name").(string)
	app := d.Get("app").(string)

	rpaasClient, err := provider.RpaasClient.SetService(serviceName)
	if err != nil {
		return diag.Errorf("Unable to create client for service %s: %v", serviceName, err)
	}

	args := buildUpstreamOptionsArgs(d)

	tflog.Info(ctx, "Create Upstream Options", map[string]interface{}{
		"service":  serviceName,
		"instance": instance,
		"app":      app,
	})

	err = rpaasRetry(ctx, d.Timeout(schema.TimeoutCreate), func() (*http.Response, error) {
		return nil, rpaasClient.AddUpstreamOptions(ctx, args)
	})

	if err != nil {
		return diag.Errorf("Unable to create upstream options for instance %s: %v", instance, err)
	}

	d.SetId(fmt.Sprintf("%s::%s::%s", serviceName, instance, app))
	return resourceRpaasUpstreamRead(ctx, d, meta)
}

func resourceRpaasUpstreamRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	id := d.Id()

	serviceName, instance, app, err := parseUpstreamOptionsID(id)
	if err != nil {
		return diag.Errorf("Unable to parse upstream options ID: %v", err)
	}
	d.SetId(fmt.Sprintf("%s::%s::%s", serviceName, instance, app))

	provider := meta.(*rpaasProvider)
	rpaasClient, err := provider.RpaasClient.SetService(serviceName)
	if err != nil {
		return diag.Errorf("Unable to create client for service %s: %v", serviceName, err)
	}

	var upstreamOptions []types.UpstreamOptions

	err = rpaasRetry(ctx, d.Timeout(schema.TimeoutRead), func() (*http.Response, error) {
		opts, nerr := rpaasClient.ListUpstreamOptions(ctx, client.ListUpstreamOptionsArgs{
			Instance: instance,
		})
		if nerr != nil {
			return nil, nerr
		}

		upstreamOptions = opts
		return nil, nil
	})

	if err != nil {
		return diag.Errorf("Unable to list upstream options for instance %s: %v", instance, err)
	}

	for _, option := range upstreamOptions {
		if option.PrimaryBind == app {
			d.Set("service_name", serviceName)
			d.Set("instance", instance)
			d.Set("app", option.PrimaryBind)
			d.Set("canary", option.CanaryBinds)

			if option.LoadBalance != "" {
				d.Set("load_balance", string(option.LoadBalance))
			}

			// Note: LoadBalanceHashKey is not available in the current UpstreamOptions types from the client

			if tsp := flattenTrafficShapingPolicy(option.TrafficShapingPolicy); len(tsp) > 0 {
				d.Set("traffic_shaping_policy", tsp)
			}

			return nil
		}
	}

	d.SetId("")
	return nil
}

func resourceRpaasUpstreamUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	provider := meta.(*rpaasProvider)

	instance := d.Get("instance").(string)
	serviceName := d.Get("service_name").(string)
	app := d.Get("app").(string)

	rpaasClient, err := provider.RpaasClient.SetService(serviceName)
	if err != nil {
		return diag.Errorf("Unable to create client for service %s: %v", serviceName, err)
	}

	args := buildUpstreamOptionsArgs(d)

	tflog.Info(ctx, "Update Upstream Options", map[string]interface{}{
		"service":  serviceName,
		"instance": instance,
		"app":      app,
	})

	err = rpaasRetry(ctx, d.Timeout(schema.TimeoutUpdate), func() (*http.Response, error) {
		return nil, rpaasClient.UpdateUpstreamOptions(ctx, args)
	})

	if err != nil {
		return diag.Errorf("Unable to update upstream options for instance %s: %v", instance, err)
	}

	return resourceRpaasUpstreamRead(ctx, d, meta)
}

func resourceRpaasUpstreamDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	id := d.Id()
	serviceName, instance, app, err := parseUpstreamOptionsID(id)

	if err != nil {
		return diag.Errorf("Unable to parse upstream options ID: %v", err)
	}

	provider := meta.(*rpaasProvider)
	rpaasClient, err := provider.RpaasClient.SetService(serviceName)
	if err != nil {
		return diag.Errorf("Unable to create client for service %s: %v", serviceName, err)
	}

	tflog.Info(ctx, "Delete Upstream Options", map[string]interface{}{
		"id":       d.Id(),
		"service":  serviceName,
		"instance": instance,
		"app":      app,
	})

	err = rpaasRetry(ctx, d.Timeout(schema.TimeoutDelete), func() (*http.Response, error) {
		return nil, rpaasClient.DeleteUpstreamOptions(ctx, client.DeleteUpstreamOptionsArgs{
			Instance:    instance,
			PrimaryBind: app,
		})
	})

	if err != nil {
		return diag.Errorf("Unable to delete upstream options for instance %s: %v", instance, err)
	}

	return nil
}

func buildUpstreamOptionsArgs(d *schema.ResourceData) client.UpstreamOptionsArgs {
	args := client.UpstreamOptionsArgs{
		Instance:    d.Get("instance").(string),
		PrimaryBind: d.Get("app").(string),
	}

	if v, ok := d.GetOk("canary"); ok {
		canaryBinds := v.([]interface{})
		args.CanaryBinds = make([]string, len(canaryBinds))
		for i, bind := range canaryBinds {
			args.CanaryBinds[i] = bind.(string)
		}
	}

	if v, ok := d.GetOk("load_balance"); ok {
		args.LoadBalance = v.(string)
	}

	if v, ok := d.GetOk("load_balance_hash_key"); ok {
		args.LoadBalanceHashKey = v.(string)
	}

	if v, ok := d.GetOk("traffic_shaping_policy"); ok {
		tspList := v.([]interface{})
		if len(tspList) > 0 && tspList[0] != nil {
			tspMap := tspList[0].(map[string]interface{})
			tsp := client.TrafficShapingPolicy{}

			if weight, ok := tspMap["weight"]; ok {
				tsp.Weight = weight.(int)
			}
			if weightTotal, ok := tspMap["weight_total"]; ok {
				tsp.WeightTotal = weightTotal.(int)
			}
			if header, ok := tspMap["header"]; ok {
				tsp.Header = header.(string)
			}
			if headerValue, ok := tspMap["header_value"]; ok {
				tsp.HeaderValue = headerValue.(string)
			}
			if headerPattern, ok := tspMap["header_pattern"]; ok {
				tsp.HeaderPattern = headerPattern.(string)
			}
			if cookie, ok := tspMap["cookie"]; ok {
				tsp.Cookie = cookie.(string)
			}

			args.TrafficShapingPolicy = tsp
		}
	}

	return args
}

func flattenTrafficShapingPolicy(tsp v1alpha1.TrafficShapingPolicy) []interface{} {
	if tsp == (v1alpha1.TrafficShapingPolicy{}) {
		return nil
	}

	result := make(map[string]interface{})
	if tsp.Weight != 0 {
		result["weight"] = tsp.Weight
	}
	if tsp.WeightTotal != 0 {
		result["weight_total"] = tsp.WeightTotal
	}
	if tsp.Header != "" {
		result["header"] = tsp.Header
	}
	if tsp.HeaderValue != "" {
		result["header_value"] = tsp.HeaderValue
	}
	if tsp.HeaderPattern != "" {
		result["header_pattern"] = tsp.HeaderPattern
	}
	if tsp.Cookie != "" {
		result["cookie"] = tsp.Cookie
	}

	if len(result) == 0 {
		return nil
	}

	return []interface{}{result}
}

func parseUpstreamOptionsID(id string) (serviceName string, instance string, app string, err error) {
	splitID := strings.Split(id, "::")

	if len(splitID) != 3 {
		return "", "", "", fmt.Errorf("Could not parse id %q. Format should be \"service::instance::app\"", id)
	}

	return splitID[0], splitID[1], splitID[2], nil
}
