# An access token owned by a service account (a machine identity), restricted
# to explicit repositories. Requires an admin token. name and scopes are
# required; any change forces a new token.
resource "artifactkeeper_service_account_token" "ci" {
  service_account_id = artifactkeeper_service_account.ci.id
  name               = "ci-pipeline"
  scopes             = ["read:artifacts", "write:artifacts"]
  expires_in_days    = 90 # omit for a non-expiring token
  description        = "CI/CD pipeline token"

  # Restrict to explicit repositories (mutually exclusive with repo_selector):
  repository_ids = [artifactkeeper_repository.docker_local.id]

  # ...or match repositories dynamically (mutually exclusive with repository_ids):
  # repo_selector = jsonencode({ match_formats = ["docker"] })
}

# The plaintext token is only available at creation.
output "ci_service_account_token" {
  value     = artifactkeeper_service_account_token.ci.token
  sensitive = true
}

# Import an existing token by "<service_account_id>/<token_id>":
#   terraform import artifactkeeper_service_account_token.ci 550e8400-e29b-41d4-a716-446655440000/8f3c9e2a-1b4d-4c6e-9f0a-2d5e7c8b1a3f
# The plaintext token cannot be recovered on import.
