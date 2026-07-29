# Install a WASM format plugin from git, enabled, with config.
resource "artifactkeeper_plugin" "cyclonedx" {
  source_git_url = "https://github.com/artifact-keeper/plugin-cyclonedx.git"
  source_git_ref = "v1.2.0"
  enabled        = true

  config = jsonencode({
    strict_mode = true
    max_depth   = 5
  })
}
