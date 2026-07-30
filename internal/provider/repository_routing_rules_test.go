package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccRepositoryRoutingRulesResource: create a remote repo, set two ordered
// rewrite rules, then import by key. Requires TF_ACC and a live instance.
func TestAccRepositoryRoutingRulesResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
provider "artifactkeeper" {}

resource "artifactkeeper_repository" "routed" {
  key          = "tf-acc-routing-repo"
  name         = "TF Acc Routing Repo"
  format       = "generic"
  repo_type    = "remote"
  upstream_url = "https://example.com"
}

resource "artifactkeeper_repository_routing_rules" "routed" {
  repository_key = artifactkeeper_repository.routed.key
  rules = [
    {
      path_pattern = "^/v1/(.*)$"
      rewrite_to   = "/$1"
    },
    {
      path_pattern = "^/legacy/(.*)$"
      rewrite_to   = "/new/$1"
    },
  ]
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("artifactkeeper_repository_routing_rules.routed", "rules.#", "2"),
					resource.TestCheckResourceAttr("artifactkeeper_repository_routing_rules.routed", "rules.0.path_pattern", "^/v1/(.*)$"),
					resource.TestCheckResourceAttr("artifactkeeper_repository_routing_rules.routed", "rules.0.rewrite_to", "/$1"),
				),
			},
			{
				ResourceName:      "artifactkeeper_repository_routing_rules.routed",
				ImportState:       true,
				ImportStateId:     "tf-acc-routing-repo",
				ImportStateVerify: true,
			},
		},
	})
}
