// Copyright 2021 tsuru authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package rpaas

import (
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
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

func TestAccRpaasFile_basic(t *testing.T) {
	fakeServer := echo.New()
	fakeServer.POST("/services/rpaasv2-be/proxy/my_rpaas", func(c echo.Context) error {
		uploadedFiles, err := parseFileMapFromContext(c)
		if err != nil {
			t.Error(err)
		}
		assert.Equal(t, map[string]string{
			"custom_file.txt": "line1\nline2\n",
		}, uploadedFiles)

		return c.JSON(http.StatusCreated, nil)
	})
	fakeServer.GET("/services/rpaasv2-be/proxy/my_rpaas", func(c echo.Context) error {
		qparam := c.Request().URL.Query()
		path := qparam["callback"][0]
		if path == "/resources/my_rpaas/files/custom_file.txt" {
			return c.JSON(http.StatusOK, types.RpaasFile{
				Name:    "custom_file.txt",
				Content: []byte("line1\nline2\n"),
			})
		}
		return c.JSON(http.StatusNotFound, nil)
	})
	fakeServer.DELETE("/services/rpaasv2-be/proxy/my_rpaas", func(c echo.Context) error {
		p := []string{}
		err := json.NewDecoder(c.Request().Body).Decode(&p)
		require.NoError(t, err)
		assert.Len(t, p, 1)
		assert.Equal(t, "custom_file.txt", p[0])

		return c.NoContent(http.StatusOK)
	})

	fakeServer.HTTPErrorHandler = func(err error, c echo.Context) {
		t.Errorf("methods=%s, path=%s, err=%s", c.Request().Method, c.Path(), err.Error())
	}
	server := httptest.NewServer(fakeServer)
	os.Setenv("TSURU_TARGET", server.URL)
	os.Setenv("TSURU_TOKEN", "asdf")

	resourceName := "rpaas_file.custom_file"
	terraformDeclaration := `
resource "rpaas_file" "custom_file" {
	instance     = "my_rpaas"
	service_name = "rpaasv2-be"
	name         = "custom_file.txt"
	content      = <<-EOF
		line1
		line2
	EOF
}
`
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		IDRefreshName:     resourceName,
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      nil,
		Steps: []resource.TestStep{
			{
				Config: terraformDeclaration,
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccResourceExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "instance", "my_rpaas"),
					resource.TestCheckResourceAttr(resourceName, "service_name", "rpaasv2-be"),
					resource.TestCheckResourceAttr(resourceName, "name", "custom_file.txt"),
				),
			},
			{
				// Testing import
				ResourceName:  "rpaas_file.custom_file",
				ImportStateId: "rpaasv2-be/my_rpaas/custom_file.txt",
				ImportState:   true,
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccResourceExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "instance", "my_rpaas"),
					resource.TestCheckResourceAttr(resourceName, "service_name", "rpaasv2-be"),
					resource.TestCheckResourceAttr(resourceName, "name", "custom_file.txt"),
				),
			},
		},
	})
}

func parseFileMapFromContext(c echo.Context) (map[string]string, error) {
	mediaType, params, err := mime.ParseMediaType(c.Request().Header.Get("Content-Type"))
	if err != nil {
		return nil, err
	}
	if mediaType != "multipart/form-data" {
		return nil, fmt.Errorf("Content-Type was not multipart/form-data. Got %q instead", mediaType)
	}

	mr := multipart.NewReader(c.Request().Body, params["boundary"])
	uploadedFiles := map[string]string{}
	for {
		p, err := mr.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			return uploadedFiles, err
		}
		slurp, err := io.ReadAll(p)
		if err != nil {
			return uploadedFiles, err
		}
		uploadedFiles[p.FileName()] = string(slurp)
	}
	return uploadedFiles, nil
}
