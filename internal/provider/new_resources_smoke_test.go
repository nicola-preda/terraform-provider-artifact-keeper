package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccNewResourcesSmoke: create + read + empty-plan for the new resources.
// Catches "inconsistent result after apply". Requires TF_ACC and a live instance.
func TestAccNewResourcesSmoke(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
provider "artifactkeeper" {}

resource "artifactkeeper_service_account" "ci" {
  name         = "tf-acc-ci"
  display_name = "TF Acc CI"
}

resource "artifactkeeper_signing_key" "global" {
  name      = "tf-acc-signing"
  key_type  = "rsa"
  algorithm = "rsa4096"
}

resource "artifactkeeper_signing_key" "gpg" {
  name      = "tf-acc-gpg"
  key_type  = "gpg"
  uid_name  = "TF Acc"
  uid_email = "tfacc@example.com"
}

resource "artifactkeeper_webhook" "notify" {
  name             = "tf-acc-webhook"
  url              = "https://hooks.example.com/tf-acc"
  events           = ["artifact_uploaded"]
  payload_template = "slack"
}

resource "artifactkeeper_ci_oidc_provider" "gl" {
  name          = "tf-acc-gitlab"
  provider_type = "gitlab"
  issuer_url    = "https://gitlab.example.com"
}

resource "artifactkeeper_remote_instance" "mirror" {
  name    = "tf-acc-mirror"
  url     = "https://ak-eu.example.com"
  api_key = "dummy-remote-key"
}

resource "artifactkeeper_quality_gate" "baseline" {
  name             = "tf-acc-quality"
  min_health_score = 70
  action           = "block"
}

resource "artifactkeeper_security_policy" "gate" {
  name          = "tf-acc-security"
  max_severity  = "high"
  block_on_fail = true
}

resource "artifactkeeper_project" "team" {
  key  = "tf-acc-project"
  name = "TF Acc Project"
}

resource "artifactkeeper_license_policy" "licenses" {
  name             = "tf-acc-licenses"
  allowed_licenses = ["MIT", "Apache-2.0"]
  denied_licenses  = ["GPL-3.0-only"]
  action           = "warn"
}

resource "artifactkeeper_repository" "sub_host" {
  key       = "tf-acc-emailsub-repo"
  name      = "TF Acc Email Sub Repo"
  format    = "generic"
  repo_type = "local"
}

resource "artifactkeeper_email_subscription" "notify" {
  repository_key = artifactkeeper_repository.sub_host.key
  recipients     = ["devops@example.com"]
  event_types    = ["artifact.uploaded", "scan.completed"]
}

resource "artifactkeeper_ci_oidc_identity_mapping" "gl_map" {
  provider_id      = artifactkeeper_ci_oidc_provider.gl.id
  name             = "tf-acc-mapping"
  claim_filters    = jsonencode({ project_path = "mygroup/myrepo" })
  allowed_repo_ids = [artifactkeeper_repository.sub_host.id]
}

resource "artifactkeeper_service_account_token" "ci_tok" {
  service_account_id = artifactkeeper_service_account.ci.id
  name               = "tf-acc-sa-token"
  scopes             = ["read:artifacts"]
}

resource "artifactkeeper_group" "grp" {
  name = "tf-acc-grp"
}

resource "artifactkeeper_user" "member" {
  username = "tf-acc-member"
  email    = "tf-acc-member@example.com"
}

resource "artifactkeeper_group_membership" "gm" {
  group_id = artifactkeeper_group.grp.id
  user_id  = artifactkeeper_user.member.id
}

resource "artifactkeeper_repository" "expanded" {
  key                = "tf-acc-expanded"
  name               = "TF Acc Expanded"
  format             = "generic"
  repo_type          = "local"
  promotion_only     = true
  versioning_enabled = true
  project_id         = artifactkeeper_project.team.id
}

resource "artifactkeeper_system_settings" "this" {
  retention_days = 30
}

resource "artifactkeeper_telemetry_settings" "this" {
  enabled     = false
  scrub_level = "standard"
}

resource "artifactkeeper_repository_signing_config" "sub" {
  repository_id = artifactkeeper_repository.sub_host.id
  sign_metadata = true
}

resource "artifactkeeper_repository" "virtual" {
  key       = "tf-acc-virtual"
  name      = "TF Acc Virtual"
  format    = "generic"
  repo_type = "virtual"
  members   = [artifactkeeper_repository.sub_host.key, artifactkeeper_repository.expanded.key]
}

resource "artifactkeeper_format_handler" "oci" {
  format_key = "oci"
  enabled    = false
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("artifactkeeper_service_account.ci", "username"),
					resource.TestCheckResourceAttrSet("artifactkeeper_signing_key.global", "fingerprint"),
					resource.TestCheckResourceAttrSet("artifactkeeper_webhook.notify", "id"),
					resource.TestCheckResourceAttrSet("artifactkeeper_ci_oidc_provider.gl", "id"),
					resource.TestCheckResourceAttrSet("artifactkeeper_remote_instance.mirror", "id"),
					resource.TestCheckResourceAttr("artifactkeeper_quality_gate.baseline", "action", "block"),
					resource.TestCheckResourceAttr("artifactkeeper_security_policy.gate", "max_severity", "high"),
					resource.TestCheckResourceAttrSet("artifactkeeper_project.team", "id"),
					resource.TestCheckResourceAttrSet("artifactkeeper_license_policy.licenses", "id"),
					resource.TestCheckResourceAttrSet("artifactkeeper_email_subscription.notify", "id"),
					resource.TestCheckResourceAttrSet("artifactkeeper_ci_oidc_identity_mapping.gl_map", "id"),
					resource.TestCheckResourceAttrSet("artifactkeeper_service_account_token.ci_tok", "token"),
					resource.TestCheckResourceAttrSet("artifactkeeper_group_membership.gm", "id"),
					resource.TestCheckResourceAttr("artifactkeeper_repository.expanded", "promotion_only", "true"),
					resource.TestCheckResourceAttr("artifactkeeper_repository.expanded", "versioning_enabled", "true"),
					resource.TestCheckResourceAttr("artifactkeeper_system_settings.this", "retention_days", "30"),
					resource.TestCheckResourceAttr("artifactkeeper_telemetry_settings.this", "enabled", "false"),
					resource.TestCheckResourceAttrSet("artifactkeeper_repository_signing_config.sub", "id"),
					resource.TestCheckResourceAttr("artifactkeeper_repository.virtual", "members.#", "2"),
					resource.TestCheckResourceAttr("artifactkeeper_format_handler.oci", "enabled", "false"),
				),
			},
		},
	})
}
