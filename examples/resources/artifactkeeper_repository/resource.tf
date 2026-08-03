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

# A Debian remote that proxies only bookworm main/amd64, with curation enforced.
resource "artifactkeeper_repository" "debian_proxy" {
  key          = "debian-remote"
  name         = "Debian (proxy)"
  format       = "debian"
  repo_type    = "remote"
  upstream_url = "https://deb.debian.org/debian"

  quarantine_enabled          = true
  quarantine_duration_minutes = 60

  curation_enabled        = true
  curation_default_action = "review"

  debian = {
    distribution_paths = ["bookworm"]
    components         = ["main"]
    architectures      = ["amd64"]
  }
}

# Import an existing repository by its key:
#   terraform import artifactkeeper_repository.docker_local docker-local
