# System settings are a singleton: exactly one record always exists on the
# server. Declare a single instance to manage the tunable values.
resource "artifactkeeper_system_settings" "this" {
  retention_days               = 365
  audit_retention_days         = 90
  backup_retention_count       = 10
  edge_stale_threshold_minutes = 5
}
