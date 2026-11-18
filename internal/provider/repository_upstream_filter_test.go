package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/nicola-preda/terraform-provider-artifact-keeper/internal/client"
)

// The API answers [] for an empty and for an unset list, #7
func TestUpstreamFilterToModelKeepsEmptyLists(t *testing.T) {
	ctx := context.Background()
	empty := types.ListValueMust(types.StringType, []attr.Value{})
	null := types.ListNull(types.StringType)
	one := types.ListValueMust(types.StringType, []attr.Value{types.StringValue("^v2/")})

	cases := []struct {
		name             string
		prior            repositoryUpstreamFilterResourceModel
		resp             client.UpstreamFilterResponse
		wantInc, wantExc types.List
	}{
		{"include set, exclude []", repositoryUpstreamFilterResourceModel{IncludePatterns: one, ExcludePatterns: empty}, client.UpstreamFilterResponse{IncludePatterns: []string{"^v2/"}, ExcludePatterns: []string{}}, one, empty},
		{"both []", repositoryUpstreamFilterResourceModel{IncludePatterns: empty, ExcludePatterns: empty}, client.UpstreamFilterResponse{IncludePatterns: []string{}, ExcludePatterns: []string{}}, empty, empty},
		{"both unset", repositoryUpstreamFilterResourceModel{IncludePatterns: null, ExcludePatterns: null}, client.UpstreamFilterResponse{IncludePatterns: []string{}, ExcludePatterns: []string{}}, null, null},
		{"import", repositoryUpstreamFilterResourceModel{IncludePatterns: null, ExcludePatterns: null}, client.UpstreamFilterResponse{IncludePatterns: []string{"^v2/"}}, one, null},
		{"drift from [] to a pattern", repositoryUpstreamFilterResourceModel{IncludePatterns: empty, ExcludePatterns: empty}, client.UpstreamFilterResponse{IncludePatterns: []string{"^v2/"}}, one, empty},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			c.prior.RepositoryKey = types.StringValue("r")
			m, diags := upstreamFilterToModel(ctx, c.prior, &c.resp)
			if diags.HasError() {
				t.Fatal(diags)
			}
			if !m.IncludePatterns.Equal(c.wantInc) {
				t.Errorf("include_patterns = %s, want %s", m.IncludePatterns, c.wantInc)
			}
			if !m.ExcludePatterns.Equal(c.wantExc) {
				t.Errorf("exclude_patterns = %s, want %s", m.ExcludePatterns, c.wantExc)
			}
		})
	}
}

// TestAccRepositoryUpstreamFilterEmptyLists applies explicit empty pattern
// lists on create, update and refresh, #7. Requires TF_ACC and a live instance
func TestAccRepositoryUpstreamFilterEmptyLists(t *testing.T) {
	config := func(patterns string) string {
		return `
provider "artifactkeeper" {}

resource "artifactkeeper_repository" "remote" {
  key          = "tf-acc-filter-empty"
  name         = "TF Acc filter empty lists"
  format       = "helm"
  repo_type    = "remote"
  upstream_url = "https://example.com/"
}

resource "artifactkeeper_repository_upstream_filter" "f" {
  repository_key = artifactkeeper_repository.remote.key
` + patterns + `
}
`
	}
	const f = "artifactkeeper_repository_upstream_filter.f"
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config(`  include_patterns = ["^v2/test/helm/"]
  exclude_patterns = []`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(f, "include_patterns.#", "1"),
					resource.TestCheckResourceAttr(f, "exclude_patterns.#", "0"),
					resource.TestCheckResourceAttr(f, "active", "true"),
				),
			},
			{
				Config: config(`  include_patterns = []
  exclude_patterns = ["\\.exe$"]`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(f, "include_patterns.#", "0"),
					resource.TestCheckResourceAttr(f, "exclude_patterns.#", "1"),
				),
			},
			{
				Config: config(`  include_patterns = []
  exclude_patterns = []`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(f, "include_patterns.#", "0"),
					resource.TestCheckResourceAttr(f, "exclude_patterns.#", "0"),
					resource.TestCheckResourceAttr(f, "active", "false"),
				),
			},
			{
				Config: config(`  include_patterns = ["^v2/"]`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(f, "include_patterns.#", "1"),
					resource.TestCheckNoResourceAttr(f, "exclude_patterns"),
				),
			},
		},
	})
}
