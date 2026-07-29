# A migration job scoped to a single repository. Terraform creates it in a
# pending state; start it yourself (UI, or POST /api/v1/migrations/{id}/start).
resource "artifactkeeper_migration_job" "myrepo_npm" {
  source_connection_id = artifactkeeper_migration_source.nexus.id
  include_repos        = ["myrepo-npm"]

  # Proxy/cached-remote artifacts are excluded by default; hosted repos only.
  include_cached_remote = false
  conflict_resolution   = "skip"
}
