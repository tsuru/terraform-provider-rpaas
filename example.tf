provider "rpaas" {

}

provider "tsuru" {

}

resource "tsuru_service_instance" "my-rpaas" {
    service_name = "rpaasv2-be"
    name = "my-rpaas"
}

resource "rpaas_autoscale" "be-autoscale" {
    service_name = tsuru_service_instance.my-rpaas.service_name
    instance = tsuru_service_instance.my-rpaas.name

    minimum_replicas = 10
    maximum_replicas = 40

    cpu_threshold = 60
    memory_threshold = "128Mb"
}

resource "rpaas_certificate" "be-custom-certificate" {
    service_name = tsuru_service_instance.my-rpaas.service_name
    instance = tsuru_service_instance.my-rpaas.name

    name = "my-certificate"
    cert = file("cert.pem")
    key = file("key.pem")
}

resource "rpaas_block" "be-http-block" {
    service_name = tsuru_service_instance.my-rpaas.service_name
    instance = tsuru_service_instance.my-rpaas.name

    name = "http" # One of [root, http, server, lua-server, lua-worker]
    content = <<EOT
    # custom http block
    EOT
}

resource "rpaas_route" "be-route-default" {
    service_name = tsuru_service_instance.my-rpaas.service_name
    instance = tsuru_service_instance.my-rpaas.name

    path = "/"
    destination = "app.test.tsuru.io"
    force_https = true
}

resource "rpaas_route" "be-route-custom" {
    service_name = tsuru_service_instance.my-rpaas.service_name
    instance = tsuru_service_instance.my-rpaas.name

    path = "/custom"
    content = <<EOT
    # nginx config
    EOT
    force_https = true
}
