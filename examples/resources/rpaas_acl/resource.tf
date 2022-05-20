resource "rpaas_acl" "example" {
  service_name = "rpaasv2-be"
  instance     = "my-rpaas"

  host = "example.com"
  port = 443
}
