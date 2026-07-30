# Mirror an upstream PyPI project on a hosted repository: the local "acme-sdk"
# project tracks its releases from pypi.org's Simple index.
resource "artifactkeeper_repository_pypi_track" "acme" {
  repository_key = artifactkeeper_repository.pypi_local.key
  project        = "acme-sdk"
  tracks_url     = "https://pypi.org/simple/acme-sdk/"
}
