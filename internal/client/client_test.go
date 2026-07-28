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
