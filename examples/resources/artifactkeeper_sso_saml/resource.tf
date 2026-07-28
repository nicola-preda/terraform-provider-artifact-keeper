resource "artifactkeeper_sso_saml" "okta" {
  name        = "okta"
  entity_id   = "https://idp.example.com/saml/metadata"
  sso_url     = "https://idp.example.com/saml/sso"
  certificate = file("${path.module}/idp.pem")
  is_enabled  = true
}
