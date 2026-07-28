package client

import (
	"context"
	"net/http"
	"net/url"
)

// RepositorySigningConfig is the per-repository signing singleton at
// /signing/repositories/{repo_id}/config (GET/POST only, no delete). One struct
// decodes both the GET and POST shapes; fields unique to either are ignored.
type RepositorySigningConfig struct {
	RepositoryID      string  `json:"repository_id"`
	SigningKeyID      *string `json:"signing_key_id"`
	SignMetadata      bool    `json:"sign_metadata"`
	SignPackages      bool    `json:"sign_packages"`
	RequireSignatures bool    `json:"require_signatures"`
}

// UpdateSigningConfigRequest maps UpdateSigningConfigPayload. Every field
// optional. POST upserts/merges: an omitted field keeps its current value, and
// an omitted signing_key_id can't clear an already-set key.
type UpdateSigningConfigRequest struct {
	SigningKeyID      *string `json:"signing_key_id,omitempty"`
	SignMetadata      *bool   `json:"sign_metadata,omitempty"`
	SignPackages      *bool   `json:"sign_packages,omitempty"`
	RequireSignatures *bool   `json:"require_signatures,omitempty"`
}

// GetRepositorySigningConfig reads via
// GET /signing/repositories/{repository_id}/config.
func (c *Client) GetRepositorySigningConfig(ctx context.Context, repositoryID string) (*RepositorySigningConfig, error) {
	var out RepositorySigningConfig
	if err := c.do(ctx, http.MethodGet, "/signing/repositories/"+url.PathEscape(repositoryID)+"/config", nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// UpdateRepositorySigningConfig upserts via
// POST /signing/repositories/{repository_id}/config. Admin only.
func (c *Client) UpdateRepositorySigningConfig(ctx context.Context, repositoryID string, req UpdateSigningConfigRequest) (*RepositorySigningConfig, error) {
	var out RepositorySigningConfig
	if err := c.do(ctx, http.MethodPost, "/signing/repositories/"+url.PathEscape(repositoryID)+"/config", req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
