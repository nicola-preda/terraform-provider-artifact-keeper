# The age gate applies only to remote (pull-through) repositories. Hold newly
# published upstream versions for 14 days before serving them.
resource "artifactkeeper_age_gate" "npm_remote" {
  repository_key = "npm-remote"
  enabled        = true
  min_age_days   = 14
}
