package provider

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccV112Fields covers the 1.12 fields added to existing resources: each
// is created with one value and updated in place to another, then imported
// where the resource supports it
func TestAccV112Fields(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip("acceptance test; set TF_ACC to run")
	}
	testAccPreCheck(t)

	config := func(first bool) string {
		pick := func(a, b string) string {
			if first {
				return a
			}
			return b
		}
		return `
provider "artifactkeeper" {}

resource "artifactkeeper_runtime_settings" "rt" {
  guest_access_enabled = true
  hidden_nav_items     = ` + pick(`["/peers", "/plugins"]`, `[]`) + `
  upstream_jobs_paused = ` + pick("true", "false") + `
}

resource "artifactkeeper_repository" "staging" {
  key       = "tf-acc-112-staging"
  name      = "tf-acc-112-staging"
  format    = "generic"
  repo_type = "staging"
}

resource "artifactkeeper_repository" "local" {
  key       = "tf-acc-112-local"
  name      = "tf-acc-112-local"
  format    = "generic"
  repo_type = "local"
}

resource "artifactkeeper_repository" "remote" {
  key          = "tf-acc-112-remote"
  name         = "tf-acc-112-remote"
  format       = "npm"
  repo_type    = "remote"
  upstream_url = "https://registry.npmjs.org"
}

resource "artifactkeeper_security_policy" "p" {
  name              = "tf-acc-112-policy"
  max_severity      = "high"
  block_on_fail     = true
  block_uncataloged = ` + pick("true", "false") + `
}

resource "artifactkeeper_quality_gate" "g" {
  name                  = "tf-acc-112-gate"
  min_health_score      = 70
  action                = "block"
  accepted_grade_status = "` + pick("incomplete", "any") + `"
}

resource "artifactkeeper_promotion_rule" "r" {
  name                  = "tf-acc-112-rule"
  source_repo_id        = artifactkeeper_repository.staging.id
  target_repo_id        = artifactkeeper_repository.local.id
  min_health_score      = 60
  accepted_grade_status = "` + pick("any", "rated") + `"
}

resource "artifactkeeper_age_gate" "ag" {
  repository_key  = artifactkeeper_repository.remote.key
  enabled         = true
  min_age_days    = 7
  instance_locked = ` + pick("false", "true") + `
}

resource "artifactkeeper_repository_security" "s" {
  repository_key  = artifactkeeper_repository.remote.key
  scan_enabled    = true
  scan_on_proxy   = true
  instance_locked = ` + pick("true", "false") + `
}

resource "artifactkeeper_peer" "far" {
  name         = "tf-acc-112-peer"
  endpoint_url = "https://peer.example.com"
  api_key      = "not-a-real-key"
  sync_filter  = jsonencode({ formats = ["maven", "npm"] })
}

resource "artifactkeeper_user" "u" {
  username = "tf-acc-112-user"
  email    = "tf-acc-112-user@example.com"
}

resource "artifactkeeper_user_api_token" "t" {
  user_id       = artifactkeeper_user.u.id
  name          = "tf-acc-112-token"
  scopes        = ["read:artifacts"]
  repo_selector = jsonencode({ match_formats = ["generic"] })
}
`
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config(true),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("artifactkeeper_runtime_settings.rt", "hidden_nav_items.#", "2"),
					resource.TestCheckTypeSetElemAttr("artifactkeeper_runtime_settings.rt", "hidden_nav_items.*", "/peers"),
					resource.TestCheckResourceAttr("artifactkeeper_runtime_settings.rt", "upstream_jobs_paused", "true"),
					resource.TestCheckResourceAttr("artifactkeeper_runtime_settings.rt", "upstream_jobs_paused_in_force", "true"),
					resource.TestCheckResourceAttr("artifactkeeper_security_policy.p", "block_uncataloged", "true"),
					resource.TestCheckResourceAttr("artifactkeeper_quality_gate.g", "accepted_grade_status", "incomplete"),
					resource.TestCheckResourceAttr("artifactkeeper_promotion_rule.r", "accepted_grade_status", "any"),
					resource.TestCheckResourceAttr("artifactkeeper_age_gate.ag", "instance_locked", "false"),
					resource.TestCheckResourceAttr("artifactkeeper_repository_security.s", "instance_locked", "true"),
					resource.TestCheckResourceAttrSet("artifactkeeper_peer.far", "id"),
					resource.TestCheckResourceAttrSet("artifactkeeper_user_api_token.t", "token"),
				),
			},
			{
				Config: config(false),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("artifactkeeper_runtime_settings.rt", "hidden_nav_items.#", "0"),
					resource.TestCheckResourceAttr("artifactkeeper_runtime_settings.rt", "upstream_jobs_paused", "false"),
					resource.TestCheckResourceAttr("artifactkeeper_security_policy.p", "block_uncataloged", "false"),
					resource.TestCheckResourceAttr("artifactkeeper_quality_gate.g", "accepted_grade_status", "any"),
					resource.TestCheckResourceAttr("artifactkeeper_promotion_rule.r", "accepted_grade_status", "rated"),
					resource.TestCheckResourceAttr("artifactkeeper_age_gate.ag", "instance_locked", "true"),
					resource.TestCheckResourceAttr("artifactkeeper_repository_security.s", "instance_locked", "false"),
				),
			},
		},
	})
}
