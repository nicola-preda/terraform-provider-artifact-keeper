package client

import (
	"context"
	"net/http"
)

// ApiTokenExpiryPolicy is the mint-time expiration policy for API tokens: the
// `policy` object of GET/PUT /admin/settings/token-policy (1.10.0, #3460).
//
// The backend declares the struct `deny_unknown_fields` and gives only
// default_days and apply_to_service_accounts a serde default, so every key is
// sent on every write and none may be renamed.
type ApiTokenExpiryPolicy struct {
	// RequireExpiration false leaves the policy entirely inert.
	RequireExpiration bool `json:"require_expiration"`
	// MinDays and MaxDays bound an accepted expires_in_days, inclusive.
	MinDays int64 `json:"min_days"`
	MaxDays int64 `json:"max_days"`
	// DefaultDays is applied when an enforced mint omits expires_in_days;
	// null rejects such a mint instead.
	DefaultDays *int64 `json:"default_days"`
	// ApplyToServiceAccounts subjects CI credentials to the policy too.
	ApplyToServiceAccounts bool `json:"apply_to_service_accounts"`
}

// TokenPolicy is the body of GET/PUT /admin/settings/token-policy. A singleton.
//
// The response also counts live never-expiring tokens; those move on their own
// (every mint and revoke changes them), so they are deliberately not modelled,
// as with the TOTP policy's enrollment counts.
type TokenPolicy struct {
	Policy ApiTokenExpiryPolicy `json:"policy"`
	// Source is database (the stored setting is in force) or environment (the
	// API_TOKEN_EXPIRATION_* variables pin it, and writes are refused).
	Source string `json:"source"`
	// Editable is false when Source is environment.
	Editable bool `json:"editable"`
}

// UpdateTokenPolicyRequest maps UpdateTokenPolicyRequest.
type UpdateTokenPolicyRequest struct {
	Policy ApiTokenExpiryPolicy `json:"policy"`
}

// GetTokenPolicy reads GET /admin/settings/token-policy. Admin only.
func (c *Client) GetTokenPolicy(ctx context.Context) (*TokenPolicy, error) {
	var out TokenPolicy
	if err := c.do(ctx, http.MethodGet, "/admin/settings/token-policy", nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// UpdateTokenPolicy writes PUT /admin/settings/token-policy and returns the
// policy now in force. 409s when API_TOKEN_EXPIRATION_REQUIRED pins it, and
// 400s when default_days falls outside [min_days, max_days].
func (c *Client) UpdateTokenPolicy(ctx context.Context, req UpdateTokenPolicyRequest) (*TokenPolicy, error) {
	var out TokenPolicy
	if err := c.do(ctx, http.MethodPut, "/admin/settings/token-policy", req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
