package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
)

// listEnvelope is the standard paginated list wrapper the Iru API returns for most
// library endpoints: {count, next, previous, results[]}.
type listEnvelope struct {
	Count    int             `json:"count"`
	Next     *string         `json:"next"`
	Previous *string         `json:"previous"`
	Results  json.RawMessage `json:"results"`
}

// listAll walks every page of a paginated endpoint and appends decoded results into
// out (which must be a pointer to a slice). It uses limit/offset paging (limit 300).
func listAll[T any](ctx context.Context, c *Client, path string) ([]T, error) {
	const limit = 300
	offset := 0
	var all []T

	for {
		q := url.Values{}
		q.Set("limit", fmt.Sprintf("%d", limit))
		q.Set("offset", fmt.Sprintf("%d", offset))

		sep := "?"
		if containsQuery(path) {
			sep = "&"
		}

		var env listEnvelope
		if err := c.DoJSON(ctx, "GET", path+sep+q.Encode(), nil, &env); err != nil {
			return nil, err
		}

		var page []T
		if len(env.Results) > 0 {
			if err := json.Unmarshal(env.Results, &page); err != nil {
				return nil, fmt.Errorf("decoding list page: %w", err)
			}
		}
		all = append(all, page...)

		if len(page) < limit {
			break
		}
		offset += len(page)
	}

	return all, nil
}

func containsQuery(path string) bool {
	for i := 0; i < len(path); i++ {
		if path[i] == '?' {
			return true
		}
	}
	return false
}
