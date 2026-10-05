package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/nicola-preda/terraform-provider-artifact-keeper/internal/client"
)

func TestKeepRedactedHeaders(t *testing.T) {
	ctx := context.Background()
	known := types.MapValueMust(types.StringType, map[string]attr.Value{
		"Authorization": types.StringValue("Bearer s3cret"),
		"X-Gone":        types.StringValue("v"),
	})
	wh := &client.Webhook{Headers: map[string]string{
		"Authorization": "***",
		"X-New":         "***",
		"X-Plain":       "visible",
	}}

	if d := keepRedactedHeaders(ctx, wh, known); d.HasError() {
		t.Fatal(d)
	}
	want := map[string]string{"Authorization": "Bearer s3cret", "X-New": "***", "X-Plain": "visible"}
	for k, v := range want {
		if wh.Headers[k] != v {
			t.Errorf("%s = %q, want %q", k, wh.Headers[k], v)
		}
	}
	if len(wh.Headers) != len(want) {
		t.Errorf("headers = %v, want only %v", wh.Headers, want)
	}
}
