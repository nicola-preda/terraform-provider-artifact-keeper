package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	dschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/nicola-preda/terraform-provider-artifact-keeper/internal/client"
)

var (
	_ datasource.DataSource              = (*projectDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*projectDataSource)(nil)
)

func NewProjectDataSource() datasource.DataSource { return &projectDataSource{} }

type projectDataSource struct {
	client *client.Client
}

type projectDataSourceModel struct {
	Key         types.String `tfsdk:"key"`
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	QuotaBytes  types.Int64  `tfsdk:"quota_bytes"`
	CreatedAt   types.String `tfsdk:"created_at"`
	UpdatedAt   types.String `tfsdk:"updated_at"`
}

func (d *projectDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_project"
}

func (d *projectDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = dschema.Schema{
		MarkdownDescription: "Looks up an existing project by its key, resolving the UUID (`id`) you pass to `repository.project_id` and project memberships.",
		Attributes: map[string]dschema.Attribute{
			"key":         dschema.StringAttribute{Required: true, MarkdownDescription: "Project key to look up."},
			"id":          dschema.StringAttribute{Computed: true, MarkdownDescription: "Resolved project UUID."},
			"name":        dschema.StringAttribute{Computed: true},
			"description": dschema.StringAttribute{Computed: true},
			"quota_bytes": dschema.Int64Attribute{Computed: true},
			"created_at":  dschema.StringAttribute{Computed: true},
			"updated_at":  dschema.StringAttribute{Computed: true},
		},
	}
}

func (d *projectDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = configureClient(req.ProviderData, &resp.Diagnostics)
}

func (d *projectDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var cfg projectDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &cfg)...)
	if resp.Diagnostics.HasError() {
		return
	}
	key := cfg.Key.ValueString()
	projects, err := d.client.ListProjects(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Error reading project", err.Error())
		return
	}
	for _, p := range projects {
		if p.Key != key {
			continue
		}
		resp.Diagnostics.Append(resp.State.Set(ctx, projectDataSourceModel{
			Key:         types.StringValue(p.Key),
			ID:          types.StringValue(p.ID),
			Name:        types.StringValue(p.Name),
			Description: stringPointerValue(p.Description),
			QuotaBytes:  int64PointerValue(p.QuotaBytes),
			CreatedAt:   types.StringValue(p.CreatedAt),
			UpdatedAt:   types.StringValue(p.UpdatedAt),
		})...)
		return
	}
	resp.Diagnostics.AddError("Error reading project", fmt.Sprintf("no project with key %q", key))
}
