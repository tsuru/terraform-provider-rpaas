// Copyright 2021 tsuru authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package provider

import (
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"testing"
	"time"

	cmv1 "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/stretchr/testify/require"
	"github.com/tsuru/rpaas-operator/api/v1alpha1"
	"github.com/tsuru/rpaas-operator/pkg/rpaas/client"
	"github.com/tsuru/rpaas-operator/pkg/web"
	"github.com/tsuru/rpaas-operator/pkg/web/target"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

var testAccProviderFactories = map[string]func() (*schema.Provider, error){
	"rpaas": func() (*schema.Provider, error) {
		return Provider(), nil
	},
}

func TestProvider(t *testing.T) {
	provider := Provider()
	require.NoError(t, provider.InternalValidate(), "failed to validate internal provider")
}

func setupTestRpaasServer(t *testing.T) (*web.Api, *rpaasProvider) {
	t.Helper()

	address := fmt.Sprintf("127.0.0.1:19%03d", rand.Intn(999))
	serverURL := fmt.Sprintf("http://%s", address)

	t.Setenv("RPAAS_URL", serverURL)
	t.Setenv("RPAAS_USER", "admin")
	t.Setenv("RPAAS_PASSWORD", "admin")

	factory, err := target.NewFakeServerFactory(fakeRuntimeObjects())
	require.NoError(t, err, "failed to create factory")

	server, err := web.NewWithTargetFactory(factory, address, "", time.Second, make(chan struct{}, 1))
	require.NoError(t, err, "could not create fake RPaaS API")

	go func() {
		nerr := server.StartWithOptions(web.APIServerStartOptions{
			ConfigEnableCertManager: true,
			DiscardLogging:          true,
		})
		require.NoError(t, nerr, "failed to start fake RPaaS web server")
	}()

	err = waitForOkStatus(fmt.Sprintf("%s/healthcheck", serverURL))
	require.NoError(t, err, "Failed connect to the fake RPaaS API")

	providerOpts := &ProviderConfigOptions{
		URL:      serverURL,
		Username: "admin",
		Password: "admin",
	}

	legacyClient, err := getLegacyClient(providerOpts)
	require.NoError(t, err)

	provider := &rpaasProvider{
		RpaasClient: legacyClient,
		opts:        providerOpts,
	}

	return server, provider
}

func setupTestAPIServer(t *testing.T) (client.Client, *web.Api) {
	t.Helper()
	server, provider := setupTestRpaasServer(t)
	return provider.RpaasClient, server
}

func testAccPreCheck(t *testing.T) {
	require.Contains(t, os.Getenv("RPAAS_URL"), "http://127.0.0.1:19")

	_, found := os.LookupEnv("RPAAS_TARGET")
	require.False(t, found, "Should not set the TSURU_TARGET env var")
}

func testAccResourceExists(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		_, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("Not found: %s", resourceName)
		}
		return nil
	}
}

func fakeRuntimeObjects() []runtime.Object {
	return []runtime.Object{
		&v1alpha1.RpaasPlan{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "my-plan",
				Namespace: "rpaasv2",
			},
		},
		&v1alpha1.RpaasInstance{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "my-rpaas",
				Namespace: "rpaasv2",
			},
		},
		&cmv1.ClusterIssuer{
			TypeMeta: metav1.TypeMeta{},
			ObjectMeta: metav1.ObjectMeta{
				Name: "my-custom-issuer",
			},
			Spec: cmv1.IssuerSpec{
				IssuerConfig: cmv1.IssuerConfig{
					SelfSigned: &cmv1.SelfSignedIssuer{},
				},
			},
		},
	}
}

func waitForOkStatus(url string) error {
	client := http.Client{
		Timeout: 2 * time.Second,
	}

	for tries := 0; tries < 5; tries++ {
		resp, err := client.Get(url)
		if err == nil && resp.StatusCode == http.StatusOK {
			return nil
		}

	}
	return fmt.Errorf("Could not get OK after too many attempts")
}
