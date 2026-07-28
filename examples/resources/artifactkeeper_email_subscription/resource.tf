# An email subscription that delivers artifact and scan events on the
# `docker-local` repository to the ops mailing list. Subscriptions are
# immutable; any change forces a new subscription.
resource "artifactkeeper_email_subscription" "ops" {
  repository_key = artifactkeeper_repository.docker_local.key
  recipients     = ["ops@example.com", "oncall@example.com"]
  event_types    = ["artifact.uploaded", "scan.completed", "scan.failed"]
  enabled        = true # optional; defaults to true
}

# Import an existing email subscription by "<repository_key>/<subscription_id>":
#   terraform import artifactkeeper_email_subscription.ops docker-local/8f3c9e2a-1b4d-4c6e-9f0a-2d5e7c8b1a3f
