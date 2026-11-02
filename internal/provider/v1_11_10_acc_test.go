package provider

import (
	"os"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccV11110Surface covers what 1.11.1 to 1.11.10 changed for the provider:
// the public-with-upstream-credentials refusal and its override, the conda
// allowlist scope, and peer_network_profile now that its route has `/:id`.
func TestAccV11110Surface(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip("acceptance test; set TF_ACC to run")
	}
	testAccPreCheck(t)

	config := func(allow, scope string) string {
		allowLine := ""
		if allow != "" {
			allowLine = "allow_public_with_upstream_credentials = " + allow
		}
		return `
provider "artifactkeeper" {}

resource "artifactkeeper_repository" "public_remote" {
  key          = "tf-acc-11110-public-remote"
  name         = "TF Acc 1.11.10 public remote"
  format       = "generic"
  repo_type    = "remote"
  upstream_url = "https://example.com/"
  is_public    = true
}

resource "artifactkeeper_repository_upstream_auth" "public_remote" {
  repository_key = artifactkeeper_repository.public_remote.key
  auth_type      = "basic"
  username       = "svc"
  password       = "not-a-real-secret"
  ` + allowLine + `
}

resource "artifactkeeper_repository" "conda_remote" {
  key          = "tf-acc-11110-conda-remote"
  name         = "TF Acc 1.11.10 conda remote"
  format       = "conda"
  repo_type    = "remote"
  upstream_url = "https://conda.anaconda.org/conda-forge"
}

resource "artifactkeeper_repository" "conda_virtual" {
  key       = "tf-acc-11110-conda-virtual"
  name      = "TF Acc 1.11.10 conda virtual"
  format    = "conda"
  repo_type = "virtual"
  members   = [artifactkeeper_repository.conda_remote.key]
}

resource "artifactkeeper_repository_allowlist" "conda" {
  repository_key = artifactkeeper_repository.conda_virtual.key
  enabled        = true
  scope          = "` + scope + `"
  entries        = [{ name = "numpy" }]
}

resource "artifactkeeper_peer" "far" {
  name         = "tf-acc-11110-peer"
  endpoint_url = "https://peer.example.com"
  api_key      = "not-a-real-key"
}

resource "artifactkeeper_peer_network_profile" "far" {
  peer_id                    = artifactkeeper_peer.far.id
  sync_window_start          = "02:00:00"
  sync_window_end            = "06:00:00"
  sync_window_timezone       = "UTC"
  concurrent_transfers_limit = 2
}
`
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Credentials on a public repository without the override are refused.
			{
				Config:      config("", "remote_only"),
				ExpectError: regexp.MustCompile(`would be public while it carries upstream\s+credentials`),
			},
			{
				Config: config("true", "remote_only"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("artifactkeeper_repository_upstream_auth.public_remote", "configured", "true"),
					resource.TestCheckResourceAttr("artifactkeeper_repository_allowlist.conda", "scope", "remote_only"),
					resource.TestCheckResourceAttr("artifactkeeper_peer_network_profile.far", "concurrent_transfers_limit", "2"),
				),
			},
			// The override is stored on the repository, so a refresh reads it back.
			{
				Config: config("true", "all_members"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("artifactkeeper_repository.public_remote", "allow_public_with_upstream_credentials", "true"),
					resource.TestCheckResourceAttr("artifactkeeper_repository_allowlist.conda", "scope", "all_members"),
				),
			},
			{ResourceName: "artifactkeeper_repository_allowlist.conda", ImportState: true, ImportStateId: "tf-acc-11110-conda-virtual", ImportStateVerify: true},
		},
	})
}
