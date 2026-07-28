# A project groups repositories under a shared, URL-safe key.
resource "artifactkeeper_project" "payments" {
  key         = "payments"
  name        = "Payments"
  description = "Repositories owned by the Payments team."
  quota_bytes = 10737418240
}
