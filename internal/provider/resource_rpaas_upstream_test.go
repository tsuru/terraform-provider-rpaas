package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/stretchr/testify/assert"
	"github.com/tsuru/rpaas-operator/pkg/rpaas/client"
)

func TestAccResourceRpaasUpstreamImplicitRoundRobin(t *testing.T) {
	testAPIClient, testAPIServer := setupTestAPIServer(t)
	defer testAPIServer.Stop()

	resourceName := "rpaas_upstream.test"
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      nil,
		Steps: []resource.TestStep{
			{
				Config: testAccResourceRpaasUpstreamNoLoadBalance(),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccResourceExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "instance", "my-rpaas"),
					resource.TestCheckResourceAttr(resourceName, "service_name", "rpaasv2-be"),
					resource.TestCheckResourceAttr(resourceName, "app", "test-app"),
					func(s *terraform.State) error {
						upstreamOptions, err := testAPIClient.ListUpstreamOptions(context.Background(), client.ListUpstreamOptionsArgs{Instance: "my-rpaas"})
						assert.NoError(t, err)
						assert.Len(t, upstreamOptions, 1)
						assert.Equal(t, "test-app", upstreamOptions[0].PrimaryBind)
						assert.Equal(t, "round_robin", string(upstreamOptions[0].LoadBalance))
						return nil
					},
				),
			},
		},
	})
}

// Test upstream with load balance round_robin
func TestAccResourceRpaasUpstreamLoadBalanceRoundRobin(t *testing.T) {
	testAPIClient, testAPIServer := setupTestAPIServer(t)
	defer testAPIServer.Stop()

	resourceName := "rpaas_upstream.test"
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      nil,
		Steps: []resource.TestStep{
			{
				Config: testAccResourceRpaasUpstreamLoadBalanceConfig("round_robin", ""),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccResourceExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "load_balance", "round_robin"),
					func(s *terraform.State) error {
						upstreamOptions, err := testAPIClient.ListUpstreamOptions(context.Background(), client.ListUpstreamOptionsArgs{Instance: "my-rpaas"})
						assert.NoError(t, err)
						assert.Len(t, upstreamOptions, 1)
						assert.Equal(t, "round_robin", string(upstreamOptions[0].LoadBalance))
						return nil
					},
				),
			},
		},
	})
}

// Test upstream with load balance chash (consistent hash)
func TestAccResourceRpaasUpstreamLoadBalanceChash(t *testing.T) {
	testAPIClient, testAPIServer := setupTestAPIServer(t)
	defer testAPIServer.Stop()

	resourceName := "rpaas_upstream.test"
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      nil,
		Steps: []resource.TestStep{
			{
				Config: testAccResourceRpaasUpstreamLoadBalanceConfig("chash", "$remote_addr"),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccResourceExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "load_balance", "chash"),
					resource.TestCheckResourceAttr(resourceName, "load_balance_hash_key", "$remote_addr"),
					func(s *terraform.State) error {
						upstreamOptions, err := testAPIClient.ListUpstreamOptions(context.Background(), client.ListUpstreamOptionsArgs{Instance: "my-rpaas"})
						assert.NoError(t, err)
						assert.Len(t, upstreamOptions, 1)
						assert.Equal(t, "chash", string(upstreamOptions[0].LoadBalance))
						return nil
					},
				),
			},
		},
	})
}

// Test upstream with load balance ewma (exponentially weighted moving average)
func TestAccResourceRpaasUpstreamLoadBalanceEWMA(t *testing.T) {
	testAPIClient, testAPIServer := setupTestAPIServer(t)
	defer testAPIServer.Stop()

	resourceName := "rpaas_upstream.test"
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      nil,
		Steps: []resource.TestStep{
			{
				Config: testAccResourceRpaasUpstreamLoadBalanceConfig("ewma", ""),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccResourceExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "load_balance", "ewma"),
					func(s *terraform.State) error {
						upstreamOptions, err := testAPIClient.ListUpstreamOptions(context.Background(), client.ListUpstreamOptionsArgs{Instance: "my-rpaas"})
						assert.NoError(t, err)
						assert.Len(t, upstreamOptions, 1)
						assert.Equal(t, "ewma", string(upstreamOptions[0].LoadBalance))
						return nil
					},
				),
			},
		},
	})
}

// Test upstream with traffic shaping policy - weight-based
func TestAccResourceRpaasUpstreamTrafficShapingWeight(t *testing.T) {
	testAPIClient, testAPIServer := setupTestAPIServer(t)
	defer testAPIServer.Stop()

	resourceName := "rpaas_upstream.test"
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      nil,
		Steps: []resource.TestStep{
			{
				Config: testAccResourceRpaasUpstreamTrafficShapingWeightSingleConfig(),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccResourceExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "app", "test-app"),
					resource.TestCheckResourceAttr(resourceName, "traffic_shaping_policy.0.weight", "50"),
					resource.TestCheckResourceAttr(resourceName, "traffic_shaping_policy.0.weight_total", "100"),
					func(s *terraform.State) error {
						upstreamOptions, err := testAPIClient.ListUpstreamOptions(context.Background(), client.ListUpstreamOptionsArgs{Instance: "my-rpaas"})
						assert.NoError(t, err)
						assert.Len(t, upstreamOptions, 1)
						assert.Equal(t, 50, upstreamOptions[0].TrafficShapingPolicy.Weight)
						assert.Equal(t, 100, upstreamOptions[0].TrafficShapingPolicy.WeightTotal)
						return nil
					},
				),
			},
		},
	})
}

// Test upstream with traffic shaping policy - header-based routing
func TestAccResourceRpaasUpstreamTrafficShapingHeader(t *testing.T) {
	testAPIClient, testAPIServer := setupTestAPIServer(t)
	defer testAPIServer.Stop()

	resourceName := "rpaas_upstream.test"
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      nil,
		Steps: []resource.TestStep{
			{
				Config: testAccResourceRpaasUpstreamTrafficShapingHeaderSingleConfig(),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccResourceExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "app", "test-app"),
					resource.TestCheckResourceAttr(resourceName, "traffic_shaping_policy.0.header", "X-Canary"),
					resource.TestCheckResourceAttr(resourceName, "traffic_shaping_policy.0.header_value", "true"),
					func(s *terraform.State) error {
						upstreamOptions, err := testAPIClient.ListUpstreamOptions(context.Background(), client.ListUpstreamOptionsArgs{Instance: "my-rpaas"})
						assert.NoError(t, err)
						assert.Len(t, upstreamOptions, 1)
						assert.Equal(t, "X-Canary", upstreamOptions[0].TrafficShapingPolicy.Header)
						assert.Equal(t, "true", upstreamOptions[0].TrafficShapingPolicy.HeaderValue)
						return nil
					},
				),
			},
		},
	})
}

// Test upstream with traffic shaping policy - header-based with regex pattern
func TestAccResourceRpaasUpstreamTrafficShapingHeaderPattern(t *testing.T) {
	testAPIClient, testAPIServer := setupTestAPIServer(t)
	defer testAPIServer.Stop()

	resourceName := "rpaas_upstream.test"
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      nil,
		Steps: []resource.TestStep{
			{
				Config: testAccResourceRpaasUpstreamTrafficShapingHeaderPatternSingleConfig(),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccResourceExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "app", "test-app"),
					resource.TestCheckResourceAttr(resourceName, "traffic_shaping_policy.0.header", "X-User-Type"),
					resource.TestCheckResourceAttr(resourceName, "traffic_shaping_policy.0.header_pattern", "^(admin|tester)$"),
					func(s *terraform.State) error {
						upstreamOptions, err := testAPIClient.ListUpstreamOptions(context.Background(), client.ListUpstreamOptionsArgs{Instance: "my-rpaas"})
						assert.NoError(t, err)
						assert.Len(t, upstreamOptions, 1)
						assert.Equal(t, "X-User-Type", upstreamOptions[0].TrafficShapingPolicy.Header)
						assert.Equal(t, "^(admin|tester)$", upstreamOptions[0].TrafficShapingPolicy.HeaderPattern)
						return nil
					},
				),
			},
		},
	})
}

// Test upstream with traffic shaping policy - cookie-based routing
func TestAccResourceRpaasUpstreamTrafficShapingCookie(t *testing.T) {
	testAPIClient, testAPIServer := setupTestAPIServer(t)
	defer testAPIServer.Stop()

	resourceName := "rpaas_upstream.test"
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      nil,
		Steps: []resource.TestStep{
			{
				Config: testAccResourceRpaasUpstreamTrafficShapingCookieSingleConfig(),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccResourceExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "app", "test-app"),
					resource.TestCheckResourceAttr(resourceName, "traffic_shaping_policy.0.cookie", "x-session-id"),
					func(s *terraform.State) error {
						upstreamOptions, err := testAPIClient.ListUpstreamOptions(context.Background(), client.ListUpstreamOptionsArgs{Instance: "my-rpaas"})
						assert.NoError(t, err)
						assert.Len(t, upstreamOptions, 1)
						assert.Equal(t, "x-session-id", upstreamOptions[0].TrafficShapingPolicy.Cookie)
						return nil
					},
				),
			},
		},
	})
}

// Test upstream update - modify load balance
func TestAccResourceRpaasUpstreamUpdate(t *testing.T) {
	testAPIClient, testAPIServer := setupTestAPIServer(t)
	defer testAPIServer.Stop()

	resourceName := "rpaas_upstream.test"
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      nil,
		Steps: []resource.TestStep{
			{
				Config: testAccResourceRpaasUpstreamConfig(),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccResourceExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "load_balance", "round_robin"),
					func(s *terraform.State) error {
						upstreamOptions, err := testAPIClient.ListUpstreamOptions(context.Background(), client.ListUpstreamOptionsArgs{Instance: "my-rpaas"})
						assert.NoError(t, err)
						assert.Len(t, upstreamOptions, 1)
						assert.Equal(t, "round_robin", string(upstreamOptions[0].LoadBalance))
						return nil
					},
				),
			},
			{
				Config: testAccResourceRpaasUpstreamUpdateConfig(),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccResourceExists(resourceName),
					resource.TestCheckResourceAttr(resourceName, "load_balance", "ewma"),
					resource.TestCheckResourceAttr(resourceName, "traffic_shaping_policy.0.weight", "50"),
					resource.TestCheckResourceAttr(resourceName, "traffic_shaping_policy.0.weight_total", "100"),
					func(s *terraform.State) error {
						upstreamOptions, err := testAPIClient.ListUpstreamOptions(context.Background(), client.ListUpstreamOptionsArgs{Instance: "my-rpaas"})
						assert.NoError(t, err)
						assert.Len(t, upstreamOptions, 1)
						assert.Equal(t, "ewma", string(upstreamOptions[0].LoadBalance))
						assert.Equal(t, 50, upstreamOptions[0].TrafficShapingPolicy.Weight)
						assert.Equal(t, 100, upstreamOptions[0].TrafficShapingPolicy.WeightTotal)
						return nil
					},
				),
			},
		},
	})
}

// Test upstream import
func TestAccResourceRpaasUpstreamImport(t *testing.T) {
	_, testAPIServer := setupTestAPIServer(t)
	defer testAPIServer.Stop()

	resourceName := "rpaas_upstream.test"
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      nil,
		Steps: []resource.TestStep{
			{
				Config: testAccResourceRpaasUpstreamConfig(),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccResourceRpaasUpstreamNoLoadBalance() string {
	return `
resource "rpaas_upstream" "test" {
  instance     = "my-rpaas"
  service_name = "rpaasv2-be"
  app          = "test-app"
}
`
}

func testAccResourceRpaasUpstreamConfig() string {
	return `
resource "rpaas_upstream" "test" {
  instance     = "my-rpaas"
  service_name = "rpaasv2-be"
  app          = "test-app"
  load_balance = "round_robin"

  traffic_shaping_policy {
    weight       = 100
    weight_total = 100
  }
}
`
}

// Config for upstream with different load balance algorithms
func testAccResourceRpaasUpstreamLoadBalanceConfig(algorithm string, hashKey string) string {
	config := `
resource "rpaas_upstream" "test" {
  instance     = "my-rpaas"
  service_name = "rpaasv2-be"
  app          = "test-app"
  load_balance = "` + algorithm + `"`

	if hashKey != "" {
		config += `
  load_balance_hash_key = "` + hashKey + `"`
	}

	config += `
}
`
	return config
}

// Config for upstream update test
func testAccResourceRpaasUpstreamUpdateConfig() string {
	return `
resource "rpaas_upstream" "test" {
  instance     = "my-rpaas"
  service_name = "rpaasv2-be"
  app          = "test-app"
  load_balance = "ewma"

  traffic_shaping_policy {
    weight       = 50
    weight_total = 100
  }
}
`
}

// Single resource config helpers for traffic shaping tests

// Config for upstream with weight-based traffic shaping (single resource)
func testAccResourceRpaasUpstreamTrafficShapingWeightSingleConfig() string {
	return `
resource "rpaas_upstream" "test" {
  instance     = "my-rpaas"
  service_name = "rpaasv2-be"
  app          = "test-app"
  load_balance = "round_robin"

  traffic_shaping_policy {
    weight       = 50
    weight_total = 100
  }
}
`
}

// Config for upstream with header-based traffic shaping (single resource)
func testAccResourceRpaasUpstreamTrafficShapingHeaderSingleConfig() string {
	return `
resource "rpaas_upstream" "test" {
  instance     = "my-rpaas"
  service_name = "rpaasv2-be"
  app          = "test-app"
  load_balance = "round_robin"

  traffic_shaping_policy {
    header       = "X-Canary"
    header_value = "true"
  }
}
`
}

// Config for upstream with header pattern traffic shaping (single resource)
func testAccResourceRpaasUpstreamTrafficShapingHeaderPatternSingleConfig() string {
	return `
resource "rpaas_upstream" "test" {
  instance     = "my-rpaas"
  service_name = "rpaasv2-be"
  app          = "test-app"
  load_balance = "round_robin"

  traffic_shaping_policy {
    header         = "X-User-Type"
    header_pattern = "^(admin|tester)$"
  }
}
`
}

// Config for upstream with cookie-based traffic shaping (single resource)
func testAccResourceRpaasUpstreamTrafficShapingCookieSingleConfig() string {
	return `
resource "rpaas_upstream" "test" {
  instance     = "my-rpaas"
  service_name = "rpaasv2-be"
  app          = "test-app"
  load_balance = "round_robin"

  traffic_shaping_policy {
    cookie = "x-session-id"
  }
}
`
}
