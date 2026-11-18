package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// writeOnlySecret returns the `<plain>_wo` and `<plain>_wo_version` siblings of a sensitive attribute. Both are optional, so clients below 1.11 keep working with `<plain>` and only fail when `_wo` is set
func writeOnlySecret(plain string, replace bool) (schema.StringAttribute, schema.Int64Attribute) {
	wo := schema.StringAttribute{
		Optional:            true,
		Sensitive:           true,
		WriteOnly:           true,
		MarkdownDescription: "Write-only alternative to `" + plain + "`, never stored in plan or state. Needs Terraform 1.11 or OpenTofu 1.11 or later. Bump `" + plain + "_wo_version` to send a new value.",
		Validators:          []validator.String{stringvalidator.ConflictsWith(path.MatchRoot(plain))},
	}
	ver := schema.Int64Attribute{
		Optional:            true,
		MarkdownDescription: "Change this to send `" + plain + "_wo` again, the write-only value itself is not compared between runs.",
		Validators:          []validator.Int64{int64validator.AlsoRequires(path.MatchRoot(plain + "_wo"))},
	}
	if replace {
		ver.MarkdownDescription += " Changing it forces a new resource, like `" + plain + "`."
		ver.PlanModifiers = []planmodifier.Int64{int64planmodifier.RequiresReplace()}
	}
	return wo, ver
}

// secretValue returns `plain` when set, else the write-only sibling, which only the config carries
func secretValue(ctx context.Context, cfg tfsdk.Config, plain types.String, wo string, diags *diag.Diagnostics) *string {
	if isSet(plain) {
		return plain.ValueStringPointer()
	}
	var v types.String
	diags.Append(cfg.GetAttribute(ctx, path.Root(wo), &v)...)
	if !isSet(v) {
		return nil
	}
	return v.ValueStringPointer()
}

// updatedSecret is secretValue for an Update that may leave the stored secret alone: the write-only value goes out only when `_wo_version` changed or when it replaces a plain value
func updatedSecret(ctx context.Context, cfg tfsdk.Config, plan, state types.String, planVersion, stateVersion types.Int64, wo string, diags *diag.Diagnostics) *string {
	if !isSet(plan) && planVersion.Equal(stateVersion) && state.IsNull() {
		return nil
	}
	return secretValue(ctx, cfg, plan, wo, diags)
}
