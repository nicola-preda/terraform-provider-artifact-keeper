# Bootstrapping a tenant: a team, a member, two repositories, an access grant,
# and a CI token, a realistic starting point.

terraform {
  required_providers {
    artifactkeeper = {
      source  = "nicola-preda/artifact-keeper"
      version = "~> 1.8.0"
    }
  }
}

variable "artifact_keeper_token" {
  type      = string
  sensitive = true
}

provider "artifactkeeper" {
  endpoint = "https://artifact-keeper.example.com"
  token    = var.artifact_keeper_token
}

resource "artifactkeeper_group" "platform" {
  name        = "platform"
  description = "Platform engineering"
}

resource "artifactkeeper_user" "alice" {
  username     = "alice"
  email        = "alice@example.com"
  display_name = "Alice Example"
}

# A local Docker repository and a pull-through npm cache.
resource "artifactkeeper_repository" "docker" {
  key       = "docker-local"
  name      = "Docker (local)"
  format    = "docker"
  repo_type = "local"
}

resource "artifactkeeper_repository" "npm" {
  key          = "npm-remote"
  name         = "npm (proxy)"
  format       = "npm"
  repo_type    = "remote"
  upstream_url = "https://registry.npmjs.org"
  is_public    = true
}

# Let the platform team read and write the Docker repo.
resource "artifactkeeper_permission" "platform_docker" {
  principal_type = "group"
  principal_id   = artifactkeeper_group.platform.id
  target_type    = "repository"
  target_id      = artifactkeeper_repository.docker.id
  actions        = ["read", "write"]
}

# A token for CI to pull and push artifacts.
resource "artifactkeeper_api_token" "ci" {
  name            = "ci-pipeline"
  scopes          = ["read:artifacts", "write:artifacts"]
  expires_in_days = 90
}

output "ci_token" {
  value     = artifactkeeper_api_token.ci.token
  sensitive = true
}
