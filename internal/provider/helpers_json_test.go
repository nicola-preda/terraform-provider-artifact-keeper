package provider

import (
	"encoding/json"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestJSONKeepPrior(t *testing.T) {
	server := json.RawMessage(`{"conda":{"require_attestation":true,"allowed_channels":[]},"origin":{"allowed_kinds":[],"denied_upstreams":[]}}`)
	for _, tc := range []struct {
		prior string
		keep  bool
	}{
		{`{"conda":{"require_attestation":true}}`, true},
		{`{"conda":{"require_attestation":false}}`, false},
		{`{"origin":{"allowed_kinds":["hosted"]}}`, false},
		{`{}`, false},
	} {
		got := jsonKeepPrior(types.StringValue(tc.prior), server)
		if (got.ValueString() == tc.prior) != tc.keep {
			t.Errorf("prior %s: kept=%v, want %v", tc.prior, got.ValueString() == tc.prior, tc.keep)
		}
	}
	if !jsonKeepPrior(types.StringNull(), json.RawMessage("null")).IsNull() {
		t.Error("null predicates must read back null")
	}
}
