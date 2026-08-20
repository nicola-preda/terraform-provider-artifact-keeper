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

// totpPolicyID is the fixed id for this singleton (no per-object id
// server-side), giving Terraform a stable address to import and refresh.
const totpPolicyID = "totp_policy"

var (
	_ resource.Resource                = (*totpPolicyResource)(nil)
	_ resource.ResourceWithConfigure   = (*totpPolicyResource)(nil)
	_ resource.ResourceWithImportState = (*totpPolicyResource)(nil)
)

func NewTotpPolicyResource() resource.Resource { return &totpPolicyResource{} }

type totpPolicyResource struct {
	client *client.Client
}

type totpPolicyResourceModel struct {
	ID       types.String `tfsdk:"id"`
	Policy   types.String `tfsdk:"policy"`
	Source   types.String `tfsdk:"source"`
	Editable types.Bool   `tfsdk:"editable"`
}

func (r *totpPolicyResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_totp_policy"
}

func (r *totpPolicyResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "System-wide 2FA enforcement policy, a singleton (one instance regardless of how many times declared; delete is a no-op and doesn't reset the backend).\n\n" +
			"The policy is evaluated only on interactive local password login. API tokens, service accounts, package-client basic auth, existing sessions and SSO/LDAP users are all exempt, so turning it on does not break CI or a running `docker pull`. A user who has not enrolled is not locked out: their login returns an enrollment ticket instead of a session.\n\n" +
			"Two things make an apply fail with `409`, both deliberate: the `TOTP_POLICY` environment variable pins the policy (`editable` is then `false`), and **tightening** the policy requires the admin whose credentials the provider is using to have TOTP enrolled already. Relaxing is never refused. If the provider authenticates with `username`/`password` rather than a token, enabling this will also lock the provider itself out at the next login, so use a token.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Fixed identifier for the TOTP policy singleton (always `totp_policy`).",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"policy": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "`disabled` (TOTP stays opt-in, the backend default), `required_for_admins`, or `required_for_all` (every local user).",
				Validators:          []validator.String{stringvalidator.OneOf("disabled", "required_for_admins", "required_for_all")},
			},
			"source": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Where the policy in force comes from: `database` (the stored setting, which this resource owns) or `environment` (`TOTP_POLICY` pins it and writes are refused).",
			},
			"editable": schema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Whether the policy can be changed over the API. `false` when `source` is `environment`.",
			},
		},
	}
}

func (r *totpPolicyResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = configureClient(req.ProviderData, &resp.Diagnostics)
}

func (r *totpPolicyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan totpPolicyResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Singleton, no create endpoint: PUT then store what the PUT echoes back.
	cfg, err := r.client.UpdateTotpPolicy(ctx, client.UpdateTotpPolicyRequest{Policy: plan.Policy.ValueString()})
	if err != nil {
		resp.Diagnostics.AddError("Error configuring TOTP policy", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, totpPolicyToModel(cfg))...)
}

func (r *totpPolicyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state totpPolicyResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Singleton always exists (backend defaults to disabled): never RemoveResource.
	cfg, err := r.client.GetTotpPolicy(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Error reading TOTP policy", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, totpPolicyToModel(cfg))...)
}

func (r *totpPolicyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan totpPolicyResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	cfg, err := r.client.UpdateTotpPolicy(ctx, client.UpdateTotpPolicyRequest{Policy: plan.Policy.ValueString()})
	if err != nil {
		resp.Diagnostics.AddError("Error updating TOTP policy", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, totpPolicyToModel(cfg))...)
}

// Delete is a no-op: relaxing the policy on destroy would be a silent security
// downgrade, so destroy just stops managing it.
func (r *totpPolicyResource) Delete(_ context.Context, _ resource.DeleteRequest, _ *resource.DeleteResponse) {
}

func (r *totpPolicyResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func totpPolicyToModel(c *client.TotpPolicy) totpPolicyResourceModel {
	return totpPolicyResourceModel{
		ID:       types.StringValue(totpPolicyID),
		Policy:   types.StringValue(c.Policy),
		Source:   types.StringValue(c.Source),
		Editable: types.BoolValue(c.Editable),
	}
}
