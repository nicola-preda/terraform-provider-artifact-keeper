# Only ever fetch release tarballs from the upstream, never installers.
resource "artifactkeeper_repository_upstream_filter" "downloads" {
  repository_key   = "downloads-remote"
  include_patterns = ["^releases/"]
  exclude_patterns = ["\\.(exe|msi)$"]
}
