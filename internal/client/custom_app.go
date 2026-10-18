package client

import (
	"context"
	"fmt"
	"net/url"
	"path/filepath"
	"strconv"
)

const customAppsPath = "/api/v1/library/custom-apps"

// CustomApp models the /api/v1/library/custom-apps contract. The binary is uploaded
// first (UploadCustomApp, or out of band) and referenced by FileKey. self_service fields ARE returned by GET here (unlike
// custom scripts), so they round-trip normally.
type CustomApp struct {
	ID                     string `json:"id,omitempty"`
	Name                   string `json:"name"`
	FileKey                string `json:"file_key,omitempty"`
	InstallType            string `json:"install_type,omitempty"`
	InstallEnforcement     string `json:"install_enforcement,omitempty"`
	UnzipLocation          string `json:"unzip_location,omitempty"`
	AuditScript            string `json:"audit_script,omitempty"`
	PreinstallScript       string `json:"preinstall_script,omitempty"`
	PostinstallScript      string `json:"postinstall_script,omitempty"`
	Active                 bool   `json:"active"`
	Restart                bool   `json:"restart"`
	ShowInSelfService      bool   `json:"show_in_self_service"`
	SelfServiceCategoryID  string `json:"self_service_category_id,omitempty"`
	SelfServiceRecommended bool   `json:"self_service_recommended,omitempty"`
	// Computed / read-only:
	FileURL     string `json:"file_url,omitempty"`
	FileSize    int64  `json:"file_size,omitempty"`
	FileUpdated string `json:"file_updated,omitempty"`
	SHA256      string `json:"sha256,omitempty"`
}

// customAppForm builds the urlencoded body the create/update endpoints expect.
func customAppForm(in CustomApp) []byte {
	v := url.Values{}
	v.Set("name", in.Name)
	if in.FileKey != "" {
		v.Set("file_key", in.FileKey)
	}
	v.Set("install_type", in.InstallType)
	v.Set("install_enforcement", in.InstallEnforcement)
	v.Set("unzip_location", in.UnzipLocation)
	v.Set("audit_script", in.AuditScript)
	v.Set("preinstall_script", in.PreinstallScript)
	v.Set("postinstall_script", in.PostinstallScript)
	v.Set("show_in_self_service", strconv.FormatBool(in.ShowInSelfService))
	if in.SelfServiceCategoryID != "" {
		v.Set("self_service_category_id", in.SelfServiceCategoryID)
	}
	// The API rejects self_service_recommended, even false, unless Self Service is on.
	if in.ShowInSelfService {
		v.Set("self_service_recommended", strconv.FormatBool(in.SelfServiceRecommended))
	}
	v.Set("active", strconv.FormatBool(in.Active))
	v.Set("restart", strconv.FormatBool(in.Restart))
	return []byte(v.Encode())
}

// CreateCustomApp creates a custom app around an already-uploaded binary (file_key).
func (c *Client) CreateCustomApp(ctx context.Context, in CustomApp) (*CustomApp, error) {
	var out CustomApp
	if err := c.DoRaw(ctx, "POST", customAppsPath, "application/x-www-form-urlencoded", customAppForm(in), &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// GetCustomApp fetches a custom app by id. Returns NotFoundError on 404.
func (c *Client) GetCustomApp(ctx context.Context, id string) (*CustomApp, error) {
	var out CustomApp
	if err := c.DoJSON(ctx, "GET", customAppsPath+"/"+id, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// UpdateCustomApp patches a custom app by id.
func (c *Client) UpdateCustomApp(ctx context.Context, id string, in CustomApp) (*CustomApp, error) {
	var out CustomApp
	if err := c.DoRaw(ctx, "PATCH", customAppsPath+"/"+id, "application/x-www-form-urlencoded", customAppForm(in), &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// DeleteCustomApp removes a custom app by id.
func (c *Client) DeleteCustomApp(ctx context.Context, id string) error {
	return c.DoJSON(ctx, "DELETE", customAppsPath+"/"+id, nil, nil)
}

// UploadCustomApp uploads the installer at path and returns its file_key. Unlike
// in-house apps there is no validation step to wait for.
func (c *Client) UploadCustomApp(ctx context.Context, path string) (string, error) {
	var up UploadTarget
	if err := c.DoJSON(ctx, "POST", customAppsPath+"/upload", map[string]string{"name": filepath.Base(path)}, &up); err != nil {
		return "", fmt.Errorf("requesting upload URL: %w", err)
	}
	if err := c.UploadFile(ctx, up, path); err != nil {
		return "", err
	}
	return up.FileKey, nil
}

// ListCustomApps returns all custom apps (paginated).
func (c *Client) ListCustomApps(ctx context.Context) ([]CustomApp, error) {
	return listAll[CustomApp](ctx, c, customAppsPath)
}
