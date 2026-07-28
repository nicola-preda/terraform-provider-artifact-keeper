# System settings are a singleton: exactly one record always exists on the
# server. Declare a single instance to manage the tunable values.
resource "artifactkeeper_system_settings" "this" {
  allow_anonymous_download     = false
  max_upload_size_bytes        = 104857600 # 100 MiB
  retention_days               = 365
  audit_retention_days         = 90
  backup_retention_count       = 10
  edge_stale_threshold_minutes = 5
}
