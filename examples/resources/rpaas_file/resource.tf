resource "rpaas_file" "example_1" {
  service_name = "rpaasv2-be"
  instance     = "my-rpaas"

  name    = "foo.txt"
  content = <<-EOF
    content of the file
  EOF
}

resource "rpaas_file" "example_2" {
  service_name = "rpaasv2-be"
  instance     = "my-rpaas"

  name    = "example.txt"
  content = file("example.txt")
}
