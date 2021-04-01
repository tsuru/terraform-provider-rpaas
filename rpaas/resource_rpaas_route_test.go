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

func TestAccRpaasRoute_basic(t *testing.T) {
	fakeServer := echo.New()
	fakeServer.POST("/services/rpaasv2-be/proxy/my_rpaas", func(c echo.Context) error {
		p := struct {
			Path, Destination, HTTPS_only, Content string
		}{}
		err := c.Bind(&p)
		require.NoError(t, err)
		assert.Equal(t, "/", p.Path)
		assert.Equal(t, "", p.HTTPS_only)
		assert.Equal(t, "	# nginx config\n", p.Content)
		return c.JSON(http.StatusCreated, nil)
	})
	fakeServer.GET("/services/rpaasv2-be/proxy/my_rpaas", func(c echo.Context) error {
		return c.JSON(http.StatusOK, struct {
			Routes []types.Route `json:"routes"`
		}{
			Routes: []types.Route{
				{Path: "/", Content: "	# nginx config\n"},
			},
		})
	})
	fakeServer.DELETE("/services/rpaasv2-be/proxy/my_rpaas", func(c echo.Context) error {
		return c.NoContent(http.StatusOK)
	})
	fakeServer.HTTPErrorHandler = func(err error, c echo.Context) {
		t.Errorf("methods=%s, path=%s, err=%s", c.Request().Method, c.Path(), err.Error())
	}
	server := httptest.NewServer(fakeServer)
	os.Setenv("TSURU_TARGET", server.URL)
	os.Setenv("TSURU_TOKEN", "asdf")

	resourceName := "rpaas_route.custom_route"
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		IDRefreshName:     resourceName,
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      nil,
		Steps: []resource.TestStep{
			{
				Config: testAccRpaasRouteConfig_basic(server.URL, "my_rpaas"),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccResourceExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "instance", "my_rpaas"),
					resource.TestCheckResourceAttr(resourceName, "service_name", "rpaasv2-be"),
					resource.TestCheckResourceAttr(resourceName, "path", "/"),
				),
			},
		},
	})
}

func testAccRpaasRouteConfig_basic(fakeServer, name string) string {
	return fmt.Sprintf(`
resource "rpaas_route" "custom_route" {
	instance = "%s"
	service_name = "rpaasv2-be"

	path = "/"

	content = <<EOF
	# nginx config
	EOF
}
`, name)
}
