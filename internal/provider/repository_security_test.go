package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccRepositorySecurityResource: set + read + empty-plan + import of a
// repository's scan config. Requires TF_ACC and a live instance.
func TestAccRepositorySecurityResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
provider "artifactkeeper" {}

resource "artifactkeeper_repository" "scanned" {
  key       = "tf-acc-scan-repo"
  name      = "TF Acc Scan Repo"
  format    = "generic"
  repo_type = "local"
}

resource "artifactkeeper_repository_security" "scanned" {
  repository_key            = artifactkeeper_repository.scanned.key
  scan_enabled              = true
  scan_on_upload            = true
  scan_on_proxy             = false
  block_on_policy_violation = true
  severity_threshold        = "high"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("artifactkeeper_repository_security.scanned", "scan_enabled", "true"),
					resource.TestCheckResourceAttr("artifactkeeper_repository_security.scanned", "scan_on_upload", "true"),
					resource.TestCheckResourceAttr("artifactkeeper_repository_security.scanned", "scan_on_proxy", "false"),
					resource.TestCheckResourceAttr("artifactkeeper_repository_security.scanned", "severity_threshold", "high"),
					resource.TestCheckResourceAttrSet("artifactkeeper_repository_security.scanned", "repository_id"),
				),
			},
			{
				ResourceName:      "artifactkeeper_repository_security.scanned",
				ImportState:       true,
				ImportStateId:     "tf-acc-scan-repo",
				ImportStateVerify: true,
			},
		},
	})
}
