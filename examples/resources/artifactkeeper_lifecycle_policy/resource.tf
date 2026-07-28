# Delete artifacts older than 90 days in one repository.
resource "artifactkeeper_lifecycle_policy" "expire_old" {
  repository_id = artifactkeeper_repository.docker_local.id
  name          = "expire-old"
  description   = "Remove artifacts older than 90 days."
  policy_type   = "max_age_days"
  config        = jsonencode({ max_age_days = 90 })
  priority      = 10
}

# Keep only the latest 5 versions per package, on a nightly cron.
resource "artifactkeeper_lifecycle_policy" "keep_recent" {
  repository_id = artifactkeeper_repository.docker_local.id
  name          = "keep-recent"
  policy_type   = "max_versions"
  config        = jsonencode({ max_versions = 5 })
  cron_schedule = "0 3 * * *"
  enabled       = true
}

# Import an existing lifecycle policy by its UUID:
#   terraform import artifactkeeper_lifecycle_policy.expire_old 550e8400-e29b-41d4-a716-446655440000
