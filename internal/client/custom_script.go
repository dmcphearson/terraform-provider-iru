package client

import "context"

const customScriptsPath = "/api/v1/library/custom-scripts"

// CreateCustomScript creates a script and returns the server representation.
func (c *Client) CreateCustomScript(ctx context.Context, in CustomScript) (*CustomScript, error) {
	var out CustomScript
	if err := c.DoJSON(ctx, "POST", customScriptsPath, in, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// GetCustomScript fetches a single script by id. Returns a NotFoundError on 404.
func (c *Client) GetCustomScript(ctx context.Context, id string) (*CustomScript, error) {
	var out CustomScript
	if err := c.DoJSON(ctx, "GET", customScriptsPath+"/"+id, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// UpdateCustomScript patches a script by id.
func (c *Client) UpdateCustomScript(ctx context.Context, id string, in CustomScript) (*CustomScript, error) {
	var out CustomScript
	if err := c.DoJSON(ctx, "PATCH", customScriptsPath+"/"+id, in, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// DeleteCustomScript removes a script by id.
func (c *Client) DeleteCustomScript(ctx context.Context, id string) error {
	return c.DoJSON(ctx, "DELETE", customScriptsPath+"/"+id, nil, nil)
}
