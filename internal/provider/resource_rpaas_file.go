// Copyright 2021 tsuru authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package provider

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"strings"
	"unicode/utf8"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	rpaas_client "github.com/tsuru/rpaas-operator/pkg/rpaas/client"
	"github.com/tsuru/rpaas-operator/pkg/rpaas/client/types"
)

func resourceRpaasFile() *schema.Resource {
	return &schema.Resource{
		CreateContext: resourceRpaasFileCreate,
		ReadContext:   resourceRpaasFileRead,
		UpdateContext: resourceRpaasFileUpdate,
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
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "Name of a persistent file in the instance filesystem",
			},
			"content": {
				Type:         schema.TypeString,
				Optional:     true,
				ExactlyOneOf: []string{"content", "content_base64"},
				Description:  "Content of the persistent file in the instance filesystem, expected to be an UTF-8 encoded string.",
			},
			"content_base64": {
				Type:         schema.TypeString,
				Optional:     true,
				ExactlyOneOf: []string{"content", "content_base64"},
				Description:  "Content of the persistent file in the instance filesystem, expected to be binary encoded as base64 string. (v0.2.3)",
			},
		},
	}
}

func resourceRpaasFileCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	provider := meta.(*rpaasProvider)

	serviceName := d.Get("service_name").(string)
	instance := d.Get("instance").(string)
	filename := d.Get("name").(string)
	content, err := resourceRpaasFileContent(d)
	if err != nil {
		return diag.Errorf("Unable to read content: %v", err)
	}

	rpaasClient, err := provider.RpaasClient.SetService(serviceName)
	if err != nil {
		return diag.Errorf("Unable to create client for service %s: %v", serviceName, err)
	}

	tflog.Info(ctx, "Create file", map[string]interface{}{
		"service":  serviceName,
		"instance": instance,
		"name":     filename,
	})

	err = rpaasRetry(ctx, d.Timeout(schema.TimeoutCreate), func() (*http.Response, error) {
		return nil, rpaasClient.AddExtraFiles(ctx, rpaas_client.ExtraFilesArgs{
			Instance: instance,
			Files: []types.RpaasFile{
				{Name: filename, Content: []byte(content)},
			},
		})
	})

	if err != nil {
		return diag.Errorf("Unable to create file %q for instance %s: %v", filename, instance, err)
	}

	d.SetId(fmt.Sprintf("%s::%s::%s", serviceName, instance, filename))
	return resourceRpaasFileRead(ctx, d, meta)
}

func resourceRpaasFileUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	provider := meta.(*rpaasProvider)

	serviceName, instance, filename, err := parseRpaasFileID(d.Id())
	if err != nil {
		return diag.Errorf("Unable to parse File ID: %v", err)
	}

	content, err := resourceRpaasFileContent(d)
	if err != nil {
		return diag.Errorf("Unable to read content: %v", err)
	}

	rpaasClient, err := provider.RpaasClient.SetService(serviceName)
	if err != nil {
		return diag.Errorf("Unable to create client for service %s: %v", serviceName, err)
	}

	tflog.Info(ctx, "Update file", map[string]interface{}{
		"id":       d.Id(),
		"service":  serviceName,
		"instance": instance,
		"name":     filename,
	})

	err = rpaasRetry(ctx, d.Timeout(schema.TimeoutUpdate), func() (*http.Response, error) {
		return nil, rpaasClient.UpdateExtraFiles(ctx, rpaas_client.ExtraFilesArgs{
			Instance: instance,
			Files: []types.RpaasFile{
				{Name: filename, Content: []byte(content)},
			},
		})
	})

	if err != nil {
		return diag.Errorf("Unable to update file %q for instance %s: %v", filename, instance, err)
	}

	return resourceRpaasFileRead(ctx, d, meta)
}

func resourceRpaasFileRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	provider := meta.(*rpaasProvider)

	serviceName, instance, filename, err := parseRpaasFileID(d.Id())
	if err != nil {
		return diag.Errorf("Unable to parse File ID: %v", err)
	}
	d.SetId(fmt.Sprintf("%s::%s::%s", serviceName, instance, filename))

	rpaasClient, err := provider.RpaasClient.SetService(serviceName)
	if err != nil {
		return diag.Errorf("Unable to create client for service %s: %v", serviceName, err)
	}

	var rpaasFile types.RpaasFile

	err = rpaasRetry(ctx, d.Timeout(schema.TimeoutRead), func() (*http.Response, error) {
		f, nerr := rpaasClient.GetExtraFile(ctx, rpaas_client.GetExtraFileArgs{
			Instance: instance,
			FileName: filename,
		})
		if nerr != nil {
			return nil, err
		}

		rpaasFile = f
		return nil, nil
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
	setResourceRpaasFileContent(d, rpaasFile.Content)
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

	tflog.Info(ctx, "Delete file", map[string]interface{}{
		"id":       d.Id(),
		"service":  serviceName,
		"instance": instance,
		"name":     filename,
	})

	err = rpaasRetry(ctx, d.Timeout(schema.TimeoutDelete), func() (*http.Response, error) {
		return nil, rpaasClient.DeleteExtraFiles(ctx,
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

func resourceRpaasFileContent(d *schema.ResourceData) ([]byte, error) {
	if contentBase64, ok := d.GetOk("content_base64"); ok {
		return base64.StdEncoding.DecodeString(contentBase64.(string))
	}

	return []byte(d.Get("content").(string)), nil
}

func setResourceRpaasFileContent(d *schema.ResourceData, content []byte) {
	if utf8.Valid(content) {
		if _, ok := d.GetOk("content_base64"); !ok {
			d.Set("content", string(content))
			d.Set("content_base64", nil)
			return
		}
	}

	d.Set("content", nil)
	d.Set("content_base64", base64.StdEncoding.EncodeToString(content))
}

func parseRpaasFileID(id string) (serviceName, instance, filename string, err error) {
	splitID := strings.Split(id, "::")
	if len(splitID) != 3 {
		serviceName, instance, filename, err = parseRpaasFileID_legacyV0(id)
		if err != nil {
			err = fmt.Errorf("Could not parse id %q. Format should be \"service::instance::file\"", id)
		}
		return
	}
	serviceName = splitID[0]
	instance = splitID[1]
	filename = splitID[2]
	return
}

func parseRpaasFileID_legacyV0(id string) (serviceName, instance, filename string, err error) {
	splitID := strings.Split(id, "/")
	if len(splitID) != 3 {
		err = fmt.Errorf("Resource ID could not be parsed. Legacy format: \"service/instance/file\"")
		return
	}
	serviceName = splitID[0]
	instance = splitID[1]
	filename = splitID[2]
	return
}
