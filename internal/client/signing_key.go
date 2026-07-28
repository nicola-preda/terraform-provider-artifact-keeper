package client

import (
	"context"
	"net/http"
	"net/url"
)

// SigningKey mirrors the API's SigningKeyPublic (the private key material is
// never serialized). Returned by POST and GET /signing/keys.
type SigningKey struct {
	ID           string  `json:"id"`
	RepositoryID *string `json:"repository_id"`
	Name         string  `json:"name"`
	KeyType      string  `json:"key_type"`
	Fingerprint  *string `json:"fingerprint"`
	KeyID        *string `json:"key_id"`
	PublicKeyPEM string  `json:"public_key_pem"`
	Algorithm    string  `json:"algorithm"`
	UIDName      *string `json:"uid_name"`
	UIDEmail     *string `json:"uid_email"`
	ExpiresAt    *string `json:"expires_at"`
	IsActive     bool    `json:"is_active"`
	CreatedAt    string  `json:"created_at"`
	LastUsedAt   *string `json:"last_used_at"`
}

// CreateSigningKeyRequest maps the API's CreateKeyPayload. The server defaults
// key_type to "rsa" and algorithm to "rsa4096" when they are omitted; omitting
// repository_id creates a global (instance-wide) key.
type CreateSigningKeyRequest struct {
	RepositoryID *string `json:"repository_id,omitempty"`
	Name         string  `json:"name"`
	KeyType      *string `json:"key_type,omitempty"`
	Algorithm    *string `json:"algorithm,omitempty"`
	UIDName      *string `json:"uid_name,omitempty"`
	UIDEmail     *string `json:"uid_email,omitempty"`
}

// CreateSigningKey generates a new signing key via POST /signing/keys. The
// response is the full public view of the key (no private material).
func (c *Client) CreateSigningKey(ctx context.Context, req CreateSigningKeyRequest) (*SigningKey, error) {
	var out SigningKey
	if err := c.do(ctx, http.MethodPost, "/signing/keys", req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) GetSigningKey(ctx context.Context, id string) (*SigningKey, error) {
	var out SigningKey
	if err := c.do(ctx, http.MethodGet, "/signing/keys/"+url.PathEscape(id), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// DeleteSigningKey removes a signing key. There is no update endpoint, so any
// change to a key forces replacement.
func (c *Client) DeleteSigningKey(ctx context.Context, id string) error {
	return c.do(ctx, http.MethodDelete, "/signing/keys/"+url.PathEscape(id), nil, nil)
}
