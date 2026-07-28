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
	_ resource.Resource                = (*signingKeyResource)(nil)
	_ resource.ResourceWithConfigure   = (*signingKeyResource)(nil)
	_ resource.ResourceWithImportState = (*signingKeyResource)(nil)
)

// NewSigningKeyResource is the factory registered with the provider.
func NewSigningKeyResource() resource.Resource { return &signingKeyResource{} }

type signingKeyResource struct {
	client *client.Client
}

// signingKeyResourceModel maps the resource schema. Attribute names mirror the
// Artifact Keeper API fields exactly.
type signingKeyResourceModel struct {
	ID           types.String `tfsdk:"id"`
	RepositoryID types.String `tfsdk:"repository_id"`
	Name         types.String `tfsdk:"name"`
	KeyType      types.String `tfsdk:"key_type"`
	Fingerprint  types.String `tfsdk:"fingerprint"`
	KeyID        types.String `tfsdk:"key_id"`
	PublicKeyPEM types.String `tfsdk:"public_key_pem"`
	Algorithm    types.String `tfsdk:"algorithm"`
	UIDName      types.String `tfsdk:"uid_name"`
	UIDEmail     types.String `tfsdk:"uid_email"`
	ExpiresAt    types.String `tfsdk:"expires_at"`
	IsActive     types.Bool   `tfsdk:"is_active"`
	CreatedAt    types.String `tfsdk:"created_at"`
	LastUsedAt   types.String `tfsdk:"last_used_at"`
}

func (r *signingKeyResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_signing_key"
}

func (r *signingKeyResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a signing key used to sign repository metadata/packages in Artifact Keeper. Signing keys are immutable: any change forces a new key. Only the public key material is exposed; the private key is never returned.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Signing key UUID assigned by Artifact Keeper.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"repository_id": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "UUID of the repository this key belongs to. Omit for a global (instance-wide) key. Changing this forces a new key.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Display name for the signing key. Changing this forces a new key.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"key_type": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Key family: `rsa` (default) or `gpg`. Debian/RPM repositories sign metadata with OpenPGP and require `gpg`. Set the RSA key size via `algorithm`, not here, an RSA size variant passed as `key_type` is normalized to `rsa` by the server and causes a plan inconsistency. Changing this forces a new key.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace(), stringplanmodifier.UseStateForUnknown()},
			},
			"algorithm": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "RSA key size: `rsa4096` (default) or `rsa2048`. Changing this forces a new key.",
				Validators:          []validator.String{stringvalidator.OneOf("rsa2048", "rsa4096")},
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace(), stringplanmodifier.UseStateForUnknown()},
			},
			"uid_name": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "OpenPGP user-ID name embedded in a `gpg` key. Changing this forces a new key.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"uid_email": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "OpenPGP user-ID email embedded in a `gpg` key. Changing this forces a new key.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"fingerprint": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Key fingerprint, if applicable.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"key_id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "OpenPGP key ID, if applicable.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"public_key_pem": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The public key in PEM format, for client import. Not secret.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"expires_at": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "RFC 3339 expiry timestamp, if the key expires.",
			},
			"is_active": schema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Whether the key is currently active (revoking or rotating the key out-of-band clears this).",
			},
			"created_at": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "RFC 3339 creation timestamp.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"last_used_at": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "RFC 3339 timestamp of the key's last use, if ever used.",
			},
		},
	}
}

func (r *signingKeyResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = configureClient(req.ProviderData, &resp.Diagnostics)
}

func (r *signingKeyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan signingKeyResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	createReq := client.CreateSigningKeyRequest{Name: plan.Name.ValueString()}
	if !plan.RepositoryID.IsNull() {
		createReq.RepositoryID = plan.RepositoryID.ValueStringPointer()
	}
	if !plan.KeyType.IsNull() && !plan.KeyType.IsUnknown() {
		createReq.KeyType = plan.KeyType.ValueStringPointer()
	}
	if !plan.Algorithm.IsNull() && !plan.Algorithm.IsUnknown() {
		createReq.Algorithm = plan.Algorithm.ValueStringPointer()
	}
	if !plan.UIDName.IsNull() {
		createReq.UIDName = plan.UIDName.ValueStringPointer()
	}
	if !plan.UIDEmail.IsNull() {
		createReq.UIDEmail = plan.UIDEmail.ValueStringPointer()
	}

	key, err := r.client.CreateSigningKey(ctx, createReq)
	if err != nil {
		resp.Diagnostics.AddError("Error creating signing key", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, signingKeyToModel(key))...)
}

func (r *signingKeyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state signingKeyResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	key, err := r.client.GetSigningKey(ctx, state.ID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading signing key", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, signingKeyToModel(key))...)
}

// Update never runs: every field forces replacement. Here to satisfy the interface.
func (r *signingKeyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan signingKeyResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *signingKeyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state signingKeyResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.DeleteSigningKey(ctx, state.ID.ValueString()); err != nil && !client.IsNotFound(err) {
		resp.Diagnostics.AddError("Error deleting signing key", err.Error())
	}
}

func (r *signingKeyResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func signingKeyToModel(k *client.SigningKey) signingKeyResourceModel {
	return signingKeyResourceModel{
		ID:           types.StringValue(k.ID),
		RepositoryID: stringPointerValue(k.RepositoryID),
		Name:         types.StringValue(k.Name),
		KeyType:      types.StringValue(k.KeyType),
		Fingerprint:  stringPointerValue(k.Fingerprint),
		KeyID:        stringPointerValue(k.KeyID),
		PublicKeyPEM: types.StringValue(k.PublicKeyPEM),
		Algorithm:    types.StringValue(k.Algorithm),
		UIDName:      stringPointerValue(k.UIDName),
		UIDEmail:     stringPointerValue(k.UIDEmail),
		ExpiresAt:    stringPointerValue(k.ExpiresAt),
		IsActive:     types.BoolValue(k.IsActive),
		CreatedAt:    types.StringValue(k.CreatedAt),
		LastUsedAt:   stringPointerValue(k.LastUsedAt),
	}
}
