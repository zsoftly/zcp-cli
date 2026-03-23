// Package apierrors defines ZCP API error types and parsing.
package apierrors

import (
	"encoding/json"
	"errors"
	"fmt"
)

// APIError represents a structured error returned by the ZCP API.
type APIError struct {
	StatusCode int
	Code       string
	Message    string
}

func (e *APIError) Error() string {
	if e.Code != "" {
		return fmt.Sprintf("API error %d (code=%s): %s", e.StatusCode, e.Code, e.Message)
	}
	return fmt.Sprintf("API error %d: %s", e.StatusCode, e.Message)
}

// IsNotFound returns true if err is an APIError with status 404.
func IsNotFound(err error) bool {
	var ae *APIError
	return errors.As(err, &ae) && ae.StatusCode == 404
}

// IsUnauthorized returns true if err is an APIError with status 401.
func IsUnauthorized(err error) bool {
	var ae *APIError
	return errors.As(err, &ae) && ae.StatusCode == 401
}

// IsForbidden returns true if err is an APIError with status 403.
func IsForbidden(err error) bool {
	var ae *APIError
	return errors.As(err, &ae) && ae.StatusCode == 403
}

// apiErrorResponse mirrors the ZCP error envelope:
// { "listErrorResponse": { "errorCode": "...", "errorMsg": "..." } }
type apiErrorResponse struct {
	ListErrorResponse *apiErrorMsg `json:"listErrorResponse"`
	ErrorCode         string       `json:"errorCode"`
	ErrorMsg          string       `json:"errorMsg"`
	Message           string       `json:"message"`
}

type apiErrorMsg struct {
	ErrorCode string `json:"errorCode"`
	ErrorMsg  string `json:"errorMsg"`
}

// ParseResponse creates an APIError from an HTTP status code and response body.
func ParseResponse(statusCode int, body []byte) error {
	ae := &APIError{StatusCode: statusCode}

	if len(body) > 0 {
		var resp apiErrorResponse
		if err := json.Unmarshal(body, &resp); err == nil {
			if resp.ListErrorResponse != nil {
				ae.Code = resp.ListErrorResponse.ErrorCode
				ae.Message = resp.ListErrorResponse.ErrorMsg
			} else if resp.ErrorMsg != "" {
				ae.Code = resp.ErrorCode
				ae.Message = resp.ErrorMsg
			} else if resp.Message != "" {
				ae.Message = resp.Message
			}
		}
		if ae.Message == "" {
			// Fall back to raw body (truncated)
			msg := string(body)
			if len(msg) > 256 {
				msg = msg[:256] + "..."
			}
			ae.Message = msg
		}
	}

	if ae.Message == "" {
		ae.Message = fmt.Sprintf("HTTP %d", statusCode)
	}

	return ae
}
