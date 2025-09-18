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
	"github.com/tsuru/rpaas-operator/pkg/rpaas/client/types"
)

func TestAccRpaasCertManager_basic(t *testing.T) {
	testAPIClient, testAPIServer := setupTestAPIServer(t)
	defer testAPIServer.Stop()

	resourceName := "rpaas_cert_manager.cert-manager-custom-issuer"
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		IDRefreshName:     resourceName,
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      nil,
		Steps: []resource.TestStep{
			{
				Config: testAccRpaasCertManagerConfigWithName("my-custom-issuer", "my-instance.test", `["*.example.com", "my-instance.test"]`),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccResourceExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "id", "rpaasv2::my-rpaas::my-custom-issuer::my-instance.test"),
					resource.TestCheckResourceAttr(resourceName, "service_name", "rpaasv2"),
					resource.TestCheckResourceAttr(resourceName, "instance", "my-rpaas"),
					resource.TestCheckResourceAttr(resourceName, "issuer", "my-custom-issuer"),
					resource.TestCheckResourceAttr(resourceName, "dns_names.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "dns_names.0", "*.example.com"),
					resource.TestCheckResourceAttr(resourceName, "dns_names.1", "my-instance.test"),
					func(s *terraform.State) error {
						certManagers, err := testAPIClient.ListCertManagerRequests(context.Background(), "my-rpaas")
						assert.NoError(t, err)
						assert.Len(t, certManagers, 1)
						certManager, found := findCertManagerRequestByIssuerAndName(certManagers, "my-custom-issuer", "")
						assert.True(t, found)
						assert.Equal(t, "my-custom-issuer", certManager.Issuer)
						assert.EqualValues(t, []string{"*.example.com", "my-instance.test"}, certManager.DNSNames)
						return nil
					},
				),
			},
			{
				// Testing Update
				Config: testAccRpaasCertManagerConfigWithName("my-custom-issuer", "my-instance.test", `["my-instance.test"]`),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccResourceExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "id", "rpaasv2::my-rpaas::my-custom-issuer::my-instance.test"),
					resource.TestCheckResourceAttr(resourceName, "service_name", "rpaasv2"),
					resource.TestCheckResourceAttr(resourceName, "instance", "my-rpaas"),
					resource.TestCheckResourceAttr(resourceName, "issuer", "my-custom-issuer"),
					resource.TestCheckResourceAttr(resourceName, "dns_names.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "dns_names.0", "my-instance.test"),
					func(s *terraform.State) error {
						certManagers, err := testAPIClient.ListCertManagerRequests(context.Background(), "my-rpaas")
						assert.NoError(t, err)
						assert.Len(t, certManagers, 1)
						certManager, found := findCertManagerRequestByIssuerAndName(certManagers, "my-custom-issuer", "")
						assert.True(t, found)
						assert.Equal(t, "my-custom-issuer", certManager.Issuer)
						assert.EqualValues(t, []string{"my-instance.test"}, certManager.DNSNames)
						return nil
					}),
			},
		},
	})
}

func TestAccRpaasCertManager_withName(t *testing.T) {
	testAPIClient, testAPIServer := setupTestAPIServer(t)
	defer testAPIServer.Stop()

	resourceName := "rpaas_cert_manager.cert-manager-custom-issuer"
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		IDRefreshName:     resourceName,
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      nil,
		Steps: []resource.TestStep{
			{
				Config: testAccRpaasCertManagerConfigWithName("my-custom-issuer", "example.com", `["*.example.com", "my-instance.test"]`),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccResourceExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "id", "rpaasv2::my-rpaas::my-custom-issuer::example.com"),
					resource.TestCheckResourceAttr(resourceName, "service_name", "rpaasv2"),
					resource.TestCheckResourceAttr(resourceName, "instance", "my-rpaas"),
					resource.TestCheckResourceAttr(resourceName, "issuer", "my-custom-issuer"),
					resource.TestCheckResourceAttr(resourceName, "certificate_name", "example.com"),
					resource.TestCheckResourceAttr(resourceName, "dns_names.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "dns_names.0", "*.example.com"),
					resource.TestCheckResourceAttr(resourceName, "dns_names.1", "my-instance.test"),
					func(s *terraform.State) error {
						certManagers, err := testAPIClient.ListCertManagerRequests(context.Background(), "my-rpaas")
						assert.NoError(t, err)
						assert.Len(t, certManagers, 1)
						certManager, found := findCertManagerRequestByIssuerAndName(certManagers, "my-custom-issuer", "")
						assert.True(t, found)
						assert.Equal(t, "example.com", certManager.Name)
						assert.Equal(t, "my-custom-issuer", certManager.Issuer)
						assert.EqualValues(t, []string{"*.example.com", "my-instance.test"}, certManager.DNSNames)
						return nil
					},
				),
			},
			{
				// Testing Update
				Config: testAccRpaasCertManagerConfigWithName("my-custom-issuer", "example.com", `["my-instance.test"]`),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccResourceExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "id", "rpaasv2::my-rpaas::my-custom-issuer::example.com"),
					resource.TestCheckResourceAttr(resourceName, "service_name", "rpaasv2"),
					resource.TestCheckResourceAttr(resourceName, "instance", "my-rpaas"),
					resource.TestCheckResourceAttr(resourceName, "issuer", "my-custom-issuer"),
					resource.TestCheckResourceAttr(resourceName, "certificate_name", "example.com"),
					resource.TestCheckResourceAttr(resourceName, "dns_names.#", "1"),
					resource.TestCheckResourceAttr(resourceName, "dns_names.0", "my-instance.test"),
					func(s *terraform.State) error {
						certManagers, err := testAPIClient.ListCertManagerRequests(context.Background(), "my-rpaas")
						assert.NoError(t, err)
						assert.Len(t, certManagers, 1)
						certManager, found := findCertManagerRequestByIssuerAndName(certManagers, "my-custom-issuer", "")
						assert.True(t, found)
						assert.Equal(t, "my-custom-issuer", certManager.Issuer)
						assert.Equal(t, "example.com", certManager.Name)
						assert.EqualValues(t, []string{"my-instance.test"}, certManager.DNSNames)
						return nil
					}),
			},
		},
	})
}

func TestAccRpaasCertManager_import(t *testing.T) {
	testAPIClient, testAPIServer := setupTestAPIServer(t)
	defer testAPIServer.Stop()

	if err := testAPIClient.UpdateCertManager(context.Background(),
		client.UpdateCertManagerArgs{
			Instance: "my-rpaas",
			CertManager: types.CertManager{
				Issuer:      "my-custom-issuer",
				DNSNames:    []string{"dns1.example.com", "dns2.example.com", "dns3.example.com"},
				IPAddresses: []string{"10.10.10.10", "192.168.90.90"}, // ignored on this provider
			},
		},
	); err != nil {
		t.Errorf("Api client failed to connect: %v", err)
	}

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      nil,
		Steps: []resource.TestStep{
			// Testing Import
			{
				Config:        `resource "rpaas_cert_manager" "imported" {}`,
				ResourceName:  "rpaas_cert_manager.imported",
				ImportStateId: "rpaasv2-be::my-rpaas::my-custom-issuer",
				ImportState:   true,
				ImportStateCheck: func(s []*terraform.InstanceState) error {
					state := s[0]
					assert.Equal(t, "rpaasv2-be", state.Attributes["service_name"])
					assert.Equal(t, "my-rpaas", state.Attributes["instance"])
					assert.Equal(t, "my-custom-issuer", state.Attributes["issuer"])
					assert.Equal(t, "3", state.Attributes["dns_names.#"])
					assert.Equal(t, "dns1.example.com", state.Attributes["dns_names.0"])
					assert.Equal(t, "dns2.example.com", state.Attributes["dns_names.1"])
					assert.Equal(t, "dns3.example.com", state.Attributes["dns_names.2"])
					return nil
				},
			},
			// Testing Import legacy ID
			{
				Config:        `resource "rpaas_cert_manager" "imported_legacy" {}`,
				ResourceName:  "rpaas_cert_manager.imported_legacy",
				ImportStateId: "rpaasv2-be my-rpaas my-custom-issuer", // legacy ID
				ImportState:   true,
				ImportStateCheck: func(s []*terraform.InstanceState) error {
					state := s[0]
					assert.Len(t, s, 1)
					assert.Equal(t, "rpaasv2-be::my-rpaas::my-custom-issuer", state.Attributes["id"])
					return nil
				},
			},
		},
	})
}

func testAccRpaasCertManagerConfigWithName(issuer, name, dnsNamesArray string) string {
	return fmt.Sprintf(`
resource "rpaas_cert_manager" "cert-manager-custom-issuer" {
	instance         = "my-rpaas"
	service_name     = "rpaasv2"
	certificate_name = "%s"
	issuer           = "%s"
	dns_names        = %s
}`, name, issuer, dnsNamesArray)
}
