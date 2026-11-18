package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestKeepEmptyList(t *testing.T) {
	empty := types.ListValueMust(types.StringType, []attr.Value{})
	null := types.ListNull(types.StringType)
	one := types.ListValueMust(types.StringType, []attr.Value{types.StringValue("a")})
	cases := []struct {
		name             string
		prior, got, want types.List
	}{
		{"configured [] read back null", empty, null, empty},
		{"unset stays null", null, null, null},
		{"unknown prior", types.ListUnknown(types.StringType), null, null},
		{"drift from []", empty, one, one},
		{"cleared outside", one, null, null},
	}
	for _, c := range cases {
		if got := keepEmptyList(c.prior, c.got); !got.Equal(c.want) {
			t.Errorf("%s: got %s, want %s", c.name, got, c.want)
		}
	}
	emptyMap := types.MapValueMust(types.StringType, map[string]attr.Value{})
	if got := keepEmptyMap(emptyMap, types.MapNull(types.StringType)); !got.Equal(emptyMap) {
		t.Errorf("map: got %s, want {}", got)
	}
}

func TestSyncPolicyKeepEmpty(t *testing.T) {
	empty := types.ListValueMust(types.StringType, []attr.Value{})
	emptyMap := types.MapValueMust(types.StringType, map[string]attr.Value{})
	null := types.ListNull(types.StringType)
	nullMap := types.MapNull(types.StringType)
	formats := types.ListValueMust(types.StringType, []attr.Value{types.StringValue("docker")})

	prior := syncPolicyResourceModel{
		RepoSelector:   &repoSelectorModel{MatchLabels: emptyMap, MatchFormats: formats, MatchPattern: types.StringNull(), MatchRepos: empty},
		PeerSelector:   &peerSelectorModel{All: types.BoolNull(), MatchLabels: emptyMap, MatchRegion: types.StringNull(), MatchPeers: empty},
		ArtifactFilter: &artifactFilterModel{MaxAgeDays: types.Int64Null(), IncludePaths: empty, ExcludePaths: null, MaxSizeBytes: types.Int64Null(), MatchTags: nullMap},
	}
	got := syncPolicyResourceModel{
		RepoSelector: &repoSelectorModel{MatchLabels: nullMap, MatchFormats: formats, MatchPattern: types.StringNull(), MatchRepos: null},
	}
	got.keepEmpty(prior)
	if got.RepoSelector.MatchRepos.IsNull() || got.RepoSelector.MatchLabels.IsNull() {
		t.Errorf("repo_selector lost its empty collections: %+v", got.RepoSelector)
	}
	if got.PeerSelector == nil || got.PeerSelector.MatchPeers.IsNull() {
		t.Errorf("an all-empty peer_selector read back null")
	}
	if got.ArtifactFilter == nil || got.ArtifactFilter.IncludePaths.IsNull() || !got.ArtifactFilter.ExcludePaths.IsNull() {
		t.Errorf("artifact_filter = %+v", got.ArtifactFilter)
	}

	withRegion := syncPolicyResourceModel{PeerSelector: &peerSelectorModel{All: types.BoolNull(), MatchLabels: nullMap, MatchRegion: types.StringValue("eu"), MatchPeers: null}}
	var cleared syncPolicyResourceModel
	cleared.keepEmpty(withRegion)
	if cleared.PeerSelector != nil {
		t.Errorf("a selector cleared outside Terraform came back from prior")
	}
}

func TestAllowlistKeepEmpty(t *testing.T) {
	ctx := context.Background()
	objType := types.ObjectType{AttrTypes: allowlistEntryAttrTypes}
	entry := func(subdirs types.List) attr.Value {
		return types.ObjectValueMust(allowlistEntryAttrTypes, map[string]attr.Value{
			"name": types.StringValue("numpy"), "version": types.StringNull(), "subdirs": subdirs,
		})
	}
	empty := types.ListValueMust(types.StringType, []attr.Value{})
	null := types.ListNull(types.StringType)

	m := repositoryAllowlistResourceModel{Entries: types.ListNull(objType)}
	if d := m.keepEmpty(ctx, repositoryAllowlistResourceModel{Entries: types.ListValueMust(objType, []attr.Value{})}); d.HasError() {
		t.Fatal(d)
	}
	if m.Entries.IsNull() {
		t.Error("entries = [] read back null")
	}

	m = repositoryAllowlistResourceModel{Entries: types.ListValueMust(objType, []attr.Value{entry(null)})}
	if d := m.keepEmpty(ctx, repositoryAllowlistResourceModel{Entries: types.ListValueMust(objType, []attr.Value{entry(empty)})}); d.HasError() {
		t.Fatal(d)
	}
	want := types.ListValueMust(objType, []attr.Value{entry(empty)})
	if !m.Entries.Equal(want) {
		t.Errorf("entries = %s, want subdirs = []", m.Entries)
	}
}

// TestAccEmptyCollections sets [] and {} on every optional collection that
// used to read back null after apply (same cause as #7), then non-empty
// values, then [] again. Requires TF_ACC and a live instance
func TestAccEmptyCollections(t *testing.T) {
	config := func(full bool) string {
		v := func(set, unset string) string {
			if full {
				return set
			}
			return unset
		}
		return `
provider "artifactkeeper" {}

resource "artifactkeeper_repository" "npm" {
  key                       = "tf-acc-empty-npm"
  name                      = "TF Acc empty npm"
  format                    = "npm"
  repo_type                 = "remote"
  upstream_url              = "https://registry.npmjs.org"
  npm_allowed_name_patterns = ` + v(`["internal-*"]`, `[]`) + `
}

resource "artifactkeeper_repository_npm_scope_policy" "s" {
  repository_key = artifactkeeper_repository.npm.key
  allowed_scopes = ` + v(`["@acme"]`, `[]`) + `
  allow_unscoped = true
}

resource "artifactkeeper_repository" "conda_remote" {
  key          = "tf-acc-empty-conda-remote"
  name         = "TF Acc empty conda remote"
  format       = "conda"
  repo_type    = "remote"
  upstream_url = "https://conda.anaconda.org/conda-forge"
}

resource "artifactkeeper_repository" "conda_virtual" {
  key       = "tf-acc-empty-conda-virtual"
  name      = "TF Acc empty conda virtual"
  format    = "conda"
  repo_type = "virtual"
  members   = [artifactkeeper_repository.conda_remote.key]
}

resource "artifactkeeper_repository_allowlist" "a" {
  repository_key = artifactkeeper_repository.conda_virtual.key
  enabled        = true
  entries        = [{ name = "numpy", subdirs = ` + v(`["linux-64"]`, `[]`) + ` }]
}

resource "artifactkeeper_repository" "staging" {
  key       = "tf-acc-empty-staging"
  name      = "TF Acc empty staging"
  format    = "generic"
  repo_type = "staging"
}

resource "artifactkeeper_repository" "local" {
  key       = "tf-acc-empty-local"
  name      = "TF Acc empty local"
  format    = "generic"
  repo_type = "local"
}

resource "artifactkeeper_promotion_rule" "p" {
  name             = "tf-acc-empty-promo"
  source_repo_id   = artifactkeeper_repository.staging.id
  target_repo_id   = artifactkeeper_repository.local.id
  allowed_licenses = ` + v(`["MIT"]`, `[]`) + `
}

resource "artifactkeeper_ci_oidc_provider" "gl" {
  name          = "tf-acc-empty-gitlab"
  provider_type = "gitlab"
  issuer_url    = "https://gitlab.example.com"
}

resource "artifactkeeper_ci_oidc_identity_mapping" "m" {
  provider_id      = artifactkeeper_ci_oidc_provider.gl.id
  name             = "tf-acc-empty-mapping"
  claim_filters    = jsonencode({ project_path = "g/r" })
  allowed_repo_ids = ` + v(`[artifactkeeper_repository.local.id]`, `[]`) + `
}

resource "artifactkeeper_sync_policy" "s" {
  name             = "tf-acc-empty-sync"
  replication_mode = "mirror"
  repo_selector = {
    match_formats = ["docker"]
    match_labels  = ` + v(`{ team = "x" }`, `{}`) + `
    match_repos   = ` + v(`[artifactkeeper_repository.local.id]`, `[]`) + `
  }
  artifact_filter = {
    include_paths = ` + v(`["^dist/"]`, `[]`) + `
    match_tags    = ` + v(`{ tier = "prod" }`, `{}`) + `
  }
}

resource "artifactkeeper_webhook" "w" {
  name    = "tf-acc-empty-webhook"
  url     = "https://hooks.example.com/tf-acc-empty"
  events  = ["artifact_uploaded"]
  headers = ` + v(`{ X-Team = "x" }`, `{}`) + `
}
`
	}
	check := func(n string) resource.TestCheckFunc {
		return resource.ComposeAggregateTestCheckFunc(
			resource.TestCheckResourceAttr("artifactkeeper_repository.npm", "npm_allowed_name_patterns.#", n),
			resource.TestCheckResourceAttr("artifactkeeper_repository_npm_scope_policy.s", "allowed_scopes.#", n),
			resource.TestCheckResourceAttr("artifactkeeper_repository_allowlist.a", "entries.0.subdirs.#", n),
			resource.TestCheckResourceAttr("artifactkeeper_promotion_rule.p", "allowed_licenses.#", n),
			resource.TestCheckResourceAttr("artifactkeeper_ci_oidc_identity_mapping.m", "allowed_repo_ids.#", n),
			resource.TestCheckResourceAttr("artifactkeeper_sync_policy.s", "repo_selector.match_repos.#", n),
			resource.TestCheckResourceAttr("artifactkeeper_sync_policy.s", "artifact_filter.include_paths.#", n),
			resource.TestCheckResourceAttr("artifactkeeper_webhook.w", "headers.%", n),
		)
	}
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{Config: config(false), Check: check("0")},
			{Config: config(true), Check: check("1")},
			{Config: config(false), Check: check("0")},
		},
	})
}
