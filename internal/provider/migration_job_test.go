package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccMigrationJobResource: create + read + empty-plan + import for a pending
// migration job against a source connection. The job is created, not started, so
// no reachable source is needed. Also gives migration_source its create/read
// coverage. Requires TF_ACC and a live instance.
func TestAccMigrationJobResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
provider "artifactkeeper" {}

resource "artifactkeeper_migration_source" "nexus" {
  name                = "tf-acc-nexus"
  url                 = "https://nexus.example.com"
  auth_type           = "basic_auth"
  source_type         = "nexus"
  credential_username = "tfacc"
  credential_password = "tfacc-secret"
}

resource "artifactkeeper_migration_job" "hosted" {
  source_connection_id  = artifactkeeper_migration_source.nexus.id
  include_repos         = ["tf-acc-repo"]
  repo_mappings         = { "tf-acc-repo" = "tf-acc-renamed" }
  include_cached_remote = false
  conflict_resolution   = "skip"
  dry_run               = true
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("artifactkeeper_migration_job.hosted", "id"),
					resource.TestCheckResourceAttrSet("artifactkeeper_migration_job.hosted", "status"),
					resource.TestCheckResourceAttrSet("artifactkeeper_migration_job.hosted", "created_at"),
					resource.TestCheckResourceAttrPair(
						"artifactkeeper_migration_job.hosted", "source_connection_id",
						"artifactkeeper_migration_source.nexus", "id"),
					resource.TestCheckResourceAttr("artifactkeeper_migration_job.hosted", "include_repos.#", "1"),
					resource.TestCheckResourceAttr("artifactkeeper_migration_job.hosted",
						"repo_mappings.tf-acc-repo", "tf-acc-renamed"),
				),
			},
			{
				ResourceName:      "artifactkeeper_migration_job.hosted",
				ImportState:       true,
				ImportStateVerify: true,
				// The API returns config as an opaque blob, so configured inputs
				// can't be recovered on import and are carried from prior state.
				ImportStateVerifyIgnore: []string{
					"include_repos", "exclude_repos", "exclude_paths", "repo_mappings",
					"include_users", "include_groups", "include_permissions",
					"include_cached_remote", "dry_run", "conflict_resolution",
					"concurrent_transfers", "throttle_delay_ms", "verify_checksums",
					"date_from", "date_to",
				},
			},
		},
	})
}
