resource "rpaas_cert_manager" "example" {
  service_name = "rpaasv2-be"
  instance     = "my-rpaas"

  certificate_name = "example.com"
  issuer           = "custom-issuer.ClusterIssuer.local"
  dns_names        = ["example.com"]
}
