package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/nicola-preda/terraform-provider-artifact-keeper/internal/client"
)

var (
	_ resource.Resource                = (*peerNetworkProfileResource)(nil)
	_ resource.ResourceWithConfigure   = (*peerNetworkProfileResource)(nil)
	_ resource.ResourceWithImportState = (*peerNetworkProfileResource)(nil)
)

func NewPeerNetworkProfileResource() resource.Resource { return &peerNetworkProfileResource{} }

type peerNetworkProfileResource struct {
	client *client.Client
}

// Singleton per peer: peer_id is the identity, id mirrors it.
type peerNetworkProfileResourceModel struct {
	ID                       types.String `tfsdk:"id"`
	PeerID                   types.String `tfsdk:"peer_id"`
	MaxBandwidthBps          types.Int64  `tfsdk:"max_bandwidth_bps"`
	SyncWindowStart          types.String `tfsdk:"sync_window_start"`
	SyncWindowEnd            types.String `tfsdk:"sync_window_end"`
	SyncWindowTimezone       types.String `tfsdk:"sync_window_timezone"`
	ConcurrentTransfersLimit types.Int64  `tfsdk:"concurrent_transfers_limit"`
}

func (r *peerNetworkProfileResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_peer_network_profile"
}

func (r *peerNetworkProfileResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Transfer scheduling and bandwidth settings for how this instance syncs with a peer, a singleton per `artifactkeeper_peer`, so `peer_id` is the identity and changing it replaces the resource. The API never returns these fields, so values are kept from config (drift isn't detected, an omitted field keeps its stored value, and delete only drops them from state).",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Resource identifier. Mirrors `peer_id` (the profile is a per-peer singleton).",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"peer_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "UUID of the peer instance this profile belongs to. Changing this forces a new resource.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"max_bandwidth_bps": schema.Int64Attribute{
				Optional:            true,
				MarkdownDescription: "Maximum outbound transfer bandwidth to the peer, in bytes per second. Omit to leave the stored value unchanged.",
			},
			"sync_window_start": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Start of the daily sync window as an `HH:MM:SS` 24-hour time (e.g. `02:00:00`). Interpreted in `sync_window_timezone`. Omit to leave the stored value unchanged.",
			},
			"sync_window_end": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "End of the daily sync window as an `HH:MM:SS` 24-hour time (e.g. `06:00:00`). A window whose end is before its start is treated as spanning midnight. Omit to leave the stored value unchanged.",
			},
			"sync_window_timezone": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "IANA timezone the sync window is interpreted in (e.g. `America/New_York`, `UTC`). Omit to leave the stored value unchanged.",
			},
			"concurrent_transfers_limit": schema.Int64Attribute{
				Optional:            true,
				MarkdownDescription: "Maximum number of concurrent transfers to the peer. Omit to leave the stored value unchanged.",
			},
		},
	}
}

func (r *peerNetworkProfileResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = configureClient(req.ProviderData, &resp.Diagnostics)
}

func (r *peerNetworkProfileResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan peerNetworkProfileResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.UpdatePeerNetworkProfile(ctx, plan.PeerID.ValueString(), buildNetworkProfile(plan)); err != nil {
		resp.Diagnostics.AddError("Error setting peer network profile", err.Error())
		return
	}

	// No response body, no read-back: state is the plan.
	plan.ID = plan.PeerID
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

// Read only verifies the parent peer still exists; no endpoint returns these
// fields, so nothing to refresh and no drift detected.
func (r *peerNetworkProfileResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state peerNetworkProfileResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	peerID := state.PeerID.ValueString()
	if _, err := r.client.GetPeer(ctx, peerID); err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading peer network profile", err.Error())
		return
	}

	// Mirror id to peer_id (after import only peer_id is set); other fields kept.
	state.ID = types.StringValue(peerID)
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

// Re-PUTs the profile with planned values (a real update).
func (r *peerNetworkProfileResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan peerNetworkProfileResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.UpdatePeerNetworkProfile(ctx, plan.PeerID.ValueString(), buildNetworkProfile(plan)); err != nil {
		resp.Diagnostics.AddError("Error updating peer network profile", err.Error())
		return
	}

	plan.ID = plan.PeerID
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

// No-op: no delete endpoint; the profile lives with the peer. Destroy just
// drops it from state.
func (r *peerNetworkProfileResource) Delete(_ context.Context, _ resource.DeleteRequest, _ *resource.DeleteResponse) {
}

// Import ID is the peer UUID. Profile fields can't be recovered (no read
// endpoint), so they import null until the next apply.
func (r *peerNetworkProfileResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("peer_id"), req, resp)
}

func buildNetworkProfile(plan peerNetworkProfileResourceModel) client.NetworkProfile {
	np := client.NetworkProfile{}
	if !plan.MaxBandwidthBps.IsNull() && !plan.MaxBandwidthBps.IsUnknown() {
		np.MaxBandwidthBps = plan.MaxBandwidthBps.ValueInt64Pointer()
	}
	if !plan.SyncWindowStart.IsNull() && !plan.SyncWindowStart.IsUnknown() {
		np.SyncWindowStart = plan.SyncWindowStart.ValueStringPointer()
	}
	if !plan.SyncWindowEnd.IsNull() && !plan.SyncWindowEnd.IsUnknown() {
		np.SyncWindowEnd = plan.SyncWindowEnd.ValueStringPointer()
	}
	if !plan.SyncWindowTimezone.IsNull() && !plan.SyncWindowTimezone.IsUnknown() {
		np.SyncWindowTimezone = plan.SyncWindowTimezone.ValueStringPointer()
	}
	if !plan.ConcurrentTransfersLimit.IsNull() && !plan.ConcurrentTransfersLimit.IsUnknown() {
		v := int32(plan.ConcurrentTransfersLimit.ValueInt64())
		np.ConcurrentTransfersLimit = &v
	}
	return np
}
