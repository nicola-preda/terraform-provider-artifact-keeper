package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccRepositoryCacheTtlResource: set + read + empty-plan + import of a
// repository's cache TTL. Requires TF_ACC and a live instance.
func TestAccRepositoryCacheTtlResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
provider "artifactkeeper" {}

resource "artifactkeeper_repository" "cached" {
  key          = "tf-acc-cachettl"
  name         = "TF Acc Cache TTL"
  format       = "generic"
  repo_type    = "remote"
  upstream_url = "https://example.com"
}

resource "artifactkeeper_repository_cache_ttl" "cached" {
  repository_key    = artifactkeeper_repository.cached.key
  cache_ttl_seconds = 3600
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("artifactkeeper_repository_cache_ttl.cached", "cache_ttl_seconds", "3600"),
					resource.TestCheckResourceAttr("artifactkeeper_repository_cache_ttl.cached", "repository_key", "tf-acc-cachettl"),
				),
			},
			{
				ResourceName:      "artifactkeeper_repository_cache_ttl.cached",
				ImportState:       true,
				ImportStateId:     "tf-acc-cachettl",
				ImportStateVerify: true,
			},
		},
	})
}
