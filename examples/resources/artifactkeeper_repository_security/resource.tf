# Enable vulnerability scanning on a repository: on upload for hosted repos, on
# proxy fetch for remotes, and block anything at or above the severity threshold.
resource "artifactkeeper_repository_security" "scanned" {
  repository_key            = artifactkeeper_repository.npm_proxy.key
  scan_enabled              = true
  scan_on_upload            = true
  scan_on_proxy             = true
  block_on_policy_violation = true
  severity_threshold        = "high"
}
