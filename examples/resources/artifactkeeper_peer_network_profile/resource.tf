# Configure the transfer scheduling and bandwidth profile for a peer instance.
# This is a singleton per peer: peer_id is the identity, so there is at most one
# artifactkeeper_peer_network_profile per artifactkeeper_peer. The API has no
# endpoint to read these fields back, so Terraform preserves the configured
# values in state and cannot detect out-of-band drift on them.
resource "artifactkeeper_peer_network_profile" "oslo" {
  peer_id = artifactkeeper_peer.oslo.id

  max_bandwidth_bps          = 100000000 # 100 MB/s
  sync_window_start          = "02:00:00"
  sync_window_end            = "06:00:00"
  sync_window_timezone       = "America/New_York"
  concurrent_transfers_limit = 4
}

# Import an existing profile by the peer UUID:
#   terraform import artifactkeeper_peer_network_profile.oslo <peer_id>
