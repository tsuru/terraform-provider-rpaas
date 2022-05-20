resource "rpaas_route" "example" {
  service_name = "rpaasv2-be"
  instance     = "my-rpaas"

  path        = "/"
  destination = "app.test.tsuru.io"
  force_https = true
}
