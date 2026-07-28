package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

// ServiceAccountToken mirrors TokenInfoResponse, an item of GET
// /service-accounts/{id}/tokens. Plaintext token is returned only on create,
// never here.
type ServiceAccountToken struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	TokenPrefix string   `json:"token_prefix"`
	Scopes      []string `json:"scopes"`
	ExpiresAt   *string  `json:"expires_at"`
	LastUsedAt  *string  `json:"last_used_at"`
	CreatedAt   string   `json:"created_at"`
	IsExpired   bool     `json:"is_expired"`
	// RepoSelector restricts the token to dynamically matched repos (free-form
	// JSON, nil when unset). Mutually exclusive with RepositoryIDs.
	RepoSelector json.RawMessage `json:"repo_selector"`
	// RepositoryIDs is the explicit repository restriction (empty when unset).
	RepositoryIDs []string `json:"repository_ids"`
}

// CreateServiceAccountTokenRequest maps CreateTokenRequest. name and scopes
// required; repo_selector and repository_ids are mutually exclusive.
type CreateServiceAccountTokenRequest struct {
	Name          string          `json:"name"`
	Scopes        []string        `json:"scopes"`
	ExpiresInDays *int64          `json:"expires_in_days,omitempty"`
	Description   *string         `json:"description,omitempty"`
	RepositoryIDs []string        `json:"repository_ids,omitempty"`
	RepoSelector  json.RawMessage `json:"repo_selector,omitempty"`
}

// ServiceAccountTokenCreated is the create-only response carrying the plaintext
// token (CreateTokenResponse).
type ServiceAccountTokenCreated struct {
	ID    string `json:"id"`
	Token string `json:"token"`
	Name  string `json:"name"`
}

type serviceAccountTokenListResponse struct {
	Items []ServiceAccountToken `json:"items"`
}

// CreateServiceAccountToken mints a token via POST /service-accounts/{id}/tokens.
// Plaintext token is returned only here.
func (c *Client) CreateServiceAccountToken(ctx context.Context, accountID string, req CreateServiceAccountTokenRequest) (*ServiceAccountTokenCreated, error) {
	var out ServiceAccountTokenCreated
	if err := c.do(ctx, http.MethodPost, "/service-accounts/"+url.PathEscape(accountID)+"/tokens", req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// GetServiceAccountToken looks up by (account id, token id): no GET-by-id, so
// list and filter. The list keeps expired tokens (drops revoked); we report
// expired as 404 so Terraform recreates instead of keeping a dead token.
func (c *Client) GetServiceAccountToken(ctx context.Context, accountID, id string) (*ServiceAccountToken, error) {
	var list serviceAccountTokenListResponse
	if err := c.do(ctx, http.MethodGet, "/service-accounts/"+url.PathEscape(accountID)+"/tokens", nil, &list); err != nil {
		return nil, err
	}
	for i := range list.Items {
		if list.Items[i].ID != id {
			continue
		}
		if list.Items[i].IsExpired {
			break // expired but still listed; treat as gone
		}
		return &list.Items[i], nil
	}
	return nil, &APIError{StatusCode: http.StatusNotFound, Message: fmt.Sprintf("service account token %s not found", id)}
}

// DeleteServiceAccountToken revokes a service-account token.
func (c *Client) DeleteServiceAccountToken(ctx context.Context, accountID, id string) error {
	return c.do(ctx, http.MethodDelete, "/service-accounts/"+url.PathEscape(accountID)+"/tokens/"+url.PathEscape(id), nil, nil)
}
