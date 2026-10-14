package provider

import (
	"context"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/nicola-preda/terraform-provider-artifact-keeper/internal/client"
)

var (
	_ resource.Resource                = (*systemBannerResource)(nil)
	_ resource.ResourceWithConfigure   = (*systemBannerResource)(nil)
	_ resource.ResourceWithImportState = (*systemBannerResource)(nil)
)

func NewSystemBannerResource() resource.Resource { return &systemBannerResource{} }

type systemBannerResource struct {
	client *client.Client
}

type systemBannerResourceModel struct {
	ID        types.String `tfsdk:"id"`
	Title     types.String `tfsdk:"title"`
	Message   types.String `tfsdk:"message"`
	Severity  types.String `tfsdk:"severity"`
	Target    types.String `tfsdk:"target"`
	LinkURL   types.String `tfsdk:"link_url"`
	StartsAt  types.String `tfsdk:"starts_at"`
	EndsAt    types.String `tfsdk:"ends_at"`
	Enabled   types.Bool   `tfsdk:"enabled"`
	Active    types.Bool   `tfsdk:"active"`
	CreatedAt types.String `tfsdk:"created_at"`
	UpdatedAt types.String `tfsdk:"updated_at"`
}

func (r *systemBannerResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_system_banner"
}

func (r *systemBannerResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "A system-wide maintenance or downtime banner (Artifact Keeper 1.11.0), shown in the web UI and/or sent as a header on API responses inside its display window. Admin-only.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Banner UUID.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"title":   schema.StringAttribute{Required: true, MarkdownDescription: "Short headline."},
			"message": schema.StringAttribute{Required: true, MarkdownDescription: "Banner text."},
			"severity": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "`info` (default), `warning` or `critical`.",
				Validators:          []validator.String{stringvalidator.OneOf("info", "warning", "critical")},
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"target": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Where it shows: `all` (default), `ui` or `api`.",
				Validators:          []validator.String{stringvalidator.OneOf("all", "ui", "api")},
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"link_url": schema.StringAttribute{Optional: true, MarkdownDescription: "Optional call-to-action link, `http` or `https` (e.g. a status page)."},
			"starts_at": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "RFC 3339 instant the banner appears. Omit to show it immediately.",
			},
			"ends_at": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "RFC 3339 instant the banner disappears. Omit to keep it until disabled or destroyed.",
			},
			"enabled": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Defaults to `true`.",
				PlanModifiers:       []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"active":     schema.BoolAttribute{Computed: true, MarkdownDescription: "Enabled and inside its display window as of the last read."},
			"created_at": schema.StringAttribute{Computed: true, PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}},
			"updated_at": schema.StringAttribute{Computed: true},
		},
	}
}

func (r *systemBannerResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = configureClient(req.ProviderData, &resp.Diagnostics)
}

func (r *systemBannerResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan systemBannerResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	b, err := r.client.CreateBanner(ctx, bannerInputFromModel(plan))
	if err != nil {
		resp.Diagnostics.AddError("Error creating banner", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, bannerToModel(b, plan))...)
}

func (r *systemBannerResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state systemBannerResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	b, err := r.client.GetBanner(ctx, state.ID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading banner", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, bannerToModel(b, state))...)
}

func (r *systemBannerResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan systemBannerResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	b, err := r.client.UpdateBanner(ctx, plan.ID.ValueString(), bannerInputFromModel(plan))
	if err != nil {
		resp.Diagnostics.AddError("Error updating banner", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, bannerToModel(b, plan))...)
}

func (r *systemBannerResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state systemBannerResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.DeleteBanner(ctx, state.ID.ValueString()); err != nil && !client.IsNotFound(err) {
		resp.Diagnostics.AddError("Error deleting banner", err.Error())
	}
}

func (r *systemBannerResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// bannerInputFromModel builds the full-replace body: the PUT resets anything
// omitted, so every field is always sent.
func bannerInputFromModel(m systemBannerResourceModel) client.BannerInput {
	in := client.BannerInput{
		Title:    m.Title.ValueString(),
		Message:  m.Message.ValueString(),
		LinkURL:  optionalString(m.LinkURL),
		StartsAt: optionalString(m.StartsAt),
		EndsAt:   optionalString(m.EndsAt),
		Enabled:  optionalBool(m.Enabled),
	}
	if v := optionalString(m.Severity); v != nil {
		in.Severity = *v
	}
	if v := optionalString(m.Target); v != nil {
		in.Target = *v
	}
	return in
}

// bannerToModel keeps the practitioner's spelling of a timestamp when the
// server's normalized form is the same instant.
func bannerToModel(b *client.Banner, prior systemBannerResourceModel) systemBannerResourceModel {
	return systemBannerResourceModel{
		ID:        types.StringValue(b.ID),
		Title:     types.StringValue(b.Title),
		Message:   types.StringValue(b.Message),
		Severity:  types.StringValue(b.Severity),
		Target:    types.StringValue(b.Target),
		LinkURL:   stringPointerValue(b.LinkURL),
		StartsAt:  sameInstantOr(prior.StartsAt, b.StartsAt),
		EndsAt:    sameInstantOr(prior.EndsAt, b.EndsAt),
		Enabled:   types.BoolValue(b.Enabled),
		Active:    types.BoolValue(b.Active),
		CreatedAt: types.StringValue(b.CreatedAt),
		UpdatedAt: types.StringValue(b.UpdatedAt),
	}
}

func sameInstantOr(prior types.String, got *string) types.String {
	if got == nil || prior.IsNull() || prior.IsUnknown() {
		return stringPointerValue(got)
	}
	a, errA := time.Parse(time.RFC3339Nano, prior.ValueString())
	b, errB := time.Parse(time.RFC3339Nano, *got)
	if errA == nil && errB == nil && a.Equal(b) {
		return prior
	}
	return types.StringValue(*got)
}
