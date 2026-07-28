// Package client is an HTTP client for the Artifact Keeper REST API
// (https://<host>/api/v1). It has no dependency on the Terraform framework, so
// resources deal only in Go types.
package client

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// Config holds the inputs needed to construct a Client.
type Config struct {
	Endpoint           string
	Token              string
	InsecureSkipVerify bool
	UserAgent          string
	HTTPClient         *http.Client
}

// Client talks to a single Artifact Keeper instance.
type Client struct {
	baseURL    *url.URL
	token      string
	userAgent  string
	httpClient *http.Client
}

// New builds a Client. The endpoint is normalized to end with /api/v1.
func New(cfg Config) (*Client, error) {
	base, err := normalizeBaseURL(cfg.Endpoint)
	if err != nil {
		return nil, err
	}

	hc := cfg.HTTPClient
	if hc == nil {
		transport := http.DefaultTransport.(*http.Transport).Clone()
		if cfg.InsecureSkipVerify {
			transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true} //nolint:gosec // opt-in for self-signed instances
		}
		hc = &http.Client{Timeout: 60 * time.Second, Transport: transport}
	}

	return &Client{baseURL: base, token: cfg.Token, userAgent: cfg.UserAgent, httpClient: hc}, nil
}

func normalizeBaseURL(raw string) (*url.URL, error) {
	if raw == "" {
		return nil, errors.New("endpoint must not be empty")
	}
	if !strings.Contains(raw, "://") {
		raw = "https://" + raw
	}
	u, err := url.Parse(raw)
	if err != nil {
		return nil, fmt.Errorf("invalid endpoint %q: %w", raw, err)
	}
	u.Path = strings.TrimRight(u.Path, "/")
	if !strings.HasSuffix(u.Path, "/api/v1") {
		u.Path += "/api/v1"
	}
	return u, nil
}

// APIError represents a non-2xx response from the API.
type APIError struct {
	StatusCode int
	Message    string
	Body       string
}

func (e *APIError) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("artifact-keeper API returned status %d: %s", e.StatusCode, e.Message)
	}
	return fmt.Sprintf("artifact-keeper API returned status %d: %s", e.StatusCode, e.Body)
}

// IsNotFound reports whether err is a 404 APIError.
func IsNotFound(err error) bool {
	var ae *APIError
	if errors.As(err, &ae) {
		return ae.StatusCode == http.StatusNotFound
	}
	return false
}

// maxAttempts bounds how many times do retries a shed/rate-limited or
// transiently-failed request (the initial try plus retries).
const maxAttempts = 4

// do performs a JSON request. body and out may be nil. A non-2xx status returns
// an *APIError.
//
// The server sheds load with 503 + Retry-After and rate-limits with 429 (both
// reject the request before acting on it), so do retries those, and transient
// transport errors, with a bounded backoff that honors Retry-After. Retrying
// is safe for every method here because a 429/503 means the request never ran.
func (c *Client) do(ctx context.Context, method, path string, body, out any) error {
	var bodyBytes []byte
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("encoding request body: %w", err)
		}
		bodyBytes = b
	}

	u := *c.baseURL
	u.Path += path

	var lastErr error
	var wait time.Duration
	for attempt := 0; attempt < maxAttempts; attempt++ {
		if attempt > 0 {
			if err := sleepCtx(ctx, wait); err != nil {
				return err
			}
		}

		var reader io.Reader
		if bodyBytes != nil {
			reader = bytes.NewReader(bodyBytes)
		}
		req, err := http.NewRequestWithContext(ctx, method, u.String(), reader)
		if err != nil {
			return err
		}
		if bodyBytes != nil {
			req.Header.Set("Content-Type", "application/json")
		}
		req.Header.Set("Accept", "application/json")
		if c.userAgent != "" {
			req.Header.Set("User-Agent", c.userAgent)
		}
		if c.token != "" {
			req.Header.Set("Authorization", "Bearer "+c.token)
		}

		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = err
			wait = backoffFor(attempt)
			continue
		}

		data, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode == http.StatusServiceUnavailable {
			lastErr = &APIError{StatusCode: resp.StatusCode, Message: extractMessage(data), Body: string(data)}
			wait = retryDelay(resp.Header.Get("Retry-After"), attempt)
			continue
		}
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return &APIError{StatusCode: resp.StatusCode, Message: extractMessage(data), Body: string(data)}
		}
		if out != nil && len(data) > 0 {
			if err := json.Unmarshal(data, out); err != nil {
				return fmt.Errorf("decoding response: %w", err)
			}
		}
		return nil
	}
	return lastErr
}

// backoffFor returns an exponential backoff (0.5s, 1s, 2s, …) capped at 30s.
func backoffFor(attempt int) time.Duration {
	d := (500 * time.Millisecond) << attempt
	if d > 30*time.Second {
		d = 30 * time.Second
	}
	return d
}

// retryDelay honors a Retry-After header (delta-seconds or HTTP-date, capped at
// 60s), falling back to exponential backoff when it's absent or unparseable.
func retryDelay(retryAfter string, attempt int) time.Duration {
	retryAfter = strings.TrimSpace(retryAfter)
	if retryAfter != "" {
		if secs, err := strconv.Atoi(retryAfter); err == nil && secs >= 0 {
			return capDelay(time.Duration(secs) * time.Second)
		}
		if t, err := http.ParseTime(retryAfter); err == nil {
			return capDelay(time.Until(t))
		}
	}
	return backoffFor(attempt)
}

func capDelay(d time.Duration) time.Duration {
	if d < 0 {
		return 0
	}
	if d > 60*time.Second {
		return 60 * time.Second
	}
	return d
}

// sleepCtx waits for d or until ctx is cancelled, whichever comes first.
func sleepCtx(ctx context.Context, d time.Duration) error {
	if d <= 0 {
		return nil
	}
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}

// extractMessage pulls a human-readable message out of an error body, tolerating
// either {"error": ...} or {"message": ...} shapes.
func extractMessage(data []byte) string {
	var e struct {
		Error   string `json:"error"`
		Message string `json:"message"`
	}
	_ = json.Unmarshal(data, &e)
	if e.Message != "" {
		return e.Message
	}
	return e.Error
}
