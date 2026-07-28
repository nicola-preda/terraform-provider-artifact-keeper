# An identity mapping under a CI OIDC provider. CI JWTs whose `ref` claim is
# `refs/heads/main` authenticate as this mapping's stable service account.
resource "artifactkeeper_ci_oidc_identity_mapping" "main_branch" {
  provider_id = artifactkeeper_ci_oidc_provider.gitlab_ci.id
  name        = "main-branch"
  priority    = 10
  claim_filters = jsonencode({
    ref = "refs/heads/main"
  })
  is_enabled = true
}

# A mapping restricted to specific repositories, matching any of several refs
# (array value = any-of match).
resource "artifactkeeper_ci_oidc_identity_mapping" "release_branches" {
  provider_id = artifactkeeper_ci_oidc_provider.gitlab_ci.id
  name        = "release-branches"
  priority    = 20
  claim_filters = jsonencode({
    ref = ["refs/heads/release", "refs/heads/hotfix"]
  })
  allowed_repo_ids = [
    artifactkeeper_repository.docker_local.id,
  ]
}
