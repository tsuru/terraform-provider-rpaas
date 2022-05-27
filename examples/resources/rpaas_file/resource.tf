resource "rpaas_file" "foo.txt" {
  service_name = "rpaasv2-be"
  instance     = "my-rpaas"

  name    = "foo.txt"
  content = <<-EOF
    content of the file
  EOF
}

resource "rpaas_file" "image_1" {
  service_name = "rpaasv2-be"
  instance     = "my-rpaas"

  name           = "image.png"
  content_base64 = filebase64("${path.module}/image.png")
}

resource "rpaas_file" "example.txt" {
  service_name = "rpaasv2-be"
  instance     = "my-rpaas"

  name    = "example.txt"
  content = file("${path.module}/example.txt")
}
