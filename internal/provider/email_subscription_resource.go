package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/nicola-preda/terraform-provider-artifact-keeper/internal/client"
)

var (
	_ resource.Resource                = (*emailSubscriptionResource)(nil)
	_ resource.ResourceWithConfigure   = (*emailSubscriptionResource)(nil)
	_ resource.ResourceWithImportState = (*emailSubscriptionResource)(nil)
)

func NewEmailSubscriptionResource() resource.Resource { return &emailSubscriptionResource{} }

type emailSubscriptionResource struct {
	client *client.Client
}

// Addressed by the (repository_key, id) pair.
type emailSubscriptionResourceModel struct {
	ID            types.String `tfsdk:"id"`
	RepositoryKey types.String `tfsdk:"repository_key"`
	Recipients    types.List   `tfsdk:"recipients"`
	EventTypes    types.List   `tfsdk:"event_types"`
	Enabled       types.Bool   `tfsdk:"enabled"`
}

func (r *emailSubscriptionResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_email_subscription"
}

func (r *emailSubscriptionResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages an email subscription scoped to a single repository in Artifact Keeper. Matching repository events are delivered to the configured recipients. The API has no update endpoint, so subscriptions are immutable: any change forces a new subscription.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Subscription UUID assigned by Artifact Keeper.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"repository_key": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Key of the repository this subscription is scoped to. Changing this forces a new subscription.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"recipients": schema.ListAttribute{
				ElementType:         types.StringType,
				Required:            true,
				MarkdownDescription: "Email addresses to deliver matching events to. Changing this forces a new subscription.",
				PlanModifiers:       []planmodifier.List{listplanmodifier.RequiresReplace()},
			},
			"event_types": schema.ListAttribute{
				ElementType:         types.StringType,
				Required:            true,
				MarkdownDescription: "Event-type tokens to subscribe to, e.g. `[\"artifact.uploaded\", \"scan.completed\"]`. Changing this forces a new subscription.",
				PlanModifiers:       []planmodifier.List{listplanmodifier.RequiresReplace()},
			},
			"enabled": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Whether the subscription delivers events. Defaults to `true`. Changing this forces a new subscription.",
				PlanModifiers:       []planmodifier.Bool{boolplanmodifier.UseStateForUnknown(), boolplanmodifier.RequiresReplace()},
			},
		},
	}
}

func (r *emailSubscriptionResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = configureClient(req.ProviderData, &resp.Diagnostics)
}

func (r *emailSubscriptionResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan emailSubscriptionResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	recipients, d := listToStringSlice(ctx, plan.Recipients)
	resp.Diagnostics.Append(d...)
	eventTypes, d := listToStringSlice(ctx, plan.EventTypes)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}

	createReq := client.CreateEmailSubscriptionRequest{
		Recipients: recipients,
		EventTypes: eventTypes,
	}
	if !plan.Enabled.IsNull() && !plan.Enabled.IsUnknown() {
		createReq.Enabled = plan.Enabled.ValueBoolPointer()
	}

	key := plan.RepositoryKey.ValueString()
	created, err := r.client.CreateEmailSubscription(ctx, key, createReq)
	if err != nil {
		resp.Diagnostics.AddError("Error creating email subscription", err.Error())
		return
	}

	state, d := emailSubscriptionToModel(ctx, key, created)
	resp.Diagnostics.Append(d...)
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *emailSubscriptionResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state emailSubscriptionResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// No single-item GET: list the repo's subscriptions and find by id. A 404 or
	// a missing id both mean it's gone; drop from state to plan a recreate.
	key := state.RepositoryKey.ValueString()
	subs, err := r.client.ListEmailSubscriptions(ctx, key)
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading email subscriptions", err.Error())
		return
	}

	id := state.ID.ValueString()
	var found *client.EmailSubscription
	for i := range subs {
		if subs[i].ID == id {
			found = &subs[i]
			break
		}
	}
	if found == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	refreshed, d := emailSubscriptionToModel(ctx, key, found)
	resp.Diagnostics.Append(d...)
	resp.Diagnostics.Append(resp.State.Set(ctx, refreshed)...)
}

// Never runs: every field forces replacement (no update endpoint). Satisfies
// the interface.
func (r *emailSubscriptionResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan emailSubscriptionResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *emailSubscriptionResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state emailSubscriptionResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.DeleteEmailSubscription(ctx, state.RepositoryKey.ValueString(), state.ID.ValueString()); err != nil && !client.IsNotFound(err) {
		resp.Diagnostics.AddError("Error deleting email subscription", err.Error())
	}
}

// Composite import ID "<repository_key>/<subscription_id>"; both parts address
// the subscription.
func (r *emailSubscriptionResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	key, id, ok := strings.Cut(req.ID, "/")
	if !ok || key == "" || id == "" {
		resp.Diagnostics.AddError(
			"Unexpected import identifier",
			fmt.Sprintf("Expected import ID in the format \"<repository_key>/<subscription_id>\", got: %q", req.ID),
		)
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("repository_key"), key)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), id)...)
}

func emailSubscriptionToModel(ctx context.Context, key string, s *client.EmailSubscription) (emailSubscriptionResourceModel, diag.Diagnostics) {
	recipients, d := stringListValue(ctx, s.Recipients)
	eventTypes, d2 := stringListValue(ctx, s.EventTypes)
	d.Append(d2...)
	return emailSubscriptionResourceModel{
		ID:            types.StringValue(s.ID),
		RepositoryKey: types.StringValue(key),
		Recipients:    recipients,
		EventTypes:    eventTypes,
		Enabled:       types.BoolValue(s.Enabled),
	}, d
}
