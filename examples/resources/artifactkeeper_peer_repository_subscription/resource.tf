# Subscribe a repository to a peer instance for replication. The (peer,
# repository) pair is the subscription identity: changing peer_id or
# repository_id forces a new subscription, while the replication settings below
# are updated in place.
resource "artifactkeeper_peer_repository_subscription" "oslo_docker" {
  peer_id       = artifactkeeper_peer.oslo.id
  repository_id = artifactkeeper_repository.docker_local.id

  sync_enabled         = true
  replication_mode     = "pull" # one of: push, pull, mirror, none
  replication_schedule = "0 */6 * * *"

  # Optional JSON filter; omit to replicate everything.
  replication_filter = jsonencode({
    include_patterns = ["^v\\d+\\."]
    exclude_patterns = [".*-SNAPSHOT$"]
  })
}

# Import an existing subscription by "<peer_id>/<repository_id>":
#   terraform import artifactkeeper_peer_repository_subscription.oslo_docker <peer_id>/<repository_id>
