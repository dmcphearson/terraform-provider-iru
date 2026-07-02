package client

import "context"

// ListSelfServiceCategories returns the tenant's Self Service categories. This
// endpoint returns a flat JSON array (not a paginated envelope).
func (c *Client) ListSelfServiceCategories(ctx context.Context) ([]SelfServiceCategory, error) {
	var out []SelfServiceCategory
	if err := c.DoJSON(ctx, "GET", "/api/v1/self-service/categories", nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}
