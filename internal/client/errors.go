package client

import (
	"errors"
	"fmt"
)

// APIError is a non-2xx response. Body is retained for diagnostics; callers should
// avoid surfacing it verbatim if it may contain sensitive data.
type APIError struct {
	StatusCode int
	Body       string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("iru API error: status=%d body=%s", e.StatusCode, e.Body)
}

// NotFoundError is a sentinel for 404 responses so Read can drop the resource from
// state instead of hard-failing the plan.
type NotFoundError struct {
	Body string
}

func (e *NotFoundError) Error() string { return "iru API error: not found" }

// IsNotFound reports whether err is (or wraps) a NotFoundError.
func IsNotFound(err error) bool {
	var nfe *NotFoundError
	return errors.As(err, &nfe)
}
