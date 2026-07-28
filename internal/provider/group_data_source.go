package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	dschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/nicola-preda/terraform-provider-artifact-keeper/internal/client"
)

var (
	_ datasource.DataSource              = (*groupDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*groupDataSource)(nil)
)

func NewGroupDataSource() datasource.DataSource { return &groupDataSource{} }

type groupDataSource struct {
	client *client.Client
}

type groupDataSourceModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	MemberCount types.Int64  `tfsdk:"member_count"`
	CreatedAt   types.String `tfsdk:"created_at"`
	UpdatedAt   types.String `tfsdk:"updated_at"`
}

func (d *groupDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_group"
}

func (d *groupDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = dschema.Schema{
		MarkdownDescription: "Looks up an existing group by its UUID.",
		Attributes: map[string]dschema.Attribute{
			"id":           dschema.StringAttribute{Required: true, MarkdownDescription: "Group UUID to look up."},
			"name":         dschema.StringAttribute{Computed: true},
			"description":  dschema.StringAttribute{Computed: true},
			"member_count": dschema.Int64Attribute{Computed: true},
			"created_at":   dschema.StringAttribute{Computed: true},
			"updated_at":   dschema.StringAttribute{Computed: true},
		},
	}
}

func (d *groupDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = configureClient(req.ProviderData, &resp.Diagnostics)
}

func (d *groupDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var cfg groupDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &cfg)...)
	if resp.Diagnostics.HasError() {
		return
	}
	g, err := d.client.GetGroup(ctx, cfg.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading group", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, groupDataSourceModel{
		ID:          types.StringValue(g.ID),
		Name:        types.StringValue(g.Name),
		Description: stringPointerValue(g.Description),
		MemberCount: types.Int64Value(g.MemberCount),
		CreatedAt:   types.StringValue(g.CreatedAt),
		UpdatedAt:   types.StringValue(g.UpdatedAt),
	})...)
}
