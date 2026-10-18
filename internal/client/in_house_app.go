package client

import (
	"context"
	"fmt"
	"path/filepath"
	"time"
)

const inHouseAppsPath = "/api/v1/library/ipa-apps"

// uploadPollInterval is how often UploadInHouseApp checks validation status. It is
// a var so tests can shorten it.
var uploadPollInterval = time.Second

// InHouseApp models /api/v1/library/ipa-apps. Writes are JSON; the .ipa is uploaded
// beforehand and referenced by FileKey.
type InHouseApp struct {
	ID           string `json:"id,omitempty"`
	Name         string `json:"name"`
	FileKey      string `json:"file_key,omitempty"`
	Active       bool   `json:"active"`
	RunsOnIPhone bool   `json:"runs_on_iphone"`
	RunsOnIPad   bool   `json:"runs_on_ipad"`
	RunsOnTV     bool   `json:"runs_on_tv"`
	// Computed / read-only:
	AppIdentifier    string `json:"app_identifier,omitempty"`
	AppName          string `json:"app_name,omitempty"`
	AppVersion       string `json:"app_version,omitempty"`
	MinimumOSVersion string `json:"minimum_os_version,omitempty"`
	SHA256           string `json:"sha256,omitempty"`
	FileSize         int64  `json:"file_size,omitempty"`
	FileUpdated      string `json:"file_updated,omitempty"`
}

type inHouseAppUpload struct {
	UploadTarget
	ID string `json:"id"`
}

type inHouseAppUploadStatus struct {
	Status string `json:"status"`
	Reason string `json:"reason"`
}

// UploadInHouseApp uploads the .ipa at path and waits until Iru has validated it.
// It returns the file_key to create or update the library item with.
func (c *Client) UploadInHouseApp(ctx context.Context, path string) (string, error) {
	var up inHouseAppUpload
	if err := c.DoJSON(ctx, "POST", inHouseAppsPath+"/upload", map[string]string{"filename": filepath.Base(path)}, &up); err != nil {
		return "", fmt.Errorf("requesting upload URL: %w", err)
	}
	if err := c.UploadFile(ctx, up.UploadTarget, path); err != nil {
		return "", err
	}
	for {
		var st inHouseAppUploadStatus
		if err := c.DoJSON(ctx, "GET", inHouseAppsPath+"/upload/"+up.ID+"/status", nil, &st); err != nil {
			return "", fmt.Errorf("checking upload status: %w", err)
		}
		switch st.Status {
		case "VALIDATED":
			return up.FileKey, nil
		case "UPLOAD_FAILED", "VALIDATE_FAILED":
			return "", fmt.Errorf("iru rejected the upload: %s %s", st.Status, st.Reason)
		}
		select {
		case <-ctx.Done():
			return "", fmt.Errorf("waiting for upload validation: %w", ctx.Err())
		case <-time.After(uploadPollInterval):
		}
	}
}

// CreateInHouseApp creates an in-house app around an uploaded file_key. The API
// documents no response body for create, so when none comes back the new item is
// found by its file_key, which is unique per upload.
func (c *Client) CreateInHouseApp(ctx context.Context, in InHouseApp) (*InHouseApp, error) {
	var out InHouseApp
	if err := c.DoJSON(ctx, "POST", inHouseAppsPath, in, &out); err != nil {
		return nil, err
	}
	if out.ID != "" {
		return &out, nil
	}
	apps, err := listAll[InHouseApp](ctx, c, inHouseAppsPath)
	if err != nil {
		return nil, fmt.Errorf("locating created in-house app: %w", err)
	}
	for i := range apps {
		if apps[i].FileKey == in.FileKey {
			return &apps[i], nil
		}
	}
	return nil, fmt.Errorf("created in-house app not found by file_key %s", in.FileKey)
}

// GetInHouseApp fetches an in-house app by id. Returns NotFoundError on 404.
func (c *Client) GetInHouseApp(ctx context.Context, id string) (*InHouseApp, error) {
	var out InHouseApp
	if err := c.DoJSON(ctx, "GET", inHouseAppsPath+"/"+id, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// UpdateInHouseApp patches an in-house app. FileKey is sent only when non-empty,
// i.e. when a new binary was uploaded. The result is re-read rather than trusting a
// possibly empty PATCH body.
func (c *Client) UpdateInHouseApp(ctx context.Context, id string, in InHouseApp) (*InHouseApp, error) {
	if err := c.DoJSON(ctx, "PATCH", inHouseAppsPath+"/"+id, in, nil); err != nil {
		return nil, err
	}
	return c.GetInHouseApp(ctx, id)
}

// DeleteInHouseApp removes an in-house app by id.
func (c *Client) DeleteInHouseApp(ctx context.Context, id string) error {
	return c.DoJSON(ctx, "DELETE", inHouseAppsPath+"/"+id, nil, nil)
}

// ListInHouseApps returns all in-house apps (paginated).
func (c *Client) ListInHouseApps(ctx context.Context) ([]InHouseApp, error) {
	return listAll[InHouseApp](ctx, c, inHouseAppsPath)
}
