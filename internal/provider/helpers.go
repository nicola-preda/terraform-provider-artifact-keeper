package provider

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/nicola-preda/terraform-provider-artifact-keeper/internal/client"
)

// configureClient extracts the *client.Client passed via ProviderData, adding a
// diagnostic on type mismatch. Returns nil when ProviderData is nil (which
// happens during early validation walks).
func configureClient(providerData any, diags *diag.Diagnostics) *client.Client {
	if providerData == nil {
		return nil
	}
	c, ok := providerData.(*client.Client)
	if !ok {
		diags.AddError(
			"Unexpected provider data type",
			fmt.Sprintf("Expected *client.Client, got %T. This is a provider bug.", providerData),
		)
		return nil
	}
	return c
}

// stringListValue converts a Go string slice into a types.List, returning a
// null list for a nil slice.
func stringListValue(ctx context.Context, vals []string) (types.List, diag.Diagnostics) {
	if vals == nil {
		return types.ListNull(types.StringType), nil
	}
	return types.ListValueFrom(ctx, types.StringType, vals)
}

// stringMapValue converts a Go string map into a types.Map, returning a null
// map for a nil map.
func stringMapValue(ctx context.Context, m map[string]string) (types.Map, diag.Diagnostics) {
	if m == nil {
		return types.MapNull(types.StringType), nil
	}
	return types.MapValueFrom(ctx, types.StringType, m)
}

// listToStringSlice reads a types.List into a Go slice (nil when null/unknown).
func listToStringSlice(ctx context.Context, l types.List) ([]string, diag.Diagnostics) {
	if l.IsNull() || l.IsUnknown() {
		return nil, nil
	}
	var out []string
	d := l.ElementsAs(ctx, &out, false)
	return out, d
}

// optionalBool returns a *bool for a set value, or nil when null/unknown.
func optionalBool(v types.Bool) *bool {
	if v.IsNull() || v.IsUnknown() {
		return nil
	}
	return v.ValueBoolPointer()
}

// optionalInt64 returns a *int64 for a set value, or nil when null/unknown.
func optionalInt64(v types.Int64) *int64 {
	if v.IsNull() || v.IsUnknown() {
		return nil
	}
	return v.ValueInt64Pointer()
}

// canonicalJSON reparses a JSON document into a stable form (object keys sorted,
// insignificant whitespace removed) so semantically-equal JSON compares equal in
// state and doesn't churn the plan. Returns an error if s isn't valid JSON.
func canonicalJSON(s string) (string, error) {
	var v any
	if err := json.Unmarshal([]byte(s), &v); err != nil {
		return "", err
	}
	b, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// mapToStringMap reads a types.Map into a Go map (nil when null/unknown).
func mapToStringMap(ctx context.Context, m types.Map) (map[string]string, diag.Diagnostics) {
	if m.IsNull() || m.IsUnknown() {
		return nil, nil
	}
	out := map[string]string{}
	d := m.ElementsAs(ctx, &out, false)
	return out, d
}
