# Label a repository with key/value metadata (used by sync policies, etc.). A
# label is identified by the repository key plus the label key; changing either
# forces a new label, while the value can be updated in place.
resource "artifactkeeper_repository_label" "env" {
  repository_key = artifactkeeper_repository.docker_local.key
  key            = "env"
  value          = "production"
}

# A key-only label (value defaults to an empty string).
resource "artifactkeeper_repository_label" "tier" {
  repository_key = artifactkeeper_repository.docker_local.key
  key            = "tier"
}
