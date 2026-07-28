# A remote Artifact Keeper instance this one can proxy to. The api_key is not
# returned by the API, and there is no update endpoint, any change replaces it.
resource "artifactkeeper_remote_instance" "mirror" {
  name    = "eu-mirror"
  url     = "https://ak-eu.example.com"
  api_key = var.mirror_api_key
}

variable "mirror_api_key" {
  type      = string
  sensitive = true
}
