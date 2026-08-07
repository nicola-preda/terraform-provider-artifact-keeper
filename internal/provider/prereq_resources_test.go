package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccPrereqResources covers the new resources that need prerequisite repos
// (promotion/sync/curation/lifecycle/age-gate/repo-token/label). It's the
// shakeout for the trickier ones: nested selectors, JSON config, staging/remote
// repo requirements. Requires TF_ACC and a live instance.
func TestAccPrereqResources(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
provider "artifactkeeper" {}

resource "artifactkeeper_repository" "local" {
  key       = "tfacc-p-local"
  name      = "tfacc p local"
  format    = "docker"
  repo_type = "local"
}

resource "artifactkeeper_repository" "staging" {
  key       = "tfacc-p-staging"
  name      = "tfacc p staging"
  format    = "docker"
  repo_type = "staging"
}

resource "artifactkeeper_repository" "remote" {
  key          = "tfacc-p-remote"
  name         = "tfacc p remote"
  format       = "npm"
  repo_type    = "remote"
  upstream_url = "https://registry.npmjs.org"
}

resource "artifactkeeper_promotion_rule" "p" {
  name           = "tfacc-promo"
  source_repo_id = artifactkeeper_repository.staging.id
  target_repo_id = artifactkeeper_repository.local.id
}

resource "artifactkeeper_sync_policy" "s" {
  name             = "tfacc-sync"
  replication_mode = "mirror"
  repo_selector = {
    match_formats = ["docker"]
  }
}

resource "artifactkeeper_curation_rule" "c" {
  package_pattern = "telnet*"
  action          = "block"
  reason          = "Cleartext protocol; disallowed."
}

resource "artifactkeeper_lifecycle_policy" "l" {
  repository_id = artifactkeeper_repository.local.id
  name          = "tfacc-lifecycle"
  policy_type   = "max_age_days"
  config        = jsonencode({ max_age_days = 90 })
}

resource "artifactkeeper_age_gate" "ag" {
  repository_key = artifactkeeper_repository.remote.key
  enabled        = true
  min_age_days   = 14
  mode           = "first_seen"
}

resource "artifactkeeper_repo_token" "t" {
  repository_key = artifactkeeper_repository.local.key
  name           = "tfacc-token"
  scopes         = ["read:artifacts"]
}

resource "artifactkeeper_repository_label" "lbl" {
  repository_key = artifactkeeper_repository.local.key
  key            = "env"
  value          = "prod"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("artifactkeeper_promotion_rule.p", "id"),
					resource.TestCheckResourceAttr("artifactkeeper_sync_policy.s", "replication_mode", "mirror"),
					resource.TestCheckResourceAttr("artifactkeeper_curation_rule.c", "action", "block"),
					resource.TestCheckResourceAttrSet("artifactkeeper_lifecycle_policy.l", "id"),
					resource.TestCheckResourceAttr("artifactkeeper_age_gate.ag", "min_age_days", "14"),
					resource.TestCheckResourceAttr("artifactkeeper_age_gate.ag", "mode", "first_seen"),
					resource.TestCheckResourceAttrSet("artifactkeeper_repo_token.t", "token"),
					resource.TestCheckResourceAttr("artifactkeeper_repository_label.lbl", "value", "prod"),
				),
			},
		},
	})
}
