package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
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
	_ resource.Resource                = (*projectMembershipResource)(nil)
	_ resource.ResourceWithConfigure   = (*projectMembershipResource)(nil)
	_ resource.ResourceWithImportState = (*projectMembershipResource)(nil)
)

func NewProjectMembershipResource() resource.Resource { return &projectMembershipResource{} }

type projectMembershipResource struct {
	client *client.Client
}

// One membership grant on a project (a permissions row with target_type
// 'project'), inherited by every repository in the project. Synthetic id is
// "<project_id>/<principal_type>/<principal_id>".
type projectMembershipResourceModel struct {
	ID            types.String `tfsdk:"id"`
	ProjectID     types.String `tfsdk:"project_id"`
	PrincipalType types.String `tfsdk:"principal_type"`
	PrincipalID   types.String `tfsdk:"principal_id"`
	Actions       types.List   `tfsdk:"actions"`
}

func (r *projectMembershipResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_project_membership"
}

func (r *projectMembershipResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "A membership grant on a project: the `actions` granted to a user or group on every repository assigned to the project. Declare one per principal. Changing the project or principal forces a new grant; `actions` update in place. Admin-only, and `principal_id` must already exist as the given `principal_type`.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Synthetic ID in the form `\"<project_id>/<principal_type>/<principal_id>\"`.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"project_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "UUID of the project. Changing this forces a new grant.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"principal_type": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Principal type the grant is for: `user` or `group`. Changing this forces a new grant.",
				Validators:          []validator.String{stringvalidator.OneOf("user", "group")},
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"principal_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "UUID of the user or group. Must already exist as the given `principal_type`. Changing this forces a new grant.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"actions": schema.ListAttribute{
				ElementType:         types.StringType,
				Required:            true,
				MarkdownDescription: "Actions granted on every repository in the project (e.g. `[\"read\"]`, `[\"read\", \"write\"]`). At least one is required.",
				Validators:          []validator.List{listvalidator.SizeAtLeast(1)},
			},
		},
	}
}

func (r *projectMembershipResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = configureClient(req.ProviderData, &resp.Diagnostics)
}

func (r *projectMembershipResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan projectMembershipResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	actions, d := listToStringSlice(ctx, plan.Actions)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}

	member, err := r.client.SetProjectMember(ctx, plan.ProjectID.ValueString(), client.AddProjectMemberRequest{
		PrincipalType: plan.PrincipalType.ValueString(),
		PrincipalID:   plan.PrincipalID.ValueString(),
		Actions:       actions,
	})
	if err != nil {
		resp.Diagnostics.AddError("Error creating project membership", err.Error())
		return
	}

	model, d := projectMembershipToModel(ctx, plan.ProjectID.ValueString(), member)
	resp.Diagnostics.Append(d...)
	resp.Diagnostics.Append(resp.State.Set(ctx, model)...)
}

func (r *projectMembershipResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state projectMembershipResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	member, err := r.client.GetProjectMember(ctx, state.ProjectID.ValueString(), state.PrincipalType.ValueString(), state.PrincipalID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading project membership", err.Error())
		return
	}

	model, d := projectMembershipToModel(ctx, state.ProjectID.ValueString(), member)
	resp.Diagnostics.Append(d...)
	resp.Diagnostics.Append(resp.State.Set(ctx, model)...)
}

func (r *projectMembershipResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan projectMembershipResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Only `actions` reaches Update; project/principal force replacement. The
	// POST upserts, so re-sending replaces the stored action set.
	actions, d := listToStringSlice(ctx, plan.Actions)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}

	member, err := r.client.SetProjectMember(ctx, plan.ProjectID.ValueString(), client.AddProjectMemberRequest{
		PrincipalType: plan.PrincipalType.ValueString(),
		PrincipalID:   plan.PrincipalID.ValueString(),
		Actions:       actions,
	})
	if err != nil {
		resp.Diagnostics.AddError("Error updating project membership", err.Error())
		return
	}

	model, d := projectMembershipToModel(ctx, plan.ProjectID.ValueString(), member)
	resp.Diagnostics.Append(d...)
	resp.Diagnostics.Append(resp.State.Set(ctx, model)...)
}

func (r *projectMembershipResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state projectMembershipResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.RemoveProjectMember(ctx, state.ProjectID.ValueString(), state.PrincipalType.ValueString(), state.PrincipalID.ValueString()); err != nil && !client.IsNotFound(err) {
		resp.Diagnostics.AddError("Error deleting project membership", err.Error())
	}
}

// Composite import ID "<project_id>/<principal_type>/<principal_id>".
func (r *projectMembershipResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.Split(req.ID, "/")
	if len(parts) != 3 || parts[0] == "" || parts[1] == "" || parts[2] == "" {
		resp.Diagnostics.AddError(
			"Unexpected import identifier",
			fmt.Sprintf("Expected import ID in the format \"<project_id>/<principal_type>/<principal_id>\", got: %q", req.ID),
		)
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("project_id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("principal_type"), parts[1])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("principal_id"), parts[2])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
}

func projectMembershipToModel(ctx context.Context, projectID string, m *client.ProjectMember) (projectMembershipResourceModel, diag.Diagnostics) {
	actions, diags := stringListValue(ctx, m.Actions)
	return projectMembershipResourceModel{
		ID:            types.StringValue(projectID + "/" + m.PrincipalType + "/" + m.PrincipalID),
		ProjectID:     types.StringValue(projectID),
		PrincipalType: types.StringValue(m.PrincipalType),
		PrincipalID:   types.StringValue(m.PrincipalID),
		Actions:       actions,
	}, diags
}
