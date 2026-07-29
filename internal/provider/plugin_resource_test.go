package provider

import "testing"

func TestPluginJSONEqual(t *testing.T) {
	cases := []struct {
		name string
		a, b string
		want bool
	}{
		{"identical", `{"a":1}`, `{"a":1}`, true},
		{"key order", `{"a":1,"b":2}`, `{"b":2,"a":1}`, true},
		{"whitespace", `{ "a": 1 }`, `{"a":1}`, true},
		{"jsonencode int vs float", `{"n":5}`, `{"n":5.0}`, true},
		{"different value", `{"a":1}`, `{"a":2}`, false},
		{"invalid falls back to string equality", `{oops`, `{oops`, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := pluginJSONEqual(c.a, c.b); got != c.want {
				t.Fatalf("pluginJSONEqual(%q,%q)=%v want %v", c.a, c.b, got, c.want)
			}
		})
	}
}

func TestNormalizePluginJSON(t *testing.T) {
	got, err := normalizePluginJSON(`{ "b": 2, "a": 1 }`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Canonical form: sorted keys, no insignificant whitespace — matches
	// Terraform's jsonencode output so config never shows a spurious diff.
	if want := `{"a":1,"b":2}`; got != want {
		t.Fatalf("normalizePluginJSON = %q, want %q", got, want)
	}
	if _, err := normalizePluginJSON(`not json`); err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestPluginStatusEnabled(t *testing.T) {
	if !pluginStatusEnabled("active") {
		t.Fatal(`"active" should be enabled`)
	}
	for _, s := range []string{"disabled", "error", ""} {
		if pluginStatusEnabled(s) {
			t.Fatalf("%q should not be enabled", s)
		}
	}
}
