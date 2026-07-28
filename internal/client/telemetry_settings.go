package client

import (
	"context"
	"net/http"
)

// TelemetrySettings mirrors the /admin/telemetry/settings singleton (GET returns
// defaults if unset, POST upserts). All fields are mandatory in the POST body,
// so change a subset by reading first and merging.
type TelemetrySettings struct {
	Enabled          bool   `json:"enabled"`
	ReviewBeforeSend bool   `json:"review_before_send"`
	ScrubLevel       string `json:"scrub_level"`
	IncludeLogs      bool   `json:"include_logs"`
}

func (c *Client) GetTelemetrySettings(ctx context.Context) (*TelemetrySettings, error) {
	var out TelemetrySettings
	if err := c.do(ctx, http.MethodGet, "/admin/telemetry/settings", nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) UpdateTelemetrySettings(ctx context.Context, req TelemetrySettings) (*TelemetrySettings, error) {
	var out TelemetrySettings
	if err := c.do(ctx, http.MethodPost, "/admin/telemetry/settings", req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
