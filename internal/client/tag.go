package client

import "context"

const tagsPath = "/api/v1/tags"

// CreateTag creates a tag.
func (c *Client) CreateTag(ctx context.Context, in Tag) (*Tag, error) {
	var out Tag
	if err := c.DoJSON(ctx, "POST", tagsPath, in, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// ListTags returns all tags (paginated). The Iru API has no GET-by-id for tags,
// so callers refresh by listing and matching on id.
func (c *Client) ListTags(ctx context.Context) ([]Tag, error) {
	return listAll[Tag](ctx, c, tagsPath)
}

// UpdateTag patches a tag by id.
func (c *Client) UpdateTag(ctx context.Context, id string, in Tag) (*Tag, error) {
	var out Tag
	if err := c.DoJSON(ctx, "PATCH", tagsPath+"/"+id, in, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// DeleteTag removes a tag by id.
func (c *Client) DeleteTag(ctx context.Context, id string) error {
	return c.DoJSON(ctx, "DELETE", tagsPath+"/"+id, nil, nil)
}

// GetTagByID lists tags and returns the one matching id, or a NotFoundError.
func (c *Client) GetTagByID(ctx context.Context, id string) (*Tag, error) {
	tags, err := c.ListTags(ctx)
	if err != nil {
		return nil, err
	}
	for i := range tags {
		if tags[i].ID == id {
			return &tags[i], nil
		}
	}
	return nil, &NotFoundError{Body: "tag " + id + " not found"}
}
