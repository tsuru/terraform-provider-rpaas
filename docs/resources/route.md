---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "rpaas_route Resource - terraform-provider-rpaas"
subcategory: ""
description: |-
  
---

# rpaas_route (Resource)



## Example Usage

```terraform
resource "rpaas_route" "example" {
  service_name = "rpaasv2-be"
  instance     = "my-rpaas"

  path        = "/"
  destination = "app.test.tsuru.io"
  force_https = true
}
```

<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `instance` (String) RPaaS Instance Name
- `path` (String) Path for this route
- `service_name` (String) RPaaS Service Name

### Optional

- `content` (String) Custom Nginx configuration content
- `destination` (String) Custom Nginx upstream destination
- `https_only` (Boolean) Only on https
- `timeouts` (Block, Optional) (see [below for nested schema](#nestedblock--timeouts))

### Read-Only

- `id` (String) The ID of this resource.

<a id="nestedblock--timeouts"></a>
### Nested Schema for `timeouts`

Optional:

- `create` (String)
- `delete` (String)
- `read` (String)
- `update` (String)

## Import

Import is supported using the following syntax:

```shell
terraform import rpaas_route.resource_name "service::instance::path"

# example
terraform import rpaas_route.myroute "rpaasv2-be::my-rpaas::/"
```
