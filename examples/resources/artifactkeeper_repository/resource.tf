# A local Docker repository.
resource "artifactkeeper_repository" "docker_local" {
  key       = "docker-local"
  name      = "Docker (local)"
  format    = "docker"
  repo_type = "local"
}

# A remote (pull-through cache) npm repository, publicly readable.
resource "artifactkeeper_repository" "npm_proxy" {
  key          = "npm-remote"
  name         = "npm (proxy)"
  format       = "npm"
  repo_type    = "remote"
  upstream_url = "https://registry.npmjs.org"
  is_public    = true
  quota_bytes  = 53687091200 # 50 GiB
}

# Import an existing repository by its key:
#   terraform import artifactkeeper_repository.docker_local docker-local
