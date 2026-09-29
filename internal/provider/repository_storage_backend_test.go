package provider

import (
	"context"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"

	"github.com/nicola-preda/terraform-provider-artifact-keeper/internal/client"
)

func TestStorageBackendRequiresReplace(t *testing.T) {
	cases := []struct {
		name        string
		state, plan types.String
		want        bool
	}{
		{"null state (after import) adopts config", types.StringNull(), types.StringValue("filesystem"), false},
		{"changed value replaces", types.StringValue("filesystem"), types.StringValue("s3"), true},
		{"removed value replaces", types.StringValue("filesystem"), types.StringNull(), true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := planmodifier.StringRequest{StateValue: tc.state, PlanValue: tc.plan, ConfigValue: tc.plan}
			var resp stringplanmodifier.RequiresReplaceIfFuncResponse
			storageBackendRequiresReplace(context.Background(), req, &resp)
			if resp.RequiresReplace != tc.want {
				t.Fatalf("RequiresReplace = %v, want %v", resp.RequiresReplace, tc.want)
			}
		})
	}
}

// TestAccRepositoryResource_storageBackendImport creates a repository out of band
// with an explicit storage_backend (which the API never returns), imports it, and
// asserts that the follow-up plan is an in-place update rather than a replacement.
// Requires TF_ACC and a live instance.
func TestAccRepositoryResource_storageBackendImport(t *testing.T) {
	const key = "tf-acc-storage-backend"
	const addr = "artifactkeeper_repository.test"
	const config = `
provider "artifactkeeper" {}

resource "artifactkeeper_repository" "test" {
  key             = "tf-acc-storage-backend"
  name            = "tf-acc storage backend"
  format          = "generic"
  repo_type       = "local"
  storage_backend = "filesystem"
}
`

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			c := testAccClient(t)
			backend := "filesystem"
			if _, err := c.CreateRepository(context.Background(), client.CreateRepositoryRequest{
				Key: key, Name: "tf-acc storage backend", Format: "generic", RepoType: "local", StorageBackend: &backend,
			}); err != nil {
				t.Fatalf("creating repository out of band: %v", err)
			}
			// Terraform destroys it once imported; this covers a failed import.
			t.Cleanup(func() { _ = c.DeleteRepository(context.Background(), key) })
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:             config,
				ResourceName:       addr,
				ImportState:        true,
				ImportStateId:      key,
				ImportStatePersist: true,
			},
			{
				Config: config,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(addr, plancheck.ResourceActionUpdate),
					},
				},
				Check: resource.TestCheckResourceAttr(addr, "storage_backend", "filesystem"),
			},
		},
	})
}

// testAccClient builds an API client from the acceptance-test environment.
func testAccClient(t *testing.T) *client.Client {
	t.Helper()
	c, err := client.New(client.Config{
		Endpoint: os.Getenv("ARTIFACT_KEEPER_ENDPOINT"),
		Token:    os.Getenv("ARTIFACT_KEEPER_TOKEN"),
	})
	if err != nil {
		t.Fatalf("building client: %v", err)
	}
	if os.Getenv("ARTIFACT_KEEPER_TOKEN") == "" {
		if err := c.Login(context.Background(), os.Getenv("ARTIFACT_KEEPER_USERNAME"), os.Getenv("ARTIFACT_KEEPER_PASSWORD")); err != nil {
			t.Fatalf("logging in: %v", err)
		}
	}
	return c
}
