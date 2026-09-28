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
			{
				// The AWS auth types (1.10.0, #1559). No AWS call happens on
				// write: the backend only checks that the repository's upstream
				// URL is the ECR host shape the config claims, so this runs
				// without credentials and still proves the widened enum and the
				// nested block reach the wire.
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

resource "artifactkeeper_repository" "ecr" {
  key          = "tf-acc-upstream-auth-ecr"
  name         = "TF Acc Upstream Auth ECR"
  format       = "docker"
  repo_type    = "remote"
  upstream_url = "https://123456789012.dkr.ecr.us-east-1.amazonaws.com"
}

resource "artifactkeeper_repository_upstream_auth" "ecr" {
  repository_key = artifactkeeper_repository.ecr.key
  auth_type      = "aws_ecr"

  aws = {
    region      = "us-east-1"
    registry_id = "123456789012"
  }
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("artifactkeeper_repository_upstream_auth.ecr", "auth_type", "aws_ecr"),
					resource.TestCheckResourceAttr("artifactkeeper_repository_upstream_auth.ecr", "aws.region", "us-east-1"),
					resource.TestCheckResourceAttr("artifactkeeper_repository_upstream_auth.ecr", "aws.registry_id", "123456789012"),
					resource.TestCheckResourceAttr("artifactkeeper_repository_upstream_auth.ecr", "configured", "true"),
					resource.TestCheckResourceAttr("artifactkeeper_repository_upstream_auth.ecr", "configured_auth_type", "aws_ecr"),
				),
			},
		},
	})
}
