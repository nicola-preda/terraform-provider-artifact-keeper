# Register a remote peer for replication. api_key is an admin-scoped token minted
# on the remote peer. There is no update endpoint, any change replaces the peer.
resource "artifactkeeper_peer" "oslo" {
  name         = "oslo"
  endpoint_url = "https://peer.example.com"
  region       = "osl"
  api_key      = var.peer_api_key
}

variable "peer_api_key" {
  type      = string
  sensitive = true
}
