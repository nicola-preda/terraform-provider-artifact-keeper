package client

import (
	"context"
	"net/http"
	"net/url"
)

// Banner mirrors AdminBanner (GET /admin/banners/{id}, 1.11.0): a system-wide
// maintenance or downtime notice.
type Banner struct {
	ID        string  `json:"id"`
	Title     string  `json:"title"`
	Message   string  `json:"message"`
	Severity  string  `json:"severity"`
	Target    string  `json:"target"`
	LinkURL   *string `json:"link_url"`
	StartsAt  *string `json:"starts_at"`
	EndsAt    *string `json:"ends_at"`
	Enabled   bool    `json:"enabled"`
	Active    bool    `json:"active"`
	CreatedAt string  `json:"created_at"`
	UpdatedAt string  `json:"updated_at"`
}

// BannerInput is the body of POST /admin/banners and PUT /admin/banners/{id}
// (a full replace).
type BannerInput struct {
	Title    string  `json:"title"`
	Message  string  `json:"message"`
	Severity string  `json:"severity,omitempty"`
	Target   string  `json:"target,omitempty"`
	LinkURL  *string `json:"link_url,omitempty"`
	StartsAt *string `json:"starts_at,omitempty"`
	EndsAt   *string `json:"ends_at,omitempty"`
	Enabled  *bool   `json:"enabled,omitempty"`
}

func (c *Client) CreateBanner(ctx context.Context, req BannerInput) (*Banner, error) {
	var out Banner
	if err := c.do(ctx, http.MethodPost, "/admin/banners", req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) GetBanner(ctx context.Context, id string) (*Banner, error) {
	var out Banner
	if err := c.do(ctx, http.MethodGet, "/admin/banners/"+url.PathEscape(id), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) UpdateBanner(ctx context.Context, id string, req BannerInput) (*Banner, error) {
	var out Banner
	if err := c.do(ctx, http.MethodPut, "/admin/banners/"+url.PathEscape(id), req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) DeleteBanner(ctx context.Context, id string) error {
	return c.do(ctx, http.MethodDelete, "/admin/banners/"+url.PathEscape(id), nil, nil)
}
