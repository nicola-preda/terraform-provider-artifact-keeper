package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/nicola-preda/terraform-provider-artifact-keeper/internal/client"
)

var (
	_ resource.Resource                = (*peerRepositorySubscriptionResource)(nil)
	_ resource.ResourceWithConfigure   = (*peerRepositorySubscriptionResource)(nil)
	_ resource.ResourceWithImportState = (*peerRepositorySubscriptionResource)(nil)
)

func NewPeerRepositorySubscriptionResource() resource.Resource {
	return &peerRepositorySubscriptionResource{}
}

type peerRepositorySubscriptionResource struct {
	client *client.Client
}

// Addressed by the (peer_id, repository_id) pair.
type peerRepositorySubscriptionResourceModel struct {
	ID                  types.String `tfsdk:"id"`
	PeerID              types.String `tfsdk:"peer_id"`
	RepositoryID        types.String `tfsdk:"repository_id"`
	SyncEnabled         types.Bool   `tfsdk:"sync_enabled"`
	ReplicationMode     types.String `tfsdk:"replication_mode"`
	ReplicationSchedule types.String `tfsdk:"replication_schedule"`
	ReplicationFilter   types.String `tfsdk:"replication_filter"`
}

func (r *peerRepositorySubscriptionResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_peer_repository_subscription"
}

func (r *peerRepositorySubscriptionResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Subscribes one repository to an `artifactkeeper_peer` for replication. The (peer, repository) pair is the identity, changing either forces a new subscription; `sync_enabled`, `replication_mode`, `replication_schedule`, and `replication_filter` update in place. Admin-only.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Subscription UUID assigned by Artifact Keeper.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"peer_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "UUID of the parent peer instance. Changing this forces a new subscription.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"repository_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "UUID of the repository to replicate to the peer. Changing this forces a new subscription.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"sync_enabled": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Whether replication is enabled for this subscription. Defaults to `true`.",
				PlanModifiers:       []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"replication_mode": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Replication mode: one of `push`, `pull`, `mirror`, `none`. Defaults to `pull`.",
				Validators:          []validator.String{stringvalidator.OneOf("push", "pull", "mirror", "none")},
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"replication_schedule": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Optional cron schedule controlling when scheduled syncs run, e.g. `0 */6 * * *`. Omit to leave unscheduled.",
			},
			"replication_filter": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Optional JSON object constraining which artifacts replicate, e.g. `jsonencode({ include_patterns = [\"^v\\\\d+\\\\.\"], exclude_patterns = [\".*-SNAPSHOT$\"] })`. Omit to replicate everything. Use `jsonencode(...)` so the encoding matches the API's normalized form.",
			},
		},
	}
}

func (r *peerRepositorySubscriptionResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = configureClient(req.ProviderData, &resp.Diagnostics)
}

func (r *peerRepositorySubscriptionResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan peerRepositorySubscriptionResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	r.assignAndRead(ctx, plan, &resp.Diagnostics, &resp.State)
}

func (r *peerRepositorySubscriptionResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state peerRepositorySubscriptionResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	peerID := state.PeerID.ValueString()
	sub, err := r.client.GetPeerRepositorySubscription(ctx, peerID, state.RepositoryID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading peer repository subscription", err.Error())
		return
	}

	refreshed := peerRepoSubToModel(peerID, sub)
	// Keep configured replication_filter encoding when semantically equal to the
	// API's, so JSONB round-trip whitespace/key-order don't churn. Overwrite only
	// on real drift or import.
	if !state.ReplicationFilter.IsNull() &&
		canonicalizeJSON(state.ReplicationFilter.ValueString()) == canonicalizeJSON(refreshed.ReplicationFilter.ValueString()) {
		refreshed.ReplicationFilter = state.ReplicationFilter
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, refreshed)...)
}

// Re-POSTs: the assign endpoint upserts on the (peer, repository) key, pushing
// mutable fields in place. peer_id/repository_id force replacement.
func (r *peerRepositorySubscriptionResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan peerRepositorySubscriptionResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	r.assignAndRead(ctx, plan, &resp.Diagnostics, &resp.State)
}

func (r *peerRepositorySubscriptionResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state peerRepositorySubscriptionResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.UnassignRepo(ctx, state.PeerID.ValueString(), state.RepositoryID.ValueString()); err != nil && !client.IsNotFound(err) {
		resp.Diagnostics.AddError("Error deleting peer repository subscription", err.Error())
	}
}

// Composite import ID "<peer_id>/<repository_id>"; both parts address the
// subscription. UUID resolved by the subsequent Read.
func (r *peerRepositorySubscriptionResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	peerID, repoID, ok := strings.Cut(req.ID, "/")
	if !ok || peerID == "" || repoID == "" {
		resp.Diagnostics.AddError(
			"Unexpected import identifier",
			fmt.Sprintf("Expected import ID in the format \"<peer_id>/<repository_id>\", got: %q", req.ID),
		)
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("peer_id"), peerID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("repository_id"), repoID)...)
}

// assignAndRead upserts via POST then reads the subscription back (assign
// returns no body), preserving the configured replication_filter encoding.
func (r *peerRepositorySubscriptionResource) assignAndRead(ctx context.Context, plan peerRepositorySubscriptionResourceModel, diags *diag.Diagnostics, state *tfsdk.State) {
	peerID := plan.PeerID.ValueString()
	repoID := plan.RepositoryID.ValueString()

	if err := r.client.AssignRepo(ctx, peerID, buildAssignRequest(plan)); err != nil {
		diags.AddError("Error assigning repository to peer", err.Error())
		return
	}

	sub, err := r.client.GetPeerRepositorySubscription(ctx, peerID, repoID)
	if err != nil {
		diags.AddError("Error reading back peer repository subscription", err.Error())
		return
	}

	newState := peerRepoSubToModel(peerID, sub)
	newState.ReplicationFilter = plan.ReplicationFilter // keep configured JSON encoding verbatim
	diags.Append(state.Set(ctx, newState)...)
}

func buildAssignRequest(plan peerRepositorySubscriptionResourceModel) client.AssignRepoRequest {
	req := client.AssignRepoRequest{RepositoryID: plan.RepositoryID.ValueString()}
	if !plan.SyncEnabled.IsNull() && !plan.SyncEnabled.IsUnknown() {
		req.SyncEnabled = plan.SyncEnabled.ValueBoolPointer()
	}
	if !plan.ReplicationMode.IsNull() && !plan.ReplicationMode.IsUnknown() {
		req.ReplicationMode = plan.ReplicationMode.ValueStringPointer()
	}
	if !plan.ReplicationSchedule.IsNull() && !plan.ReplicationSchedule.IsUnknown() {
		req.ReplicationSchedule = plan.ReplicationSchedule.ValueStringPointer()
	}
	if !plan.ReplicationFilter.IsNull() && !plan.ReplicationFilter.IsUnknown() {
		req.ReplicationFilter = json.RawMessage(plan.ReplicationFilter.ValueString())
	}
	return req
}

func peerRepoSubToModel(peerID string, sub *client.PeerRepositorySubscription) peerRepositorySubscriptionResourceModel {
	return peerRepositorySubscriptionResourceModel{
		ID:                  types.StringValue(sub.ID),
		PeerID:              types.StringValue(peerID),
		RepositoryID:        types.StringValue(sub.RepositoryID),
		SyncEnabled:         types.BoolValue(sub.SyncEnabled),
		ReplicationMode:     stringPointerValue(sub.ReplicationMode),
		ReplicationSchedule: stringPointerValue(sub.ReplicationSchedule),
		ReplicationFilter:   rawJSONToStringValue(sub.ReplicationFilter),
	}
}

// rawJSONToStringValue renders raw JSONB as a Terraform string; absent or
// JSON-null maps to a null string.
func rawJSONToStringValue(raw json.RawMessage) types.String {
	s := strings.TrimSpace(string(raw))
	if s == "" || s == "null" {
		return types.StringNull()
	}
	return types.StringValue(s)
}

// canonicalizeJSON returns a stable serialization so semantically-equal
// encodings compare equal. Non-JSON is returned unchanged.
func canonicalizeJSON(s string) string {
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
