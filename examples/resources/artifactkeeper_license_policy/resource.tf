# A global license policy: allow permissive licenses, deny copyleft, and block
# on violation. Managing policies needs an admin token. The API has no update
# endpoint (POST is an upsert), so changing any field replaces the policy.
resource "artifactkeeper_license_policy" "global" {
  name             = "global-license-gate"
  allowed_licenses = ["MIT", "Apache-2.0", "BSD-3-Clause"]
  denied_licenses  = ["GPL-3.0-only", "AGPL-3.0-only"]
  allow_unknown    = false
  action           = "block"
}

# A policy scoped to one repository, warning (not blocking) on unknown licenses.
# repository_id is immutable, changing it replaces the policy.
resource "artifactkeeper_license_policy" "docker_licenses" {
  name             = "docker-license-gate"
  repository_id    = artifactkeeper_repository.docker_local.id
  allowed_licenses = ["MIT", "Apache-2.0"]
  denied_licenses  = ["GPL-2.0-only"]
  action           = "warn"
  is_enabled       = true
}
