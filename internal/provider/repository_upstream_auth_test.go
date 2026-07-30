package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccRepositoryUpstreamAuthResource sets basic upstream credentials on a
// remote repository. The resource is write-only (no GET), so there is nothing to
// read back and no ImportStateVerify. Requires TF_ACC and a live instance.
func TestAccRepositoryUpstreamAuthResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
provider "artifactkeeper" {}

resource "artifactkeeper_repository" "remote" {
  key          = "tf-acc-upstream-auth"
  name         = "TF Acc Upstream Auth"
  format       = "npm"
  repo_type    = "remote"
  upstream_url = "https://registry.npmjs.org"
}

resource "artifactkeeper_repository_upstream_auth" "creds" {
  repository_key = artifactkeeper_repository.remote.key
  auth_type      = "basic"
  username       = "u"
  password       = "p"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("artifactkeeper_repository_upstream_auth.creds", "auth_type", "basic"),
				),
			},
		},
	})
}
