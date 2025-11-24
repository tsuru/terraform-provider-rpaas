# rpaas_upstream Resource

Manages upstream options for an RPaaS instance, including traffic shaping, load balancing, and canary deployments.

## Example Usage

### Basic Upstream Configuration

```terraform
resource "rpaas_upstream" "app" {
  instance     = "my-rpaas-instance"
  service_name = "my-rpaas-service"
  app          = "my-app"
  load_balance = "round_robin"
}
```

### Canary Deployment Example

**Important**: For canary deployments, you must create the canary app upstream configuration **first**, then reference it from the primary app. The `traffic_shaping_policy` is only applied to the **canary** app, not the primary app.

```terraform
# Canary app upstream (with traffic shaping policy) - CREATE FIRST
resource "rpaas_upstream" "app_canary" {
  instance     = "my-rpaas-instance"
  service_name = "my-rpaas-service"
  app          = "my-app-canary"
  load_balance = "round_robin"

  traffic_shaping_policy {
    weight       = 10
    weight_total = 100
  }
}

# Primary app upstream (references canary) - CREATE SECOND
resource "rpaas_upstream" "app_primary" {
  instance     = "my-rpaas-instance"
  service_name = "my-rpaas-service"
  app          = "my-app"
  canary       = ["my-app-canary"]
  load_balance = "round_robin"

  depends_on = [rpaas_upstream.app_canary]
}
```

### Header-based Traffic Routing

```terraform
# Beta app with header-based routing - CREATE FIRST
resource "rpaas_upstream" "app_beta" {
  instance     = "my-rpaas-instance"
  service_name = "my-rpaas-service"
  app          = "my-app-beta"
  load_balance = "round_robin"

  traffic_shaping_policy {
    header       = "X-Beta-User"
    header_value = "true"
  }
}

# Primary app (references beta) - CREATE SECOND
resource "rpaas_upstream" "app_primary" {
  instance     = "my-rpaas-instance"
  service_name = "my-rpaas-service"
  app          = "my-app"
  canary       = ["my-app-beta"]
  load_balance = "round_robin"

  depends_on = [rpaas_upstream.app_beta]
}
```

### Upstream with Consistent Hashing

```terraform
resource "rpaas_upstream" "consistent_hash" {
  instance               = "my-rpaas-instance"
  service_name          = "my-rpaas-service"
  app                   = "my-app"
  load_balance          = "chash"
  load_balance_hash_key = "$remote_addr"
}
```

## Argument Reference

* `instance` - (Required) RPaaS Instance Name
* `service_name` - (Required) RPaaS Service Name  
* `app` - (Required) Primary app name
* `canary` - (Optional) Canary app names that participate in traffic distribution
* `load_balance` - (Optional) Load balancing algorithm. Valid values: `round_robin`, `chash`, `ewma`
* `load_balance_hash_key` - (Optional) Nginx variable for consistent hashing when `load_balance` is `chash`. Required when `load_balance` is `chash`
* `traffic_shaping_policy` - (Optional) Traffic shaping policy configuration block. **Only used for canary apps**

### traffic_shaping_policy

**Note**: Traffic shaping policies should only be applied to canary apps, not primary apps.

* `weight` - (Optional) Weight for weight-based routing (only for canary apps)
* `weight_total` - (Optional) Total weight for weight-based routing
* `header` - (Optional) Header on which to redirect requests to this backend
* `header_value` - (Optional) Header value on which to redirect requests to this backend (mutually exclusive with `header_pattern`)
* `header_pattern` - (Optional) Header value match pattern using regex (mutually exclusive with `header_value`)
* `cookie` - (Optional) Cookie on which to redirect requests to this backend

## Attributes Reference

In addition to all arguments above, the following attributes are exported:

* `id` - The ID of the upstream resource in the format `service::instance::app`

## Import

Upstream resources can be imported using the format `service::instance::app`:

```bash
terraform import rpaas_upstream.example my-service::my-instance::my-app
```

## Important Notes

1. **Canary Deployments**: Both primary and canary apps must exist as separate upstream configurations
2. **Traffic Shaping**: Only apply `traffic_shaping_policy` to canary apps, not primary apps
3. **Dependencies**: Canary apps must be created **before** the primary app that references them. Use `depends_on` to ensure canary app exists before creating primary app
4. **Header Routing**: `header_value` and `header_pattern` are mutually exclusive