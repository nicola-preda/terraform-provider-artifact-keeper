# Link a staging repository to the release repository that artifacts promote
# into. Omit release_repository_key to leave the staging repo unlinked.
resource "artifactkeeper_repository_release_target" "staging" {
  repository_key         = artifactkeeper_repository.docker_staging.key
  release_repository_key = artifactkeeper_repository.docker_release.key
}
