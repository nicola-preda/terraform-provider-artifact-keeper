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
}

// Login exchanges username/password for a bearer token via
// POST /api/v1/auth/login and stores it on the client.
func (c *Client) Login(ctx context.Context, username, password string) error {
	var out loginResponse
	if err := c.do(ctx, http.MethodPost, "/auth/login", loginRequest{Username: username, Password: password}, &out); err != nil {
		return err
	}
	if out.AccessToken == "" {
		return errors.New("login succeeded but no access_token was returned")
	}
	c.token = out.AccessToken
	return nil
}
