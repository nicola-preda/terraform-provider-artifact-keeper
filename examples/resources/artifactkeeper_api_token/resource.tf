resource "artifactkeeper_api_token" "ci" {
  name            = "ci-pipeline"
  scopes          = ["read:artifacts", "write:artifacts"]
  expires_in_days = 90 # omit for a non-expiring token
}

# The plaintext token is only available at creation.
output "ci_token" {
  value     = artifactkeeper_api_token.ci.token
  sensitive = true
}
