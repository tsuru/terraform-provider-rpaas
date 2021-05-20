// Copyright 2021 tsuru authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package rpaas

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	rpaas_client "github.com/tsuru/rpaas-operator/pkg/rpaas/client"
)

func resourceRpaasAutoscale() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceRpaasAutoscaleCreate,
		ReadContext:   resourceRpaasAutoscaleRead,
		UpdateContext: resourceRpaasAutoscaleUpdate,
		DeleteContext: resourceRpaasAutoscaleDelete,
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
			"min_replicas": {
				Type:        schema.TypeInt,
				Optional:    true,
				Description: "Minimum number of replicas",
			},
			"max_replicas": {
				Type:        schema.TypeInt,
				Optional:    true,
				Description: "Maximum number of replicas",
			},
			"target_cpu_utilization_percentage": {
				Type:        schema.TypeInt,
				Optional:    true,
				Description: "Target average CPU utilization (represented as a percentage of requested CPU) over all the pods.",
			},
		},
	}
}

func resourceRpaasAutoscaleCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	provider := meta.(*rpaasProvider)

	instance := d.Get("instance").(string)
	serviceName := d.Get("service_name").(string)

	rpaasClient, err := provider.RpaasClient.SetService(serviceName)
	if err != nil {
		return diag.Errorf("Unable to create client for service %s: %v", serviceName, err)
	}

	args := rpaas_client.UpdateAutoscaleArgs{
		Instance: instance,
	}

	if v, ok := d.GetOk("min_replicas"); ok {
		args.MinReplicas = pointerToInt32(int32(v.(int)))
	}

	if v, ok := d.GetOk("max_replicas"); ok {
		args.MaxReplicas = pointerToInt32(int32(v.(int)))
	}

	if v, ok := d.GetOk("target_cpu_utilization_percentage"); ok {
		args.CPU = pointerToInt32(int32(v.(int)))
	}

	err = rpaasRetry(ctx, d, func() error {
		return rpaasClient.UpdateAutoscale(ctx, args)
	})

	if err != nil {
		return diag.Errorf("Unable to create autoscale for instance %s: %v", instance, err)
	}

	d.SetId(fmt.Sprintf("%s/%s", serviceName, instance))
	return nil
}

func resourceRpaasAutoscaleRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	provider := meta.(*rpaasProvider)

	instance := d.Get("instance").(string)
	serviceName := d.Get("service_name").(string)

	rpaasClient, err := provider.RpaasClient.SetService(serviceName)
	if err != nil {
		return diag.Errorf("Unable to create client for service %s: %v", serviceName, err)
	}
	autoscale, err := rpaasClient.GetAutoscale(ctx, rpaas_client.GetAutoscaleArgs{Instance: instance})
	if err != nil {
		return diag.Errorf("Unable to get autoscale for %s: %v", instance, err)
	}

	if autoscale.MinReplicas != nil {
		d.Set("min_replicas", *autoscale.MinReplicas)
	}

	if autoscale.MaxReplicas != nil {
		d.Set("max_replicas", *autoscale.MaxReplicas)
	}

	if autoscale.CPU != nil {
		d.Set("target_cpu_utilization_percentage", *autoscale.CPU)
	}

	return nil
}

func resourceRpaasAutoscaleUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	provider := meta.(*rpaasProvider)

	instance := d.Get("instance").(string)
	serviceName := d.Get("service_name").(string)
	rpaasClient, err := provider.RpaasClient.SetService(serviceName)
	if err != nil {
		return diag.Errorf("Unable to create client for service %s: %v", serviceName, err)
	}

	args := rpaas_client.UpdateAutoscaleArgs{
		Instance: instance,
	}

	if v, ok := d.GetOk("min_replicas"); ok {
		args.MinReplicas = pointerToInt32(int32(v.(int)))
	}

	if v, ok := d.GetOk("max_replicas"); ok {
		args.MaxReplicas = pointerToInt32(int32(v.(int)))
	}

	if v, ok := d.GetOk("target_cpu_utilization_percentage"); ok {
		args.CPU = pointerToInt32(int32(v.(int)))
	}

	err = rpaasRetry(ctx, d, func() error {
		return rpaasClient.UpdateAutoscale(ctx, args)
	})
	if err != nil {
		return diag.Errorf("Unable to update autoscale for instance %s: %v", instance, err)
	}
	return nil
}

func resourceRpaasAutoscaleDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	provider := meta.(*rpaasProvider)

	instance := d.Get("instance").(string)
	serviceName := d.Get("service_name").(string)
	rpaasClient, err := provider.RpaasClient.SetService(serviceName)
	if err != nil {
		return diag.Errorf("Unable to create client for service %s: %v", serviceName, err)
	}

	err = rpaasRetry(ctx, d, func() error {
		return rpaasClient.RemoveAutoscale(ctx, rpaas_client.RemoveAutoscaleArgs{
			Instance: instance,
		})
	})

	if err != nil {
		return diag.Errorf("Unable to remove autoscale for instance %s: %v", instance, err)
	}
	return nil
}

func pointerToInt32(x int32) *int32 {
	return &x
}
