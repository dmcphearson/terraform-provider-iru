package client

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
)

const usersPath = "/api/v1/users"

// UserIntegration is the directory integration a user comes from.
type UserIntegration struct {
	UUID string `json:"uuid"`
	Name string `json:"name"`
	Type string `json:"type"`
}

// User models /api/v1/users: users synced from directory integrations.
type User struct {
	ID          string           `json:"id"`
	Name        string           `json:"name"`
	Email       string           `json:"email"`
	Active      bool             `json:"active"`
	Archived    bool             `json:"archived"`
	Department  string           `json:"department"`
	JobTitle    string           `json:"job_title"`
	DeviceCount int64            `json:"device_count"`
	Integration *UserIntegration `json:"integration"`
}

// UserFilter narrows ListUsers. Empty fields are not sent.
type UserFilter struct {
	Email         string
	IntegrationID string
	Archived      *bool
}

// ListUsers returns every user matching f. The endpoint pages with a cursor in an
// absolute next URL, which may name another host, so only its path and query are
// followed.
func (c *Client) ListUsers(ctx context.Context, f UserFilter) ([]User, error) {
	q := url.Values{}
	if f.Email != "" {
		q.Set("email", f.Email)
	}
	if f.IntegrationID != "" {
		q.Set("integration_id", f.IntegrationID)
	}
	if f.Archived != nil {
		q.Set("archived", strconv.FormatBool(*f.Archived))
	}
	next := usersPath
	if len(q) > 0 {
		next += "?" + q.Encode()
	}

	var all []User
	seen := map[string]bool{}
	for next != "" && !seen[next] {
		seen[next] = true
		var page struct {
			Next    *string `json:"next"`
			Results []User  `json:"results"`
		}
		if err := c.DoJSON(ctx, "GET", next, nil, &page); err != nil {
			return nil, err
		}
		all = append(all, page.Results...)
		next = ""
		if page.Next != nil && *page.Next != "" {
			u, err := url.Parse(*page.Next)
			if err != nil {
				return nil, fmt.Errorf("parsing next page URL: %w", err)
			}
			next = u.EscapedPath()
			if u.RawQuery != "" {
				next += "?" + u.RawQuery
			}
		}
	}
	return all, nil
}

// GetUser fetches a user by id. Returns NotFoundError on 404.
func (c *Client) GetUser(ctx context.Context, id string) (*User, error) {
	var out User
	if err := c.DoJSON(ctx, "GET", usersPath+"/"+id, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
