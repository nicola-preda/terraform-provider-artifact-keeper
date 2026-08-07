# Label a peer instance so sync policies can target it. A label is identified by
# the peer UUID plus the label key; changing either forces a new label, while the
# value can be updated in place. The backend re-evaluates sync policies on every
# label write.
resource "artifactkeeper_peer_instance_label" "region" {
  peer_id = artifactkeeper_peer.osl1.id
  key     = "region"
  value   = "osl1"
}

# A key-only label (value defaults to an empty string).
resource "artifactkeeper_peer_instance_label" "dr" {
  peer_id = artifactkeeper_peer.osl1.id
  key     = "dr-target"
}
