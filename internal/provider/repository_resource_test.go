package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccRepositoryResource covers create, import, a data-source read-back, and an
// in-place update. The testing framework asserts an empty plan after each apply,
// which catches "provider produced inconsistent result" bugs. Requires TF_ACC and
// a live instance.
func TestAccRepositoryResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
provider "artifactkeeper" {}

resource "artifactkeeper_repository" "test" {
  key       = "tf-acc-docker"
  name      = "tf-acc docker"
  format    = "docker"
  repo_type = "local"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("artifactkeeper_repository.test", "key", "tf-acc-docker"),
					resource.TestCheckResourceAttr("artifactkeeper_repository.test", "repo_type", "local"),
					resource.TestCheckResourceAttrSet("artifactkeeper_repository.test", "id"),
				),
			},
			{
				ResourceName:      "artifactkeeper_repository.test",
				ImportState:       true,
				ImportStateId:     "tf-acc-docker",
				ImportStateVerify: true,
			},
			{
				Config: `
provider "artifactkeeper" {}

resource "artifactkeeper_repository" "test" {
  key       = "tf-acc-docker"
  name      = "tf-acc docker"
  format    = "docker"
  repo_type = "local"
}

data "artifactkeeper_repository" "by_key" {
  key = artifactkeeper_repository.test.key
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.artifactkeeper_repository.by_key", "repo_type", "local"),
					resource.TestCheckResourceAttrPair("data.artifactkeeper_repository.by_key", "id", "artifactkeeper_repository.test", "id"),
				),
			},
			{
				Config: `
provider "artifactkeeper" {}

resource "artifactkeeper_repository" "test" {
  key       = "tf-acc-docker"
  name      = "tf-acc docker (renamed)"
  format    = "docker"
  repo_type = "local"
}
`,
				Check: resource.TestCheckResourceAttr("artifactkeeper_repository.test", "name", "tf-acc docker (renamed)"),
			},
		},
	})
}
