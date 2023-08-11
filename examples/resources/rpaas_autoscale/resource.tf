resource "rpaas_autoscale" "example" {
  service_name = "rpaasv2-be"
  instance     = "my-rpaas"

  min_replicas                      = 3
  max_replicas                      = 10
  target_cpu_utilization_percentage = 50
  target_requests_per_second        = 500

  scheduled_window {
    min_replicas = 5
    start        = "00 20 * * 2"
    end          = "00 01 * * 3"
  }
}
