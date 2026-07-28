package provider

import (
	"context"

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
	_ resource.Resource                = (*peerResource)(nil)
	_ resource.ResourceWithConfigure   = (*peerResource)(nil)
	_ resource.ResourceWithImportState = (*peerResource)(nil)
)

func NewPeerResource() resource.Resource { return &peerResource{} }

type peerResource struct {
	client *client.Client
}

type peerResourceModel struct {
	ID              types.String `tfsdk:"id"`
	Name            types.String `tfsdk:"name"`
	EndpointURL     types.String `tfsdk:"endpoint_url"`
	Region          types.String `tfsdk:"region"`
	CacheSizeBytes  types.Int64  `tfsdk:"cache_size_bytes"`
	APIKey          types.String `tfsdk:"api_key"`
	Status          types.String `tfsdk:"status"`
	CacheUsedBytes  types.Int64  `tfsdk:"cache_used_bytes"`
	LastHeartbeatAt types.String `tfsdk:"last_heartbeat_at"`
	LastSyncAt      types.String `tfsdk:"last_sync_at"`
	CreatedAt       types.String `tfsdk:"created_at"`
	IsLocal         types.Bool   `tfsdk:"is_local"`
}

func (r *peerResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_peer"
}

func (r *peerResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Registers a remote peer instance for replication. The API has no update endpoint, so any change forces a new peer. Note: a peer's `status` only becomes `online` once the backend receives heartbeats from it.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Unique peer name.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"endpoint_url": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Base URL of the remote peer (e.g. `https://peer.example.com`).",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"region": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Optional region identifier.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"cache_size_bytes": schema.Int64Attribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Cache capacity in bytes. Defaults to 10 GiB.",
				PlanModifiers:       []planmodifier.Int64{int64planmodifier.RequiresReplace(), int64planmodifier.UseStateForUnknown()},
			},
			"api_key": schema.StringAttribute{
				Required:            true,
				Sensitive:           true,
				MarkdownDescription: "API token (Bearer) the local instance uses to authenticate to this peer. Should be an admin-scoped token issued on the remote peer.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"status":            schema.StringAttribute{Computed: true, MarkdownDescription: "online, offline, syncing, or degraded."},
			"cache_used_bytes":  schema.Int64Attribute{Computed: true},
			"last_heartbeat_at": schema.StringAttribute{Computed: true},
			"last_sync_at":      schema.StringAttribute{Computed: true},
			"created_at": schema.StringAttribute{
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"is_local": schema.BoolAttribute{
				Computed:      true,
				PlanModifiers: []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
		},
	}
}

func (r *peerResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = configureClient(req.ProviderData, &resp.Diagnostics)
}

func (r *peerResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan peerResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	createReq := client.RegisterPeerRequest{
		Name:        plan.Name.ValueString(),
		EndpointURL: plan.EndpointURL.ValueString(),
		APIKey:      plan.APIKey.ValueString(),
	}
	if !plan.Region.IsNull() {
		createReq.Region = plan.Region.ValueStringPointer()
	}
	if !plan.CacheSizeBytes.IsNull() && !plan.CacheSizeBytes.IsUnknown() {
		createReq.CacheSizeBytes = plan.CacheSizeBytes.ValueInt64Pointer()
	}

	peer, err := r.client.CreatePeer(ctx, createReq)
	if err != nil {
		resp.Diagnostics.AddError("Error registering peer", err.Error())
		return
	}

	state := peerToModel(peer)
	state.APIKey = plan.APIKey // API never returns it; preserved from config
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *peerResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state peerResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	peer, err := r.client.GetPeer(ctx, state.ID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading peer", err.Error())
		return
	}

	refreshed := peerToModel(peer)
	refreshed.APIKey = state.APIKey // preserved; never returned by the API
	resp.Diagnostics.Append(resp.State.Set(ctx, refreshed)...)
}

// Update is unreachable (no update endpoint; all attributes force replacement).
func (r *peerResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan peerResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *peerResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state peerResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.DeletePeer(ctx, state.ID.ValueString()); err != nil && !client.IsNotFound(err) {
		resp.Diagnostics.AddError("Error deleting peer", err.Error())
	}
}

func (r *peerResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func peerToModel(p *client.Peer) peerResourceModel {
	return peerResourceModel{
		ID:              types.StringValue(p.ID),
		Name:            types.StringValue(p.Name),
		EndpointURL:     types.StringValue(p.EndpointURL),
		Region:          stringPointerValue(p.Region),
		CacheSizeBytes:  types.Int64Value(p.CacheSizeBytes),
		Status:          types.StringValue(p.Status),
		CacheUsedBytes:  types.Int64Value(p.CacheUsedBytes),
		LastHeartbeatAt: stringPointerValue(p.LastHeartbeatAt),
		LastSyncAt:      stringPointerValue(p.LastSyncAt),
		CreatedAt:       types.StringValue(p.CreatedAt),
		IsLocal:         types.BoolValue(p.IsLocal),
	}
}
