package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
)

// CiOidcMapping mirrors CiOidcMappingResponse: an identity mapping on a CI OIDC
// provider. On token exchange, mappings run in priority order (lower = higher);
// the first enabled mapping whose claim_filters all match the CI JWT wins.
type CiOidcMapping struct {
	ID         string `json:"id"`
	ProviderID string `json:"provider_id"`
	Name       string `json:"name"`
	Priority   int64  `json:"priority"`
	// ClaimFilters is a free-form JSON object: claim name -> string (exact match)
	// or array of strings (any-of). Passed through verbatim.
	ClaimFilters json.RawMessage `json:"claim_filters"`
	// AllowedRepoIDs is an optional repository restriction (nil = unrestricted).
	AllowedRepoIDs []string `json:"allowed_repo_ids"`
	IsEnabled      bool     `json:"is_enabled"`
	CreatedAt      string   `json:"created_at"`
	UpdatedAt      string   `json:"updated_at"`
}

// CreateCiOidcMappingRequest maps POST .../ci-oidc/{provider_id}/mappings.
// name and claim_filters required; omitted priority defaults to 100, is_enabled
// to true.
type CreateCiOidcMappingRequest struct {
	Name           string          `json:"name"`
	Priority       *int64          `json:"priority,omitempty"`
	ClaimFilters   json.RawMessage `json:"claim_filters"`
	AllowedRepoIDs []string        `json:"allowed_repo_ids,omitempty"`
	IsEnabled      *bool           `json:"is_enabled,omitempty"`
}

// UpdateCiOidcMappingRequest maps PUT .../ci-oidc/{provider_id}/mappings/{mapping_id}.
// Omitted fields keep their current value.
type UpdateCiOidcMappingRequest struct {
	Name           *string         `json:"name,omitempty"`
	Priority       *int64          `json:"priority,omitempty"`
	ClaimFilters   json.RawMessage `json:"claim_filters,omitempty"`
	AllowedRepoIDs []string        `json:"allowed_repo_ids,omitempty"`
	IsEnabled      *bool           `json:"is_enabled,omitempty"`
}

func (c *Client) CreateCiOidcMapping(ctx context.Context, providerID string, req CreateCiOidcMappingRequest) (*CiOidcMapping, error) {
	var out CiOidcMapping
	if err := c.do(ctx, http.MethodPost, "/admin/ci-oidc/"+url.PathEscape(providerID)+"/mappings", req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) GetCiOidcMapping(ctx context.Context, providerID, id string) (*CiOidcMapping, error) {
	var out CiOidcMapping
	if err := c.do(ctx, http.MethodGet, "/admin/ci-oidc/"+url.PathEscape(providerID)+"/mappings/"+url.PathEscape(id), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) UpdateCiOidcMapping(ctx context.Context, providerID, id string, req UpdateCiOidcMappingRequest) (*CiOidcMapping, error) {
	var out CiOidcMapping
	if err := c.do(ctx, http.MethodPut, "/admin/ci-oidc/"+url.PathEscape(providerID)+"/mappings/"+url.PathEscape(id), req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) DeleteCiOidcMapping(ctx context.Context, providerID, id string) error {
	return c.do(ctx, http.MethodDelete, "/admin/ci-oidc/"+url.PathEscape(providerID)+"/mappings/"+url.PathEscape(id), nil, nil)
}
