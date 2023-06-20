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
	"github.com/stretchr/testify/require"

	"github.com/tsuru/rpaas-operator/pkg/rpaas/client"
	"github.com/tsuru/rpaas-operator/pkg/rpaas/client/types"
)

func TestAccRpaasAutoscale_basic(t *testing.T) {
	testAPIClient, testAPIServer := setupTestAPIServer(t)
	defer testAPIServer.Stop()

	resourceName := "rpaas_autoscale.be_autoscale"

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		IDRefreshName:     resourceName,
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      nil,
		Steps: []resource.TestStep{
			{
				Config: testAccRpaasRouterConfig(10),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccResourceExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "id", "rpaasv2-be::my-rpaas"),
					resource.TestCheckResourceAttr(resourceName, "instance", "my-rpaas"),
					resource.TestCheckResourceAttr(resourceName, "service_name", "rpaasv2-be"),
					resource.TestCheckResourceAttr(resourceName, "min_replicas", "10"),
					resource.TestCheckResourceAttr(resourceName, "max_replicas", "50"),
					resource.TestCheckResourceAttr(resourceName, "target_cpu_utilization_percentage", "60"),
					resource.TestCheckResourceAttr(resourceName, "target_requests_per_second", "100"),
					func(s *terraform.State) error {
						autoscale, err := testAPIClient.GetAutoscale(context.Background(), client.GetAutoscaleArgs{Instance: "my-rpaas"})
						if assert.NoError(t, err) {
							return err
						}

						assert.Equal(t, &types.Autoscale{
							MinReplicas: int32ToPointer(10),
							MaxReplicas: int32ToPointer(50),
							CPU:         int32ToPointer(60),
							RPS:         int32ToPointer(100),
						}, autoscale)
						return nil
					},
				),
			},
			{
				// Testing Update
				Config: testAccRpaasRouterConfig(1),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccResourceExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "id", "rpaasv2-be::my-rpaas"),
					resource.TestCheckResourceAttr(resourceName, "instance", "my-rpaas"),
					resource.TestCheckResourceAttr(resourceName, "service_name", "rpaasv2-be"),
					resource.TestCheckResourceAttr(resourceName, "min_replicas", "1"), //changed
					resource.TestCheckResourceAttr(resourceName, "max_replicas", "50"),
					resource.TestCheckResourceAttr(resourceName, "target_cpu_utilization_percentage", "60"),
					resource.TestCheckResourceAttr(resourceName, "target_requests_per_second", "100"),
					func(s *terraform.State) error {
						autoscale, err := testAPIClient.GetAutoscale(context.Background(), client.GetAutoscaleArgs{Instance: "my-rpaas"})
						if assert.NoError(t, err) {
							return err
						}

						assert.Equal(t, &types.Autoscale{
							MinReplicas: int32ToPointer(1),
							MaxReplicas: int32ToPointer(50),
							CPU:         int32ToPointer(60),
							RPS:         int32ToPointer(100),
						}, autoscale)
						return nil
					},
				),
			},
		},
	})
}

func TestAccRpaasAutoscale_import(t *testing.T) {
	testAPIClient, testAPIServer := setupTestAPIServer(t)
	defer testAPIServer.Stop()

	err := testAPIClient.UpdateAutoscale(context.Background(),
		client.UpdateAutoscaleArgs{
			Instance:    "my-rpaas",
			MinReplicas: int32ToPointer(1),
			MaxReplicas: int32ToPointer(5),
			CPU:         int32ToPointer(50),
			RPS:         int32ToPointer(500),
		},
	)
	require.NoError(t, err)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      nil,
		Steps: []resource.TestStep{
			{
				// Testing Import
				Config:        `resource "rpaas_autoscale" "imported" {}`,
				ResourceName:  "rpaas_autoscale.imported",
				ImportStateId: "rpaasv2-be::my-rpaas",
				ImportState:   true,
				ImportStateCheck: func(s []*terraform.InstanceState) error {
					state := s[0]
					assert.Equal(t, "rpaasv2-be", state.Attributes["service_name"])
					assert.Equal(t, "my-rpaas", state.Attributes["instance"])
					assert.Equal(t, "1", state.Attributes["min_replicas"])
					assert.Equal(t, "5", state.Attributes["max_replicas"])
					assert.Equal(t, "50", state.Attributes["target_cpu_utilization_percentage"])
					assert.Equal(t, "500", state.Attributes["target_requests_per_second"])
					return nil
				},
			},
			{
				// Testing Import legacy ID
				Config:        `resource "rpaas_autoscale" "imported" {}`,
				ResourceName:  "rpaas_autoscale.imported",
				ImportStateId: "rpaasv2-be/my-rpaas", //legacy id
				ImportState:   true,
				ImportStateCheck: func(s []*terraform.InstanceState) error {
					state := s[0]
					assert.Len(t, s, 1)
					assert.Equal(t, "rpaasv2-be::my-rpaas", state.Attributes["id"])
					return nil
				},
			},
		},
	})
}

func testAccRpaasRouterConfig(min_replicas int) string {
	return fmt.Sprintf(`
resource "rpaas_autoscale" "be_autoscale" {
	instance = "my-rpaas"
	service_name = "rpaasv2-be"

	min_replicas = %d
	max_replicas = 50

	target_cpu_utilization_percentage = 60
	target_requests_per_second        = 100
}
`, min_replicas)
}
