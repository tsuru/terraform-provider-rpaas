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
)

func TestAccRpaasBlock_basic(t *testing.T) {
	testAPIClient, testAPIServer := setupTestAPIServer(t)
	defer testAPIServer.Stop()

	resourceName := "rpaas_block.custom_block_server"
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		IDRefreshName:     resourceName,
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccRpaasBlockConfig("server", "# nginx config"),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccResourceExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "id", "rpaasv2-be::my-rpaas::server"),
					resource.TestCheckResourceAttr(resourceName, "instance", "my-rpaas"),
					resource.TestCheckResourceAttr(resourceName, "service_name", "rpaasv2-be"),
					resource.TestCheckResourceAttr(resourceName, "name", "server"),
					resource.TestCheckResourceAttr(resourceName, "content", "# nginx config\n"),
					func(s *terraform.State) error {
						blocks, err := testAPIClient.ListBlocks(context.Background(), client.ListBlocksArgs{Instance: "my-rpaas"})
						assert.NoError(t, err)
						assert.Len(t, blocks, 1)
						assert.Equal(t, "server", blocks[0].Name)
						assert.Equal(t, "# nginx config\n", blocks[0].Content)
						return nil
					},
				),
			},
			{
				// Testing Update - block content
				Config: testAccRpaasBlockConfig("server", "# a different content"),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccResourceExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "id", "rpaasv2-be::my-rpaas::server"),
					resource.TestCheckResourceAttr(resourceName, "instance", "my-rpaas"),
					resource.TestCheckResourceAttr(resourceName, "service_name", "rpaasv2-be"),
					resource.TestCheckResourceAttr(resourceName, "name", "server"),
					resource.TestCheckResourceAttr(resourceName, "content", "# a different content\n"),
					func(s *terraform.State) error {
						blocks, err := testAPIClient.ListBlocks(context.Background(), client.ListBlocksArgs{Instance: "my-rpaas"})
						assert.NoError(t, err)
						assert.Len(t, blocks, 1)
						assert.Equal(t, "server", blocks[0].Name)
						assert.Equal(t, "# a different content\n", blocks[0].Content)
						return nil
					},
				),
			},
			{
				// Testing Update - block name
				Config: testAccRpaasBlockConfig("http", "# a different content"),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccResourceExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "id", "rpaasv2-be::my-rpaas::http"),
					resource.TestCheckResourceAttr(resourceName, "instance", "my-rpaas"),
					resource.TestCheckResourceAttr(resourceName, "service_name", "rpaasv2-be"),
					resource.TestCheckResourceAttr(resourceName, "name", "http"),
					resource.TestCheckResourceAttr(resourceName, "content", "# a different content\n"),
					func(s *terraform.State) error {
						blocks, err := testAPIClient.ListBlocks(context.Background(), client.ListBlocksArgs{Instance: "my-rpaas"})
						assert.NoError(t, err)
						assert.Len(t, blocks, 1)
						assert.Equal(t, "http", blocks[0].Name)
						assert.Equal(t, "# a different content\n", blocks[0].Content)
						return nil
					},
				),
			},
		},
	})
}

func TestAccRpaasBlock_import(t *testing.T) {
	testAPIClient, testAPIServer := setupTestAPIServer(t)
	defer testAPIServer.Stop()

	if err := testAPIClient.UpdateBlock(context.Background(),
		client.UpdateBlockArgs{Instance: "my-rpaas", Name: "lua-worker", Content: "imported"},
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
				Config:        `resource "rpaas_block" "imported" {}`,
				ResourceName:  "rpaas_block.imported",
				ImportStateId: "rpaasv2-be::my-rpaas::lua-worker",
				ImportState:   true,
				ImportStateCheck: func(s []*terraform.InstanceState) error {
					state := s[0]
					assert.Equal(t, "rpaasv2-be", state.Attributes["service_name"])
					assert.Equal(t, "my-rpaas", state.Attributes["instance"])
					assert.Equal(t, "lua-worker", state.Attributes["name"])
					assert.Equal(t, "imported", state.Attributes["content"])
					return nil
				},
			},
			{
				// Testing Import legacy ID
				Config:        `resource "rpaas_block" "imported" {}`,
				ResourceName:  "rpaas_block.imported",
				ImportStateId: "rpaasv2-be/my-rpaas", //legacy id
				ImportState:   true,
				ImportStateCheck: func(s []*terraform.InstanceState) error {
					state := s[0]
					assert.Len(t, s, 1)
					assert.Equal(t, "rpaasv2-be::my-rpaas::lua-worker", state.Attributes["id"])
					return nil
				},
			},
		},
	})
}

func testAccRpaasBlockConfig(block, content string) string {
	return fmt.Sprintf(`
resource "rpaas_block" "custom_block_server" {
	instance = "my-rpaas"
	service_name = "rpaasv2-be"

	name = "%s"

	content = <<-EOF
	%s
	EOF
}
`, block, content)
}
