package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
)

const blueprintRoutingPath = "/api/v1/blueprint-routing/"

// BlueprintRouting is the tenant's one Blueprint Routing setting: the enrollment
// code used for Manual and Android Work Profile enrollment. Routing rules are not
// exposed by the API.
type BlueprintRouting struct {
	EnrollmentCode EnrollmentCode `json:"enrollment_code"`
}

// GetBlueprintRouting returns the tenant's Blueprint Routing setting.
func (c *Client) GetBlueprintRouting(ctx context.Context) (*BlueprintRouting, error) {
	var out BlueprintRouting
	if err := c.DoJSON(ctx, "GET", blueprintRoutingPath, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// UpdateBlueprintRouting patches the enrollment code. Only the keys in code are
// sent: "code" and/or "is_active".
func (c *Client) UpdateBlueprintRouting(ctx context.Context, code map[string]any) (*BlueprintRouting, error) {
	var out BlueprintRouting
	if err := c.DoJSON(ctx, "PATCH", blueprintRoutingPath, map[string]any{"enrollment_code": code}, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// RoutingActivityUser is the admin behind a routing activity event, if any.
type RoutingActivityUser struct {
	ID    string `json:"id"`
	Email string `json:"email"`
}

// RoutingActivity is one Blueprint Routing activity event. Details vary by
// activity_type, so they are kept as raw JSON.
type RoutingActivity struct {
	ID           int64                `json:"id"`
	ActivityTime string               `json:"activity_time"`
	ActivityType string               `json:"activity_type"`
	User         *RoutingActivityUser `json:"user"`
	DeviceID     *string              `json:"device_id"`
	Details      json.RawMessage      `json:"details"`
}

// ListBlueprintRoutingActivity returns up to max events, newest first, paging
// with limit/offset.
func (c *Client) ListBlueprintRoutingActivity(ctx context.Context, max int) ([]RoutingActivity, error) {
	const pageSize = 300
	var all []RoutingActivity
	for offset := 0; len(all) < max; offset += pageSize {
		q := url.Values{}
		q.Set("limit", fmt.Sprint(min(pageSize, max-len(all))))
		q.Set("offset", fmt.Sprint(offset))
		var env listEnvelope
		if err := c.DoJSON(ctx, "GET", blueprintRoutingPath+"activity?"+q.Encode(), nil, &env); err != nil {
			return nil, err
		}
		var page []RoutingActivity
		if len(env.Results) > 0 {
			if err := json.Unmarshal(env.Results, &page); err != nil {
				return nil, fmt.Errorf("decoding activity page: %w", err)
			}
		}
		all = append(all, page...)
		if env.Next == nil || len(page) == 0 {
			break
		}
	}
	return all, nil
}
