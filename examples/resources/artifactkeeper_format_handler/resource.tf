# Container images go through Harbor, so disable the OCI/Docker format here.
resource "artifactkeeper_format_handler" "docker" {
  format_key = "docker"
  enabled    = false
}
