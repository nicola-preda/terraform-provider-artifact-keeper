package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
)

// A 503/429 with Retry-After means the request was shed before running, so do
// retries it and eventually succeeds.
func TestDoRetriesOnShedThenSucceeds(t *testing.T) {
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if atomic.AddInt32(&calls, 1) < 3 {
			w.Header().Set("Retry-After", "0") // keep the test fast
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		_, _ = w.Write([]byte(`{"key":"docker-local"}`))
	}))
	defer srv.Close()

	c, err := New(Config{Endpoint: srv.URL, Token: "x"})
	if err != nil {
		t.Fatal(err)
	}
	repo, err := c.GetRepository(context.Background(), "docker-local")
	if err != nil {
		t.Fatalf("want success after retries, got %v", err)
	}
	if repo.Key != "docker-local" {
		t.Fatalf("unexpected repo: %+v", repo)
	}
	if got := atomic.LoadInt32(&calls); got != 3 {
		t.Fatalf("want 3 attempts, got %d", got)
	}
}

// A request shed on every attempt gives up after maxAttempts and surfaces the
// APIError rather than looping forever.
func TestDoGivesUpAfterMaxAttempts(t *testing.T) {
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		atomic.AddInt32(&calls, 1)
		w.Header().Set("Retry-After", "0")
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer srv.Close()

	c, err := New(Config{Endpoint: srv.URL, Token: "x"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c.GetRepository(context.Background(), "x"); err == nil {
		t.Fatal("want error after exhausting retries, got nil")
	}
	if got := atomic.LoadInt32(&calls); got != maxAttempts {
		t.Fatalf("want %d attempts, got %d", maxAttempts, got)
	}
}

// do splits a "?query" suffix off the path into the URL query, and
// ListGroupMembers pages through with member_limit/member_offset until it has
// members_total, concatenating the pages.
func TestListGroupMembersPaginates(t *testing.T) {
	var lastQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		lastQuery = r.URL.RawQuery
		offset := r.URL.Query().Get("member_offset")
		// Two pages: 200 members, then the 201st, total 201.
		if offset == "0" {
			w.Write([]byte(`{"members":[` + repeatMember(200) + `],"members_total":201}`))
			return
		}
		w.Write([]byte(`{"members":[{"user_id":"u200","username":"u200","joined_at":"t"}],"members_total":201}`))
	}))
	defer srv.Close()

	c, err := New(Config{Endpoint: srv.URL, Token: "x"})
	if err != nil {
		t.Fatal(err)
	}
	members, err := c.ListGroupMembers(context.Background(), "g1")
	if err != nil {
		t.Fatal(err)
	}
	if len(members) != 201 {
		t.Fatalf("want 201 members across pages, got %d", len(members))
	}
	if lastQuery == "" || lastQuery == "member_limit=200&member_offset=0" {
		t.Fatalf("second page query not sent, last query was %q", lastQuery)
	}
}

func repeatMember(n int) string {
	out := ""
	for i := 0; i < n; i++ {
		if i > 0 {
			out += ","
		}
		out += `{"user_id":"u","username":"u","joined_at":"t"}`
	}
	return out
}
