package client

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
)

// RepoToken mirrors RepoTokenResponse (GET
// /repositories/{key}/tokens/{token_id}). The plaintext token is never returned
// here, only once at creation.
type RepoToken struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	TokenPrefix string   `json:"token_prefix"`
	Scopes      []string `json:"scopes"`
	ExpiresAt   *string  `json:"expires_at"`
	LastUsedAt  *string  `json:"last_used_at"`
	CreatedAt   string   `json:"created_at"`
	IsExpired   bool     `json:"is_expired"`
	IsRevoked   bool     `json:"is_revoked"`
	Description *string  `json:"description"`
	CreatedBy   *string  `json:"created_by"`
}

// CreateRepoTokenRequest maps the API's CreateRepoTokenRequest. scopes is
// required by the API.
type CreateRepoTokenRequest struct {
	Name          string   `json:"name"`
	Scopes        []string `json:"scopes"`
	ExpiresInDays *int64   `json:"expires_in_days,omitempty"`
	Description   *string  `json:"description,omitempty"`
}

// RepoTokenCreated is the create-only response that carries the plaintext token.
type RepoTokenCreated struct {
	ID            string `json:"id"`
	Token         string `json:"token"`
	Name          string `json:"name"`
	RepositoryKey string `json:"repository_key"`
}

// CreateRepoToken mints an access token scoped to repository `key` via
// POST /repositories/{key}/tokens. The plaintext token is only returned here.
func (c *Client) CreateRepoToken(ctx context.Context, key string, req CreateRepoTokenRequest) (*RepoTokenCreated, error) {
	var out RepoTokenCreated
	if err := c.do(ctx, http.MethodPost, "/repositories/"+url.PathEscape(key)+"/tokens", req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// GetRepoToken fetches a token by (repository key, token id). This endpoint keeps
// returning revoked/expired tokens; auth rejects them, so we report those as 404
// (like api_token) so Terraform recreates rather than keeping a dead token in state.
func (c *Client) GetRepoToken(ctx context.Context, key, id string) (*RepoToken, error) {
	var out RepoToken
	if err := c.do(ctx, http.MethodGet, "/repositories/"+url.PathEscape(key)+"/tokens/"+url.PathEscape(id), nil, &out); err != nil {
		return nil, err
	}
	if out.IsRevoked || out.IsExpired {
		return nil, &APIError{StatusCode: http.StatusNotFound, Message: fmt.Sprintf("repo token %s is revoked or expired", id)}
	}
	return &out, nil
}

// DeleteRepoToken revokes (soft-deletes) a repository-scoped token.
func (c *Client) DeleteRepoToken(ctx context.Context, key, id string) error {
	return c.do(ctx, http.MethodDelete, "/repositories/"+url.PathEscape(key)+"/tokens/"+url.PathEscape(id), nil, nil)
}
