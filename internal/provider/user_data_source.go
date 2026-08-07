package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	dschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/nicola-preda/terraform-provider-artifact-keeper/internal/client"
)

var (
	_ datasource.DataSource              = (*userDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*userDataSource)(nil)
)

func NewUserDataSource() datasource.DataSource { return &userDataSource{} }

type userDataSource struct {
	client *client.Client
}

// userDataSourceModel omits the write-in-only password fields the resource has.
type userDataSourceModel struct {
	ID                 types.String `tfsdk:"id"`
	Username           types.String `tfsdk:"username"`
	Email              types.String `tfsdk:"email"`
	DisplayName        types.String `tfsdk:"display_name"`
	AuthProvider       types.String `tfsdk:"auth_provider"`
	IsActive           types.Bool   `tfsdk:"is_active"`
	IsAdmin            types.Bool   `tfsdk:"is_admin"`
	MustChangePassword types.Bool   `tfsdk:"must_change_password"`
	LastLoginAt        types.String `tfsdk:"last_login_at"`
	CreatedAt          types.String `tfsdk:"created_at"`
}

func (d *userDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_user"
}

func (d *userDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = dschema.Schema{
		MarkdownDescription: "Looks up an existing user by its username, resolving the UUID (`id`) you pass to memberships and role assignments.",
		Attributes: map[string]dschema.Attribute{
			"username":             dschema.StringAttribute{Required: true, MarkdownDescription: "Username to look up."},
			"id":                   dschema.StringAttribute{Computed: true, MarkdownDescription: "Resolved user UUID."},
			"email":                dschema.StringAttribute{Computed: true},
			"display_name":         dschema.StringAttribute{Computed: true},
			"auth_provider":        dschema.StringAttribute{Computed: true},
			"is_active":            dschema.BoolAttribute{Computed: true},
			"is_admin":             dschema.BoolAttribute{Computed: true},
			"must_change_password": dschema.BoolAttribute{Computed: true},
			"last_login_at":        dschema.StringAttribute{Computed: true},
			"created_at":           dschema.StringAttribute{Computed: true},
		},
	}
}

func (d *userDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = configureClient(req.ProviderData, &resp.Diagnostics)
}

func (d *userDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var cfg userDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &cfg)...)
	if resp.Diagnostics.HasError() {
		return
	}
	u, err := d.client.FindUserByUsername(ctx, cfg.Username.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading user", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, userDataSourceModel{
		ID:                 types.StringValue(u.ID),
		Username:           types.StringValue(u.Username),
		Email:              types.StringValue(u.Email),
		DisplayName:        stringPointerValue(u.DisplayName),
		AuthProvider:       types.StringValue(u.AuthProvider),
		IsActive:           types.BoolValue(u.IsActive),
		IsAdmin:            types.BoolValue(u.IsAdmin),
		MustChangePassword: types.BoolValue(u.MustChangePassword),
		LastLoginAt:        stringPointerValue(u.LastLoginAt),
		CreatedAt:          types.StringValue(u.CreatedAt),
	})...)
}
