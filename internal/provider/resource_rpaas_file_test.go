// Copyright 2021 tsuru authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package provider

import (
	"context"
	"encoding/base64"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/stretchr/testify/assert"
	"github.com/tsuru/rpaas-operator/pkg/rpaas/client"
	"github.com/tsuru/rpaas-operator/pkg/rpaas/client/types"
)

var b64Image = "iVBORw0KGgoAAAANSUhEUgAAABQAAAAPCAYAAADkmO9VAAAACXBIWXMAAAS1AAAEtQHKlkJQAAAAGXRFWHRTb2Z0d2FyZQB3d3cuaW5rc2NhcGUub3Jnm+48GgAAAZhJREFUOI2tkTFrU1EYhp/vpAkoxOJScXHwajuIiL03dPBHiIhjRxscVIo6FFutKBUXaSnE3CtFtDgqFrq7uVQzi+29ikNbwUmENhDu69AbiE3SVOk7nu85z3k/Dn4S3uIA4xDlII4uH5wQbQotBEll6H8lpS+Vk+fjatBsuAH0S7klPw77/1UWxOF42pe75LDlC58Xis7M1rPZEGYv0bTbr2x4LXwEnFDKUeDwdqFxp9lwJ9LF4eT41L5kcfjEjGv5Qu5h7VR5sugKA6S8tyCJRiW9amFl2JWP3tibbjI/rs6ATQjGa155tnXmrLXhTkzoRWk1OtNmksyPq7NgExjJr7T+bDfiSNuEAMXU6e1fnySZ//X5HNhNAKV2e+30jXqbcLvQWN99mGXQYBFNOyQLkmge6TqA4EPNu/qu0yVDMj+J6kC+M8CDTHK/2dVhIyve2Eon3pW+VY51k2WiezL7DfzMHnjdTQbgpNzZbsOWLaZAj4Etwd29YCfcuR5CgCOYmcTTT175+15gH9CrIQBCPw418lEv7g+w+5bLDG72lgAAAABJRU5ErkJggg=="

func TestAccRpaasFile_basic(t *testing.T) {
	testAPIClient, testAPIServer := setupTestAPIServer(t)
	defer testAPIServer.Stop()

	resourceName := "rpaas_file.custom_file"
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		IDRefreshName:     resourceName,
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      nil,
		Steps: []resource.TestStep{
			{
				// Testing Create
				Config: testAccRpaasFileConfig("custom_file.txt", "content"),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccResourceExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "id", "rpaasv2-be::my-rpaas::custom_file.txt"),
					resource.TestCheckResourceAttr(resourceName, "instance", "my-rpaas"),
					resource.TestCheckResourceAttr(resourceName, "service_name", "rpaasv2-be"),
					resource.TestCheckResourceAttr(resourceName, "name", "custom_file.txt"),
					resource.TestCheckResourceAttr(resourceName, "content", "content\n"),
					func(s *terraform.State) error {
						extraFiles, err := testAPIClient.ListExtraFiles(context.Background(),
							client.ListExtraFilesArgs{Instance: "my-rpaas", ShowContent: true},
						)
						assert.NoError(t, err)
						assert.Len(t, extraFiles, 1)
						assert.Equal(t, "custom_file.txt", extraFiles[0].Name)
						assert.EqualValues(t, "content\n", extraFiles[0].Content)
						return nil
					},
				),
			},
			{
				// Testing Update - content
				Config: testAccRpaasFileConfig("custom_file.txt", "changed"),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccResourceExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "id", "rpaasv2-be::my-rpaas::custom_file.txt"),
					resource.TestCheckResourceAttr(resourceName, "instance", "my-rpaas"),
					resource.TestCheckResourceAttr(resourceName, "service_name", "rpaasv2-be"),
					resource.TestCheckResourceAttr(resourceName, "name", "custom_file.txt"),
					resource.TestCheckResourceAttr(resourceName, "content", "changed\n"),
					func(s *terraform.State) error {
						extraFiles, err := testAPIClient.ListExtraFiles(context.Background(),
							client.ListExtraFilesArgs{Instance: "my-rpaas", ShowContent: true},
						)
						assert.NoError(t, err)
						assert.Len(t, extraFiles, 1)
						assert.Equal(t, "custom_file.txt", extraFiles[0].Name)
						assert.EqualValues(t, "changed\n", extraFiles[0].Content)
						return nil
					},
				),
			},
			{
				// Testing Update - file name
				Config: testAccRpaasFileConfig("different.txt", "changed"),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccResourceExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "id", "rpaasv2-be::my-rpaas::different.txt"),
					resource.TestCheckResourceAttr(resourceName, "instance", "my-rpaas"),
					resource.TestCheckResourceAttr(resourceName, "service_name", "rpaasv2-be"),
					resource.TestCheckResourceAttr(resourceName, "name", "different.txt"),
					resource.TestCheckResourceAttr(resourceName, "content", "changed\n"),
					func(s *terraform.State) error {
						extraFiles, err := testAPIClient.ListExtraFiles(context.Background(),
							client.ListExtraFilesArgs{Instance: "my-rpaas", ShowContent: true},
						)
						assert.NoError(t, err)
						assert.Len(t, extraFiles, 1)
						assert.Equal(t, "different.txt", extraFiles[0].Name)
						assert.EqualValues(t, "changed\n", extraFiles[0].Content)
						return nil
					},
				),
			},
			{
				// content_base64
				Config: testAccRpaasFileConfigExtraParam("base64.txt", `content_base64 = "aGVsbG8gd29ybGQ="`),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccResourceExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "name", "base64.txt"),
					resource.TestCheckResourceAttr(resourceName, "content_base64", "aGVsbG8gd29ybGQ="),
					resource.TestCheckResourceAttr(resourceName, "content", ""),
					func(s *terraform.State) error {
						extraFiles, err := testAPIClient.ListExtraFiles(context.Background(),
							client.ListExtraFilesArgs{Instance: "my-rpaas", ShowContent: true},
						)
						assert.NoError(t, err)
						assert.Len(t, extraFiles, 1)
						assert.Equal(t, "base64.txt", extraFiles[0].Name)
						assert.EqualValues(t, "hello world", extraFiles[0].Content)
						return nil
					},
				),
			},
		},
	})
}

func TestAccRpaasFile_import(t *testing.T) {
	testAPIClient, testAPIServer := setupTestAPIServer(t)
	defer testAPIServer.Stop()

	setupRpaasFilesWithClient(t, testAPIClient)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      nil,
		Steps: []resource.TestStep{
			{
				// Testing Import
				Config:        `resource "rpaas_file" "imported" {}`,
				ResourceName:  "rpaas_file.imported",
				ImportStateId: "rpaasv2-be::my-rpaas::imported.txt",
				ImportState:   true,
				ImportStateCheck: func(s []*terraform.InstanceState) error {
					state := s[0]
					assert.Equal(t, "rpaasv2-be", state.Attributes["service_name"])
					assert.Equal(t, "my-rpaas", state.Attributes["instance"])
					assert.Equal(t, "imported.txt", state.Attributes["name"])
					assert.Equal(t, "imported content", state.Attributes["content"])
					return nil
				},
			},
			{
				// Testing Import legacy ID
				Config:        `resource "rpaas_file" "imported_legacy" {}`,
				ResourceName:  "rpaas_file.imported_legacy",
				ImportStateId: "rpaasv2-be/my-rpaas/imported.txt", // legacy id
				ImportState:   true,
				ImportStateCheck: func(s []*terraform.InstanceState) error {
					state := s[0]
					assert.Len(t, s, 1)
					assert.Equal(t, "rpaasv2-be::my-rpaas::imported.txt", state.Attributes["id"])
					return nil
				},
			},
			{
				// Testing Import file with binary content (image)
				Config:        `resource "rpaas_file" "image" {}`,
				ResourceName:  "rpaas_file.image",
				ImportStateId: "rpaasv2-be::my-rpaas::image.png",
				ImportState:   true,
				ImportStateCheck: func(s []*terraform.InstanceState) error {
					state := s[0]
					assert.Equal(t, "rpaasv2-be", state.Attributes["service_name"])
					assert.Equal(t, "my-rpaas", state.Attributes["instance"])
					assert.Equal(t, "image.png", state.Attributes["name"])
					assert.Equal(t, "", state.Attributes["content"])
					assert.Equal(t, b64Image, state.Attributes["content_base64"])
					return nil
				},
			},
		},
	})
}

func setupRpaasFilesWithClient(t *testing.T, testAPIClient client.Client) {
	if err := testAPIClient.AddExtraFiles(context.Background(),
		client.ExtraFilesArgs{
			Instance: "my-rpaas",
			Files:    []types.RpaasFile{{Name: "imported.txt", Content: []byte("imported content")}},
		},
	); err != nil {
		t.Errorf("Api client failed to connect: %v", err)
	}
	image, _ := base64.StdEncoding.DecodeString(b64Image)
	if err := testAPIClient.AddExtraFiles(context.Background(),
		client.ExtraFilesArgs{
			Instance: "my-rpaas",
			Files:    []types.RpaasFile{{Name: "image.png", Content: image}},
		},
	); err != nil {
		t.Errorf("Api client failed to connect: %v", err)
	}
}

func testAccRpaasFileConfig(filename, content string) string {
	return fmt.Sprintf(`
resource "rpaas_file" "custom_file" {
	instance     = "my-rpaas"
	service_name = "rpaasv2-be"
	name         = "%s"
	content      = <<-EOF
		%s
	EOF
}
`, filename, content)
}

func testAccRpaasFileConfigExtraParam(filename, param string) string {
	return fmt.Sprintf(`
resource "rpaas_file" "custom_file" {
	instance     = "my-rpaas"
	service_name = "rpaasv2-be"
	name         = "%s"
	%s
}
`, filename, param)
}
