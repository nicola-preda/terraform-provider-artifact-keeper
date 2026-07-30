package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/nicola-preda/terraform-provider-artifact-keeper/internal/client"
)

var (
	_ resource.Resource                = (*repositoryRoutingRulesResource)(nil)
	_ resource.ResourceWithConfigure   = (*repositoryRoutingRulesResource)(nil)
	_ resource.ResourceWithImportState = (*repositoryRoutingRulesResource)(nil)
)

func NewRepositoryRoutingRulesResource() resource.Resource {
	return &repositoryRoutingRulesResource{}
}

type repositoryRoutingRulesResource struct {
	client *client.Client
}

type repositoryRoutingRulesResourceModel struct {
	ID            types.String `tfsdk:"id"`
	RepositoryKey types.String `tfsdk:"repository_key"`
	Rules         types.List   `tfsdk:"rules"`
}

type routingRuleModel struct {
	PathPattern types.String `tfsdk:"path_pattern"`
	RewriteTo   types.String `tfsdk:"rewrite_to"`
}

var routingRuleAttrTypes = map[string]attr.Type{
	"path_pattern": types.StringType,
	"rewrite_to":   types.StringType,
}

func (r *repositoryRoutingRulesResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_repository_routing_rules"
}

func (r *repositoryRoutingRulesResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Ordered path-rewrite rules applied to a repository's proxy requests. A singleton keyed by repository key: a write replaces the whole ordered list, and destroying it clears every rule. Rules are evaluated in order, first match wins, so list order is significant.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Resource identifier. Equal to `repository_key`, since the rules are a singleton per repository.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"repository_key": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Key of the repository whose routing rules are managed. Changing this forces a new resource.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"rules": schema.ListNestedAttribute{
				Required:            true,
				MarkdownDescription: "Ordered list of rewrite rules. Evaluated top to bottom, first match wins.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"path_pattern": schema.StringAttribute{
							Required:            true,
							MarkdownDescription: "Pattern matched against the incoming request path.",
						},
						"rewrite_to": schema.StringAttribute{
							Required:            true,
							MarkdownDescription: "Path the request is rewritten to when the pattern matches.",
						},
					},
				},
			},
		},
	}
}

func (r *repositoryRoutingRulesResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = configureClient(req.ProviderData, &resp.Diagnostics)
}

func (r *repositoryRoutingRulesResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan repositoryRoutingRulesResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	repoKey := plan.RepositoryKey.ValueString()
	rules, d := routingRulesFromList(ctx, plan.Rules)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.SetRepositoryRoutingRules(ctx, repoKey, rules); err != nil {
		resp.Diagnostics.AddError("Error setting repository routing rules", err.Error())
		return
	}
	out, err := r.client.GetRepositoryRoutingRules(ctx, repoKey)
	if err != nil {
		resp.Diagnostics.AddError("Error reading repository routing rules", err.Error())
		return
	}
	model, d := routingRulesToModel(ctx, repoKey, out.Rules)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, model)...)
}

func (r *repositoryRoutingRulesResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state repositoryRoutingRulesResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	repoKey := state.RepositoryKey.ValueString()
	out, err := r.client.GetRepositoryRoutingRules(ctx, repoKey)
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading repository routing rules", err.Error())
		return
	}
	model, d := routingRulesToModel(ctx, repoKey, out.Rules)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, model)...)
}

func (r *repositoryRoutingRulesResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan repositoryRoutingRulesResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	repoKey := plan.RepositoryKey.ValueString()
	rules, d := routingRulesFromList(ctx, plan.Rules)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}
	// The POST replaces the whole ordered list.
	if err := r.client.SetRepositoryRoutingRules(ctx, repoKey, rules); err != nil {
		resp.Diagnostics.AddError("Error updating repository routing rules", err.Error())
		return
	}
	out, err := r.client.GetRepositoryRoutingRules(ctx, repoKey)
	if err != nil {
		resp.Diagnostics.AddError("Error reading repository routing rules", err.Error())
		return
	}
	model, d := routingRulesToModel(ctx, repoKey, out.Rules)
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, model)...)
}

func (r *repositoryRoutingRulesResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state repositoryRoutingRulesResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.DeleteRepositoryRoutingRules(ctx, state.RepositoryKey.ValueString()); err != nil {
		if client.IsNotFound(err) {
			return
		}
		resp.Diagnostics.AddError("Error deleting repository routing rules", err.Error())
	}
}

func (r *repositoryRoutingRulesResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Addressed by repository key in the API; import by that key.
	resource.ImportStatePassthroughID(ctx, path.Root("repository_key"), req, resp)
}

// routingRulesFromList reads the nested-object list into an ordered slice of
// client rules.
func routingRulesFromList(ctx context.Context, l types.List) ([]client.RoutingRule, diag.Diagnostics) {
	var models []routingRuleModel
	diags := l.ElementsAs(ctx, &models, false)
	if diags.HasError() {
		return nil, diags
	}
	rules := make([]client.RoutingRule, len(models))
	for i, m := range models {
		rules[i] = client.RoutingRule{
			PathPattern: m.PathPattern.ValueString(),
			RewriteTo:   m.RewriteTo.ValueString(),
		}
	}
	return rules, diags
}

func routingRulesToModel(ctx context.Context, repoKey string, rules []client.RoutingRule) (repositoryRoutingRulesResourceModel, diag.Diagnostics) {
	models := make([]routingRuleModel, len(rules))
	for i, rule := range rules {
		models[i] = routingRuleModel{
			PathPattern: types.StringValue(rule.PathPattern),
			RewriteTo:   types.StringValue(rule.RewriteTo),
		}
	}
	lv, diags := types.ListValueFrom(ctx, types.ObjectType{AttrTypes: routingRuleAttrTypes}, models)
	return repositoryRoutingRulesResourceModel{
		ID:            types.StringValue(repoKey),
		RepositoryKey: types.StringValue(repoKey),
		Rules:         lv,
	}, diags
}
