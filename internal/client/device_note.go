package client

import (
	"context"
	"net/url"
)

// DeviceNote models /api/v1/devices/{id}/notes. Notes written in the Iru UI come
// back as HTML; notes written through the API come back as sent.
type DeviceNote struct {
	ID        string `json:"note_id,omitempty"`
	Content   string `json:"content"`
	Author    string `json:"author,omitempty"`
	CreatedAt string `json:"created_at,omitempty"`
	UpdatedAt string `json:"updated_at,omitempty"`
}

func notesPath(deviceID string) string {
	return devicesPath + "/" + url.PathEscape(deviceID) + "/notes"
}

func notePath(deviceID, noteID string) string {
	return notesPath(deviceID) + "/" + url.PathEscape(noteID)
}

// ListDeviceNotes returns every note on a device. The endpoint returns a bare
// array.
func (c *Client) ListDeviceNotes(ctx context.Context, deviceID string) ([]DeviceNote, error) {
	var out []DeviceNote
	if err := c.DoJSON(ctx, "GET", notesPath(deviceID), nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// CreateDeviceNote adds a note to a device.
func (c *Client) CreateDeviceNote(ctx context.Context, deviceID, content string) (*DeviceNote, error) {
	var out DeviceNote
	if err := c.DoJSON(ctx, "POST", notesPath(deviceID), DeviceNote{Content: content}, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// GetDeviceNote fetches one note. Returns NotFoundError on 404.
func (c *Client) GetDeviceNote(ctx context.Context, deviceID, noteID string) (*DeviceNote, error) {
	var out DeviceNote
	if err := c.DoJSON(ctx, "GET", notePath(deviceID, noteID), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// UpdateDeviceNote replaces a note's content.
func (c *Client) UpdateDeviceNote(ctx context.Context, deviceID, noteID, content string) (*DeviceNote, error) {
	var out DeviceNote
	if err := c.DoJSON(ctx, "PATCH", notePath(deviceID, noteID), DeviceNote{Content: content}, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// DeleteDeviceNote removes a note.
func (c *Client) DeleteDeviceNote(ctx context.Context, deviceID, noteID string) error {
	return c.DoJSON(ctx, "DELETE", notePath(deviceID, noteID), nil, nil)
}
