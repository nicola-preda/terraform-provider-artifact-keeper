# Gate promotion from a staging repository to a release repository. The rule
# only auto-promotes artifacts that clear every configured gate. source_repo_id
# and target_repo_id are immutable, changing either forces a new rule.
resource "artifactkeeper_promotion_rule" "staging_to_release" {
  name           = "staging-to-release"
  source_repo_id = artifactkeeper_repository.docker_staging.id
  target_repo_id = artifactkeeper_repository.docker_release.id

  is_enabled            = true
  max_cve_severity      = "high"
  allowed_licenses      = ["MIT", "Apache-2.0"]
  require_signature     = true
  min_staging_hours     = 48
  max_artifact_age_days = 90
  min_health_score      = 80
  auto_promote          = false
}
