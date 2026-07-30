package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccRepositoryReleaseTargetResource: link a staging repository to a
// release (local) repository, read it back, and import by staging repo key.
// Requires TF_ACC and a live instance.
func TestAccRepositoryReleaseTargetResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
provider "artifactkeeper" {}

resource "artifactkeeper_repository" "release" {
  key       = "tf-acc-release-repo"
  name      = "TF Acc Release Repo"
  format    = "generic"
  repo_type = "local"
}

resource "artifactkeeper_repository" "staging" {
  key       = "tf-acc-staging-repo"
  name      = "TF Acc Staging Repo"
  format    = "generic"
  repo_type = "staging"
}

resource "artifactkeeper_repository_release_target" "staging" {
  repository_key         = artifactkeeper_repository.staging.key
  release_repository_key = artifactkeeper_repository.release.key
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("artifactkeeper_repository_release_target.staging", "linked", "true"),
					resource.TestCheckResourceAttr("artifactkeeper_repository_release_target.staging", "release_repository_key", "tf-acc-release-repo"),
					resource.TestCheckResourceAttrSet("artifactkeeper_repository_release_target.staging", "release_repository_id"),
				),
			},
			{
				ResourceName:      "artifactkeeper_repository_release_target.staging",
				ImportState:       true,
				ImportStateId:     "tf-acc-staging-repo",
				ImportStateVerify: true,
			},
		},
	})
}
