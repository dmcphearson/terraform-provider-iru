// Package client is a typed HTTP client for the Iru (Kandji) Endpoint Management
// REST API. It handles bearer auth, retry/backoff on transient errors, and typed
// error mapping so resources can detect 404s and refresh cleanly.
package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const (
	defaultTimeout  = 60 * time.Second
	maxRetries      = 4
	userAgentPrefix = "terraform-provider-iru"
)

// baseRetryDelay is the exponential-backoff base. It is a var (not const) so tests
// can shrink it; production keeps the 2s default per the API retry guidance.
var baseRetryDelay = 2 * time.Second

// Client talks to a single Iru tenant.
type Client struct {
	baseURL    string
	token      string
	userAgent  string
	httpClient *http.Client
}

// New builds a Client. baseURL is the full tenant API URL
// (e.g. https://acme.api.kandji.io); it is trimmed of a trailing slash.
func New(baseURL, token, version string) *Client {
	return &Client{
		baseURL:   strings.TrimRight(baseURL, "/"),
		token:     token,
		userAgent: userAgentPrefix + "/" + version,
		httpClient: &http.Client{
			Timeout: defaultTimeout,
		},
	}
}

// SetHTTPClient swaps the underlying http.Client (used by tests to point at an
// httptest server).
func (c *Client) SetHTTPClient(h *http.Client) { c.httpClient = h }

// DoJSON performs a request with a JSON body (or nil) and decodes a JSON response
// into out (or nil to ignore the body). It retries transient failures.
func (c *Client) DoJSON(ctx context.Context, method, path string, body, out interface{}) error {
	var reqBody []byte
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("encoding request body: %w", err)
		}
		reqBody = b
	}
	return c.do(ctx, method, path, "application/json", reqBody, out)
}

// DoRaw performs a request with a caller-provided content type and raw body bytes.
// Used for urlencoded (custom-apps, blueprints) and multipart (profiles) requests.
func (c *Client) DoRaw(ctx context.Context, method, path, contentType string, body []byte, out interface{}) error {
	return c.do(ctx, method, path, contentType, body, out)
}

func (c *Client) do(ctx context.Context, method, path, contentType string, body []byte, out interface{}) error {
	url := c.baseURL + path

	var lastErr error
	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			delay := time.Duration(math.Pow(2, float64(attempt-1))) * baseRetryDelay
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(delay):
			}
		}

		var reader io.Reader
		if body != nil {
			reader = bytes.NewReader(body)
		}
		req, err := http.NewRequestWithContext(ctx, method, url, reader)
		if err != nil {
			return fmt.Errorf("building request: %w", err)
		}
		req.Header.Set("Authorization", "Bearer "+c.token)
		req.Header.Set("Accept", "application/json")
		req.Header.Set("User-Agent", c.userAgent)
		if contentType != "" && body != nil {
			req.Header.Set("Content-Type", contentType)
		}

		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = err
			continue // network error — retry
		}

		respBody, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()

		if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500 {
			lastErr = &APIError{StatusCode: resp.StatusCode, Body: string(respBody)}
			if ra := resp.Header.Get("Retry-After"); ra != "" {
				if secs, perr := strconv.Atoi(ra); perr == nil {
					select {
					case <-ctx.Done():
						return ctx.Err()
					case <-time.After(time.Duration(secs) * time.Second):
					}
				}
			}
			continue // transient — retry
		}

		if resp.StatusCode == http.StatusNotFound {
			return &NotFoundError{Body: string(respBody)}
		}

		if resp.StatusCode >= 400 {
			return &APIError{StatusCode: resp.StatusCode, Body: string(respBody)}
		}

		if out != nil && resp.StatusCode != http.StatusNoContent && len(respBody) > 0 {
			if err := json.Unmarshal(respBody, out); err != nil {
				return fmt.Errorf("decoding response: %w", err)
			}
		}
		return nil
	}

	return fmt.Errorf("request failed after %d retries: %w", maxRetries, lastErr)
}
