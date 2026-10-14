package provider

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccV111Surface exercises everything 1.11.0 added to the provider against
// a live instance: create, empty re-plan, an in-place update, and import.
func TestAccV111Surface(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip("acceptance test; set TF_ACC to run")
	}
	testAccPreCheck(t)
	testAccSkipIfEndpointMissing(t, "GET", "/admin/settings/system")

	config := func(visibility, filter, bannerSeverity string) string {
		return `
provider "artifactkeeper" {}

resource "artifactkeeper_repository" "local" {
  key        = "tf-acc-111-local"
  name       = "TF Acc 1.11 local"
  format     = "generic"
  repo_type  = "local"
  visibility = "` + visibility + `"
}

resource "artifactkeeper_repository" "remote" {
  key          = "tf-acc-111-remote"
  name         = "TF Acc 1.11 remote"
  format       = "generic"
  repo_type    = "remote"
  upstream_url = "https://example.com/"
}

resource "artifactkeeper_repository" "conda_remote" {
  key          = "tf-acc-111-conda-remote"
  name         = "TF Acc 1.11 conda remote"
  format       = "conda"
  repo_type    = "remote"
  upstream_url = "https://conda.anaconda.org/conda-forge"
}

resource "artifactkeeper_repository" "conda_virtual" {
  key       = "tf-acc-111-conda-virtual"
  name      = "TF Acc 1.11 conda virtual"
  format    = "conda"
  repo_type = "virtual"
  members   = [artifactkeeper_repository.conda_remote.key]
}

resource "artifactkeeper_repository_security" "remote" {
  repository_key    = artifactkeeper_repository.remote.key
  scan_enabled      = true
  scan_on_proxy     = true
  proxy_scan_action = "record_only"
}

resource "artifactkeeper_repository_upstream_filter" "remote" {
  repository_key   = artifactkeeper_repository.remote.key
  include_patterns = ["` + filter + `"]
  exclude_patterns = ["\\.exe$"]
}

resource "artifactkeeper_repository_allowlist" "conda" {
  repository_key = artifactkeeper_repository.conda_virtual.key
  enabled        = true
  entries = [
    { name = "numpy", version = ">=1.26", subdirs = ["linux-64"] },
    { name = "py*" },
  ]
}

resource "artifactkeeper_lifecycle_policy" "assigned" {
  name           = "tf-acc-111-assigned"
  policy_type    = "max_age_days"
  config         = jsonencode({ max_age_days = 90 })
  repository_ids = [artifactkeeper_repository.local.id, artifactkeeper_repository.remote.id]
  enabled        = false
}

resource "artifactkeeper_lifecycle_policy" "global" {
  name           = "tf-acc-111-global"
  policy_type    = "max_age_days"
  config         = jsonencode({ max_age_days = 365 })
  applies_to_all = true
  enabled        = false
}

resource "artifactkeeper_permission" "anonymous" {
  principal_type = "anonymous"
  principal_id   = "00000000-0000-0000-0000-000000000000"
  target_type    = "repository"
  target_id      = artifactkeeper_repository.local.id
  actions        = ["read"]
  allowed_cidrs  = ["10.0.0.0/8", "fd00::/8"]
}

resource "artifactkeeper_runtime_settings" "this" {
  guest_access_enabled = true
}

resource "artifactkeeper_system_banner" "maint" {
  title     = "Maintenance"
  message   = "Upgrade to 1.11.0 tonight."
  severity  = "` + bannerSeverity + `"
  target    = "ui"
  starts_at = "2030-01-01T22:00:00Z"
  ends_at   = "2030-01-01T23:00:00+00:00"
}

resource "artifactkeeper_signing_key" "gpg_ed25519" {
  name      = "tf-acc-111-ed25519"
  key_type  = "gpg"
  algorithm = "ed25519"
  uid_name  = "TF Acc"
  uid_email = "tf-acc@example.com"
}

resource "artifactkeeper_security_policy" "origin" {
  name         = "tf-acc-111-origin"
  max_severity = "critical"
  predicates   = jsonencode({ origin = { allowed_kinds = ["hosted", "proxy"] } })
}
`
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config("internal", "^pub/", "warning"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("artifactkeeper_repository.local", "visibility", "internal"),
					resource.TestCheckResourceAttr("artifactkeeper_repository.local", "is_public", "false"),
					resource.TestCheckResourceAttr("artifactkeeper_repository_security.remote", "proxy_scan_action", "record_only"),
					resource.TestCheckResourceAttr("artifactkeeper_repository_upstream_filter.remote", "active", "true"),
					resource.TestCheckResourceAttr("artifactkeeper_repository_allowlist.conda", "entries.#", "2"),
					resource.TestCheckResourceAttr("artifactkeeper_lifecycle_policy.assigned", "repository_ids.#", "2"),
					resource.TestCheckResourceAttr("artifactkeeper_lifecycle_policy.assigned", "applies_to_all", "false"),
					resource.TestCheckResourceAttr("artifactkeeper_lifecycle_policy.global", "applies_to_all", "true"),
					resource.TestCheckResourceAttr("artifactkeeper_permission.anonymous", "allowed_cidrs.#", "2"),
					resource.TestCheckResourceAttr("artifactkeeper_runtime_settings.this", "guest_access_in_force", "true"),
					resource.TestCheckResourceAttr("artifactkeeper_system_banner.maint", "active", "false"),
					resource.TestCheckResourceAttr("artifactkeeper_signing_key.gpg_ed25519", "algorithm", "ed25519"),
				),
			},
			// In-place updates: visibility flips is_public, the filter, the banner.
			{
				Config: config("public", "^dist/", "critical"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("artifactkeeper_repository.local", "visibility", "public"),
					resource.TestCheckResourceAttr("artifactkeeper_repository.local", "is_public", "true"),
					resource.TestCheckResourceAttr("artifactkeeper_system_banner.maint", "severity", "critical"),
				),
			},
			{ResourceName: "artifactkeeper_system_banner.maint", ImportState: true, ImportStateVerify: true, ImportStateVerifyIgnore: []string{"starts_at", "ends_at"}},
			{ResourceName: "artifactkeeper_repository_upstream_filter.remote", ImportState: true, ImportStateId: "tf-acc-111-remote", ImportStateVerify: true},
			{ResourceName: "artifactkeeper_repository_allowlist.conda", ImportState: true, ImportStateId: "tf-acc-111-conda-virtual", ImportStateVerify: true},
			{ResourceName: "artifactkeeper_lifecycle_policy.assigned", ImportState: true, ImportStateVerify: true, ImportStateVerifyIgnore: []string{"config"}},
			{ResourceName: "artifactkeeper_permission.anonymous", ImportState: true, ImportStateVerify: true},
		},
	})
}
