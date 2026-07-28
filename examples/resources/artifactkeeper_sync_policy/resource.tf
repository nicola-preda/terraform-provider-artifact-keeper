# Replicate production Docker/Maven repositories to every peer instance. The
# repo_selector, peer_selector, and artifact_filter blocks combine their active
# criteria with AND semantics. `filter` is a computed mirror of
# artifact_filter.include_paths.
resource "artifactkeeper_sync_policy" "prod_to_all_peers" {
  name             = "prod-to-all-peers"
  description      = "Mirror production artifacts to all peers"
  enabled          = true
  replication_mode = "mirror"
  priority         = 5
  precedence       = 10

  repo_selector = {
    match_labels  = { env = "prod" }
    match_formats = ["docker", "maven"]
    match_pattern = "libs-*"
  }

  peer_selector = {
    all = true
  }

  artifact_filter = {
    max_age_days  = 30
    include_paths = ["release/*"]
  }
}
