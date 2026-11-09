package provider

import (
	"os"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccV11112Surface covers the 1.11.12 replication modes on both resources:
// pull is refused at plan time, none and push round-trip
func TestAccV11112Surface(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip("acceptance test; set TF_ACC to run")
	}
	testAccPreCheck(t)

	config := func(mode string) string {
		return `
provider "artifactkeeper" {}

resource "artifactkeeper_repository" "local" {
  key       = "tf-acc-11112-local"
  name      = "TF Acc 1.11.12 local"
  format    = "generic"
  repo_type = "local"
}

resource "artifactkeeper_peer" "far" {
  name         = "tf-acc-11112-peer"
  endpoint_url = "https://peer.example.com"
  api_key      = "not-a-real-key"
}

resource "artifactkeeper_peer_repository_subscription" "far" {
  peer_id          = artifactkeeper_peer.far.id
  repository_id    = artifactkeeper_repository.local.id
  replication_mode = "` + mode + `"
}

resource "artifactkeeper_sync_policy" "s" {
  name             = "tf-acc-11112-sync"
  replication_mode = "` + mode + `"
  repo_selector = {
    match_formats = ["docker"]
  }
}
`
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      config("pull"),
				ExpectError: regexp.MustCompile(`value must be one of`),
			},
			{
				Config: config("none"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("artifactkeeper_peer_repository_subscription.far", "replication_mode", "none"),
					resource.TestCheckResourceAttr("artifactkeeper_sync_policy.s", "replication_mode", "none"),
				),
			},
			{
				Config: config("push"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("artifactkeeper_peer_repository_subscription.far", "replication_mode", "push"),
					resource.TestCheckResourceAttr("artifactkeeper_sync_policy.s", "replication_mode", "push"),
				),
			},
		},
	})
}
