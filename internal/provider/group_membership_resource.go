package provider

import (
	"context"
	"fmt"
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
	_ resource.Resource                = (*groupMembershipResource)(nil)
	_ resource.ResourceWithConfigure   = (*groupMembershipResource)(nil)
	_ resource.ResourceWithImportState = (*groupMembershipResource)(nil)
)

func NewGroupMembershipResource() resource.Resource { return &groupMembershipResource{} }

type groupMembershipResource struct {
	client *client.Client
}

// A single (group_id, user_id) edge; synthetic id is "<group_id>/<user_id>".
type groupMembershipResourceModel struct {
	ID      types.String `tfsdk:"id"`
	GroupID types.String `tfsdk:"group_id"`
	UserID  types.String `tfsdk:"user_id"`
}

func (r *groupMembershipResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_group_membership"
}

func (r *groupMembershipResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Membership of a single user in a group, one (group, user) edge. Declare one `artifactkeeper_group_membership` per member. There is no in-place update: changing either id detaches the old membership and creates a new one. Adding a member is idempotent server-side.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Synthetic ID in the form `\"<group_id>/<user_id>\"`.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"group_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "UUID of the group. Changing this forces a new membership.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"user_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "UUID of the user to add to the group. Changing this forces a new membership.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
		},
	}
}

func (r *groupMembershipResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = configureClient(req.ProviderData, &resp.Diagnostics)
}

func (r *groupMembershipResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan groupMembershipResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	groupID := plan.GroupID.ValueString()
	userID := plan.UserID.ValueString()
	if err := r.client.AddGroupMember(ctx, groupID, userID); err != nil {
		resp.Diagnostics.AddError("Error creating group membership", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, groupMembershipToModel(groupID, userID))...)
}

func (r *groupMembershipResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state groupMembershipResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	groupID := state.GroupID.ValueString()
	userID := state.UserID.ValueString()
	if _, err := r.client.GetGroupMember(ctx, groupID, userID); err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading group membership", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, groupMembershipToModel(groupID, userID))...)
}

// Never runs: both attributes force replacement. Satisfies the interface.
func (r *groupMembershipResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan groupMembershipResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *groupMembershipResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state groupMembershipResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.RemoveGroupMember(ctx, state.GroupID.ValueString(), state.UserID.ValueString()); err != nil && !client.IsNotFound(err) {
		resp.Diagnostics.AddError("Error deleting group membership", err.Error())
	}
}

// Composite import ID "<group_id>/<user_id>"; both parts address the membership
// and also form the synthetic id.
func (r *groupMembershipResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	groupID, userID, ok := strings.Cut(req.ID, "/")
	if !ok || groupID == "" || userID == "" {
		resp.Diagnostics.AddError(
			"Unexpected import identifier",
			fmt.Sprintf("Expected import ID in the format \"<group_id>/<user_id>\", got: %q", req.ID),
		)
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("group_id"), groupID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("user_id"), userID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
}

func groupMembershipToModel(groupID, userID string) groupMembershipResourceModel {
	return groupMembershipResourceModel{
		ID:      types.StringValue(groupID + "/" + userID),
		GroupID: types.StringValue(groupID),
		UserID:  types.StringValue(userID),
	}
}
