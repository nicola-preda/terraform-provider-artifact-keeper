# Serve only vetted packages from conda-forge through the virtual channel.
resource "artifactkeeper_repository_allowlist" "conda" {
  repository_key = "conda-virtual"
  enabled        = true
  entries = [
    { name = "numpy", version = ">=1.26,<3" },
    { name = "pandas" },
    { name = "pytorch", subdirs = ["linux-64"] },
  ]
}
