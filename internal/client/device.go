package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
)

const devicesPath = "/api/v1/devices"

// DeviceUser is the user assigned to a device.
type DeviceUser struct {
	ID         string `json:"id"`
	Email      string `json:"email"`
	Name       string `json:"name"`
	IsArchived bool   `json:"is_archived"`
}

// Device models /api/v1/devices. The endpoint is polymorphic: fields that don't
// apply to a platform come back blank.
type Device struct {
	ID              string      `json:"device_id"`
	Name            string      `json:"device_name"`
	Model           string      `json:"model"`
	SerialNumber    string      `json:"serial_number"`
	Platform        string      `json:"platform"`
	OSVersion       string      `json:"os_version"`
	LastCheckIn     string      `json:"last_check_in"`
	User            *DeviceUser `json:"-"`
	AssetTag        string      `json:"asset_tag"`
	BlueprintID     string      `json:"blueprint_id"`
	BlueprintName   string      `json:"blueprint_name"`
	MDMEnabled      bool        `json:"mdm_enabled"`
	AgentInstalled  bool        `json:"agent_installed"`
	AgentVersion    string      `json:"agent_version"`
	IsMissing       bool        `json:"is_missing"`
	IsRemoved       bool        `json:"is_removed"`
	FirstEnrollment string      `json:"first_enrollment"`
	LastEnrollment  string      `json:"last_enrollment"`
	Tags            []string    `json:"tags"`
}

// UnmarshalJSON accepts "user" as an object, or as "" or null for an unassigned
// device.
func (d *Device) UnmarshalJSON(b []byte) error {
	type plain Device
	aux := struct {
		*plain
		User json.RawMessage `json:"user"`
	}{plain: (*plain)(d)}
	if err := json.Unmarshal(b, &aux); err != nil {
		return err
	}
	d.User = nil
	if len(aux.User) > 0 && aux.User[0] == '{' {
		var u DeviceUser
		if err := json.Unmarshal(aux.User, &u); err != nil {
			return fmt.Errorf("decoding device user: %w", err)
		}
		d.User = &u
	}
	return nil
}

// DeviceFilter narrows ListDevices. Empty fields are not sent. The API matches
// serial_number, device_name and user_email by substring.
type DeviceFilter struct {
	BlueprintID  string
	Platform     string
	SerialNumber string
	DeviceName   string
	UserEmail    string
	AssetTag     string
}

func (f DeviceFilter) query() url.Values {
	q := url.Values{}
	for k, v := range map[string]string{
		"blueprint_id":  f.BlueprintID,
		"platform":      f.Platform,
		"serial_number": f.SerialNumber,
		"device_name":   f.DeviceName,
		"user_email":    f.UserEmail,
		"asset_tag":     f.AssetTag,
	} {
		if v != "" {
			q.Set(k, v)
		}
	}
	return q
}

// ListDevices returns every device matching f. Unlike library endpoints, devices
// come back as a bare array, paged with limit/offset (limit 300).
func (c *Client) ListDevices(ctx context.Context, f DeviceFilter) ([]Device, error) {
	const limit = 300
	var all []Device
	for offset := 0; ; offset += limit {
		q := f.query()
		q.Set("limit", fmt.Sprint(limit))
		q.Set("offset", fmt.Sprint(offset))
		var page []Device
		if err := c.DoJSON(ctx, "GET", devicesPath+"?"+q.Encode(), nil, &page); err != nil {
			return nil, err
		}
		all = append(all, page...)
		if len(page) < limit {
			return all, nil
		}
	}
}

// GetDevice fetches a device by id. Returns NotFoundError on 404.
func (c *Client) GetDevice(ctx context.Context, id string) (*Device, error) {
	var out Device
	if err := c.DoJSON(ctx, "GET", devicesPath+"/"+id, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// GetDeviceDetails returns a device's full detail record as raw JSON. Its shape
// varies by platform and enabled features.
func (c *Client) GetDeviceDetails(ctx context.Context, id string) (json.RawMessage, error) {
	var out json.RawMessage
	if err := c.DoJSON(ctx, "GET", devicesPath+"/"+id+"/details", nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// UpdateDevice patches a device. Only the keys in patch are sent; a nil value
// clears that field. Accepted keys: user, asset_tag, blueprint_id, tags.
func (c *Client) UpdateDevice(ctx context.Context, id string, patch map[string]any) (*Device, error) {
	var out Device
	if err := c.DoJSON(ctx, "PATCH", devicesPath+"/"+id, patch, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
