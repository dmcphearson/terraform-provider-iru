package client

import (
	"context"
	"net/url"
	"strconv"
)

const blueprintsPath = "/api/v1/blueprints"

// Blueprint models /api/v1/blueprints. type is "classic" or "map".
type Blueprint struct {
	ID             string          `json:"id,omitempty"`
	Name           string          `json:"name"`
	Icon           string          `json:"icon,omitempty"`
	Color          string          `json:"color,omitempty"`
	Description    string          `json:"description"`
	Type           string          `json:"type,omitempty"`
	ComputersCount int64           `json:"computers_count,omitempty"`
	EnrollmentCode *EnrollmentCode `json:"enrollment_code,omitempty"`
}

// EnrollmentCode is the nested enrollment_code object.
type EnrollmentCode struct {
	Code     string `json:"code,omitempty"`
	IsActive bool   `json:"is_active"`
}

// BlueprintCreate carries fields for creating a blueprint. Maps require a source
// (type + id) to seed from; the API create is urlencoded.
type BlueprintCreate struct {
	Blueprint
	SourceType string // e.g. "blueprint"
	SourceID   string
}

func blueprintForm(b Blueprint, sourceType, sourceID string) []byte {
	v := url.Values{}
	v.Set("name", b.Name)
	v.Set("description", b.Description)
	if b.Icon != "" {
		v.Set("icon", b.Icon)
	}
	if b.Color != "" {
		v.Set("color", b.Color)
	}
	if b.Type != "" {
		v.Set("type", b.Type)
	}
	if b.EnrollmentCode != nil {
		v.Set("enrollment_code.is_active", strconv.FormatBool(b.EnrollmentCode.IsActive))
		if b.EnrollmentCode.Code != "" {
			v.Set("enrollment_code.code", b.EnrollmentCode.Code)
		}
	}
	if sourceType != "" {
		v.Set("source.type", sourceType)
		v.Set("source.id", sourceID)
	}
	return []byte(v.Encode())
}

// CreateBlueprint creates a blueprint.
func (c *Client) CreateBlueprint(ctx context.Context, in BlueprintCreate) (*Blueprint, error) {
	var out Blueprint
	body := blueprintForm(in.Blueprint, in.SourceType, in.SourceID)
	if err := c.DoRaw(ctx, "POST", blueprintsPath, "application/x-www-form-urlencoded", body, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// GetBlueprint fetches a blueprint by id. Returns NotFoundError on 404.
func (c *Client) GetBlueprint(ctx context.Context, id string) (*Blueprint, error) {
	var out Blueprint
	if err := c.DoJSON(ctx, "GET", blueprintsPath+"/"+id, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// UpdateBlueprint patches a blueprint by id (urlencoded, no source).
func (c *Client) UpdateBlueprint(ctx context.Context, id string, in Blueprint) (*Blueprint, error) {
	var out Blueprint
	body := blueprintForm(in, "", "")
	if err := c.DoRaw(ctx, "PATCH", blueprintsPath+"/"+id, "application/x-www-form-urlencoded", body, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// DeleteBlueprint removes a blueprint by id.
func (c *Client) DeleteBlueprint(ctx context.Context, id string) error {
	return c.DoJSON(ctx, "DELETE", blueprintsPath+"/"+id, nil, nil)
}

// --- Library item assignment -------------------------------------------------
// Per the Iru API docs, assign and remove are two distinct POST endpoints:
//   POST .../assign-library-item  and  POST .../remove-library-item
// Read membership via list-library-items.

type assignmentBody struct {
	LibraryItemID    string `json:"library_item_id"`
	AssignmentNodeID string `json:"assignment_node_id,omitempty"`
}

// AssignLibraryItem attaches a library item to a blueprint.
func (c *Client) AssignLibraryItem(ctx context.Context, blueprintID, libraryItemID, assignmentNodeID string) error {
	body := assignmentBody{LibraryItemID: libraryItemID, AssignmentNodeID: assignmentNodeID}
	return c.DoJSON(ctx, "POST", blueprintsPath+"/"+blueprintID+"/assign-library-item", body, nil)
}

// RemoveLibraryItem detaches a library item from a blueprint.
func (c *Client) RemoveLibraryItem(ctx context.Context, blueprintID, libraryItemID, assignmentNodeID string) error {
	body := assignmentBody{LibraryItemID: libraryItemID, AssignmentNodeID: assignmentNodeID}
	return c.DoJSON(ctx, "POST", blueprintsPath+"/"+blueprintID+"/remove-library-item", body, nil)
}

// LibraryItemRef is an item returned by list-library-items.
type LibraryItemRef struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// ListBlueprintLibraryItems returns the library items assigned to a blueprint.
func (c *Client) ListBlueprintLibraryItems(ctx context.Context, blueprintID string) ([]LibraryItemRef, error) {
	return listAll[LibraryItemRef](ctx, c, blueprintsPath+"/"+blueprintID+"/list-library-items")
}

// IsLibraryItemAssigned reports whether a library item is currently assigned.
func (c *Client) IsLibraryItemAssigned(ctx context.Context, blueprintID, libraryItemID string) (bool, error) {
	items, err := c.ListBlueprintLibraryItems(ctx, blueprintID)
	if err != nil {
		return false, err
	}
	for _, it := range items {
		if it.ID == libraryItemID {
			return true, nil
		}
	}
	return false, nil
}
