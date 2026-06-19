package apierrors_test

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/zsoftly/zcp-cli/pkg/api/apierrors"
)

func TestParseResponseSTKCNSLFormat(t *testing.T) {
	body, _ := json.Marshal(map[string]interface{}{
		"status":  "Error",
		"message": "The given data was invalid.",
		"errors": map[string]interface{}{
			"email": []string{"The email field is required."},
		},
	})

	err := apierrors.ParseResponse(422, body)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var ae *apierrors.APIError
	if !errors.As(err, &ae) {
		t.Fatalf("expected *APIError, got %T", err)
	}

	if ae.StatusCode != 422 {
		t.Errorf("StatusCode = %d, want 422", ae.StatusCode)
	}
	if ae.Code != "Error" {
		t.Errorf("Code = %q, want %q", ae.Code, "Error")
	}
	if !strings.Contains(ae.Message, "The given data was invalid.") {
		t.Errorf("Message = %q, want it to contain %q", ae.Message, "The given data was invalid.")
	}
	if !strings.Contains(ae.Message, "email") {
		t.Errorf("Message = %q, want it to contain field-level errors", ae.Message)
	}
}

func TestParseResponseValidationErrorsInData(t *testing.T) {
	// Laravel 422 validation shape: field errors live under "data", no "status".
	body, _ := json.Marshal(map[string]interface{}{
		"success": false,
		"message": "Validation errors",
		"data": map[string]interface{}{
			"name":       []string{"The name field must not be greater than 20 characters."},
			"public_key": []string{"The public key has already been taken."},
		},
	})

	err := apierrors.ParseResponse(422, body)
	var ae *apierrors.APIError
	if !errors.As(err, &ae) {
		t.Fatalf("expected *APIError, got %T", err)
	}
	if !strings.Contains(ae.Message, "Validation errors") {
		t.Errorf("Message = %q, want it to contain %q", ae.Message, "Validation errors")
	}
	if !strings.Contains(ae.Message, "must not be greater than 20 characters") {
		t.Errorf("Message = %q, want it to contain the name field detail", ae.Message)
	}
	if !strings.Contains(ae.Message, "public key has already been taken") {
		t.Errorf("Message = %q, want it to contain the public_key detail", ae.Message)
	}
}

func TestParseResponseSTKCNSLSimple(t *testing.T) {
	body, _ := json.Marshal(map[string]interface{}{
		"status":  "Error",
		"message": "Unauthenticated.",
	})

	err := apierrors.ParseResponse(401, body)
	var ae *apierrors.APIError
	if !errors.As(err, &ae) {
		t.Fatalf("expected *APIError, got %T", err)
	}
	if ae.StatusCode != 401 {
		t.Errorf("StatusCode = %d, want 401", ae.StatusCode)
	}
	if ae.Message != "Unauthenticated." {
		t.Errorf("Message = %q, want %q", ae.Message, "Unauthenticated.")
	}
}

func TestParseResponseLegacyZCPEnvelope(t *testing.T) {
	body, _ := json.Marshal(map[string]interface{}{
		"listErrorResponse": map[string]string{
			"errorCode": "INVALID_CREDENTIALS",
			"errorMsg":  "API key is invalid",
		},
	})

	err := apierrors.ParseResponse(401, body)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var ae *apierrors.APIError
	if !errors.As(err, &ae) {
		t.Fatalf("expected *APIError, got %T", err)
	}

	if ae.StatusCode != 401 {
		t.Errorf("StatusCode = %d, want 401", ae.StatusCode)
	}
	if ae.Code != "INVALID_CREDENTIALS" {
		t.Errorf("Code = %q, want %q", ae.Code, "INVALID_CREDENTIALS")
	}
	if ae.Message != "API key is invalid" {
		t.Errorf("Message = %q, want %q", ae.Message, "API key is invalid")
	}
}

func TestParseResponseHTTP550(t *testing.T) {
	body, _ := json.Marshal(map[string]interface{}{
		"listErrorResponse": map[string]string{
			"errorCode": "INTERNAL_ERROR",
			"errorMsg":  "Unexpected server error",
		},
	})

	err := apierrors.ParseResponse(550, body)
	var ae *apierrors.APIError
	if !errors.As(err, &ae) {
		t.Fatalf("expected *APIError, got %T", err)
	}
	if ae.StatusCode != 550 {
		t.Errorf("StatusCode = %d, want 550", ae.StatusCode)
	}
}

func TestParseResponseRawBody(t *testing.T) {
	err := apierrors.ParseResponse(500, []byte("Internal Server Error"))
	var ae *apierrors.APIError
	if !errors.As(err, &ae) {
		t.Fatalf("expected *APIError, got %T", err)
	}
	if ae.Message == "" {
		t.Error("expected non-empty message from raw body fallback")
	}
}

func TestIsUnauthorized(t *testing.T) {
	err := apierrors.ParseResponse(401, []byte(`{}`))
	if !apierrors.IsUnauthorized(err) {
		t.Error("IsUnauthorized() = false, want true for 401")
	}
	if apierrors.IsNotFound(err) {
		t.Error("IsNotFound() = true, want false for 401")
	}
}

func TestIsNotFound(t *testing.T) {
	err := apierrors.ParseResponse(404, []byte(`{}`))
	if !apierrors.IsNotFound(err) {
		t.Error("IsNotFound() = false, want true for 404")
	}
}

func TestAPIErrorString(t *testing.T) {
	ae := &apierrors.APIError{StatusCode: 400, Code: "BAD_REQUEST", Message: "bad input"}
	got := ae.Error()
	if got == "" {
		t.Error("Error() returned empty string")
	}
}

func TestIsTransientRoutingError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "exact CMP routing message",
			err:  &apierrors.APIError{StatusCode: 403, Message: "The route virtual-machines/my-vm could not be found."},
			want: true,
		},
		{
			name: "case-insensitive match",
			err:  &apierrors.APIError{StatusCode: 403, Message: "THE ROUTE api/v1 COULD NOT BE FOUND."},
			want: true,
		},
		{
			name: "404 with routing-style message",
			err:  &apierrors.APIError{StatusCode: 404, Message: "The route virtual-machines/x could not be found."},
			want: false,
		},
		{
			name: "403 not-found (IsResourceNotFound territory)",
			err:  &apierrors.APIError{StatusCode: 403, Message: "kubernetes-cluster::k8s.not-found"},
			want: false,
		},
		{
			name: "403 generic forbidden",
			err:  &apierrors.APIError{StatusCode: 403, Message: "Access denied."},
			want: false,
		},
		{
			name: "403 contains only 'route'",
			err:  &apierrors.APIError{StatusCode: 403, Message: "route misconfigured"},
			want: false,
		},
		{
			name: "403 contains only 'could not be found'",
			err:  &apierrors.APIError{StatusCode: 403, Message: "resource could not be found"},
			want: false,
		},
		{
			name: "403 broad 'route…could not be found' without 'the' prefix",
			err:  &apierrors.APIError{StatusCode: 403, Message: "route /v1/foo could not be found"},
			want: false,
		},
		{
			name: "non-APIError",
			err:  errors.New("plain error"),
			want: false,
		},
		{
			name: "nil error",
			err:  nil,
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := apierrors.IsTransientRoutingError(tt.err)
			if got != tt.want {
				t.Errorf("IsTransientRoutingError(%v) = %v, want %v", tt.err, got, tt.want)
			}
		})
	}
}

func TestIsResourceNotFoundProvidedInvalid(t *testing.T) {
	cases := []struct {
		status int
		msg    string
		want   bool
	}{
		// Route-model-binding miss: the API's 403 phrasing for a missing resource.
		{403, "The provided VPC is invalid.", true},
		{403, "The provided network is invalid.", true},
		// Validation errors use "selected", not "provided" — must NOT match.
		{403, "The selected vpc is invalid.", false},
		{422, "The provided VPC is invalid.", false},
		{403, "Something went wrong. Please try again later.", false},
		// Anchored match: phrase embedded in a longer message must NOT match.
		{403, "Error: The provided VPC is invalid. Try again.", false},
	}
	for _, c := range cases {
		err := &apierrors.APIError{StatusCode: c.status, Message: c.msg}
		if got := apierrors.IsResourceNotFound(err); got != c.want {
			t.Errorf("IsResourceNotFound(%d, %q) = %v, want %v", c.status, c.msg, got, c.want)
		}
	}
}
