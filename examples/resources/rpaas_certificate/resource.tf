resource "rpaas_certificate" "example" {
  service_name = "rpaasv2-be"
  instance     = "my-rpaas"

  name        = "localhost"
  certificate = file("certificate.crt")
  key         = file("certificate.key")
}
