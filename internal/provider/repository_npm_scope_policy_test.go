package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccRepositoryNpmScopePolicyResource: set + read + empty-plan + import of a
// repository's npm scope policy. The policy is only valid for a remote member of
// an npm virtual repository, so the config stands up a remote and a virtual that
// aggregates it, then restricts the remote to a single scope. Requires TF_ACC and
// a live instance.
func TestAccRepositoryNpmScopePolicyResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
provider "artifactkeeper" {}

resource "artifactkeeper_repository" "npm_remote" {
  key          = "tf-acc-npm-remote"
  name         = "TF Acc npm remote"
  format       = "npm"
  repo_type    = "remote"
  upstream_url = "https://registry.npmjs.org"
}

resource "artifactkeeper_repository" "npm_virtual" {
  key       = "tf-acc-npm-virtual"
  name      = "TF Acc npm virtual"
  format    = "npm"
  repo_type = "virtual"
  members   = [artifactkeeper_repository.npm_remote.key]
}

resource "artifactkeeper_repository_npm_scope_policy" "scoped" {
  repository_key = artifactkeeper_repository.npm_remote.key
  allowed_scopes = ["@myorg"]
  allow_unscoped = false
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("artifactkeeper_repository_npm_scope_policy.scoped", "allowed_scopes.#", "1"),
					resource.TestCheckResourceAttr("artifactkeeper_repository_npm_scope_policy.scoped", "allowed_scopes.0", "@myorg"),
					resource.TestCheckResourceAttr("artifactkeeper_repository_npm_scope_policy.scoped", "allow_unscoped", "false"),
					resource.TestCheckResourceAttrSet("artifactkeeper_repository_npm_scope_policy.scoped", "active"),
				),
			},
			{
				ResourceName:      "artifactkeeper_repository_npm_scope_policy.scoped",
				ImportState:       true,
				ImportStateId:     "tf-acc-npm-remote",
				ImportStateVerify: true,
			},
		},
	})
}
