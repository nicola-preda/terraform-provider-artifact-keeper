resource "artifactkeeper_sso_oidc" "google" {
  name              = "google"
  issuer_url        = "https://accounts.google.com"
  client_id         = var.oidc_client_id
  client_secret     = var.oidc_client_secret
  scopes            = ["openid", "email", "profile"]
  auto_create_users = true
  is_enabled        = true
}

variable "oidc_client_id" { type = string }
variable "oidc_client_secret" {
  type      = string
  sensitive = true
}
