package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
)

// Plugin mirrors PluginResponse (GET /plugins/{id}). Everything except the
// enabled state (from status) and config is server-owned. The install source
// (git url/ref) and format_key aren't returned on read, only at install time.
type Plugin struct {
	ID           string          `json:"id"`
	Name         string          `json:"name"`
	Version      string          `json:"version"`
	DisplayName  string          `json:"display_name"`
	Description  *string         `json:"description"`
	Author       *string         `json:"author"`
	Homepage     *string         `json:"homepage"`
	Status       string          `json:"status"`
	PluginType   string          `json:"plugin_type"`
	ConfigSchema json.RawMessage `json:"config_schema"`
	InstalledAt  string          `json:"installed_at"`
	EnabledAt    *string         `json:"enabled_at"`
}

// PluginInstall mirrors PluginInstallResponse (POST /plugins/install/git).
type PluginInstall struct {
	PluginID  string `json:"plugin_id"`
	Name      string `json:"name"`
	Version   string `json:"version"`
	FormatKey string `json:"format_key"`
	Message   string `json:"message"`
}

type installFromGitRequest struct {
	URL string  `json:"url"`
	Ref *string `json:"ref,omitempty"`
}

type pluginConfigResponse struct {
	Config json.RawMessage `json:"config"`
}

type updatePluginConfigRequest struct {
	Config json.RawMessage `json:"config"`
}

// InstallPluginFromGit installs a WASM plugin from a git repository.
func (c *Client) InstallPluginFromGit(ctx context.Context, gitURL string, ref *string) (*PluginInstall, error) {
	var out PluginInstall
	if err := c.do(ctx, http.MethodPost, "/plugins/install/git", installFromGitRequest{URL: gitURL, Ref: ref}, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) GetPlugin(ctx context.Context, id string) (*Plugin, error) {
	var out Plugin
	if err := c.do(ctx, http.MethodGet, "/plugins/"+url.PathEscape(id), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// SetPluginEnabled toggles a plugin via POST /plugins/{id}/{enable,disable}.
func (c *Client) SetPluginEnabled(ctx context.Context, id string, enabled bool) error {
	action := "disable"
	if enabled {
		action = "enable"
	}
	return c.do(ctx, http.MethodPost, "/plugins/"+url.PathEscape(id)+"/"+action, nil, nil)
}

// GetPluginConfig returns the plugin's stored config as raw JSON.
func (c *Client) GetPluginConfig(ctx context.Context, id string) (json.RawMessage, error) {
	var out pluginConfigResponse
	if err := c.do(ctx, http.MethodGet, "/plugins/"+url.PathEscape(id)+"/config", nil, &out); err != nil {
		return nil, err
	}
	return out.Config, nil
}

// UpdatePluginConfig replaces the plugin config (POST /plugins/{id}/config).
func (c *Client) UpdatePluginConfig(ctx context.Context, id string, config json.RawMessage) error {
	return c.do(ctx, http.MethodPost, "/plugins/"+url.PathEscape(id)+"/config", updatePluginConfigRequest{Config: config}, nil)
}

func (c *Client) UninstallPlugin(ctx context.Context, id string) error {
	return c.do(ctx, http.MethodDelete, "/plugins/"+url.PathEscape(id), nil, nil)
}
