# A global security gate: block anything with a high-or-worse finding, block
# unscanned artifacts, and require a signature. Managing policies needs an admin
# token.
resource "artifactkeeper_security_policy" "global_gate" {
  name              = "global-gate"
  max_severity      = "high"
  block_on_fail     = true
  block_unscanned   = true
  require_signature = true
}

# A policy scoped to one repository, allowing medium findings and requiring a
# 24h staging soak. repository_id is immutable, changing it replaces the policy.
resource "artifactkeeper_security_policy" "docker_release" {
  name              = "docker-release-gate"
  repository_id     = artifactkeeper_repository.docker_local.id
  max_severity      = "medium"
  block_on_fail     = true
  min_staging_hours = 24
  is_enabled        = true
}
