# Announce a maintenance window in the web UI for its duration.
resource "artifactkeeper_system_banner" "upgrade" {
  title     = "Scheduled maintenance"
  message   = "Artifact Keeper is upgraded tonight; pulls may be slow for a few minutes."
  severity  = "warning"
  target    = "ui"
  link_url  = "https://status.example.com"
  starts_at = "2026-10-20T20:00:00Z"
  ends_at   = "2026-10-20T21:00:00Z"
}
