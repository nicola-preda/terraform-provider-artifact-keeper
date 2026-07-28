# A service account is a machine identity that owns API tokens. Requires an
# admin token. The server derives the immutable username (svc-ci-runner) from
# name; changing name replaces the account.
resource "artifactkeeper_service_account" "ci" {
  name         = "ci-runner"
  display_name = "CI Runner"
  is_active    = true
}

output "ci_username" {
  value = artifactkeeper_service_account.ci.username
}
