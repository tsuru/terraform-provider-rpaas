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
	echo "github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/tsuru/rpaas-operator/pkg/rpaas/client/types"
)

func TestAccRpaasAutoscale_basic(t *testing.T) {
	fakeServer := echo.New()
	getCount := 0
	fakeServer.POST("/services/rpaasv2-be/proxy/be_autoscale", func(c echo.Context) error {
		p := struct {
			Min, Max, Cpu, Memory *int32
		}{}
		err := c.Bind(&p)
		require.NoError(t, err)
		assert.Equal(t, int32(10), *p.Min)
		assert.Equal(t, int32(50), *p.Max)
		assert.Equal(t, int32(60), *p.Cpu)
		assert.Nil(t, p.Memory)
		return c.JSON(http.StatusOK, nil)
	})
	fakeServer.GET("/services/rpaasv2-be/proxy/be_autoscale", func(c echo.Context) error {
		if getCount == 0 {
			getCount++
			return c.JSON(http.StatusNotFound, nil)
		}
		return c.JSON(http.StatusOK, &types.Autoscale{
			MinReplicas: pointerToInt32(10),
			MaxReplicas: pointerToInt32(50),
			CPU:         pointerToInt32(60),
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

	resourceName := "rpaas_autoscale.be_autoscale"
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		IDRefreshName:     resourceName,
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      nil,
		Steps: []resource.TestStep{
			{
				Config: testAccRpaasRouterConfig_basic(server.URL, "be_autoscale"),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccResourceExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "instance", "be_autoscale"),
					resource.TestCheckResourceAttr(resourceName, "service_name", "rpaasv2-be"),
				),
			},
		},
	})
}

func testAccRpaasRouterConfig_basic(fakeServer, name string) string {
	return fmt.Sprintf(`
resource "rpaas_autoscale" "be_autoscale" {
	instance = "%s"
	service_name = "rpaasv2-be"

	min_replicas = 10
	max_replicas = 50

	target_cpu_utilization_percentage = 60
}
`, name)
}
