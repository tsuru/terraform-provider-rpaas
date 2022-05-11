// Copyright 2021 tsuru authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package rpaas

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	echo "github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/tsuru/rpaas-operator/pkg/rpaas/client/types"
)

func TestAccRpaasAutoscale_basic(t *testing.T) {
	setupFakeServerRpaasAutoscale(t)

	resourceName := "rpaas_autoscale.be_autoscale"

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		IDRefreshName:     resourceName,
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      nil,
		Steps: []resource.TestStep{
			{
				Config: testAccRpaasRouterConfig_basic("be_autoscale", 10),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccResourceExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "instance", "be_autoscale"),
					resource.TestCheckResourceAttr(resourceName, "service_name", "rpaasv2-be"),
					resource.TestCheckResourceAttr(resourceName, "min_replicas", "10"),
					resource.TestCheckResourceAttr(resourceName, "max_replicas", "50"),
					resource.TestCheckResourceAttr(resourceName, "target_cpu_utilization_percentage", "60"),
				),
			},
			{
				// Testing Import
				Config:        `resource "rpaas_autoscale" "imported" {}`,
				ResourceName:  "rpaas_autoscale.imported",
				ImportStateId: "rpaasv2-be/be_autoscale",
				ImportState:   true,
				ImportStateCheck: func(s []*terraform.InstanceState) error {
					state := s[0]
					assert.Equal(t, "rpaasv2-be", state.Attributes["service_name"])
					assert.Equal(t, "be_autoscale", state.Attributes["instance"])
					assert.Equal(t, "10", state.Attributes["min_replicas"])
					assert.Equal(t, "50", state.Attributes["max_replicas"])
					assert.Equal(t, "60", state.Attributes["target_cpu_utilization_percentage"])
					return nil
				},
			},
			{
				// Testing Update
				Config: testAccRpaasRouterConfig_basic("be_autoscale", 1),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccResourceExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "instance", "be_autoscale"),
					resource.TestCheckResourceAttr(resourceName, "service_name", "rpaasv2-be"),
					resource.TestCheckResourceAttr(resourceName, "min_replicas", "1"), //changed
					resource.TestCheckResourceAttr(resourceName, "max_replicas", "50"),
					resource.TestCheckResourceAttr(resourceName, "target_cpu_utilization_percentage", "60"),
				),
			},
		},
	})
}

func setupFakeServerRpaasAutoscale(t *testing.T) {
	fakeServer := echo.New()
	getCount := 0
	fakeServer.POST("/services/rpaasv2-be/proxy/be_autoscale", func(c echo.Context) error {
		p := types.Autoscale{}
		err := c.Bind(&p)
		require.NoError(t, err)
		assert.Equal(t, int32ToPointer(10), p.MinReplicas)
		assert.Equal(t, int32ToPointer(50), p.MaxReplicas)
		assert.Equal(t, int32ToPointer(60), p.CPU)
		assert.Nil(t, p.Memory)
		return c.JSON(http.StatusOK, nil)
	})
	fakeServer.PATCH("/services/rpaasv2-be/proxy/be_autoscale", func(c echo.Context) error {
		p := types.Autoscale{}
		err := c.Bind(&p)
		require.NoError(t, err)
		assert.Equal(t, int32ToPointer(1), p.MinReplicas)
		assert.Equal(t, int32ToPointer(50), p.MaxReplicas)
		assert.Equal(t, int32ToPointer(60), p.CPU)
		assert.Nil(t, p.Memory)
		return c.JSON(http.StatusOK, nil)
	})
	fakeServer.GET("/services/rpaasv2-be/proxy/be_autoscale", func(c echo.Context) error {
		getCount++

		if getCount == 1 {
			return c.JSON(http.StatusNotFound, nil)
		}

		if getCount > 6 {
			return c.JSON(http.StatusOK, &types.Autoscale{
				MinReplicas: int32ToPointer(1),
				MaxReplicas: int32ToPointer(50),
				CPU:         int32ToPointer(60),
			})
		}

		return c.JSON(http.StatusOK, &types.Autoscale{
			MinReplicas: int32ToPointer(10),
			MaxReplicas: int32ToPointer(50),
			CPU:         int32ToPointer(60),
		})
	})
	fakeServer.DELETE("/services/rpaasv2-be/proxy/be_autoscale", func(c echo.Context) error {
		return c.NoContent(http.StatusOK)
	})
	fakeServer.HTTPErrorHandler = func(err error, c echo.Context) {
		t.Errorf("methods=%s, path=%s, err=%s", c.Request().Method, c.Path(), err.Error())
	}
	server := httptest.NewServer(fakeServer)
	os.Setenv("TSURU_TARGET", server.URL)
	os.Setenv("TSURU_TOKEN", "asdf")
}

func testAccRpaasRouterConfig_basic(name string, min_replicas int) string {
	return fmt.Sprintf(`
resource "rpaas_autoscale" "be_autoscale" {
	instance = "%s"
	service_name = "rpaasv2-be"

	min_replicas = %d
	max_replicas = 50

	target_cpu_utilization_percentage = 60
}
`, name, min_replicas)
}
