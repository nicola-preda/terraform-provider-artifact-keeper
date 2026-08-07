package client

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

// CreateUserApiTokenRequest maps CreateUserApiTokenRequest on
// POST /users/{id}/tokens. Unlike the profile endpoint, `scopes` is a bare
// (non-defaulted) field server side, so it must always be sent.
type CreateUserApiTokenRequest struct {
	Name          string   `json:"name"`
	Scopes        []string `json:"scopes"`
	ExpiresInDays *int64   `json:"expires_in_days,omitempty"`
}

// CreateUserApiToken mints a token for another user (admin), or for yourself,
// via POST /users/{id}/tokens. The plaintext token is only returned here.
//
// Authorization: self-or-admin. Admin-class scopes (`*`, `admin`, `delete:*`,
// `write:users`) are refused for non-admin callers, and a scoped credential
// cannot mint a token exceeding its own scopes (the 1.7.0 mint ceiling).
func (c *Client) CreateUserApiToken(ctx context.Context, userID string, req CreateUserApiTokenRequest) (*ApiTokenCreated, error) {
	var out ApiTokenCreated
	if err := c.do(ctx, http.MethodPost, "/users/"+url.PathEscape(userID)+"/tokens", req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// ListUserApiTokens returns a user's token metadata (never the secret).
func (c *Client) ListUserApiTokens(ctx context.Context, userID string) ([]ApiToken, error) {
	var list apiTokenListResponse
	if err := c.do(ctx, http.MethodGet, "/users/"+url.PathEscape(userID)+"/tokens", nil, &list); err != nil {
		return nil, err
	}
	return list.Items, nil
}

// GetUserApiToken looks a user's token up by id. There's no GET-by-id endpoint,
// so it lists and filters, and (like GetApiToken) reports an expired-but-listed
// token as a 404 so Terraform recreates a credential that auth would reject.
func (c *Client) GetUserApiToken(ctx context.Context, userID, id string) (*ApiToken, error) {
	items, err := c.ListUserApiTokens(ctx, userID)
	if err != nil {
		return nil, err
	}
	for i := range items {
		if items[i].ID != id {
			continue
		}
		if t := items[i]; t.ExpiresAt != nil {
			if exp, err := time.Parse(time.RFC3339, *t.ExpiresAt); err == nil && exp.Before(time.Now()) {
				break
			}
		}
		return &items[i], nil
	}
	return nil, &APIError{StatusCode: http.StatusNotFound, Message: fmt.Sprintf("api token %s not found for user %s", id, userID)}
}

func (c *Client) DeleteUserApiToken(ctx context.Context, userID, id string) error {
	return c.do(ctx, http.MethodDelete, "/users/"+url.PathEscape(userID)+"/tokens/"+url.PathEscape(id), nil, nil)
}
