// Copyright 2021 tsuru authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package provider

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"

	"github.com/stretchr/testify/assert"

	"github.com/tsuru/rpaas-operator/pkg/rpaas/client"
	"github.com/tsuru/rpaas-operator/pkg/rpaas/client/types"
)

func TestAccRpaasFile_basic(t *testing.T) {
	testAPIClient, testAPIServer := setupTestAPIServer(t)
	defer testAPIServer.Stop()

	resourceName := "rpaas_file.custom_file"
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		IDRefreshName:     resourceName,
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      nil,
		Steps: []resource.TestStep{
			{
				// Testing Create
				Config: testAccRpaasFileConfig("custom_file.txt", "content"),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccResourceExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "instance", "my-rpaas"),
					resource.TestCheckResourceAttr(resourceName, "service_name", "rpaasv2-be"),
					resource.TestCheckResourceAttr(resourceName, "name", "custom_file.txt"),
					resource.TestCheckResourceAttr(resourceName, "content", "content\n"),
					func(s *terraform.State) error {
						extraFiles, err := testAPIClient.ListExtraFiles(context.Background(),
							client.ListExtraFilesArgs{Instance: "my-rpaas", ShowContent: true},
						)
						assert.NoError(t, err)
						assert.Len(t, extraFiles, 1)
						assert.Equal(t, "custom_file.txt", extraFiles[0].Name)
						assert.EqualValues(t, "content\n", extraFiles[0].Content)
						return nil
					},
				),
			},
			{
				// Testing Update - content
				Config: testAccRpaasFileConfig("custom_file.txt", "changed"),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccResourceExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "instance", "my-rpaas"),
					resource.TestCheckResourceAttr(resourceName, "service_name", "rpaasv2-be"),
					resource.TestCheckResourceAttr(resourceName, "name", "custom_file.txt"),
					resource.TestCheckResourceAttr(resourceName, "content", "changed\n"),
					func(s *terraform.State) error {
						extraFiles, err := testAPIClient.ListExtraFiles(context.Background(),
							client.ListExtraFilesArgs{Instance: "my-rpaas", ShowContent: true},
						)
						assert.NoError(t, err)
						assert.Len(t, extraFiles, 1)
						assert.Equal(t, "custom_file.txt", extraFiles[0].Name)
						assert.EqualValues(t, "changed\n", extraFiles[0].Content)
						return nil
					},
				),
			},
			{
				// Testing Update - file name
				Config: testAccRpaasFileConfig("different.txt", "changed"),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccResourceExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "instance", "my-rpaas"),
					resource.TestCheckResourceAttr(resourceName, "service_name", "rpaasv2-be"),
					resource.TestCheckResourceAttr(resourceName, "name", "different.txt"),
					resource.TestCheckResourceAttr(resourceName, "content", "changed\n"),
					func(s *terraform.State) error {
						extraFiles, err := testAPIClient.ListExtraFiles(context.Background(),
							client.ListExtraFilesArgs{Instance: "my-rpaas", ShowContent: true},
						)
						assert.NoError(t, err)
						assert.Len(t, extraFiles, 1)
						assert.Equal(t, "different.txt", extraFiles[0].Name)
						assert.EqualValues(t, "changed\n", extraFiles[0].Content)
						return nil
					},
				),
			},
		},
	})
}

func TestAccRpaasFile_import(t *testing.T) {
	testAPIClient, testAPIServer := setupTestAPIServer(t)
	defer testAPIServer.Stop()

	if err := testAPIClient.AddExtraFiles(context.Background(),
		client.ExtraFilesArgs{
			Instance: "my-rpaas",
			Files:    []types.RpaasFile{{Name: "imported.txt", Content: []byte("imported content")}},
		},
	); err != nil {
		t.Errorf("Api client failed to connect: %v", err)
	}

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      nil,
		Steps: []resource.TestStep{
			{
				// Testing Import
				Config:        `resource "rpaas_file" "imported" {}`,
				ResourceName:  "rpaas_file.imported",
				ImportStateId: "rpaasv2-be/my-rpaas/imported.txt",
				ImportState:   true,
				ImportStateCheck: func(s []*terraform.InstanceState) error {
					state := s[0]
					assert.Equal(t, "rpaasv2-be", state.Attributes["service_name"])
					assert.Equal(t, "my-rpaas", state.Attributes["instance"])
					assert.Equal(t, "imported.txt", state.Attributes["name"])
					assert.Equal(t, "imported content", state.Attributes["content"])
					return nil
				},
			},
		},
	})
}

func testAccRpaasFileConfig(filename, content string) string {
	return fmt.Sprintf(`
resource "rpaas_file" "custom_file" {
	instance     = "my-rpaas"
	service_name = "rpaasv2-be"
	name         = "%s"
	content      = <<-EOF
		%s
	EOF
}
`, filename, content)
}
