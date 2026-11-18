package provider

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/nicola-preda/terraform-provider-artifact-keeper/internal/client"
)

func decodeSettings(t *testing.T, body string) *client.SystemSettings {
	t.Helper()
	var s client.SystemSettings
	if err := json.Unmarshal([]byte(body), &s); err != nil {
		t.Fatal(err)
	}
	return &s
}

func encodeSettings(t *testing.T, s client.SystemSettings) string {
	t.Helper()
	b, err := json.Marshal(s)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// AK 1.12 dropped allow_anonymous_download from GET and answers 400 when a POST
// carries it, so a GET without it must give a body without it
func TestSystemSettingsRequestOmitsRemovedField(t *testing.T) {
	cur := decodeSettings(t, `{"storage_backend":"filesystem","storage_path":"/data","environment":"dev","max_upload_size_bytes":10737418240,"retention_days":365,"audit_retention_days":0,"backup_retention_count":10,"edge_stale_threshold_minutes":5}`)
	plan := systemSettingsResourceModel{AllowAnonymousDownload: types.BoolUnknown(), RetentionDays: types.Int64Value(30)}

	body := encodeSettings(t, systemSettingsRequestFromModel(plan, cur))
	if strings.Contains(body, "allow_anonymous_download") {
		t.Fatalf("body sends allow_anonymous_download: %s", body)
	}
	if !strings.Contains(body, `"retention_days":30`) || !strings.Contains(body, `"max_upload_size_bytes":10737418240`) {
		t.Fatalf("body lost a setting: %s", body)
	}
	if m := systemSettingsToModel(cur); !m.AllowAnonymousDownload.IsNull() {
		t.Fatalf("allow_anonymous_download = %s, want null", m.AllowAnonymousDownload)
	}
}

// Before 1.12 the field is in the GET and goes back unchanged, or as configured
func TestSystemSettingsRequestKeepsFieldOnOlderBackends(t *testing.T) {
	cur := decodeSettings(t, `{"storage_backend":"filesystem","storage_path":"/data","allow_anonymous_download":true,"max_upload_size_bytes":1,"retention_days":365,"audit_retention_days":0,"backup_retention_count":10,"edge_stale_threshold_minutes":5}`)

	body := encodeSettings(t, systemSettingsRequestFromModel(systemSettingsResourceModel{}, cur))
	if !strings.Contains(body, `"allow_anonymous_download":true`) {
		t.Fatalf("body dropped the current value: %s", body)
	}
	body = encodeSettings(t, systemSettingsRequestFromModel(systemSettingsResourceModel{AllowAnonymousDownload: types.BoolValue(false)}, cur))
	if !strings.Contains(body, `"allow_anonymous_download":false`) {
		t.Fatalf("body dropped the configured value: %s", body)
	}
	if m := systemSettingsToModel(cur); m.AllowAnonymousDownload.ValueBool() != true {
		t.Fatalf("allow_anonymous_download = %s, want true", m.AllowAnonymousDownload)
	}
}
