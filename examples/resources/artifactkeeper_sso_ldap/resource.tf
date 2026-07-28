resource "artifactkeeper_sso_ldap" "corp" {
  name          = "corp-ad"
  server_url    = "ldaps://ldap.example.com:636"
  bind_dn       = "cn=svc-artifactkeeper,ou=service,dc=example,dc=com"
  bind_password = var.ldap_bind_password
  user_base_dn  = "ou=people,dc=example,dc=com"
  user_filter   = "(uid=%s)"
  is_enabled    = true
}

variable "ldap_bind_password" {
  type      = string
  sensitive = true
}
