provider "rpaas" {}
provider "tsuru" {}

resource "tsuru_service_instance" "my_rpaas" {
  service_name = "rpaasv2-be"
  name         = "my-rpaas"
}

resource "rpaas_autoscale" "be_autoscale" {
  service_name = tsuru_service_instance.my_rpaas.service_name
  instance     = tsuru_service_instance.my_rpaas.name

  min_replicas = 10
  max_replicas = 40

  target_cpu_utilization_percentage = 60
}

resource "rpaas_certificate" "be_custom_certificate" {
  service_name = tsuru_service_instance.my_rpaas.service_name
  instance     = tsuru_service_instance.my_rpaas.name

  name = "my-certificate"
  cert = file("cert.pem")
  key  = file("key.pem")
}

resource "rpaas_block" "be_http_block" {
  service_name = tsuru_service_instance.my_rpaas.service_name
  instance     = tsuru_service_instance.my_rpaas.name

  name    = "http" # One of [root, http, server, lua-server, lua-worker]
  content = <<EOT
    # custom http block
    EOT
}

resource "rpaas_route" "be_route_default" {
  service_name = tsuru_service_instance.my_rpaas.service_name
  instance     = tsuru_service_instance.my_rpaas.name

  path        = "/"
  destination = "app.test.tsuru.io"
  force_https = true
}

resource "rpaas_route" "be_route_custom" {
  service_name = tsuru_service_instance.my_rpaas.service_name
  instance     = tsuru_service_instance.my_rpaas.name

  path        = "/custom"
  content     = <<EOT
    # nginx config
    EOT
  force_https = true
}
