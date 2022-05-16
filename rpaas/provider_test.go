// Copyright 2021 tsuru authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package rpaas

import (
	"fmt"
	"math/rand"
	"os"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/stretchr/testify/require"
	"github.com/tsuru/rpaas-operator/pkg/rpaas/client"
	"github.com/tsuru/rpaas-operator/pkg/web"
	"github.com/tsuru/rpaas-operator/pkg/web/target"
)

var testAccProvider *schema.Provider
var testAccProviderFactories = map[string]func() (*schema.Provider, error){
	"rpaas": func() (*schema.Provider, error) {
		return testAccProvider, nil
	},
}

func init() {
	testAccProvider = Provider()
}

func TestProvider(t *testing.T) {
	provider := Provider()
	if err := provider.InternalValidate(); err != nil {
		t.Fatalf("err: %s", err)
	}
}

func setupTestAPIServer(t *testing.T) (client.Client, *web.Api) {
	apiServerListen := fmt.Sprintf("127.0.0.1:199%02d", rand.Intn(99))
	os.Setenv("RPAAS_TARGET", "http://"+apiServerListen)
	os.Setenv("TSURU_TARGET", "http://"+apiServerListen)
	os.Setenv("TSURU_TOKEN", "asdf")
	os.Setenv("PROVIDER_SKIP_TSURU_PASSTHROUGH", "true")

	factory, _ := target.NewFakeServerFactory()
	apiServer, err := web.NewWithTargetFactory(factory, apiServerListen, "", 2*time.Second, make(chan struct{}))
	if err != nil {
		t.Errorf("Fail to create the api")
	}
	go apiServer.StartWithOptions(web.APIServerStartOptions{DiscardLogging: true})

	testAPIClient, err := client.NewClient("http://"+apiServerListen, "", "")
	if err != nil {
		t.Errorf("failed to create new rpaas client")
	}
	return testAPIClient, apiServer
}

func testAccPreCheck(t *testing.T) {
	tsuruTarget := os.Getenv("TSURU_TARGET")
	require.Contains(t, tsuruTarget, "http://127.0.0.1:")
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
