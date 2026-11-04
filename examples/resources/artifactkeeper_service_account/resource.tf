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

# An administrator service account, e.g. for the automation that applies this
# configuration. Its token only acts as an administrator when minted with the
# admin scope.
resource "artifactkeeper_service_account" "automation" {
  name         = "automation"
  display_name = "Configuration automation"
  is_admin     = true
}

resource "artifactkeeper_service_account_token" "automation" {
  service_account_id = artifactkeeper_service_account.automation.id
  name               = "automation-token"
  scopes             = ["admin"]
}
