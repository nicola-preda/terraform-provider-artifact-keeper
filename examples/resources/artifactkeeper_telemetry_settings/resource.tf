resource "artifactkeeper_telemetry_settings" "this" {
  enabled            = true
  review_before_send = true
  scrub_level        = "standard"
  include_logs       = false
}
