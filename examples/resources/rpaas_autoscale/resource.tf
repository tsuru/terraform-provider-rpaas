resource "rpaas_autoscale" "example" {
  service_name = "rpaasv2-be"
  instance     = "my-rpaas"

  min_replicas                      = 3
  max_replicas                      = 10
  target_cpu_utilization_percentage = 50
}
