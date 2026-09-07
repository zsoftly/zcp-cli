// Package response defines common STKCNSL API response envelope types.
package response

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// Envelope is the standard STKCNSL API response wrapper for list endpoints.
type Envelope[T any] struct {
	Status       string  `json:"status"`
	Message      string  `json:"message"`
	Timezone     string  `json:"timezone"`
	CurrentPage  int     `json:"current_page"`
	Data         []T     `json:"data"`
	FirstPageURL *string `json:"first_page_url"`
	From         *int    `json:"from"`
	LastPage     int     `json:"last_page"`
	LastPageURL  *string `json:"last_page_url"`
	NextPageURL  *string `json:"next_page_url"`
	Path         string  `json:"path"`
	PerPage      int     `json:"per_page"`
	PrevPageURL  *string `json:"prev_page_url"`
	To           *int    `json:"to"`
	Total        int     `json:"total"`
}

// Single is for endpoints that return a single object in data.
type Single[T any] struct {
	Status  string `json:"status"`
	Message string `json:"message"`
	Data    T      `json:"data"`
}

// ParseFlexInt decodes a JSON value into an int, tolerating the STKCNSL API's
// inconsistent numeric encoding: a JSON number (e.g. 3), a quoted numeric
// string (e.g. "3", list responses), or an integral float in either form
// (e.g. 3.0 or "3.0"). A missing, null, or empty value decodes to 0. A
// non-integral or non-numeric value is an error.
func ParseFlexInt(raw json.RawMessage) (int, error) {
	s := strings.TrimSpace(string(raw))
	if len(s) == 0 || s == "null" {
		return 0, nil
	}
	if s[0] == '"' {
		var inner string
		if err := json.Unmarshal(raw, &inner); err != nil {
			return 0, fmt.Errorf("invalid quoted value: %w", err)
		}
		inner = strings.TrimSpace(inner)
		if inner == "" {
			return 0, nil
		}
		return parseIntegralFloat(inner)
	}
	return parseIntegralFloat(s)
}

// parseIntegralFloat parses s as a float64 and requires it to represent a
// whole number, rejecting fractional values like "3.5".
func parseIntegralFloat(s string) (int, error) {
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, fmt.Errorf("non-numeric value %q", s)
	}
	n := int(f)
	if float64(n) != f {
		return 0, fmt.Errorf("non-integral value %q", s)
	}
	return n, nil
}
