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
	_ resource.Resource                = (*userRoleAssignmentResource)(nil)
	_ resource.ResourceWithConfigure   = (*userRoleAssignmentResource)(nil)
	_ resource.ResourceWithImportState = (*userRoleAssignmentResource)(nil)
)

func NewUserRoleAssignmentResource() resource.Resource { return &userRoleAssignmentResource{} }

type userRoleAssignmentResource struct {
	client *client.Client
}

// A single user↔role edge, addressed by (user_id, role_id). No mutable
// attributes, so any change forces a new edge.
type userRoleAssignmentResourceModel struct {
	ID     types.String `tfsdk:"id"`
	UserID types.String `tfsdk:"user_id"`
	RoleID types.String `tfsdk:"role_id"`
}

func (r *userRoleAssignmentResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_user_role_assignment"
}

func (r *userRoleAssignmentResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Assigns a role to a user in Artifact Keeper. This models a single user↔role edge; the assignment is not repository-scoped and has no mutable attributes, so any change forces a new assignment.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Synthetic identifier in the form `<user_id>/<role_id>`.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"user_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "UUID of the user the role is granted to. The user must already exist. Changing this forces a new assignment.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"role_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "UUID of the role to grant. Changing this forces a new assignment.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
		},
	}
}

func (r *userRoleAssignmentResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = configureClient(req.ProviderData, &resp.Diagnostics)
}

func (r *userRoleAssignmentResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan userRoleAssignmentResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	userID := plan.UserID.ValueString()
	roleID := plan.RoleID.ValueString()
	if err := r.client.AssignUserRole(ctx, userID, roleID); err != nil {
		resp.Diagnostics.AddError("Error assigning role to user", err.Error())
		return
	}

	plan.ID = types.StringValue(userID + "/" + roleID)
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *userRoleAssignmentResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state userRoleAssignmentResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// No mutable data; just confirm the edge still exists. GetUserRole 404s when gone.
	userID := state.UserID.ValueString()
	roleID := state.RoleID.ValueString()
	if _, err := r.client.GetUserRole(ctx, userID, roleID); err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading user role assignment", err.Error())
		return
	}

	state.ID = types.StringValue(userID + "/" + roleID)
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

// Never runs: both attributes force replacement. Satisfies the interface.
func (r *userRoleAssignmentResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan userRoleAssignmentResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *userRoleAssignmentResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state userRoleAssignmentResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.RemoveUserRole(ctx, state.UserID.ValueString(), state.RoleID.ValueString()); err != nil && !client.IsNotFound(err) {
		resp.Diagnostics.AddError("Error removing user role assignment", err.Error())
	}
}

// Composite import ID "<user_id>/<role_id>"; both parts address the assignment.
func (r *userRoleAssignmentResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	userID, roleID, ok := strings.Cut(req.ID, "/")
	if !ok || userID == "" || roleID == "" {
		resp.Diagnostics.AddError(
			"Unexpected import identifier",
			fmt.Sprintf("Expected import ID in the format \"<user_id>/<role_id>\", got: %q", req.ID),
		)
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("user_id"), userID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("role_id"), roleID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
}
