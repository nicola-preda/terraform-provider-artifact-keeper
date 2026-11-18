package provider

import (
	"context"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// validateConfig sends a resource config with only `set` non-null through ValidateResourceConfig, the way a client with or without write-only support would
func validateConfig(t *testing.T, typeName string, set map[string]tftypes.Value, writeOnlyClient bool) []*tfprotov6.Diagnostic {
	t.Helper()
	ctx := context.Background()
	server, err := providerserver.NewProtocol6WithError(New("test")())()
	if err != nil {
		t.Fatal(err)
	}
	schemas, err := server.GetProviderSchema(ctx, &tfprotov6.GetProviderSchemaRequest{})
	if err != nil {
		t.Fatal(err)
	}
	s, ok := schemas.ResourceSchemas[typeName]
	if !ok {
		t.Fatalf("no schema for %s", typeName)
	}
	typ := s.ValueType().(tftypes.Object)
	vals := map[string]tftypes.Value{}
	for name, at := range typ.AttributeTypes {
		if v, ok := set[name]; ok {
			vals[name] = v
		} else {
			vals[name] = tftypes.NewValue(at, nil)
		}
	}
	cfg, err := tfprotov6.NewDynamicValue(typ, tftypes.NewValue(typ, vals))
	if err != nil {
		t.Fatal(err)
	}
	resp, err := server.ValidateResourceConfig(ctx, &tfprotov6.ValidateResourceConfigRequest{
		TypeName:           typeName,
		Config:             &cfg,
		ClientCapabilities: &tfprotov6.ValidateResourceConfigClientCapabilities{WriteOnlyAttributesAllowed: writeOnlyClient},
	})
	if err != nil {
		t.Fatal(err)
	}
	return resp.Diagnostics
}

func diagSummaries(diags []*tfprotov6.Diagnostic) string {
	var out []string
	for _, d := range diags {
		if d.Severity == tfprotov6.DiagnosticSeverityError {
			out = append(out, d.Summary)
		}
	}
	return strings.Join(out, "; ")
}

func TestWriteOnlySecretsOnOldAndNewClients(t *testing.T) {
	str := func(s string) tftypes.Value { return tftypes.NewValue(tftypes.String, s) }
	num := func(n int64) tftypes.Value { return tftypes.NewValue(tftypes.Number, n) }
	events := tftypes.NewValue(tftypes.List{ElementType: tftypes.String}, []tftypes.Value{str("artifact_uploaded")})

	secrets := []struct {
		typ, plain string
		required   bool
		base       map[string]tftypes.Value
	}{
		{"artifactkeeper_repository_upstream_auth", "password", false, map[string]tftypes.Value{"repository_key": str("r"), "auth_type": str("basic"), "username": str("u")}},
		{"artifactkeeper_sso_oidc", "client_secret", true, map[string]tftypes.Value{"name": str("o"), "issuer_url": str("https://idp.example.com"), "client_id": str("c")}},
		{"artifactkeeper_sso_ldap", "bind_password", false, map[string]tftypes.Value{"name": str("l"), "server_url": str("ldaps://ldap.example.com"), "user_base_dn": str("dc=example"), "bind_dn": str("cn=svc")}},
		{"artifactkeeper_repository_egress_proxy", "proxy_url", false, map[string]tftypes.Value{"repository_key": str("r"), "mode": str("explicit")}},
		{"artifactkeeper_user", "password", false, map[string]tftypes.Value{"username": str("u"), "email": str("u@example.com")}},
		{"artifactkeeper_webhook", "secret", false, map[string]tftypes.Value{"name": str("w"), "url": str("https://hooks.example.com"), "events": events}},
		{"artifactkeeper_peer", "api_key", true, map[string]tftypes.Value{"name": str("p"), "endpoint_url": str("https://peer.example.com")}},
		{"artifactkeeper_remote_instance", "api_key", true, map[string]tftypes.Value{"name": str("ri"), "url": str("https://ak.example.com")}},
		{"artifactkeeper_migration_source", "credential_token", false, map[string]tftypes.Value{"name": str("m"), "url": str("https://jfrog.example.com"), "auth_type": str("api_token")}},
		{"artifactkeeper_migration_source", "credential_password", false, map[string]tftypes.Value{"name": str("m"), "url": str("https://nexus.example.com"), "auth_type": str("basic_auth"), "credential_username": str("u")}},
	}

	for _, sec := range secrets {
		wo, ver := sec.plain+"_wo", sec.plain+"_wo_version"
		with := func(extra map[string]tftypes.Value) map[string]tftypes.Value {
			m := map[string]tftypes.Value{}
			for k, v := range sec.base {
				m[k] = v
			}
			for k, v := range extra {
				m[k] = v
			}
			return m
		}
		neither := ""
		if sec.required {
			neither = "Invalid Attribute Combination"
		}
		cases := []struct {
			name     string
			set      map[string]tftypes.Value
			woClient bool
			wantErr  string
		}{
			{"plain on an old client", with(map[string]tftypes.Value{sec.plain: str("s")}), false, ""},
			{"plain on a new client", with(map[string]tftypes.Value{sec.plain: str("s")}), true, ""},
			{"wo on a new client", with(map[string]tftypes.Value{wo: str("s"), ver: num(1)}), true, ""},
			{"wo without version on a new client", with(map[string]tftypes.Value{wo: str("s")}), true, ""},
			{"wo on an old client", with(map[string]tftypes.Value{wo: str("s")}), false, "WriteOnly Attribute Not Allowed"},
			{"both", with(map[string]tftypes.Value{sec.plain: str("s"), wo: str("s")}), true, "Invalid Attribute Combination"},
			{"version without wo", with(map[string]tftypes.Value{sec.plain: str("s"), ver: num(1)}), true, "Invalid Attribute Combination"},
			{"neither", with(nil), true, neither},
		}
		for _, tc := range cases {
			t.Run(sec.typ+" "+sec.plain+" "+tc.name, func(t *testing.T) {
				got := diagSummaries(validateConfig(t, sec.typ, tc.set, tc.woClient))
				if tc.wantErr == "" && got != "" {
					t.Fatalf("unexpected errors: %s", got)
				}
				if tc.wantErr != "" && !strings.Contains(got, tc.wantErr) {
					t.Fatalf("want %q, got %q", tc.wantErr, got)
				}
			})
		}
	}
}

// A `_wo_version` bump is in place where the plain attribute changes in place, and forces a new resource where it does
func TestWriteOnlyVersionReplacement(t *testing.T) {
	ctx := context.Background()
	p := New("test")()
	resources := map[string]resource.Resource{}
	for _, f := range p.Resources(ctx) {
		r := f()
		md := &resource.MetadataResponse{}
		r.Metadata(ctx, resource.MetadataRequest{ProviderTypeName: "artifactkeeper"}, md)
		resources[md.TypeName] = r
	}
	cases := map[string]bool{
		"artifactkeeper_repository_upstream_auth.password":    false,
		"artifactkeeper_sso_oidc.client_secret":               false,
		"artifactkeeper_sso_ldap.bind_password":               false,
		"artifactkeeper_repository_egress_proxy.proxy_url":    false,
		"artifactkeeper_user.password":                        false,
		"artifactkeeper_webhook.secret":                       true,
		"artifactkeeper_peer.api_key":                         true,
		"artifactkeeper_remote_instance.api_key":              true,
		"artifactkeeper_migration_source.credential_token":    true,
		"artifactkeeper_migration_source.credential_password": true,
	}
	nonNull := tftypes.NewValue(tftypes.Object{}, map[string]tftypes.Value{})
	for name, replace := range cases {
		typ, plain, _ := strings.Cut(name, ".")
		r, ok := resources[typ]
		if !ok {
			t.Fatalf("no resource %s", typ)
		}
		sr := &resource.SchemaResponse{}
		r.Schema(ctx, resource.SchemaRequest{}, sr)
		attrs := sr.Schema.Attributes

		plainAttr, ok := attrs[plain].(schema.StringAttribute)
		if !ok || !plainAttr.Sensitive || plainAttr.Required {
			t.Errorf("%s: want an optional sensitive string", name)
		}
		wo, ok := attrs[plain+"_wo"].(schema.StringAttribute)
		if !ok || !wo.WriteOnly || !wo.Sensitive || !wo.Optional {
			t.Errorf("%s_wo: want optional, sensitive and write-only", name)
		}
		ver, ok := attrs[plain+"_wo_version"].(schema.Int64Attribute)
		if !ok || !ver.Optional || ver.Computed || ver.WriteOnly {
			t.Fatalf("%s_wo_version: want a plain optional number", name)
		}

		req := planmodifier.Int64Request{
			State:      tfsdk.State{Raw: nonNull},
			Plan:       tfsdk.Plan{Raw: nonNull},
			StateValue: types.Int64Value(1),
			PlanValue:  types.Int64Value(2),
		}
		got := false
		for _, m := range ver.PlanModifiers {
			resp := &planmodifier.Int64Response{PlanValue: req.PlanValue}
			m.PlanModifyInt64(ctx, req, resp)
			got = got || resp.RequiresReplace
		}
		if got != replace {
			t.Errorf("%s_wo_version: requires replace = %v, want %v", name, got, replace)
		}
		plainReplace := false
		for _, m := range plainAttr.PlanModifiers {
			sreq := planmodifier.StringRequest{State: req.State, Plan: req.Plan, StateValue: types.StringValue("a"), PlanValue: types.StringValue("b"), ConfigValue: types.StringValue("b")}
			resp := &planmodifier.StringResponse{PlanValue: sreq.PlanValue}
			m.PlanModifyString(ctx, sreq, resp)
			plainReplace = plainReplace || resp.RequiresReplace
		}
		if plainReplace != replace {
			t.Errorf("%s: the plain attribute requires replace = %v, the version %v", name, plainReplace, replace)
		}
	}
}

func TestUpdatedSecret(t *testing.T) {
	ctx := context.Background()
	cfgSchema := schema.Schema{Attributes: map[string]schema.Attribute{
		"s":    schema.StringAttribute{Optional: true},
		"s_wo": schema.StringAttribute{Optional: true, WriteOnly: true},
	}}
	typ := tftypes.Object{AttributeTypes: map[string]tftypes.Type{"s": tftypes.String, "s_wo": tftypes.String}}
	config := func(plain, wo any) tfsdk.Config {
		return tfsdk.Config{Schema: cfgSchema, Raw: tftypes.NewValue(typ, map[string]tftypes.Value{
			"s":    tftypes.NewValue(tftypes.String, plain),
			"s_wo": tftypes.NewValue(tftypes.String, wo),
		})}
	}
	null, one, two := types.Int64Null(), types.Int64Value(1), types.Int64Value(2)
	cases := []struct {
		name              string
		cfg               tfsdk.Config
		plan, state       types.String
		planVer, stateVer types.Int64
		want              string
	}{
		{"plain is sent as before", config("p", nil), types.StringValue("p"), types.StringValue("p"), null, null, "p"},
		{"wo with the same version stays out", config(nil, "w"), types.StringNull(), types.StringNull(), one, one, ""},
		{"wo without a version stays out", config(nil, "w"), types.StringNull(), types.StringNull(), null, null, ""},
		{"wo version bump sends it", config(nil, "w"), types.StringNull(), types.StringNull(), two, one, "w"},
		{"wo version added sends it", config(nil, "w"), types.StringNull(), types.StringNull(), one, null, "w"},
		{"switch from plain sends it", config(nil, "w"), types.StringNull(), types.StringValue("p"), null, null, "w"},
		{"dropping the secret sends nothing", config(nil, nil), types.StringNull(), types.StringValue("p"), null, null, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var diags diag.Diagnostics
			got := updatedSecret(ctx, tc.cfg, tc.plan, tc.state, tc.planVer, tc.stateVer, "s_wo", &diags)
			if diags.HasError() {
				t.Fatal(diags)
			}
			if (got == nil) != (tc.want == "") || (got != nil && *got != tc.want) {
				t.Fatalf("got %v, want %q", got, tc.want)
			}
		})
	}
}
