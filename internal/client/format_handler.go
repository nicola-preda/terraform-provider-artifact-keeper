package client

import (
	"context"
	"net/http"
	"net/url"
)

// FormatHandler mirrors FormatHandlerResponse (GET /formats/{key}). Only
// is_enabled is manageable; the rest is server-owned.
type FormatHandler struct {
	ID              string  `json:"id"`
	FormatKey       string  `json:"format_key"`
	DisplayName     string  `json:"display_name"`
	HandlerType     string  `json:"handler_type"`
	Description     *string `json:"description"`
	IsEnabled       bool    `json:"is_enabled"`
	Priority        int64   `json:"priority"`
	RepositoryCount *int64  `json:"repository_count"`
}

func (c *Client) GetFormatHandler(ctx context.Context, formatKey string) (*FormatHandler, error) {
	var out FormatHandler
	if err := c.do(ctx, http.MethodGet, "/formats/"+url.PathEscape(formatKey), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// SetFormatHandlerEnabled toggles a handler via POST
// /formats/{key}/{enable,disable}; returns the updated handler.
func (c *Client) SetFormatHandlerEnabled(ctx context.Context, formatKey string, enabled bool) (*FormatHandler, error) {
	action := "disable"
	if enabled {
		action = "enable"
	}
	var out FormatHandler
	if err := c.do(ctx, http.MethodPost, "/formats/"+url.PathEscape(formatKey)+"/"+action, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
