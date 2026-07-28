package provider

import (
	"context"

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
	_ resource.Resource                = (*permissionResource)(nil)
	_ resource.ResourceWithConfigure   = (*permissionResource)(nil)
	_ resource.ResourceWithImportState = (*permissionResource)(nil)
)

func NewPermissionResource() resource.Resource { return &permissionResource{} }

type permissionResource struct {
	client *client.Client
}

type permissionResourceModel struct {
	ID            types.String `tfsdk:"id"`
	PrincipalType types.String `tfsdk:"principal_type"`
	PrincipalID   types.String `tfsdk:"principal_id"`
	PrincipalName types.String `tfsdk:"principal_name"`
	TargetType    types.String `tfsdk:"target_type"`
	TargetID      types.String `tfsdk:"target_id"`
	TargetName    types.String `tfsdk:"target_name"`
	Actions       types.List   `tfsdk:"actions"`
	CreatedAt     types.String `tfsdk:"created_at"`
	UpdatedAt     types.String `tfsdk:"updated_at"`
}

func (r *permissionResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_permission"
}

func (r *permissionResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Grants actions to a principal (user/group) on a target (repository/group/system/...).",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{Computed: true, PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}},
			"principal_type": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Principal kind: `user`, `group`, or `service_account`. The principal must already exist; the server rejects a grant to an unknown id.",
				Validators:          []validator.String{stringvalidator.OneOf("user", "group", "service_account")},
			},
			"principal_id":   schema.StringAttribute{Required: true, MarkdownDescription: "UUID of the principal."},
			"principal_name": schema.StringAttribute{Computed: true},
			"target_type":    schema.StringAttribute{Required: true, MarkdownDescription: "`repository`, `group`, `artifact`, or `system`."},
			"target_id":      schema.StringAttribute{Required: true, MarkdownDescription: "UUID of the target resource."},
			"target_name":    schema.StringAttribute{Computed: true},
			"actions":        schema.ListAttribute{ElementType: types.StringType, Required: true, MarkdownDescription: "Actions: `read`, `write`, `delete`, `admin`."},
			"created_at":     schema.StringAttribute{Computed: true, PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}},
			"updated_at":     schema.StringAttribute{Computed: true},
		},
	}
}

func (r *permissionResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = configureClient(req.ProviderData, &resp.Diagnostics)
}

func (r *permissionResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan permissionResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	apiReq, d := permissionRequestFromModel(ctx, plan)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}
	p, err := r.client.CreatePermission(ctx, apiReq)
	if err != nil {
		resp.Diagnostics.AddError("Error creating permission", err.Error())
		return
	}
	state, d := permissionToModel(ctx, p)
	resp.Diagnostics.Append(d...)
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *permissionResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state permissionResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	p, err := r.client.GetPermission(ctx, state.ID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading permission", err.Error())
		return
	}
	refreshed, d := permissionToModel(ctx, p)
	resp.Diagnostics.Append(d...)
	resp.Diagnostics.Append(resp.State.Set(ctx, refreshed)...)
}

func (r *permissionResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan permissionResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	apiReq, d := permissionRequestFromModel(ctx, plan)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}
	p, err := r.client.UpdatePermission(ctx, plan.ID.ValueString(), apiReq)
	if err != nil {
		resp.Diagnostics.AddError("Error updating permission", err.Error())
		return
	}
	state, d := permissionToModel(ctx, p)
	resp.Diagnostics.Append(d...)
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *permissionResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state permissionResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.DeletePermission(ctx, state.ID.ValueString()); err != nil && !client.IsNotFound(err) {
		resp.Diagnostics.AddError("Error deleting permission", err.Error())
	}
}

func (r *permissionResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func permissionRequestFromModel(ctx context.Context, m permissionResourceModel) (client.PermissionRequest, diag.Diagnostics) {
	actions, d := listToStringSlice(ctx, m.Actions)
	return client.PermissionRequest{
		PrincipalType: m.PrincipalType.ValueString(),
		PrincipalID:   m.PrincipalID.ValueString(),
		TargetType:    m.TargetType.ValueString(),
		TargetID:      m.TargetID.ValueString(),
		Actions:       actions,
	}, d
}

func permissionToModel(ctx context.Context, p *client.Permission) (permissionResourceModel, diag.Diagnostics) {
	actions, d := stringListValue(ctx, p.Actions)
	return permissionResourceModel{
		ID:            types.StringValue(p.ID),
		PrincipalType: types.StringValue(p.PrincipalType),
		PrincipalID:   types.StringValue(p.PrincipalID),
		PrincipalName: stringPointerValue(p.PrincipalName),
		TargetType:    types.StringValue(p.TargetType),
		TargetID:      types.StringValue(p.TargetID),
		TargetName:    stringPointerValue(p.TargetName),
		Actions:       actions,
		CreatedAt:     types.StringValue(p.CreatedAt),
		UpdatedAt:     types.StringValue(p.UpdatedAt),
	}, d
}
