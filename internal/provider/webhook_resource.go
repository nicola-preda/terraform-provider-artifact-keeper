package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/mapplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/nicola-preda/terraform-provider-artifact-keeper/internal/client"
)

var (
	_ resource.Resource                = (*webhookResource)(nil)
	_ resource.ResourceWithConfigure   = (*webhookResource)(nil)
	_ resource.ResourceWithImportState = (*webhookResource)(nil)
)

func NewWebhookResource() resource.Resource { return &webhookResource{} }

type webhookResource struct {
	client *client.Client
}

type webhookResourceModel struct {
	ID                   types.String `tfsdk:"id"`
	Name                 types.String `tfsdk:"name"`
	URL                  types.String `tfsdk:"url"`
	Events               types.List   `tfsdk:"events"`
	Secret               types.String `tfsdk:"secret"`
	RepositoryID         types.String `tfsdk:"repository_id"`
	Headers              types.Map    `tfsdk:"headers"`
	PayloadTemplate      types.String `tfsdk:"payload_template"`
	EventSchemaVersion   types.String `tfsdk:"event_schema_version"`
	IsEnabled            types.Bool   `tfsdk:"is_enabled"`
	SecretDigest         types.String `tfsdk:"secret_digest"`
	SecretRotationActive types.Bool   `tfsdk:"secret_rotation_active"`
	LastTriggeredAt      types.String `tfsdk:"last_triggered_at"`
	CreatedAt            types.String `tfsdk:"created_at"`
}

func (r *webhookResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_webhook"
}

func (r *webhookResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "An outbound webhook. There's no update for the definition, so changing anything other than `is_enabled` forces a new webhook; `is_enabled` toggles in place. Admin-only.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Webhook UUID assigned by Artifact Keeper.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Webhook name. Changing this forces a new webhook.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"url": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Target URL that receives delivery POSTs. Must not resolve to a private/internal address. Changing this forces a new webhook.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"events": schema.ListAttribute{
				ElementType:         types.StringType,
				Required:            true,
				MarkdownDescription: "Event types that trigger this webhook (e.g. `artifact_uploaded`, `repository_created`, `build_failed`). At least one is required. Changing this forces a new webhook.",
				PlanModifiers:       []planmodifier.List{listplanmodifier.RequiresReplace()},
			},
			"secret": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Sensitive:           true,
				MarkdownDescription: "Signing secret used to HMAC delivery bodies. If omitted, the server generates one and it is captured into state here (the API returns it only once, in the raw create response, and never again). Not returned by the API on read; preserved from state. Changing this forces a new webhook.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace(), stringplanmodifier.UseStateForUnknown()},
			},
			"repository_id": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "UUID of the repository this webhook is scoped to. Omit for a global webhook. Changing this forces a new webhook.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"headers": schema.MapAttribute{
				ElementType:         types.StringType,
				Optional:            true,
				MarkdownDescription: "Custom HTTP headers to send with each delivery. Changing this forces a new webhook.",
				PlanModifiers:       []planmodifier.Map{mapplanmodifier.RequiresReplace()},
			},
			"payload_template": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Payload layout for the target platform. Defaults to `generic`. Changing this forces a new webhook.",
				Validators:          []validator.String{stringvalidator.OneOf("generic", "slack", "microsoft_teams", "discord", "mattermost")},
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace(), stringplanmodifier.UseStateForUnknown()},
			},
			"event_schema_version": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Pinned event payload version. Defaults to `2026-04-01`. Changing this forces a new webhook.",
				Validators:          []validator.String{stringvalidator.OneOf("2026-04-01")},
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace(), stringplanmodifier.UseStateForUnknown()},
			},
			"is_enabled": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Whether the webhook is enabled. Defaults to `true`. This is the only attribute that can be changed in place.",
				PlanModifiers:       []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"secret_digest": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Short non-reversible identifier for the current signing secret (e.g. `whsec_...abcd`). Null when the webhook is unsigned.",
			},
			"secret_rotation_active": schema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "True while a previous signing secret is still accepted during a rotation overlap window.",
			},
			"last_triggered_at": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "RFC 3339 timestamp of the last delivery attempt, if any.",
			},
			"created_at": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "RFC 3339 creation timestamp.",
			},
		},
	}
}

func (r *webhookResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = configureClient(req.ProviderData, &resp.Diagnostics)
}

func (r *webhookResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan webhookResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	events, d := listToStringSlice(ctx, plan.Events)
	resp.Diagnostics.Append(d...)
	headers, d := mapToStringMap(ctx, plan.Headers)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}

	createReq := client.CreateWebhookRequest{
		Name:    plan.Name.ValueString(),
		URL:     plan.URL.ValueString(),
		Events:  events,
		Headers: headers,
	}
	if !plan.Secret.IsNull() {
		createReq.Secret = plan.Secret.ValueStringPointer()
	}
	if !plan.RepositoryID.IsNull() {
		createReq.RepositoryID = plan.RepositoryID.ValueStringPointer()
	}
	if !plan.PayloadTemplate.IsNull() && !plan.PayloadTemplate.IsUnknown() {
		createReq.PayloadTemplate = plan.PayloadTemplate.ValueStringPointer()
	}
	if !plan.EventSchemaVersion.IsNull() && !plan.EventSchemaVersion.IsUnknown() {
		createReq.EventSchemaVersion = plan.EventSchemaVersion.ValueStringPointer()
	}

	wh, generatedSecret, err := r.client.CreateWebhook(ctx, createReq)
	if err != nil {
		resp.Diagnostics.AddError("Error creating webhook", err.Error())
		return
	}

	// Webhooks are created enabled; reconcile if the caller asked otherwise.
	if !plan.IsEnabled.IsNull() && !plan.IsEnabled.IsUnknown() && plan.IsEnabled.ValueBool() != wh.IsEnabled {
		if err := r.client.SetWebhookEnabled(ctx, wh.ID, plan.IsEnabled.ValueBool()); err != nil {
			resp.Diagnostics.AddError("Error setting webhook enabled state", err.Error())
			return
		}
		wh, err = r.client.GetWebhook(ctx, wh.ID)
		if err != nil {
			resp.Diagnostics.AddError("Error reading back created webhook", err.Error())
			return
		}
	}

	state, d := webhookToModel(ctx, wh)
	resp.Diagnostics.Append(d...)
	// The API never returns the secret on read. Preserve the configured secret,
	// or capture the server-generated one (returned only once, on create).
	switch {
	case !plan.Secret.IsNull() && !plan.Secret.IsUnknown():
		state.Secret = plan.Secret
	case generatedSecret != nil:
		state.Secret = types.StringValue(*generatedSecret)
	default:
		state.Secret = types.StringNull()
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *webhookResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state webhookResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	wh, err := r.client.GetWebhook(ctx, state.ID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading webhook", err.Error())
		return
	}

	refreshed, d := webhookToModel(ctx, wh)
	resp.Diagnostics.Append(d...)
	refreshed.Secret = state.Secret // not returned by the API; preserved from config
	resp.Diagnostics.Append(resp.State.Set(ctx, refreshed)...)
}

// Update only ever reconciles is_enabled: every other attribute forces
// replacement, so Terraform never routes their changes through here.
func (r *webhookResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan webhookResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if !plan.IsEnabled.IsNull() && !plan.IsEnabled.IsUnknown() {
		if err := r.client.SetWebhookEnabled(ctx, plan.ID.ValueString(), plan.IsEnabled.ValueBool()); err != nil {
			resp.Diagnostics.AddError("Error setting webhook enabled state", err.Error())
			return
		}
	}

	wh, err := r.client.GetWebhook(ctx, plan.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading webhook", err.Error())
		return
	}

	state, d := webhookToModel(ctx, wh)
	resp.Diagnostics.Append(d...)
	state.Secret = plan.Secret // not returned by the API; preserved from config
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *webhookResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state webhookResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.DeleteWebhook(ctx, state.ID.ValueString()); err != nil && !client.IsNotFound(err) {
		resp.Diagnostics.AddError("Error deleting webhook", err.Error())
	}
}

func (r *webhookResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func webhookToModel(ctx context.Context, wh *client.Webhook) (webhookResourceModel, diag.Diagnostics) {
	var diags diag.Diagnostics
	events, d := stringListValue(ctx, wh.Events)
	diags.Append(d...)
	headers, d := stringMapValue(ctx, wh.Headers)
	diags.Append(d...)
	return webhookResourceModel{
		ID:                   types.StringValue(wh.ID),
		Name:                 types.StringValue(wh.Name),
		URL:                  types.StringValue(wh.URL),
		Events:               events,
		RepositoryID:         stringPointerValue(wh.RepositoryID),
		Headers:              headers,
		PayloadTemplate:      types.StringValue(wh.PayloadTemplate),
		EventSchemaVersion:   types.StringValue(wh.EventSchemaVersion),
		IsEnabled:            types.BoolValue(wh.IsEnabled),
		SecretDigest:         stringPointerValue(wh.SecretDigest),
		SecretRotationActive: types.BoolValue(wh.SecretRotationActive),
		LastTriggeredAt:      stringPointerValue(wh.LastTriggeredAt),
		CreatedAt:            types.StringValue(wh.CreatedAt),
	}, diags
}
