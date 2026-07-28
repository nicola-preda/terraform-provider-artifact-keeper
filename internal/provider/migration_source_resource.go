package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/nicola-preda/terraform-provider-artifact-keeper/internal/client"
)

var (
	_ resource.Resource                = (*migrationSourceResource)(nil)
	_ resource.ResourceWithConfigure   = (*migrationSourceResource)(nil)
	_ resource.ResourceWithImportState = (*migrationSourceResource)(nil)
)

func NewMigrationSourceResource() resource.Resource { return &migrationSourceResource{} }

type migrationSourceResource struct {
	client *client.Client
}

type migrationSourceResourceModel struct {
	ID                 types.String `tfsdk:"id"`
	Name               types.String `tfsdk:"name"`
	URL                types.String `tfsdk:"url"`
	AuthType           types.String `tfsdk:"auth_type"`
	SourceType         types.String `tfsdk:"source_type"`
	CredentialToken    types.String `tfsdk:"credential_token"`
	CredentialUsername types.String `tfsdk:"credential_username"`
	CredentialPassword types.String `tfsdk:"credential_password"`
	CreatedAt          types.String `tfsdk:"created_at"`
	VerifiedAt         types.String `tfsdk:"verified_at"`
}

func (r *migrationSourceResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_migration_source"
}

func (r *migrationSourceResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "A persistent migration source connection (e.g. a Nexus or Artifactory instance) used to import repositories and artifacts. Credentials are not returned by the API. The API has no update endpoint, so changes force a new connection. Running a migration is a separate imperative operation, not managed here.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Connection name.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"url": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Base URL of the source registry (e.g. `https://nexus.example.com`).",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"auth_type": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Authentication type: `basic_auth` (Nexus) or `api_token` (Artifactory).",
				Validators:          []validator.String{stringvalidator.OneOf("api_token", "basic_auth")},
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"source_type": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Source registry type: `nexus` or `artifactory`. Defaults to `artifactory`.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace(), stringplanmodifier.UseStateForUnknown()},
			},
			"credential_token": schema.StringAttribute{
				Optional:            true,
				Sensitive:           true,
				MarkdownDescription: "API token for `api_token` auth. Not returned by the API.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"credential_username": schema.StringAttribute{
				Optional:            true,
				Sensitive:           true,
				MarkdownDescription: "Username for `basic_auth`. Not returned by the API.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"credential_password": schema.StringAttribute{
				Optional:            true,
				Sensitive:           true,
				MarkdownDescription: "Password for `basic_auth`. Not returned by the API.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"created_at": schema.StringAttribute{
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"verified_at": schema.StringAttribute{Computed: true},
		},
	}
}

func (r *migrationSourceResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = configureClient(req.ProviderData, &resp.Diagnostics)
}

func (r *migrationSourceResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan migrationSourceResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	createReq := client.CreateMigrationSourceRequest{
		Name:     plan.Name.ValueString(),
		URL:      plan.URL.ValueString(),
		AuthType: plan.AuthType.ValueString(),
		Credentials: client.MigrationCredentials{
			Token:    optionalString(plan.CredentialToken),
			Username: optionalString(plan.CredentialUsername),
			Password: optionalString(plan.CredentialPassword),
		},
	}
	if !plan.SourceType.IsNull() && !plan.SourceType.IsUnknown() {
		createReq.SourceType = plan.SourceType.ValueStringPointer()
	}

	src, err := r.client.CreateMigrationSource(ctx, createReq)
	if err != nil {
		resp.Diagnostics.AddError("Error creating migration source", err.Error())
		return
	}

	state := migrationSourceToModel(src)
	state.CredentialToken = plan.CredentialToken
	state.CredentialUsername = plan.CredentialUsername
	state.CredentialPassword = plan.CredentialPassword
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *migrationSourceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state migrationSourceResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	src, err := r.client.GetMigrationSource(ctx, state.ID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading migration source", err.Error())
		return
	}

	refreshed := migrationSourceToModel(src)
	refreshed.CredentialToken = state.CredentialToken
	refreshed.CredentialUsername = state.CredentialUsername
	refreshed.CredentialPassword = state.CredentialPassword
	resp.Diagnostics.Append(resp.State.Set(ctx, refreshed)...)
}

// Update is unreachable (no update endpoint; all attributes force replacement).
func (r *migrationSourceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan migrationSourceResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *migrationSourceResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state migrationSourceResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.DeleteMigrationSource(ctx, state.ID.ValueString()); err != nil && !client.IsNotFound(err) {
		resp.Diagnostics.AddError("Error deleting migration source", err.Error())
	}
}

func (r *migrationSourceResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func migrationSourceToModel(s *client.MigrationSource) migrationSourceResourceModel {
	return migrationSourceResourceModel{
		ID:         types.StringValue(s.ID),
		Name:       types.StringValue(s.Name),
		URL:        types.StringValue(s.URL),
		AuthType:   types.StringValue(s.AuthType),
		SourceType: types.StringValue(s.SourceType),
		CreatedAt:  types.StringValue(s.CreatedAt),
		VerifiedAt: stringPointerValue(s.VerifiedAt),
	}
}

// optionalString returns a *string for a set value, or nil when null/unknown.
func optionalString(v types.String) *string {
	if v.IsNull() || v.IsUnknown() {
		return nil
	}
	return v.ValueStringPointer()
}
