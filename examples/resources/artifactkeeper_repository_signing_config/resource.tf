# Sign this repository's metadata and packages with a repository-scoped key,
# and require artifacts to carry a valid signature.
resource "artifactkeeper_signing_key" "docker_local" {
  repository_id = artifactkeeper_repository.docker_local.id
  name          = "docker-local signing key"
}

resource "artifactkeeper_repository_signing_config" "docker_local" {
  repository_id      = artifactkeeper_repository.docker_local.id
  signing_key_id     = artifactkeeper_signing_key.docker_local.id
  sign_metadata      = true
  sign_packages      = true
  require_signatures = true
}
