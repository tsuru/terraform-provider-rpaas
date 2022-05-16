// Copyright 2021 tsuru authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package rpaas

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/stretchr/testify/assert"
)

func TestAccRpaasACL_basic(t *testing.T) {
	testAPIClient, testAPIServer := setupTestAPIServer(t)
	defer testAPIServer.Stop()

	resourceName := "rpaas_acl.myacl"
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		IDRefreshName:     resourceName,
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      nil,
		Steps: []resource.TestStep{
			{
				Config: testAccRpaasACLConfig_basic("test-host.globoi.com", "80"),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccResourceExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "instance", "my-rpaas"),
					resource.TestCheckResourceAttr(resourceName, "service_name", "rpaasv2-be"),
					resource.TestCheckResourceAttr(resourceName, "host", "test-host.globoi.com"),
					resource.TestCheckResourceAttr(resourceName, "port", "80"),
					func(s *terraform.State) error {
						acls, err := testAPIClient.ListAccessControlList(context.Background(), "my-rpaas")
						assert.NoError(t, err)
						assert.Len(t, acls, 1)
						assert.Equal(t, "test-host.globoi.com", acls[0].Host)
						assert.Equal(t, 80, acls[0].Port)
						return nil
					},
				),
			},
			{
				// Testing Update
				Config: testAccRpaasACLConfig_basic("test-host.globoi.com", "333"),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccResourceExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "instance", "my-rpaas"),
					resource.TestCheckResourceAttr(resourceName, "service_name", "rpaasv2-be"),
					resource.TestCheckResourceAttr(resourceName, "host", "test-host.globoi.com"),
					resource.TestCheckResourceAttr(resourceName, "port", "333"),
				),
			},
		},
	})
}

func TestAccRpaasACL_import(t *testing.T) {
	testAPIClient, testAPIServer := setupTestAPIServer(t)
	defer testAPIServer.Stop()

	if err := testAPIClient.AddAccessControlList(context.Background(), "my-rpaas", "imported-host.globoi.com", 500); err != nil {
		t.Errorf("Api client failed to connect: %v", err)
	}

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      nil,
		Steps: []resource.TestStep{
			{
				// Testing Import
				Config:        `resource "rpaas_acl" "imported_acl" {}`,
				ResourceName:  "rpaas_acl.imported_acl",
				ImportStateId: "rpaasv2-be/my-rpaas imported-host.globoi.com:500",
				ImportState:   true,
				ImportStateCheck: func(s []*terraform.InstanceState) error {
					state := s[0]
					assert.Equal(t, "rpaasv2-be", state.Attributes["service_name"])
					assert.Equal(t, "my-rpaas", state.Attributes["instance"])
					assert.Equal(t, "imported-host.globoi.com", state.Attributes["host"])
					assert.Equal(t, "500", state.Attributes["port"])
					return nil
				},
			},
		},
	})
}

func testAccRpaasACLConfig_basic(host, port string) string {
	return fmt.Sprintf(`
resource "rpaas_acl" "myacl" {
	service_name = "rpaasv2-be"
	instance     = "my-rpaas"

	host = "%s"
	port = %s
}
`, host, port)
}
