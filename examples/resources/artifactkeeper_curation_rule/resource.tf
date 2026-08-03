# Block a package by name across every repository (global rule).
resource "artifactkeeper_curation_rule" "block_telnet" {
  package_pattern = "telnet*"
  action          = "block"
  reason          = "Cleartext protocol; disallowed by policy."
}

# Allow a specific package/version into one staging repository.
resource "artifactkeeper_curation_rule" "allow_nginx" {
  staging_repo_id    = artifactkeeper_repository.rpm_staging.id
  package_pattern    = "nginx"
  version_constraint = ">= 1.24"
  architecture       = "x86_64"
  action             = "allow"
  priority           = 10
  reason             = "Approved web server build."
  enabled            = true
}

# A typed "popularity" rule: block low-download and typosquatting packages.
resource "artifactkeeper_curation_rule" "popularity_gate" {
  staging_repo_id = artifactkeeper_repository.rpm_staging.id
  package_pattern = "*"
  action          = "review"
  reason          = "Quarantine unpopular or typosquatting packages for review."
  rule_type       = "popularity"
  config = jsonencode({
    min_downloads   = 1000
    typosquat_check = true
    action          = "review"
  })
}

# Import an existing curation rule by its UUID:
#   terraform import artifactkeeper_curation_rule.allow_nginx 550e8400-e29b-41d4-a716-446655440000
