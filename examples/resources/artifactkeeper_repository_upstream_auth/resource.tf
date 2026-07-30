# Credentials a remote repository uses to reach its upstream. Write-only: the
# API never returns them, so every apply re-sends username/password.
resource "artifactkeeper_repository_upstream_auth" "npm" {
  repository_key = artifactkeeper_repository.npm_proxy.key
  auth_type      = "basic"
  username       = var.upstream_username
  password       = var.upstream_password
}

variable "upstream_username" { type = string }
variable "upstream_password" {
  type      = string
  sensitive = true
}
