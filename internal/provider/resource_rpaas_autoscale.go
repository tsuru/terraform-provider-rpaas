// Copyright 2021 tsuru authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package provider

import (
	"context"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/tsuru/rpaas-operator/pkg/rpaas/client/autogenerated"
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
				Required:    true,
				Description: "Minimum number of replicas",
			},
			"max_replicas": {
				Type:        schema.TypeInt,
				Required:    true,
				Description: "Maximum number of replicas",
			},
			"target_cpu_utilization_percentage": {
				Type:        schema.TypeInt,
				Optional:    true,
				Description: "Target average CPU utilization (represented as a percentage of requested CPU) over all the pods.",
			},
			"target_requests_per_second": {
				Type:        schema.TypeInt,
				Optional:    true,
				Description: "Target average of HTTP requests per second over the serving pods",
			},
			"scheduled_window": {
				Type: schema.TypeList,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"min_replicas": {
							Type:        schema.TypeInt,
							Required:    true,
							Description: "Min number of running pods while this window is active. It cannot be greater than `max_replicas`.",
						},
						"start": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "An Cron expression defining the start of the scheduled window. Example: `00 20 * * * 1-5`.",
						},
						"end": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "An Cron expression defining the end of the scheduled window. Example: `00 00 * * * 1-5`.",
						},
					},
				},
				Optional:    true,
				Description: "Scheduled windows are recurring (or not) time windows where the instance can scale in/out your min replicas regardless of traffic or resource utilization.",
			},
		},
	}
}

func resourceRpaasAutoscaleCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	var instance string
	if v, ok := d.GetOk("instance"); ok {
		instance = v.(string)
	}

	var service string
	if v, ok := d.GetOk("service_name"); ok {
		service = v.(string)
	}

	autoscale := extractAutoscaleFromState(d)

	provider, ok := meta.(*rpaasProvider)
	if !ok {
		return diag.Errorf("could not type assert meta as RPaaS provider")
	}

	err := rpaasRetry(ctx, d.Timeout(schema.TimeoutCreate), func() (*http.Response, error) {
		return provider.Client(service, instance).RpaasApi.UpdateAutoscale(ctx, instance).Autoscale(autoscale).Execute()
	})

	if err != nil {
		return diag.Errorf("could not update the autoscale config on RPaaS: %s", err)
	}

	d.SetId(fmt.Sprintf("%s::%s", service, instance))
	return nil
}

func resourceRpaasAutoscaleRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	service, instance, err := parseRpaasInstanceID(d.Id())
	if err != nil {
		return diag.Errorf("could not read autoscale ID: %s", err)
	}

	d.Set("service_name", service)
	d.Set("instance", instance)

	d.SetId(fmt.Sprintf("%s::%s", service, instance)) // ensure the new ID format

	provider, ok := meta.(*rpaasProvider)
	if !ok {
		return diag.Errorf("could not type assert meta as RPaaS provider")
	}

	var autoscale *autogenerated.Autoscale

	err = rpaasRetry(ctx, d.Timeout(schema.TimeoutRead), func() (*http.Response, error) {
		a, response, nerr := provider.Client(service, instance).RpaasApi.GetAutoscale(ctx, instance).Execute()
		if nerr != nil {
			return response, nerr
		}

		autoscale = a
		return nil, nil
	})

	if err != nil {
		return diag.Errorf("could not get autoscale params from RPaaS API: %s", err)
	}

	if autoscale == nil {
		d.SetId("")
		return nil
	}

	d.Set("min_replicas", autoscale.MinReplicas)
	d.Set("max_replicas", autoscale.MaxReplicas)

	if cpu := autoscale.Cpu; cpu != nil {
		d.Set("target_cpu_utilization_percentage", *cpu)
	}

	if rps := autoscale.Rps; rps != nil {
		d.Set("target_requests_per_second", *rps)
	}

	var sws []any
	for _, sw := range autoscale.Schedules {
		sws = append(sws, map[string]any{
			"min_replicas": int(sw.MinReplicas),
			"start":        sw.Start,
			"end":          sw.End,
		})
	}

	d.Set("scheduled_window", sws)

	return nil
}

func resourceRpaasAutoscaleUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	service, instance, err := parseRpaasInstanceID(d.Id())
	if err != nil {
		return diag.Errorf("could not read autoscale ID: %s", err)
	}

	provider, ok := meta.(*rpaasProvider)
	if !ok {
		return diag.Errorf("could not type assert meta as RPaaS provider")
	}

	autoscale := extractAutoscaleFromState(d)

	err = rpaasRetry(ctx, d.Timeout(schema.TimeoutUpdate), func() (*http.Response, error) {
		return provider.Client(service, instance).RpaasApi.UpdateAutoscale(ctx, instance).Autoscale(autoscale).Execute()
	})

	if err != nil {
		return diag.Errorf("could not update the autoscale config on RPaaS API: %s", err)
	}

	return nil
}

func resourceRpaasAutoscaleDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	service, instance, err := parseRpaasInstanceID(d.Id())
	if err != nil {
		return diag.Errorf("could not read autoscale ID: %s", err)
	}

	provider, ok := meta.(*rpaasProvider)
	if !ok {
		return diag.Errorf("could not type assert meta as RPaaS provider")
	}

	err = rpaasRetry(ctx, d.Timeout(schema.TimeoutDelete), func() (*http.Response, error) {
		return provider.Client(service, instance).RpaasApi.RemoveAutoscale(ctx, instance).Execute()
	})

	if err != nil {
		return diag.Errorf("could not remove the autoscale config from RPaaS API: %s", err)
	}

	d.SetId("")
	return nil
}

func extractAutoscaleFromState(d *schema.ResourceData) (a autogenerated.Autoscale) {
	if v, ok := d.GetOk("min_replicas"); ok {
		a.MinReplicas = int32(v.(int))
	}

	if v, ok := d.GetOk("max_replicas"); ok {
		a.MaxReplicas = int32(v.(int))
	}

	if v, ok := d.GetOk("target_cpu_utilization_percentage"); ok {
		a.Cpu = autogenerated.PtrInt32(int32(v.(int)))
	}

	if v, ok := d.GetOk("target_requests_per_second"); ok {
		a.Rps = autogenerated.PtrInt32(int32(v.(int)))
	}

	if v, ok := d.GetOk("scheduled_window"); ok {
		a.Schedules = asScheduledWindows(v.([]any))
	}

	return
}

func asScheduledWindows(v []any) (sws []autogenerated.ScheduledWindow) {
	for _, sw := range v {
		s := sw.(map[string]any)

		var minReplicas int32
		if n, found := s["min_replicas"]; found {
			minReplicas = int32(n.(int))
		}

		var start string
		if s, found := s["start"]; found {
			start = s.(string)
		}

		var end string
		if s, found := s["end"]; found {
			end = s.(string)
		}

		sws = append(sws, autogenerated.ScheduledWindow{
			MinReplicas: minReplicas,
			Start:       start,
			End:         end,
		})
	}

	return
}
