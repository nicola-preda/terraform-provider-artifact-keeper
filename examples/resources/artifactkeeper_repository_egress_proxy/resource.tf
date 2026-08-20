# Send this repository's upstream fetches through a corporate proxy, bypassing it
# for internal hosts. The per-repository setting replaces HTTP_PROXY/NO_PROXY for
# this repository rather than merging with them.
resource "artifactkeeper_repository_egress_proxy" "npm_proxy" {
  repository_key = artifactkeeper_repository.npm_proxy.key
  mode           = "explicit"
  proxy_url      = "http://proxy.internal:3128"
  no_proxy       = "localhost,.internal,10.0.0.0/8"
}

# Let one repository out directly while the rest of the deployment stays proxied.
resource "artifactkeeper_repository_egress_proxy" "pypi_proxy" {
  repository_key = artifactkeeper_repository.pypi_proxy.key
  mode           = "direct"
}
