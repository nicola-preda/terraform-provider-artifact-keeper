# Cache upstream artifacts for one hour before the remote repository re-fetches them.
resource "artifactkeeper_repository_cache_ttl" "npm_proxy" {
  repository_key    = artifactkeeper_repository.npm_proxy.key
  cache_ttl_seconds = 3600
}
