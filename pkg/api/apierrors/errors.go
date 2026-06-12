// Package apierrors defines ZCP API error types and parsing.
package apierrors

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
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

// providedInvalidRE matches the CMP's route-model-binding miss message,
// which is exactly "The provided <resource> is invalid." — anchored to the
// full message so unrelated 403s containing similar words don't match.
var providedInvalidRE = regexp.MustCompile(`(?i)^the provided [a-z0-9 _-]+ is invalid\.?$`)

// transientRoutingRE matches the CMP's known routing-layer error phrase.
// Expected format: "The route <path> could not be found."
var transientRoutingRE = regexp.MustCompile(`(?i)\bthe route\b.*could not be found`)

// IsTransientRoutingError returns true for the CMP's "The route … could not be found" 403,
// which occurs during the brief window after resource creation when the routing layer
// has not yet indexed the new slug. It is safe to retry this error.
func IsTransientRoutingError(err error) bool {
	var ae *APIError
	if !errors.As(err, &ae) {
		return false
	}
	if ae.StatusCode != 403 {
		return false
	}
	return transientRoutingRE.MatchString(ae.Message)
}

// IsResourceNotFound returns true if the error indicates the resource does not exist.
// It handles the CMP API's inconsistent use of 403 for not-found responses
// in addition to the standard 404: "kubernetes-cluster::k8s.not-found" style
// translation keys, and route-model-binding misses phrased as
// "The provided <resource> is invalid."
func IsResourceNotFound(err error) bool {
	var ae *APIError
	if !errors.As(err, &ae) {
		return false
	}
	if ae.StatusCode == 404 {
		return true
	}
	if ae.StatusCode != 403 {
		return false
	}
	if strings.Contains(strings.ToLower(ae.Message), "not-found") {
		return true
	}
	return providedInvalidRE.MatchString(strings.TrimSpace(ae.Message))
}

// apiErrorResponse mirrors the STKCNSL error envelope:
// { "status": "Error", "message": "...", "errors": { "field": ["..."] } }
// It also supports the legacy STKBILL format for backward compatibility.
type apiErrorResponse struct {
	// STKCNSL format
	Status  string                     `json:"status"`
	Message string                     `json:"message"`
	Errors  map[string]json.RawMessage `json:"errors"`

	// Legacy STKBILL format
	ListErrorResponse *apiErrorMsg `json:"listErrorResponse"`
	ErrorCode         string       `json:"errorCode"`
	ErrorMsg          string       `json:"errorMsg"`
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
			switch {
			// STKCNSL format: {"status":"Error","message":"...","errors":{...}}
			case resp.Status != "" && resp.Message != "":
				ae.Code = resp.Status
				ae.Message = resp.Message
				// Append field-level errors if present.
				if len(resp.Errors) > 0 {
					if detail, err := json.Marshal(resp.Errors); err == nil {
						ae.Message += " — " + string(detail)
					}
				}
			// Legacy STKBILL envelope
			case resp.ListErrorResponse != nil:
				ae.Code = resp.ListErrorResponse.ErrorCode
				ae.Message = resp.ListErrorResponse.ErrorMsg
			case resp.ErrorMsg != "":
				ae.Code = resp.ErrorCode
				ae.Message = resp.ErrorMsg
			case resp.Message != "":
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
