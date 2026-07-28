# A connection to a legacy Nexus to import from. Credentials aren't returned by
# the API, and there's no update endpoint, changing any field replaces it.
resource "artifactkeeper_migration_source" "nexus" {
  name        = "legacy-nexus"
  url         = "https://nexus.example.com"
  source_type = "nexus"
  auth_type   = "basic_auth"

  credential_username = var.nexus_username
  credential_password = var.nexus_password
}

variable "nexus_username" { type = string }
variable "nexus_password" {
  type      = string
  sensitive = true
}
