package provider

import (
	"context"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/nicola-preda/terraform-provider-artifact-keeper/internal/client"
)

var (
	_ resource.Resource                = (*peerInstanceLabelResource)(nil)
	_ resource.ResourceWithConfigure   = (*peerInstanceLabelResource)(nil)
	_ resource.ResourceWithImportState = (*peerInstanceLabelResource)(nil)
)

func NewPeerInstanceLabelResource() resource.Resource { return &peerInstanceLabelResource{} }

type peerInstanceLabelResource struct {
	client *client.Client
}

type peerInstanceLabelResourceModel struct {
	ID        types.String `tfsdk:"id"`
	PeerID    types.String `tfsdk:"peer_id"`
	Key       types.String `tfsdk:"key"`
	Value     types.String `tfsdk:"value"`
	CreatedAt types.String `tfsdk:"created_at"`
}

func (r *peerInstanceLabelResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_peer_instance_label"
}

func (r *peerInstanceLabelResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "A single key/value label on a peer instance. `sync_policy` match rules select peers by these labels, and the backend re-evaluates the policies on every label write. A label is addressed by the peer UUID plus the label key; changing either forces a new label, while the value can be updated in place. Admin-only.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Label UUID assigned by Artifact Keeper.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"peer_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "UUID of the peer instance the label is attached to. Changing this forces a new label.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"key": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Label key. Changing this forces a new label.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"value": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Label value. Defaults to an empty string when omitted.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"created_at": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "RFC 3339 creation timestamp.",
			},
		},
	}
}

func (r *peerInstanceLabelResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = configureClient(req.ProviderData, &resp.Diagnostics)
}

func (r *peerInstanceLabelResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan peerInstanceLabelResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	label, err := r.client.SetPeerInstanceLabel(ctx, plan.PeerID.ValueString(), plan.Key.ValueString(), plan.Value.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error creating peer instance label", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, peerInstanceLabelToModel(label))...)
}

func (r *peerInstanceLabelResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state peerInstanceLabelResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	label, err := r.client.GetPeerInstanceLabel(ctx, state.PeerID.ValueString(), state.Key.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading peer instance label", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, peerInstanceLabelToModel(label))...)
}

func (r *peerInstanceLabelResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan peerInstanceLabelResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// The add endpoint upserts, so it also serves in-place value changes.
	label, err := r.client.SetPeerInstanceLabel(ctx, plan.PeerID.ValueString(), plan.Key.ValueString(), plan.Value.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error updating peer instance label", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, peerInstanceLabelToModel(label))...)
}

func (r *peerInstanceLabelResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state peerInstanceLabelResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.DeletePeerInstanceLabel(ctx, state.PeerID.ValueString(), state.Key.ValueString()); err != nil && !client.IsNotFound(err) {
		resp.Diagnostics.AddError("Error deleting peer instance label", err.Error())
	}
}

// ImportState accepts a composite ID of the form "<peer_id>/<label_key>".
func (r *peerInstanceLabelResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.SplitN(req.ID, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		resp.Diagnostics.AddError(
			"Invalid import ID",
			"Expected import ID in the form \"<peer_id>/<label_key>\".",
		)
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("peer_id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("key"), parts[1])...)
}

func peerInstanceLabelToModel(l *client.PeerInstanceLabel) peerInstanceLabelResourceModel {
	return peerInstanceLabelResourceModel{
		ID:        types.StringValue(l.ID),
		PeerID:    types.StringValue(l.PeerInstanceID),
		Key:       types.StringValue(l.Key),
		Value:     types.StringValue(l.Value),
		CreatedAt: types.StringValue(l.CreatedAt),
	}
}
