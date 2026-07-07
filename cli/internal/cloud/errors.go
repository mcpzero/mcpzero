package cloud

import (
	"encoding/json"
	"errors"
	"fmt"
)

// APIError is a structured error from the MCPZERO web API.
type APIError struct {
	Code   string
	Status int
	Body   string
}

func (e *APIError) Error() string {
	if e.Code != "" {
		return fmt.Sprintf("API error %s (HTTP %d)", e.Code, e.Status)
	}
	return fmt.Sprintf("API error (HTTP %d): %s", e.Status, e.Body)
}

// AsAPIError returns a structured API error when err came from the web client.
func AsAPIError(err error) (*APIError, bool) {
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		return apiErr, true
	}
	return nil, false
}

// IsAPIError reports whether err is an API error with the given code.
func IsAPIError(err error, code string) bool {
	apiErr, ok := AsAPIError(err)
	return ok && apiErr.Code == code
}

func newAPIError(status int, body string) error {
	var parsed apiError
	code := ""
	if json.Unmarshal([]byte(body), &parsed) == nil && parsed.Error != "" {
		code = parsed.Error
	}
	snippet := body
	if len(snippet) > 200 {
		snippet = snippet[:200] + "..."
	}
	return &APIError{Code: code, Status: status, Body: snippet}
}
