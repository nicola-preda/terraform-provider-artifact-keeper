# An outbound webhook. Requires an admin token to create. Only is_enabled can
# change in place; any other change replaces the webhook. If secret is omitted
# the server generates one that is not retrievable afterwards.
resource "artifactkeeper_webhook" "ci_notify" {
  name   = "ci-notify"
  url    = "https://hooks.example.com/artifact-keeper"
  events = ["artifact_uploaded", "build_failed"]
  secret = var.webhook_secret

  payload_template = "slack"

  headers = {
    "X-Environment" = "production"
  }

  is_enabled = true
}

variable "webhook_secret" {
  type      = string
  sensitive = true
}
