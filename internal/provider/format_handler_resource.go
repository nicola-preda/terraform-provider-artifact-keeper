package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/nicola-preda/terraform-provider-artifact-keeper/internal/client"
)

var (
	_ resource.Resource                = (*formatHandlerResource)(nil)
	_ resource.ResourceWithConfigure   = (*formatHandlerResource)(nil)
	_ resource.ResourceWithImportState = (*formatHandlerResource)(nil)
)

func NewFormatHandlerResource() resource.Resource { return &formatHandlerResource{} }

type formatHandlerResource struct {
	client *client.Client
}

// Format handlers are seeded by the backend; this resource only manages the
// enabled flag (e.g. disable `docker` when container images live in Harbor).
type formatHandlerResourceModel struct {
	ID              types.String `tfsdk:"id"`
	FormatKey       types.String `tfsdk:"format_key"`
	Enabled         types.Bool   `tfsdk:"enabled"`
	DisplayName     types.String `tfsdk:"display_name"`
	HandlerType     types.String `tfsdk:"handler_type"`
	Description     types.String `tfsdk:"description"`
	Priority        types.Int64  `tfsdk:"priority"`
	RepositoryCount types.Int64  `tfsdk:"repository_count"`
}

func (r *formatHandlerResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_format_handler"
}

func (r *formatHandlerResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Enables or disables a built-in format handler (e.g. `docker`, `npm`, `maven`). The handler itself is seeded by the backend; this resource manages only whether it is enabled. Destroying the resource re-enables the handler (the backend default).",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Format handler UUID.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"format_key": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Format handler key, e.g. `docker`, `npm`, `pypi`, `rpm`. Changing this forces a new resource.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"enabled": schema.BoolAttribute{
				Required:            true,
				MarkdownDescription: "Whether the format handler is enabled. Set to `false` to block creating/serving repositories of this format.",
			},
			"display_name":     schema.StringAttribute{Computed: true, MarkdownDescription: "Human-readable handler name."},
			"handler_type":     schema.StringAttribute{Computed: true, MarkdownDescription: "Handler type (e.g. `core`, `plugin`)."},
			"description":      schema.StringAttribute{Computed: true, MarkdownDescription: "Handler description, if any."},
			"priority":         schema.Int64Attribute{Computed: true, MarkdownDescription: "Resolution priority."},
			"repository_count": schema.Int64Attribute{Computed: true, MarkdownDescription: "Number of repositories using this format."},
		},
	}
}

func (r *formatHandlerResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	c, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError("Unexpected provider data type", fmt.Sprintf("Expected *client.Client, got %T. This is a provider bug.", req.ProviderData))
		return
	}
	r.client = c
}

func (r *formatHandlerResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan formatHandlerResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	h, err := r.client.SetFormatHandlerEnabled(ctx, plan.FormatKey.ValueString(), plan.Enabled.ValueBool())
	if err != nil {
		resp.Diagnostics.AddError("Error setting format handler state", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, formatHandlerToModel(h))...)
}

func (r *formatHandlerResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state formatHandlerResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	h, err := r.client.GetFormatHandler(ctx, state.FormatKey.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading format handler", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, formatHandlerToModel(h))...)
}

func (r *formatHandlerResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan formatHandlerResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	h, err := r.client.SetFormatHandlerEnabled(ctx, plan.FormatKey.ValueString(), plan.Enabled.ValueBool())
	if err != nil {
		resp.Diagnostics.AddError("Error setting format handler state", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, formatHandlerToModel(h))...)
}

func (r *formatHandlerResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state formatHandlerResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	// Restore backend default (enabled) so destroy undoes the toggle.
	if _, err := r.client.SetFormatHandlerEnabled(ctx, state.FormatKey.ValueString(), true); err != nil {
		if client.IsNotFound(err) {
			return
		}
		resp.Diagnostics.AddError("Error re-enabling format handler", err.Error())
	}
}

func (r *formatHandlerResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("format_key"), req, resp)
}

func formatHandlerToModel(h *client.FormatHandler) formatHandlerResourceModel {
	return formatHandlerResourceModel{
		ID:              types.StringValue(h.ID),
		FormatKey:       types.StringValue(h.FormatKey),
		Enabled:         types.BoolValue(h.IsEnabled),
		DisplayName:     types.StringValue(h.DisplayName),
		HandlerType:     types.StringValue(h.HandlerType),
		Description:     stringPointerValue(h.Description),
		Priority:        types.Int64Value(h.Priority),
		RepositoryCount: int64PointerValue(h.RepositoryCount),
	}
}
