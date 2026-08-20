package client

import (
	"context"
	"net/http"
)

// TotpPolicy is the system-wide 2FA enforcement policy: the body of
// GET/PUT /admin/settings/totp-policy (1.8.0, #2805). A singleton.
//
// The response also carries enrollment counts and whether the calling admin has
// TOTP enabled; those are monitoring data that changes on its own, so they are
// deliberately not modelled.
type TotpPolicy struct {
	// Policy is disabled, required_for_admins or required_for_all.
	Policy string `json:"policy"`
	// Source is database (the stored setting is in force) or environment (the
	// TOTP_POLICY variable pins it, and writes are refused).
	Source string `json:"source"`
	// Editable is false when Source is environment.
	Editable bool `json:"editable"`
}

// UpdateTotpPolicyRequest maps UpdateTotpPolicyRequest.
type UpdateTotpPolicyRequest struct {
	Policy string `json:"policy"`
}

// GetTotpPolicy reads GET /admin/settings/totp-policy. Admin only.
func (c *Client) GetTotpPolicy(ctx context.Context) (*TotpPolicy, error) {
	var out TotpPolicy
	if err := c.do(ctx, http.MethodGet, "/admin/settings/totp-policy", nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// UpdateTotpPolicy writes PUT /admin/settings/totp-policy and returns the policy
// now in force. 409s when TOTP_POLICY pins the policy, and when tightening it
// while the calling admin has not enrolled in TOTP themselves (the lockout
// guard). Relaxing is never refused.
func (c *Client) UpdateTotpPolicy(ctx context.Context, req UpdateTotpPolicyRequest) (*TotpPolicy, error) {
	var out TotpPolicy
	if err := c.do(ctx, http.MethodPut, "/admin/settings/totp-policy", req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
