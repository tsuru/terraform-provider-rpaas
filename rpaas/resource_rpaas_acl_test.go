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

func TestAccRpaasACL_basic(t *testing.T) {
	fakeServer := echo.New()
	fakeServer.POST("/services/rpaasv2-be/proxy/myacl", func(c echo.Context) error {
		p := types.AllowedUpstream{}
		err := c.Bind(&p)
		require.NoError(t, err)
		assert.Equal(t, "test-host.globoi.com", p.Host)
		assert.Equal(t, 80, p.Port)

		return c.JSON(http.StatusCreated, nil)
	})
	fakeServer.GET("/services/rpaasv2-be/proxy/myacl", func(c echo.Context) error {
		return c.JSON(http.StatusOK, []types.AllowedUpstream{
			{
				Host: "test-host.globoi.com",
				Port: 80,
			},
		})
	})
	fakeServer.DELETE("/services/rpaasv2-be/proxy/myacl", func(c echo.Context) error {
		p := types.AllowedUpstream{}
		err := c.Bind(&p)
		require.NoError(t, err)
		assert.Equal(t, "test-host.globoi.com", p.Host)
		assert.Equal(t, 80, p.Port)

		return c.NoContent(http.StatusNoContent)
	})
	fakeServer.HTTPErrorHandler = func(err error, c echo.Context) {
		t.Errorf("methods=%s, path=%s, err=%s", c.Request().Method, c.Path(), err.Error())
	}
	server := httptest.NewServer(fakeServer)
	os.Setenv("TSURU_TARGET", server.URL)
	os.Setenv("TSURU_TOKEN", "asdf")

	resourceName := "rpaas_acl.myacl"
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		IDRefreshName:     resourceName,
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      nil,
		Steps: []resource.TestStep{
			{
				Config: testAccRpaasACLConfig_basic("myacl"),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccResourceExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "instance", "myacl"),
					resource.TestCheckResourceAttr(resourceName, "service_name", "rpaasv2-be"),
				),
			},
		},
	})
}

func testAccRpaasACLConfig_basic(name string) string {
	return fmt.Sprintf(`
resource "rpaas_acl" "myacl" {
	service_name = "rpaasv2-be"
	instance     = "%s"

	host = "test-host.globoi.com"
	port = 80
}
`, name)
}
