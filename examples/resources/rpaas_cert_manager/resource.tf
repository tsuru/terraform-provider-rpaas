resource "rpaas_cert_manager" "example" {
  service_name = "rpaasv2-be"
  instance     = "my-rpaas"

  issuer    = "custom-issuer.ClusterIssuer.local"
  dns_names = ["*.example.com", "my-instance.test"]
}
