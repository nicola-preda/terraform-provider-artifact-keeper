package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	dschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/nicola-preda/terraform-provider-artifact-keeper/internal/client"
)

var (
	_ datasource.DataSource              = (*repositoryDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*repositoryDataSource)(nil)
)

func NewRepositoryDataSource() datasource.DataSource { return &repositoryDataSource{} }

type repositoryDataSource struct {
	client *client.Client
}

func (d *repositoryDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_repository"
}

func (d *repositoryDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = dschema.Schema{
		MarkdownDescription: "Looks up an existing repository by its key.",
		Attributes: map[string]dschema.Attribute{
			"key":                      dschema.StringAttribute{Required: true, MarkdownDescription: "Repository key to look up."},
			"id":                       dschema.StringAttribute{Computed: true},
			"name":                     dschema.StringAttribute{Computed: true},
			"description":              dschema.StringAttribute{Computed: true},
			"format":                   dschema.StringAttribute{Computed: true},
			"repo_type":                dschema.StringAttribute{Computed: true},
			"is_public":                dschema.BoolAttribute{Computed: true},
			"allow_anonymous_access":   dschema.BoolAttribute{Computed: true},
			"storage_used_bytes":       dschema.Int64Attribute{Computed: true},
			"quota_bytes":              dschema.Int64Attribute{Computed: true},
			"upstream_url":             dschema.StringAttribute{Computed: true},
			"upstream_auth_type":       dschema.StringAttribute{Computed: true},
			"upstream_auth_configured": dschema.BoolAttribute{Computed: true},
			"created_at":               dschema.StringAttribute{Computed: true},
			"updated_at":               dschema.StringAttribute{Computed: true},
			"members": dschema.ListAttribute{
				Computed:            true,
				ElementType:         types.StringType,
				MarkdownDescription: "For `virtual` repositories: ordered member repository keys. Null for non-virtual repositories.",
			},
			"promotion_only":      dschema.BoolAttribute{Computed: true, MarkdownDescription: "Whether direct uploads are rejected (promotion-only)."},
			"versioning_enabled":  dschema.BoolAttribute{Computed: true, MarkdownDescription: "Whether first-class artifact versioning is enabled."},
			"project_id":          dschema.StringAttribute{Computed: true, MarkdownDescription: "UUID of the assigned project, if any."},
			"has_trusted_gpg_key": dschema.BoolAttribute{Computed: true, MarkdownDescription: "Whether a trusted upstream GPG public key is configured."},
			"custom_user_agent":   dschema.StringAttribute{Computed: true, MarkdownDescription: "Custom outbound User-Agent, if set."},
			"apt_origin":          dschema.StringAttribute{Computed: true, MarkdownDescription: "Custom APT Release Origin, if set."},
			"apt_label":           dschema.StringAttribute{Computed: true, MarkdownDescription: "Custom APT Release Label, if set."},
			"apt_release_version": dschema.StringAttribute{Computed: true, MarkdownDescription: "Custom APT Release Version, if set."},
			"apt_description":     dschema.StringAttribute{Computed: true, MarkdownDescription: "Custom APT Release Description, if set."},
			// Write-only on the resource; never returned by the API, so always null here.
			"trusted_gpg_key":          dschema.StringAttribute{Computed: true, Sensitive: true, MarkdownDescription: "Write-only; never returned (see `has_trusted_gpg_key`)."},
			"storage_backend":          dschema.StringAttribute{Computed: true, MarkdownDescription: "Write-only; not returned by the API."},
			"format_key":               dschema.StringAttribute{Computed: true, MarkdownDescription: "Write-only; not returned by the API."},
			"index_upstream_url":       dschema.StringAttribute{Computed: true, MarkdownDescription: "Write-only; not returned by the API."},
			"pypi_upstream_index_path": dschema.StringAttribute{Computed: true, MarkdownDescription: "Write-only; not returned by the API."},
		},
	}
}

func (d *repositoryDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = configureClient(req.ProviderData, &resp.Diagnostics)
}

func (d *repositoryDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var cfg repositoryResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &cfg)...)
	if resp.Diagnostics.HasError() {
		return
	}
	repo, err := d.client.GetRepository(ctx, cfg.Key.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading repository", err.Error())
		return
	}
	// repositoryResourceModel matches this data source's schema, so reuse the mapper.
	model := repositoryToModel(repo)
	if repo.RepoType == "virtual" {
		members, err := d.client.GetVirtualMembers(ctx, repo.Key)
		if err != nil {
			resp.Diagnostics.AddError("Error reading virtual repository members", err.Error())
			return
		}
		keys := make([]string, len(members))
		for i, m := range members {
			keys[i] = m.MemberRepoKey
		}
		lv, d := types.ListValueFrom(ctx, types.StringType, keys)
		resp.Diagnostics.Append(d...)
		if resp.Diagnostics.HasError() {
			return
		}
		model.Members = lv
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, model)...)
}
