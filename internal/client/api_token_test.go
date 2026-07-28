package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// GetApiToken must treat an expired-but-still-listed token as gone (the list
// endpoint filters only on revoked_at), while returning live ones.
func TestGetApiTokenSkipsExpired(t *testing.T) {
	past := time.Now().Add(-time.Hour).UTC().Format(time.RFC3339)
	future := time.Now().Add(time.Hour).UTC().Format(time.RFC3339)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"items":[
			{"id":"expired","expires_at":"` + past + `"},
			{"id":"live","expires_at":"` + future + `"},
			{"id":"forever","expires_at":null}
		]}`))
	}))
	defer srv.Close()

	c, err := New(Config{Endpoint: srv.URL, Token: "x"})
	if err != nil {
		t.Fatal(err)
	}

	if _, err := c.GetApiToken(context.Background(), "expired"); !IsNotFound(err) {
		t.Fatalf("expired token: want 404, got %v", err)
	}
	for _, id := range []string{"live", "forever"} {
		if tok, err := c.GetApiToken(context.Background(), id); err != nil || tok.ID != id {
			t.Fatalf("%s token: want present, got tok=%v err=%v", id, tok, err)
		}
	}
}
