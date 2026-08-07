# Mint a token for another user (admin-only). For the provider's own credential
# use artifactkeeper_api_token, and for machine identities use
# artifactkeeper_service_account_token.
resource "artifactkeeper_user_api_token" "release_bot_readonly" {
  user_id         = artifactkeeper_user.release_bot.id
  name            = "read-only artifact access"
  scopes          = ["read:artifacts", "read:repositories"]
  expires_in_days = 90
}

output "release_bot_token" {
  value     = artifactkeeper_user_api_token.release_bot_readonly.token
  sensitive = true
}
