package client

import (
	"context"
	"net/http"
)

// SystemSettings mirrors the /admin/settings singleton (always exists; no
// create/delete). storage_backend/storage_path/environment are read-only, but
// storage_backend and storage_path must still be sent on update (no serde
// default).
type SystemSettings struct {
	StorageBackend            string `json:"storage_backend"`
	StoragePath               string `json:"storage_path"`
	Environment               string `json:"environment"`
	AllowAnonymousDownload    bool   `json:"allow_anonymous_download"`
	MaxUploadSizeBytes        int64  `json:"max_upload_size_bytes"`
	RetentionDays             int64  `json:"retention_days"`
	AuditRetentionDays        int64  `json:"audit_retention_days"`
	BackupRetentionCount      int64  `json:"backup_retention_count"`
	EdgeStaleThresholdMinutes int64  `json:"edge_stale_threshold_minutes"`
}

// GetSystemSettings returns the current system settings singleton.
func (c *Client) GetSystemSettings(ctx context.Context) (*SystemSettings, error) {
	var out SystemSettings
	if err := c.do(ctx, http.MethodGet, "/admin/settings", nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// UpdateSystemSettings writes settings via POST /admin/settings, returning the
// persisted values. Sends the full struct: the backend requires storage_backend
// and storage_path in the body even though it doesn't persist them.
func (c *Client) UpdateSystemSettings(ctx context.Context, req SystemSettings) (*SystemSettings, error) {
	var out SystemSettings
	if err := c.do(ctx, http.MethodPost, "/admin/settings", req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
