// Copyright 2021 tsuru authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package rpaas

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	rpaas_client "github.com/tsuru/rpaas-operator/pkg/rpaas/client"
	"github.com/tsuru/rpaas-operator/pkg/rpaas/client/types"
)

func resourceRpaasFile() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceRpaasFileCreate,
		ReadContext:   resourceRpaasFileRead,
		UpdateContext: resourceRpaasFileCreate,
		DeleteContext: resourceRpaasFileDelete,
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
				Type:             schema.TypeString,
				Required:         true,
				Description:      "Name of a persistent file in the instance filesystem",
				ValidateDiagFunc: validateResourceRpaasFileName,
			},
			"content": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Content of the persistent file in the instance filesystem",
			},
		},
	}
}

func resourceRpaasFileCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	provider := meta.(*rpaasProvider)

	serviceName := d.Get("service_name").(string)
	instance := d.Get("instance").(string)
	filename := d.Get("name").(string)
	content := d.Get("content").(string)

	rpaasClient, err := provider.RpaasClient.SetService(serviceName)
	if err != nil {
		return diag.Errorf("Unable to create client for service %s: %v", serviceName, err)
	}

	rpaasFiles := []types.RpaasFile{
		{
			Name:    filename,
			Content: []byte(content),
		},
	}
	extraFileArgs := rpaas_client.ExtraFilesArgs{
		Instance: instance,
		Files:    rpaasFiles,
	}

	err = rpaasRetry(ctx, d, func() error {
		return rpaasClient.AddExtraFiles(ctx,
			extraFileArgs,
		)
	})

	if err != nil {
		return diag.Errorf("Unable to create file %q for instance %s: %v", filename, instance, err)
	}

	d.SetId(fmt.Sprintf("%s/%s/%s", serviceName, instance, filename))
	return nil
}

func resourceRpaasFileRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	provider := meta.(*rpaasProvider)

	splitID := strings.Split(d.Id(), "/")
	if len(splitID) != 3 {
		return diag.Errorf("Resource ID could not be parsed. Format should be \"service/instance/file\"")
	}
	serviceName := splitID[0]
	instance := splitID[1]
	filename := splitID[2]

	rpaasClient, err := provider.RpaasClient.SetService(serviceName)
	if err != nil {
		return diag.Errorf("Unable to create client for service %s: %v", serviceName, err)
	}

	rpaasFile, err := rpaasClient.GetExtraFile(ctx, rpaas_client.GetExtraFileArgs{
		Instance: instance,
		FileName: filename,
	})

	if rpaas_client.IsNotFoundError(err) {
		d.SetId("")
		return nil
	}
	if err != nil {
		return diag.Errorf("Error getting file %q from %s/%s: %v", filename, serviceName, instance, err)
	}

	d.Set("service_name", serviceName)
	d.Set("instance", instance)
	d.Set("name", filename)
	d.Set("content", string(rpaasFile.Content))
	return nil
}

func resourceRpaasFileDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	provider := meta.(*rpaasProvider)

	instance := d.Get("instance").(string)
	serviceName := d.Get("service_name").(string)
	filename := d.Get("name").(string)

	rpaasClient, err := provider.RpaasClient.SetService(serviceName)
	if err != nil {
		return diag.Errorf("Unable to create client for service %s: %v", serviceName, err)
	}

	err = rpaasRetry(ctx, d, func() error {
		return rpaasClient.DeleteExtraFiles(ctx,
			rpaas_client.DeleteExtraFilesArgs{
				Instance: instance,
				Files:    []string{filename},
			},
		)
	})

	if err != nil {
		return diag.Errorf("Unable to remove file %q for instance %s: %v", filename, instance, err)
	}

	return nil
}

func validateResourceRpaasFileName(v interface{}, p cty.Path) diag.Diagnostics {
	value := v.(string)

	if len(value) < 1 {
		return diag.Diagnostics{{
			Severity: diag.Error,
			Summary:  "Invalid filename",
			Detail:   "Filename cannot be empty string",
		}}
	}

	re := regexp.MustCompile(`[^\w._-]`)
	invalidFilename := re.MatchString(value)
	if invalidFilename {
		return diag.Diagnostics{{
			Severity: diag.Error,
			Summary:  "Invalid filename",
			Detail:   fmt.Sprintf("%q contains invalid characters.", value),
		}}
	}

	return nil
}
