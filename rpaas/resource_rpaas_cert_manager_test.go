// Copyright 2021 tsuru authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package rpaas

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tsuru/rpaas-operator/pkg/rpaas/client/types"
	rpaasTypes "github.com/tsuru/rpaas-operator/pkg/rpaas/client/types"
)

func TestAccRpaasCertManager_with_custom_issuer(t *testing.T) {
	var count int

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() { count++ }()

		if count == 0 && r.Method == "POST" && r.URL.Path == "/services/rpaasv2/proxy/my-instance" && r.URL.RawQuery == "callback=/resources/my-instance/cert-manager" {
			var p types.CertManager
			err := json.NewDecoder(r.Body).Decode(&p)
			require.NoError(t, err)
			defer r.Body.Close()

			assert.Equal(t, "my-custom-issuer.ClusterIssuer.example.com", p.Issuer)
			assert.Equal(t, []string{"*.example.com", "my-instance.test"}, p.DNSNames)

			w.WriteHeader(http.StatusOK)
			return
		}

		if (count >= 1 && count <= 2) && r.Method == "GET" && r.URL.Path == "/services/rpaasv2/proxy/my-instance" && r.URL.RawQuery == "callback=/resources/my-instance/cert-manager" {
			err := json.NewEncoder(w).Encode([]rpaasTypes.CertManager{
				{
					Issuer:   "lets-encrypt",
					DNSNames: []string{"my-instance.example.com"},
				},
				{
					Issuer:   "my-custom-issuer.ClusterIssuer.example.com",
					DNSNames: []string{"*.example.com", "my-instance.test"},
				},
			})
			require.NoError(t, err)
			return
		}

		if count == 3 && r.Method == "POST" && r.URL.Path == "/services/rpaasv2/proxy/my-instance" && r.URL.RawQuery == "callback=/resources/my-instance/cert-manager" {
			var p types.CertManager
			err := json.NewDecoder(r.Body).Decode(&p)
			require.NoError(t, err)
			defer r.Body.Close()

			assert.Equal(t, "my-custom-issuer.ClusterIssuer.example.com", p.Issuer)
			assert.Equal(t, []string{"*.example.com", "*.example.org", "my-instance.test"}, p.DNSNames)

			w.WriteHeader(http.StatusOK)
			return
		}

		if count == 4 && r.Method == "GET" && r.URL.Path == "/services/rpaasv2/proxy/my-instance" && r.URL.RawQuery == "callback=/resources/my-instance/cert-manager" {
			err := json.NewEncoder(w).Encode([]rpaasTypes.CertManager{
				{
					Issuer:   "lets-encrypt",
					DNSNames: []string{"my-instance.example.com"},
				},
				{
					Issuer:   "my-custom-issuer.ClusterIssuer.example.com",
					DNSNames: []string{"*.example.com", "*.example.org", "my-instance.test"},
				},
			})
			require.NoError(t, err)
			return
		}

		if count == 5 && r.Method == "DELETE" && r.URL.Path == "/services/rpaasv2/proxy/my-instance" && r.URL.RawQuery == "callback=/resources/my-instance/cert-manager&issuer=my-custom-issuer.ClusterIssuer.example.com" {
			w.WriteHeader(http.StatusOK)
			return
		}

		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "not implemented")
	}))
	defer server.Close()

	os.Setenv("TSURU_TARGET", server.URL)
	os.Setenv("TSURU_TOKEN", "foo bar")

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
resource "rpaas_cert_manager" "cert-manager-custom-issuer" {
  instance     = "my-instance"
  service_name = "rpaasv2"
  issuer       = "my-custom-issuer.ClusterIssuer.example.com"
  dns_names    = ["*.example.com", "my-instance.test"]
}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("rpaas_cert_manager.cert-manager-custom-issuer", "id", "rpaasv2 my-instance my-custom-issuer.ClusterIssuer.example.com"),
					resource.TestCheckResourceAttr("rpaas_cert_manager.cert-manager-custom-issuer", "service_name", "rpaasv2"),
					resource.TestCheckResourceAttr("rpaas_cert_manager.cert-manager-custom-issuer", "instance", "my-instance"),
					resource.TestCheckResourceAttr("rpaas_cert_manager.cert-manager-custom-issuer", "issuer", "my-custom-issuer.ClusterIssuer.example.com"),
					resource.TestCheckResourceAttr("rpaas_cert_manager.cert-manager-custom-issuer", "dns_names.#", "2"),
					resource.TestCheckResourceAttr("rpaas_cert_manager.cert-manager-custom-issuer", "dns_names.0", "*.example.com"),
					resource.TestCheckResourceAttr("rpaas_cert_manager.cert-manager-custom-issuer", "dns_names.1", "my-instance.test"),
				),
			},

			{
				Config: `
resource "rpaas_cert_manager" "cert-manager-custom-issuer" {
  instance     = "my-instance"
  service_name = "rpaasv2"
  issuer       = "my-custom-issuer.ClusterIssuer.example.com"
  dns_names    = ["*.example.com", "*.example.org", "my-instance.test"]
}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("rpaas_cert_manager.cert-manager-custom-issuer", "id", "rpaasv2 my-instance my-custom-issuer.ClusterIssuer.example.com"),
					resource.TestCheckResourceAttr("rpaas_cert_manager.cert-manager-custom-issuer", "service_name", "rpaasv2"),
					resource.TestCheckResourceAttr("rpaas_cert_manager.cert-manager-custom-issuer", "instance", "my-instance"),
					resource.TestCheckResourceAttr("rpaas_cert_manager.cert-manager-custom-issuer", "issuer", "my-custom-issuer.ClusterIssuer.example.com"),
					resource.TestCheckResourceAttr("rpaas_cert_manager.cert-manager-custom-issuer", "dns_names.#", "3"),
					resource.TestCheckResourceAttr("rpaas_cert_manager.cert-manager-custom-issuer", "dns_names.0", "*.example.com"),
					resource.TestCheckResourceAttr("rpaas_cert_manager.cert-manager-custom-issuer", "dns_names.1", "*.example.org"),
					resource.TestCheckResourceAttr("rpaas_cert_manager.cert-manager-custom-issuer", "dns_names.2", "my-instance.test"),
				),
			},
		},
	})
}
