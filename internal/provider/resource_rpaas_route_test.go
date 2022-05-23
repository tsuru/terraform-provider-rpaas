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

func TestAccRpaasRoute_basic(t *testing.T) {
	testAPIClient, testAPIServer := setupTestAPIServer(t)
	defer testAPIServer.Stop()

	resourceName := "rpaas_route.custom_route"
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		IDRefreshName:     resourceName,
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      nil,
		Steps: []resource.TestStep{
			{
				Config: testAccRpaasRouteConfig("/", "original content"),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccResourceExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "id", "rpaasv2-be::my-rpaas::/"),
					resource.TestCheckResourceAttr(resourceName, "instance", "my-rpaas"),
					resource.TestCheckResourceAttr(resourceName, "service_name", "rpaasv2-be"),
					resource.TestCheckResourceAttr(resourceName, "path", "/"),
					resource.TestCheckResourceAttr(resourceName, "https_only", "false"),
					resource.TestCheckResourceAttr(resourceName, "content", "original content\n"),
					resource.TestCheckResourceAttr(resourceName, "destination", ""),
					func(s *terraform.State) error {
						routes, err := testAPIClient.ListRoutes(context.Background(), client.ListRoutesArgs{Instance: "my-rpaas"})
						assert.NoError(t, err)
						assert.Len(t, routes, 1)
						assert.Equal(t, "/", routes[0].Path)
						assert.Equal(t, false, routes[0].HTTPSOnly)
						assert.Equal(t, "original content\n", routes[0].Content)
						assert.Equal(t, "", routes[0].Destination)
						return nil
					},
				),
			},
			{
				Config: testAccRpaasRouteConfig("/", "change content"),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccResourceExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "id", "rpaasv2-be::my-rpaas::/"),
					resource.TestCheckResourceAttr(resourceName, "instance", "my-rpaas"),
					resource.TestCheckResourceAttr(resourceName, "service_name", "rpaasv2-be"),
					resource.TestCheckResourceAttr(resourceName, "path", "/"),
					resource.TestCheckResourceAttr(resourceName, "https_only", "false"),
					resource.TestCheckResourceAttr(resourceName, "content", "change content\n"),
					resource.TestCheckResourceAttr(resourceName, "destination", ""),
					func(s *terraform.State) error {
						routes, err := testAPIClient.ListRoutes(context.Background(), client.ListRoutesArgs{Instance: "my-rpaas"})
						assert.NoError(t, err)
						assert.Len(t, routes, 1)
						assert.Equal(t, "/", routes[0].Path)
						assert.Equal(t, false, routes[0].HTTPSOnly)
						assert.Equal(t, "change content\n", routes[0].Content)
						assert.Equal(t, "", routes[0].Destination)
						return nil
					},
				),
			},
			{
				Config: testAccRpaasRouteConfig("/another/path", "change content"),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccResourceExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "id", "rpaasv2-be::my-rpaas::/another/path"),
					resource.TestCheckResourceAttr(resourceName, "instance", "my-rpaas"),
					resource.TestCheckResourceAttr(resourceName, "service_name", "rpaasv2-be"),
					resource.TestCheckResourceAttr(resourceName, "path", "/another/path"),
					resource.TestCheckResourceAttr(resourceName, "https_only", "false"),
					resource.TestCheckResourceAttr(resourceName, "content", "change content\n"),
					resource.TestCheckResourceAttr(resourceName, "destination", ""),
					func(s *terraform.State) error {
						routes, err := testAPIClient.ListRoutes(context.Background(), client.ListRoutesArgs{Instance: "my-rpaas"})
						assert.NoError(t, err)
						assert.Len(t, routes, 1)
						assert.Equal(t, "/another/path", routes[0].Path)
						assert.Equal(t, false, routes[0].HTTPSOnly)
						assert.Equal(t, "change content\n", routes[0].Content)
						assert.Equal(t, "", routes[0].Destination)
						return nil
					},
				),
			},
		},
	})
}

func TestAccRpaasRoute_import(t *testing.T) {
	testAPIClient, testAPIServer := setupTestAPIServer(t)
	defer testAPIServer.Stop()

	if err := testAPIClient.UpdateRoute(context.Background(),
		client.UpdateRouteArgs{Instance: "my-rpaas", Path: "/path/1", Destination: "http://infinity-and-beyond:5555", HTTPSOnly: true},
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
				Config:        `resource "rpaas_route" "imported" {}`,
				ResourceName:  "rpaas_route.imported",
				ImportStateId: "rpaasv2-be::my-rpaas::/path/1",
				ImportState:   true,
				ImportStateCheck: func(s []*terraform.InstanceState) error {
					state := s[0]
					assert.Equal(t, "rpaasv2-be", state.Attributes["service_name"])
					assert.Equal(t, "my-rpaas", state.Attributes["instance"])
					assert.Equal(t, "/path/1", state.Attributes["path"])
					assert.Equal(t, "true", state.Attributes["https_only"])
					assert.Equal(t, "http://infinity-and-beyond:5555", state.Attributes["destination"])
					return nil
				},
			},
			{
				// Testing Import legacy ID
				Config:        `resource "rpaas_route" "imported_legacy" {}`,
				ResourceName:  "rpaas_route.imported_legacy",
				ImportStateId: "rpaasv2-be/my-rpaas", // legacy id
				ImportState:   true,
				ImportStateCheck: func(s []*terraform.InstanceState) error {
					state := s[0]
					assert.Len(t, s, 1)
					assert.Equal(t, "rpaasv2-be::my-rpaas::/path/1", state.Attributes["id"])
					return nil
				},
			},
		},
	})
}

func testAccRpaasRouteConfig(path, content string) string {
	return fmt.Sprintf(`
resource "rpaas_route" "custom_route" {
	instance = "my-rpaas"
	service_name = "rpaasv2-be"

	path = "%s"

	content = <<-EOF
		%s
	EOF
}
`, path, content)
}
