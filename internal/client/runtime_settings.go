package client

import (
	"context"
	"net/http"
)

// RuntimeSettings is the body of GET/PATCH /admin/settings/system (1.11.0,
// #867): settings an administrator changes without a restart. A singleton.
type RuntimeSettings struct {
	// GuestAccessEnabled is the value in force right now.
	GuestAccessEnabled bool `json:"guest_access_enabled"`
	// GuestAccessSource is environment (AK_GUEST_ACCESS_ENABLED pins it),
	// database or default.
	GuestAccessSource string `json:"guest_access_source"`
	// GuestAccessStored is what this API stored, reported even while the
	// environment pins the value; it takes over once the variable is unset.
	GuestAccessStored *bool `json:"guest_access_stored"`
	// GuestAccessEditable is false while the environment pins the value.
	GuestAccessEditable bool `json:"guest_access_editable"`
	// HiddenNavItems are web UI navigation entries hidden for every user
	// (#4574), display only.
	HiddenNavItems []string `json:"hidden_nav_items"`
	// UpstreamJobsPaused is whether every scheduled upstream job is paused
	// right now (#5288).
	UpstreamJobsPaused bool `json:"upstream_jobs_paused"`
	// UpstreamJobsPausedSource is environment (UPSTREAM_JOBS_PAUSED=true pins
	// it on), database, default or unavailable.
	UpstreamJobsPausedSource string `json:"upstream_jobs_paused_source"`
	// UpstreamJobsPausedStored is what this API stored, if anything.
	UpstreamJobsPausedStored *bool `json:"upstream_jobs_paused_stored"`
}

// UpdateRuntimeSettingsRequest maps UpdateRuntimeSettingsRequest. The backend
// rejects unknown fields and leaves omitted ones unchanged.
type UpdateRuntimeSettingsRequest struct {
	GuestAccessEnabled *bool     `json:"guest_access_enabled,omitempty"`
	HiddenNavItems     *[]string `json:"hidden_nav_items,omitempty"`
	UpstreamJobsPaused *bool     `json:"upstream_jobs_paused,omitempty"`
}

// GetRuntimeSettings reads GET /admin/settings/system. Admin only.
func (c *Client) GetRuntimeSettings(ctx context.Context) (*RuntimeSettings, error) {
	var out RuntimeSettings
	if err := c.do(ctx, http.MethodGet, "/admin/settings/system", nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// UpdateRuntimeSettings writes PATCH /admin/settings/system. A write the
// environment pins is still stored, and the response says so.
func (c *Client) UpdateRuntimeSettings(ctx context.Context, req UpdateRuntimeSettingsRequest) (*RuntimeSettings, error) {
	var out RuntimeSettings
	if err := c.do(ctx, http.MethodPatch, "/admin/settings/system", req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
