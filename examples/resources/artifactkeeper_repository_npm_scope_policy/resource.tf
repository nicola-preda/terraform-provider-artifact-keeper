# Restrict an npm remote (a member of an npm virtual repo) to the @myorg scope,
# and disallow unscoped packages. Omit allowed_scopes to impose no restriction.
resource "artifactkeeper_repository_npm_scope_policy" "scoped" {
  repository_key = artifactkeeper_repository.npm_remote.key
  allowed_scopes = ["@myorg"]
  allow_unscoped = false
}
