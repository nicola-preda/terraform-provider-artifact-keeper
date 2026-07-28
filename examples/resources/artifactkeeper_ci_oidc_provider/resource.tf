resource "artifactkeeper_ci_oidc_provider" "gitlab_ci" {
  name          = "gitlab-ci"
  provider_type = "gitlab"
  issuer_url    = "https://gitlab.example.com"
  audience      = "artifact-keeper"
  is_enabled    = true
}
