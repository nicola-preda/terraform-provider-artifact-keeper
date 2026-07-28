package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/nicola-preda/terraform-provider-artifact-keeper/internal/client"
)

var (
	_ resource.Resource                = (*ciOidcMappingResource)(nil)
	_ resource.ResourceWithConfigure   = (*ciOidcMappingResource)(nil)
	_ resource.ResourceWithImportState = (*ciOidcMappingResource)(nil)
)

func NewCiOidcIdentityMappingResource() resource.Resource { return &ciOidcMappingResource{} }

type ciOidcMappingResource struct {
	client *client.Client
}

type ciOidcMappingResourceModel struct {
	ID             types.String `tfsdk:"id"`
	ProviderID     types.String `tfsdk:"provider_id"`
	Name           types.String `tfsdk:"name"`
	Priority       types.Int64  `tfsdk:"priority"`
	ClaimFilters   types.String `tfsdk:"claim_filters"`
	AllowedRepoIDs types.List   `tfsdk:"allowed_repo_ids"`
	IsEnabled      types.Bool   `tfsdk:"is_enabled"`
	CreatedAt      types.String `tfsdk:"created_at"`
	UpdatedAt      types.String `tfsdk:"updated_at"`
}

func (r *ciOidcMappingResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_ci_oidc_identity_mapping"
}

func (r *ciOidcMappingResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "A CI OIDC identity mapping under an `artifactkeeper_ci_oidc_provider`. On token exchange the provider evaluates its mappings in priority order (lower number first); the first enabled mapping whose `claim_filters` all match the incoming CI JWT wins. Admin-only.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Identity mapping UUID assigned by Artifact Keeper.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"provider_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "UUID of the parent CI OIDC provider. Changing this forces a new mapping.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Human-readable mapping name. Also derives the stable CI service-account identity.",
			},
			"priority": schema.Int64Attribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Evaluation priority; lower numbers are evaluated first. Defaults to `100`.",
				PlanModifiers:       []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
			},
			"claim_filters": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Claim filter policy as a JSON object, e.g. `jsonencode({ ref = \"refs/heads/main\" })`. Each key is a JWT claim name; the value is either a single string (exact match) or an array of strings (any-of match). Use `jsonencode(...)` so the encoding matches the API's normalized form.",
			},
			"allowed_repo_ids": schema.ListAttribute{
				ElementType:         types.StringType,
				Optional:            true,
				MarkdownDescription: "Optional list of repository UUIDs this mapping is restricted to. Omit for an unrestricted mapping.",
			},
			"is_enabled": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Whether the mapping is enabled. Defaults to `true`.",
				PlanModifiers:       []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"created_at": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "RFC 3339 creation timestamp.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"updated_at": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "RFC 3339 last-update timestamp.",
			},
		},
	}
}

func (r *ciOidcMappingResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = configureClient(req.ProviderData, &resp.Diagnostics)
}

func (r *ciOidcMappingResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ciOidcMappingResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	repoIDs, d := listToStringSlice(ctx, plan.AllowedRepoIDs)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}

	createReq := client.CreateCiOidcMappingRequest{
		Name:           plan.Name.ValueString(),
		ClaimFilters:   json.RawMessage(plan.ClaimFilters.ValueString()),
		AllowedRepoIDs: repoIDs,
	}
	if !plan.Priority.IsNull() && !plan.Priority.IsUnknown() {
		createReq.Priority = plan.Priority.ValueInt64Pointer()
	}
	if !plan.IsEnabled.IsNull() && !plan.IsEnabled.IsUnknown() {
		createReq.IsEnabled = plan.IsEnabled.ValueBoolPointer()
	}

	mapping, err := r.client.CreateCiOidcMapping(ctx, plan.ProviderID.ValueString(), createReq)
	if err != nil {
		resp.Diagnostics.AddError("Error creating CI OIDC identity mapping", err.Error())
		return
	}

	state, d := ciOidcMappingToModel(ctx, mapping)
	resp.Diagnostics.Append(d...)
	state.ClaimFilters = plan.ClaimFilters // keep configured JSON encoding verbatim
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *ciOidcMappingResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ciOidcMappingResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	mapping, err := r.client.GetCiOidcMapping(ctx, state.ProviderID.ValueString(), state.ID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading CI OIDC identity mapping", err.Error())
		return
	}

	refreshed, d := ciOidcMappingToModel(ctx, mapping)
	resp.Diagnostics.Append(d...)
	// Keep configured claim_filters encoding when semantically equal to the API's,
	// so whitespace/key-order don't churn as perpetual diffs. Overwrite only on
	// real drift.
	canon := func(s string) string {
		var v interface{}
		if json.Unmarshal([]byte(s), &v) != nil {
			return s
		}
		b, err := json.Marshal(v)
		if err != nil {
			return s
		}
		return string(b)
	}
	if !state.ClaimFilters.IsNull() && canon(state.ClaimFilters.ValueString()) == canon(refreshed.ClaimFilters.ValueString()) {
		refreshed.ClaimFilters = state.ClaimFilters
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, refreshed)...)
}

func (r *ciOidcMappingResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan ciOidcMappingResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	repoIDs, d := listToStringSlice(ctx, plan.AllowedRepoIDs)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}

	updateReq := client.UpdateCiOidcMappingRequest{
		Name:           plan.Name.ValueStringPointer(),
		Priority:       plan.Priority.ValueInt64Pointer(),
		ClaimFilters:   json.RawMessage(plan.ClaimFilters.ValueString()),
		AllowedRepoIDs: repoIDs,
		IsEnabled:      plan.IsEnabled.ValueBoolPointer(),
	}

	mapping, err := r.client.UpdateCiOidcMapping(ctx, plan.ProviderID.ValueString(), plan.ID.ValueString(), updateReq)
	if err != nil {
		resp.Diagnostics.AddError("Error updating CI OIDC identity mapping", err.Error())
		return
	}

	state, d := ciOidcMappingToModel(ctx, mapping)
	resp.Diagnostics.Append(d...)
	state.ClaimFilters = plan.ClaimFilters // keep configured JSON encoding verbatim
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *ciOidcMappingResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state ciOidcMappingResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.DeleteCiOidcMapping(ctx, state.ProviderID.ValueString(), state.ID.ValueString()); err != nil && !client.IsNotFound(err) {
		resp.Diagnostics.AddError("Error deleting CI OIDC identity mapping", err.Error())
	}
}

// Composite import ID "<provider_id>/<mapping_id>"; both parts address the
// mapping under its parent provider.
func (r *ciOidcMappingResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	providerID, id, ok := strings.Cut(req.ID, "/")
	if !ok || providerID == "" || id == "" {
		resp.Diagnostics.AddError(
			"Unexpected import identifier",
			fmt.Sprintf("Expected import ID in the format \"<provider_id>/<mapping_id>\", got: %q", req.ID),
		)
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("provider_id"), providerID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), id)...)
}

func ciOidcMappingToModel(ctx context.Context, m *client.CiOidcMapping) (ciOidcMappingResourceModel, diag.Diagnostics) {
	repoIDs, d := stringListValue(ctx, m.AllowedRepoIDs)
	return ciOidcMappingResourceModel{
		ID:             types.StringValue(m.ID),
		ProviderID:     types.StringValue(m.ProviderID),
		Name:           types.StringValue(m.Name),
		Priority:       types.Int64Value(m.Priority),
		ClaimFilters:   types.StringValue(string(m.ClaimFilters)),
		AllowedRepoIDs: repoIDs,
		IsEnabled:      types.BoolValue(m.IsEnabled),
		CreatedAt:      types.StringValue(m.CreatedAt),
		UpdatedAt:      types.StringValue(m.UpdatedAt),
	}, d
}
