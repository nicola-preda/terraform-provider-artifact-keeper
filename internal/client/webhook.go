package client

import (
	"context"
	"net/http"
	"net/url"
)

// Webhook mirrors the API's WebhookResponse. The raw signing secret is only
// ever present in the create response and is never returned by GET/LIST, so it
// is not a field here.
type Webhook struct {
	ID                   string            `json:"id"`
	Name                 string            `json:"name"`
	URL                  string            `json:"url"`
	Events               []string          `json:"events"`
	IsEnabled            bool              `json:"is_enabled"`
	RepositoryID         *string           `json:"repository_id"`
	Headers              map[string]string `json:"headers"`
	PayloadTemplate      string            `json:"payload_template"`
	EventSchemaVersion   string            `json:"event_schema_version"`
	SecretDigest         *string           `json:"secret_digest"`
	SecretRotationActive bool              `json:"secret_rotation_active"`
	LastTriggeredAt      *string           `json:"last_triggered_at"`
	CreatedAt            string            `json:"created_at"`
}

// CreateWebhookRequest maps the API's CreateWebhookRequest.
type CreateWebhookRequest struct {
	Name               string            `json:"name"`
	URL                string            `json:"url"`
	Events             []string          `json:"events"`
	Secret             *string           `json:"secret,omitempty"`
	RepositoryID       *string           `json:"repository_id,omitempty"`
	Headers            map[string]string `json:"headers,omitempty"`
	PayloadTemplate    *string           `json:"payload_template,omitempty"`
	EventSchemaVersion *string           `json:"event_schema_version,omitempty"`
}

// CreateWebhook creates a webhook. The create response flattens the webhook
// fields alongside a one-time raw `secret`; only the webhook fields are decoded
// here (the caller preserves the configured secret in state).
func (c *Client) CreateWebhook(ctx context.Context, req CreateWebhookRequest) (*Webhook, error) {
	var out Webhook
	if err := c.do(ctx, http.MethodPost, "/webhooks", req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) GetWebhook(ctx context.Context, id string) (*Webhook, error) {
	var out Webhook
	if err := c.do(ctx, http.MethodGet, "/webhooks/"+url.PathEscape(id), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) DeleteWebhook(ctx context.Context, id string) error {
	return c.do(ctx, http.MethodDelete, "/webhooks/"+url.PathEscape(id), nil, nil)
}

// SetWebhookEnabled toggles a webhook's enabled state via the dedicated
// enable/disable action endpoints (the only mutable aspect of a webhook).
func (c *Client) SetWebhookEnabled(ctx context.Context, id string, enabled bool) error {
	action := "disable"
	if enabled {
		action = "enable"
	}
	return c.do(ctx, http.MethodPost, "/webhooks/"+url.PathEscape(id)+"/"+action, nil, nil)
}
