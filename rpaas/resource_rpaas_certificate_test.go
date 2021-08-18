// Copyright 2021 tsuru authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package rpaas

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"

	echo "github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	rpaas_client "github.com/tsuru/rpaas-operator/pkg/rpaas/client"
	"github.com/tsuru/rpaas-operator/pkg/rpaas/client/types"
)

func TestAccRpaasCertificate_basic(t *testing.T) {
	fakeServer := echo.New()
	fakeServer.POST("/services/rpaasv2-be/proxy/my_rpaas", func(c echo.Context) error {
		cert, err := readMultiPartFile(c, "cert")
		require.NoError(t, err)
		key, err := readMultiPartFile(c, "key")
		require.NoError(t, err)

		p := rpaas_client.UpdateCertificateArgs{}
		err = c.Bind(&p)
		require.NoError(t, err)
		assert.Equal(t, "example.org", p.Name)
		assert.Equal(t, "	# the certificate\n", string(cert))
		assert.Equal(t, "	# the key\n", string(key))
		return c.JSON(http.StatusOK, nil)
	})
	fakeServer.GET("/services/rpaasv2-be/proxy/my_rpaas", func(c echo.Context) error {
		return c.JSON(http.StatusOK, struct {
			Routes       []types.Route           `json:"routes"`
			Certificates []types.CertificateInfo `json:"certificates"`
		}{
			Routes: []types.Route{
				{Path: "/", Content: "	# nginx config\n"},
			},
			Certificates: []types.CertificateInfo{
				{
					Name: "example.org",
				},
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

	resourceName := "rpaas_certificate.custom_route"
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		IDRefreshName:     resourceName,
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      nil,
		Steps: []resource.TestStep{
			{
				Config: testAccRpaasCertificateConfig_basic("my_rpaas"),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccResourceExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "instance", "my_rpaas"),
					resource.TestCheckResourceAttr(resourceName, "service_name", "rpaasv2-be"),
					resource.TestCheckResourceAttr(resourceName, "name", "example.org"),
				),
			},
		},
	})
}

func testAccRpaasCertificateConfig_basic(name string) string {
	return fmt.Sprintf(`
resource "rpaas_certificate" "custom_route" {
	instance = "%s"
	service_name = "rpaasv2-be"

	name = "example.org"

	certificate = <<EOF
	# the certificate
	EOF

	key = <<EOF
	# the key
	EOF
}
`, name)
}

func readMultiPartFile(c echo.Context, file string) (string, error) {
	formFile, err := c.FormFile(file)
	if err != nil {
		return "", err
	}
	stream, err := formFile.Open()
	if err != nil {
		return "", err
	}
	defer stream.Close()
	buffer, err := ioutil.ReadAll(stream)
	if err != nil {
		return "", err
	}

	return string(buffer), nil
}
