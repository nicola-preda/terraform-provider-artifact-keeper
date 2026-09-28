# Require an expiration on newly minted API tokens, between a week and six
# months, defaulting to 90 days when the caller does not ask for one. Tokens
# that already exist are untouched: the policy is evaluated at mint time only.
# Service accounts stay exempt, because expiring a CI credential is an outage
# on a schedule.
resource "artifactkeeper_token_policy" "this" {
  require_expiration = true
  min_days           = 7
  max_days           = 180
  default_days       = 90
}
