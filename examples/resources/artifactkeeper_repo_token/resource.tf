# A repository-scoped access token for a CI pipeline, restricted to the
# `docker-local` repository. Scopes are required.
resource "artifactkeeper_repo_token" "ci" {
  repository_key  = artifactkeeper_repository.docker_local.key
  name            = "ci-pipeline"
  scopes          = ["read:artifacts", "write:artifacts"]
  expires_in_days = 90 # omit for a non-expiring token
  description     = "CI/CD pipeline token"
}

# The plaintext token is only available at creation.
output "ci_repo_token" {
  value     = artifactkeeper_repo_token.ci.token
  sensitive = true
}

# Import an existing repository token by "<repository_key>/<token_id>":
#   terraform import artifactkeeper_repo_token.ci docker-local/8f3c9e2a-1b4d-4c6e-9f0a-2d5e7c8b1a3f
# The plaintext token cannot be recovered on import.
