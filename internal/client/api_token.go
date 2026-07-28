package client

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

// ApiToken is the metadata for an access token (the secret is never returned
// after creation). Mirrors ApiTokenResponse.
type ApiToken struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	TokenPrefix string   `json:"token_prefix"`
	Scopes      []string `json:"scopes"`
	ExpiresAt   *string  `json:"expires_at"`
	LastUsedAt  *string  `json:"last_used_at"`
	CreatedAt   string   `json:"created_at"`
}

// CreateApiTokenRequest maps CreateAccessTokenRequest (profile endpoint).
type CreateApiTokenRequest struct {
	Name          string   `json:"name"`
	Scopes        []string `json:"scopes,omitempty"`
	ExpiresInDays *int64   `json:"expires_in_days,omitempty"`
}

// ApiTokenCreated is the create-only response that includes the plaintext token.
type ApiTokenCreated struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Token string `json:"token"`
}

type apiTokenListResponse struct {
	Items []ApiToken `json:"items"`
}

// CreateApiToken mints a token for the authenticated user via
// POST /profile/access-tokens. The plaintext token is only returned here.
func (c *Client) CreateApiToken(ctx context.Context, req CreateApiTokenRequest) (*ApiTokenCreated, error) {
	var out ApiTokenCreated
	if err := c.do(ctx, http.MethodPost, "/profile/access-tokens", req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// GetApiToken looks a token up by id. There's no GET-by-id endpoint, so we list
// and filter.
//
// The list only drops revoked tokens, not expired ones, so an expired token
// keeps showing up here even though auth rejects it with a 401. We report it as
// a 404 so Read sees it as gone and Terraform recreates it; otherwise a peer's
// api_key could quietly stop working with no drift shown. Tokens with no expiry
// (the default) are unaffected.
func (c *Client) GetApiToken(ctx context.Context, id string) (*ApiToken, error) {
	var list apiTokenListResponse
	if err := c.do(ctx, http.MethodGet, "/profile/access-tokens", nil, &list); err != nil {
		return nil, err
	}
	for i := range list.Items {
		if list.Items[i].ID != id {
			continue
		}
		// expired but still listed, treat as gone (see above)
		if t := list.Items[i]; t.ExpiresAt != nil {
			if exp, err := time.Parse(time.RFC3339, *t.ExpiresAt); err == nil && exp.Before(time.Now()) {
				break
			}
		}
		return &list.Items[i], nil
	}
	return nil, &APIError{StatusCode: http.StatusNotFound, Message: fmt.Sprintf("api token %s not found", id)}
}

func (c *Client) DeleteApiToken(ctx context.Context, id string) error {
	return c.do(ctx, http.MethodDelete, "/profile/access-tokens/"+url.PathEscape(id), nil, nil)
}
