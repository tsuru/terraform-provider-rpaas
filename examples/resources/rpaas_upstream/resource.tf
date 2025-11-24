# Basic upstream configuration
resource "rpaas_upstream" "app_primary" {
  instance     = "my-rpaas-instance"
  service_name = "my-rpaas-service"
  app          = "my-app"
  load_balance = "round_robin"
}

# Canary deployment example (canary app with traffic shaping) - CREATE FIRST
resource "rpaas_upstream" "canary_app" {
  instance     = "my-rpaas-instance"
  service_name = "my-rpaas-service"
  app          = "my-app-canary"
  load_balance = "round_robin"

  traffic_shaping_policy {
    weight       = 10
    weight_total = 100
  }
}

# Canary deployment example (primary app references canary) - CREATE SECOND
resource "rpaas_upstream" "production_app" {
  instance     = "my-rpaas-instance"
  service_name = "my-rpaas-service"
  app          = "my-app-production"
  canary       = ["my-app-canary"]
  load_balance = "round_robin"

  depends_on = [rpaas_upstream.canary_app]
}

# Header-based routing example (beta app) - CREATE FIRST
resource "rpaas_upstream" "beta_app" {
  instance     = "my-rpaas-instance"
  service_name = "my-rpaas-service"
  app          = "my-app-beta"
  load_balance = "round_robin"

  traffic_shaping_policy {
    header       = "X-Beta-User"
    header_value = "true"
  }
}

# Primary app that references beta app - CREATE SECOND
resource "rpaas_upstream" "main_with_beta" {
  instance     = "my-rpaas-instance"
  service_name = "my-rpaas-service"
  app          = "my-app-main"
  canary       = ["my-app-beta"]
  load_balance = "round_robin"

  depends_on = [rpaas_upstream.beta_app]
}

# Example with consistent hashing
resource "rpaas_upstream" "hash_example" {
  instance               = "my-rpaas-instance"
  service_name          = "my-rpaas-service"
  app                   = "my-app-sticky"
  load_balance          = "chash"
  load_balance_hash_key = "$remote_addr"
}