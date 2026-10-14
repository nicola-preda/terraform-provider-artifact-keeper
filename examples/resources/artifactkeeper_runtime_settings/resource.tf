# Refuse anonymous access server-wide, without a restart.
resource "artifactkeeper_runtime_settings" "this" {
  guest_access_enabled = false
}
