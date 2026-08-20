# Require 2FA of every administrator with a local password. The admin whose
# credentials the provider uses must already have TOTP enrolled, or the apply
# fails with a 409.
resource "artifactkeeper_totp_policy" "this" {
  policy = "required_for_admins"
}
