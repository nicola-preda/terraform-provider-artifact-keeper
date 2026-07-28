# A global quality gate that blocks promotion of low-health artifacts.
resource "artifactkeeper_quality_gate" "baseline" {
  name                = "baseline"
  description         = "Minimum health bar enforced on promotion."
  min_health_score    = 70
  max_critical_issues = 0
  required_checks     = ["metadata_completeness"]
  action              = "block"
}

# A repository-scoped gate that only warns on download.
resource "artifactkeeper_quality_gate" "docker_warn" {
  repository_id        = artifactkeeper_repository.docker_local.id
  name                 = "docker-warn"
  min_security_score   = 80
  max_high_issues      = 5
  enforce_on_promotion = false
  enforce_on_download  = true
  action               = "warn"
  is_enabled           = true
}

# Import an existing quality gate by its UUID:
#   terraform import artifactkeeper_quality_gate.baseline 550e8400-e29b-41d4-a716-446655440000
