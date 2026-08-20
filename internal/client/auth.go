package client

import (
	"context"
	"errors"
	"net/http"
)

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type loginResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	// Set by 1.8.0 when the account owes a TOTP challenge or, under the
	// system-wide policy, an enrollment. Either way the response is a 200 with
	// an empty access_token and a short-lived ticket the provider can't use.
	TotpRequired           bool `json:"totp_required"`
	TotpEnrollmentRequired bool `json:"totp_enrollment_required"`
}

// Login exchanges username/password for a bearer token via
// POST /api/v1/auth/login and stores it on the client.
func (c *Client) Login(ctx context.Context, username, password string) error {
	var out loginResponse
	if err := c.do(ctx, http.MethodPost, "/auth/login", loginRequest{Username: username, Password: password}, &out); err != nil {
		return err
	}
	if out.AccessToken == "" {
		if out.TotpRequired || out.TotpEnrollmentRequired {
			return errors.New("this account requires 2FA, which username/password login cannot complete; configure the provider with an API token instead")
		}
		return errors.New("login succeeded but no access_token was returned")
	}
	c.token = out.AccessToken
	return nil
}
