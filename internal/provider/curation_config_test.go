package provider

import "testing"

// curationConfigJSON validates a JSON object, canonicalizes it (sorted keys,
// whitespace stripped), treats "" as {}, and rejects non-objects / malformed JSON.
func TestCurationConfigJSON(t *testing.T) {
	if got, err := curationConfigJSON(`{ "b": 1, "a": 2 }`); err != nil || string(got) != `{"a":2,"b":1}` {
		t.Fatalf("canonicalize: got %q err %v", got, err)
	}
	if got, err := curationConfigJSON(""); err != nil || string(got) != "{}" {
		t.Fatalf("empty: got %q err %v", got, err)
	}
	if _, err := curationConfigJSON(`[1,2]`); err == nil {
		t.Fatal("want error for a JSON array (not an object)")
	}
	if _, err := curationConfigJSON(`{nope}`); err == nil {
		t.Fatal("want error for malformed JSON")
	}
}
