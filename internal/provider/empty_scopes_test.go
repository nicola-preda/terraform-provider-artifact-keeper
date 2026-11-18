package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// The backend stores its default scopes for [], which then read back as new
// elements after apply
func TestEmptyScopesRefused(t *testing.T) {
	ctx := context.Background()
	for name, r := range map[string]resource.Resource{
		"api_token": NewApiTokenResource(),
		"sso_oidc":  NewSsoOidcResource(),
	} {
		var resp resource.SchemaResponse
		r.Schema(ctx, resource.SchemaRequest{}, &resp)
		scopes := resp.Schema.Attributes["scopes"].(schema.ListAttribute)
		req := validator.ListRequest{Path: path.Root("scopes"), ConfigValue: types.ListValueMust(types.StringType, []attr.Value{})}
		refused := false
		for _, v := range scopes.Validators {
			var out validator.ListResponse
			v.ValidateList(ctx, req, &out)
			refused = refused || out.Diagnostics.HasError()
		}
		if !refused {
			t.Errorf("%s: scopes = [] passes validation", name)
		}
	}
}
