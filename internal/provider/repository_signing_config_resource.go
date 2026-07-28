package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/nicola-preda/terraform-provider-artifact-keeper/internal/client"
)

var (
	_ resource.Resource                = (*repositorySigningConfigResource)(nil)
	_ resource.ResourceWithConfigure   = (*repositorySigningConfigResource)(nil)
	_ resource.ResourceWithImportState = (*repositorySigningConfigResource)(nil)
)

func NewRepositorySigningConfigResource() resource.Resource {
	return &repositorySigningConfigResource{}
}

type repositorySigningConfigResource struct {
	client *client.Client
}

type repositorySigningConfigResourceModel struct {
	ID                types.String `tfsdk:"id"`
	RepositoryID      types.String `tfsdk:"repository_id"`
	SigningKeyID      types.String `tfsdk:"signing_key_id"`
	SignMetadata      types.Bool   `tfsdk:"sign_metadata"`
	SignPackages      types.Bool   `tfsdk:"sign_packages"`
	RequireSignatures types.Bool   `tfsdk:"require_signatures"`
}

func (r *repositorySigningConfigResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_repository_signing_config"
}

func (r *repositorySigningConfigResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Per-repository signing configuration, which key signs, and whether metadata/packages are signed or signatures required. A singleton keyed by repository ID: no create/delete (the config is reset when the repository is removed), and writes upsert. Admin-only.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Resource identifier. Equal to `repository_id`, since the config is a singleton per repository.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"repository_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "UUID of the repository whose signing config is managed. Changing this forces a new resource.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"signing_key_id": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "UUID of the `artifactkeeper_signing_key` used to sign this repository's artifacts/metadata. Defaults to none (unconfigured). Note: the backend upsert merges over the existing config, so omitting this keeps the previously configured key rather than clearing it.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"sign_metadata": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Whether repository metadata is signed. Defaults to `false`.",
				PlanModifiers:       []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"sign_packages": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Whether packages/artifacts are signed. Defaults to `false`.",
				PlanModifiers:       []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"require_signatures": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Whether artifacts must carry a valid signature (e.g. to pass the promotion gate). Defaults to `false`.",
				PlanModifiers:       []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
		},
	}
}

func (r *repositorySigningConfigResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = configureClient(req.ProviderData, &resp.Diagnostics)
}

func (r *repositorySigningConfigResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan repositorySigningConfigResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	repositoryID := plan.RepositoryID.ValueString()

	// No create endpoint: config always exists (defaulted). Upsert POST establishes it.
	if _, err := r.client.UpdateRepositorySigningConfig(ctx, repositoryID, signingConfigRequestFromModel(plan)); err != nil {
		resp.Diagnostics.AddError("Error configuring repository signing config", err.Error())
		return
	}

	// Read back via GET; Create's POST and Read's GET return different shapes.
	cfg, err := r.client.GetRepositorySigningConfig(ctx, repositoryID)
	if err != nil {
		resp.Diagnostics.AddError("Error reading repository signing config", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, signingConfigToModel(cfg))...)
}

func (r *repositorySigningConfigResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state repositorySigningConfigResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	cfg, err := r.client.GetRepositorySigningConfig(ctx, state.RepositoryID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading repository signing config", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, signingConfigToModel(cfg))...)
}

func (r *repositorySigningConfigResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan repositorySigningConfigResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// POST upserts, so it's the real update path.
	cfg, err := r.client.UpdateRepositorySigningConfig(ctx, plan.RepositoryID.ValueString(), signingConfigRequestFromModel(plan))
	if err != nil {
		resp.Diagnostics.AddError("Error updating repository signing config", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, signingConfigToModel(cfg))...)
}

// No-op: no delete/reset endpoint. Config lives with the repository; destroy
// just drops it from state.
func (r *repositorySigningConfigResource) Delete(_ context.Context, _ resource.DeleteRequest, _ *resource.DeleteResponse) {
}

func (r *repositorySigningConfigResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Addressed by repository ID in the API; import by that ID.
	resource.ImportStatePassthroughID(ctx, path.Root("repository_id"), req, resp)
}

// signingConfigRequestFromModel builds the upsert payload, sending only fields
// the caller set. Omitted fields keep the server's current/default value.
func signingConfigRequestFromModel(m repositorySigningConfigResourceModel) client.UpdateSigningConfigRequest {
	var req client.UpdateSigningConfigRequest
	if !m.SigningKeyID.IsNull() && !m.SigningKeyID.IsUnknown() {
		req.SigningKeyID = m.SigningKeyID.ValueStringPointer()
	}
	if !m.SignMetadata.IsNull() && !m.SignMetadata.IsUnknown() {
		req.SignMetadata = m.SignMetadata.ValueBoolPointer()
	}
	if !m.SignPackages.IsNull() && !m.SignPackages.IsUnknown() {
		req.SignPackages = m.SignPackages.ValueBoolPointer()
	}
	if !m.RequireSignatures.IsNull() && !m.RequireSignatures.IsUnknown() {
		req.RequireSignatures = m.RequireSignatures.ValueBoolPointer()
	}
	return req
}

func signingConfigToModel(c *client.RepositorySigningConfig) repositorySigningConfigResourceModel {
	return repositorySigningConfigResourceModel{
		ID:                types.StringValue(c.RepositoryID),
		RepositoryID:      types.StringValue(c.RepositoryID),
		SigningKeyID:      stringPointerValue(c.SigningKeyID),
		SignMetadata:      types.BoolValue(c.SignMetadata),
		SignPackages:      types.BoolValue(c.SignPackages),
		RequireSignatures: types.BoolValue(c.RequireSignatures),
	}
}
