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
	_ resource.Resource                = (*repositoryUpstreamAuthResource)(nil)
	_ resource.ResourceWithConfigure   = (*repositoryUpstreamAuthResource)(nil)
	_ resource.ResourceWithImportState = (*repositoryUpstreamAuthResource)(nil)
)

func NewRepositoryUpstreamAuthResource() resource.Resource {
	return &repositoryUpstreamAuthResource{}
}

type repositoryUpstreamAuthResource struct {
	client *client.Client
}

type repositoryUpstreamAuthResourceModel struct {
	ID                 types.String `tfsdk:"id"`
	RepositoryKey      types.String `tfsdk:"repository_key"`
	AuthType           types.String `tfsdk:"auth_type"`
	Username           types.String `tfsdk:"username"`
	Password           types.String `tfsdk:"password"`
	Configured         types.Bool   `tfsdk:"configured"`
	ConfiguredAuthType types.String `tfsdk:"configured_auth_type"`
}

func (r *repositoryUpstreamAuthResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_repository_upstream_auth"
}

func (r *repositoryUpstreamAuthResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Upstream credentials a remote repository uses to authenticate to its origin. The credentials themselves are write-only (the backend never returns them, so every apply re-sends `username`/`password`), but the repository object reports whether auth is configured and its type, surfaced here as `configured`/`configured_auth_type` for drift detection. Setting `auth_type` to `none` clears the auth.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Resource identifier. Equal to `repository_key`.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"repository_key": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Key of the remote repository whose upstream credentials are managed. Changing this forces a new resource.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"auth_type": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Authentication type: `basic` (username + password), `bearer` (token in `password`), or `none` (removes the auth).",
				Validators:          []validator.String{stringvalidator.OneOf("basic", "bearer", "none")},
			},
			"username": schema.StringAttribute{
				Optional:            true,
				Sensitive:           true,
				MarkdownDescription: "Username for `basic` auth. Not returned by the API.",
			},
			"password": schema.StringAttribute{
				Optional:            true,
				Sensitive:           true,
				MarkdownDescription: "Password for `basic` auth or the token for `bearer` auth. Not returned by the API.",
			},
			"configured": schema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Whether upstream credentials are currently configured on the repository, read back from the repository object. Flips to `false` if the auth is cleared out of band.",
			},
			"configured_auth_type": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The upstream auth type the server currently reports for the repository (`basic` or `bearer`), or null when none is configured.",
			},
		},
	}
}

func (r *repositoryUpstreamAuthResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = configureClient(req.ProviderData, &resp.Diagnostics)
}

func (r *repositoryUpstreamAuthResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan repositoryUpstreamAuthResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	repoKey := plan.RepositoryKey.ValueString()
	if err := r.client.SetUpstreamAuth(ctx, repoKey, upstreamAuthRequestFromModel(plan)); err != nil {
		resp.Diagnostics.AddError("Error setting repository upstream auth", err.Error())
		return
	}

	plan.ID = types.StringValue(repoKey)
	if err := r.refreshConfigured(ctx, &plan); err != nil {
		resp.Diagnostics.AddError("Error reading back repository upstream auth", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

// Read refreshes the observable auth posture from the repository object. The
// credentials themselves have no GET and stay as prior state; `configured` /
// `configured_auth_type` give drift detection (and detect a deleted repository).
func (r *repositoryUpstreamAuthResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state repositoryUpstreamAuthResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	repo, err := r.client.GetRepository(ctx, state.RepositoryKey.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading repository upstream auth", err.Error())
		return
	}
	state.Configured = types.BoolValue(repo.UpstreamAuthConfigured)
	state.ConfiguredAuthType = stringPointerValue(repo.UpstreamAuthType)
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *repositoryUpstreamAuthResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan repositoryUpstreamAuthResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	repoKey := plan.RepositoryKey.ValueString()
	if err := r.client.SetUpstreamAuth(ctx, repoKey, upstreamAuthRequestFromModel(plan)); err != nil {
		resp.Diagnostics.AddError("Error updating repository upstream auth", err.Error())
		return
	}

	plan.ID = types.StringValue(repoKey)
	if err := r.refreshConfigured(ctx, &plan); err != nil {
		resp.Diagnostics.AddError("Error reading back repository upstream auth", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

// refreshConfigured reads the repository object and fills the observable auth
// posture (configured + type). The credentials themselves have no read endpoint.
func (r *repositoryUpstreamAuthResource) refreshConfigured(ctx context.Context, m *repositoryUpstreamAuthResourceModel) error {
	repo, err := r.client.GetRepository(ctx, m.RepositoryKey.ValueString())
	if err != nil {
		return err
	}
	m.Configured = types.BoolValue(repo.UpstreamAuthConfigured)
	m.ConfiguredAuthType = stringPointerValue(repo.UpstreamAuthType)
	return nil
}

func (r *repositoryUpstreamAuthResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state repositoryUpstreamAuthResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Clearing the auth is a PUT with auth_type "none" and no credentials.
	req2 := client.UpstreamAuthRequest{AuthType: "none"}
	if err := r.client.SetUpstreamAuth(ctx, state.RepositoryKey.ValueString(), req2); err != nil {
		resp.Diagnostics.AddError("Error clearing repository upstream auth", err.Error())
	}
}

func (r *repositoryUpstreamAuthResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Addressed by repository key. The credentials cannot be read back (no GET),
	// so username/password stay unset after import until the next apply re-sends them.
	resource.ImportStatePassthroughID(ctx, path.Root("repository_key"), req, resp)
}

func upstreamAuthRequestFromModel(m repositoryUpstreamAuthResourceModel) client.UpstreamAuthRequest {
	return client.UpstreamAuthRequest{
		AuthType: m.AuthType.ValueString(),
		Username: optionalString(m.Username),
		Password: optionalString(m.Password),
	}
}
