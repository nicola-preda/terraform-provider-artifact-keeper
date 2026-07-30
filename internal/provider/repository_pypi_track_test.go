package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccRepositoryPypiTrackResource: create a hosted PyPI repo, track a local
// project against an upstream Simple index, then re-import it. Requires TF_ACC
// and a live instance.
func TestAccRepositoryPypiTrackResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
provider "artifactkeeper" {}

resource "artifactkeeper_repository" "pypi" {
  key       = "tf-acc-pypi"
  name      = "TF Acc PyPI Repo"
  format    = "pypi"
  repo_type = "local"
}

resource "artifactkeeper_repository_pypi_track" "acme" {
  repository_key = artifactkeeper_repository.pypi.key
  project        = "acme-sdk"
  tracks_url     = "https://pypi.org/simple/acme-sdk/"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("artifactkeeper_repository_pypi_track.acme", "tracks_url", "https://pypi.org/simple/acme-sdk/"),
					resource.TestCheckResourceAttr("artifactkeeper_repository_pypi_track.acme", "normalized_name", "acme-sdk"),
				),
			},
			{
				ResourceName:      "artifactkeeper_repository_pypi_track.acme",
				ImportState:       true,
				ImportStateId:     "tf-acc-pypi/acme-sdk",
				ImportStateVerify: true,
			},
		},
	})
}
