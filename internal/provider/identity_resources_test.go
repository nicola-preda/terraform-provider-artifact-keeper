package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccIdentityResources wires group + user + repository + permission in one
// config. The permission references the group and repository, so Terraform must
// create them first, exercising the server-side principal validation added after
// v1.2.0 (a grant to an unknown principal 400s). Requires TF_ACC and a live instance.
func TestAccIdentityResources(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
provider "artifactkeeper" {}

resource "artifactkeeper_group" "team" {
  name        = "tf-acc-team"
  description = "acc-test team"
}

resource "artifactkeeper_user" "u" {
  username = "tf-acc-user"
  email    = "tf-acc-user@example.com"
}

resource "artifactkeeper_repository" "r" {
  key       = "tf-acc-perm-repo"
  name      = "tf-acc perm repo"
  format    = "docker"
  repo_type = "local"
}

resource "artifactkeeper_permission" "p" {
  principal_type = "group"
  principal_id   = artifactkeeper_group.team.id
  target_type    = "repository"
  target_id      = artifactkeeper_repository.r.id
  actions        = ["read", "write"]
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("artifactkeeper_group.team", "id"),
					resource.TestCheckResourceAttrSet("artifactkeeper_user.u", "id"),
					resource.TestCheckResourceAttrPair("artifactkeeper_permission.p", "principal_id", "artifactkeeper_group.team", "id"),
					resource.TestCheckResourceAttr("artifactkeeper_permission.p", "actions.#", "2"),
				),
			},
		},
	})
}
