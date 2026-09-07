package httpclient

import (
	"strings"
	"testing"
)

func TestRedactSecrets(t *testing.T) {
	client := New(Options{BearerToken: "tok-SECRET-1234567"})

	tests := []struct {
		name       string
		body       string
		wantSubstr string
		wantAbsent []string
	}{
		{
			name:       "spaced key and value",
			body:       `{"access_key_token": "tok-SECRET-123"}`,
			wantSubstr: `"access_key_token": "[REDACTED]"`,
			wantAbsent: []string{"tok-SECRET-123"},
		},
		{
			name:       "unspaced key and value",
			body:       `{"access_key_token":"tok-SECRET-123"}`,
			wantSubstr: `"access_key_token":"[REDACTED]"`,
			wantAbsent: []string{"tok-SECRET-123"},
		},
		{
			name:       "mixed case key",
			body:       `{"Access_Key_Token":"tok-SECRET-123"}`,
			wantSubstr: `"Access_Key_Token":"[REDACTED]"`,
			wantAbsent: []string{"tok-SECRET-123"},
		},
		{
			name:       "value with escaped quote",
			body:       `{"password":"a\"b-SECRET"}`,
			wantSubstr: `"password":"[REDACTED]"`,
			wantAbsent: []string{"a\\\"b-SECRET"},
		},
		{
			name:       "null value left as null",
			body:       `{"refresh_token":null}`,
			wantSubstr: `"refresh_token":null`,
		},
		{
			name:       "numeric value redacted",
			body:       `{"token":123456}`,
			wantSubstr: `"token":"[REDACTED]"`,
			wantAbsent: []string{"123456"},
		},
		{
			name:       "boolean value redacted",
			body:       `{"secret":true}`,
			wantSubstr: `"secret":"[REDACTED]"`,
			wantAbsent: []string{`"secret":true`},
		},
		{
			name:       "substring key left untouched",
			body:       `{"tokenizer":"keep"}`,
			wantSubstr: `"tokenizer":"keep"`,
		},
		{
			name:       "client_secret redacted",
			body:       `{"client_secret":"tok-SECRET-123"}`,
			wantSubstr: `"client_secret":"[REDACTED]"`,
			wantAbsent: []string{"tok-SECRET-123"},
		},
		{
			name:       "secret_access_key redacted",
			body:       `{"secret_access_key":"tok-SECRET-123"}`,
			wantSubstr: `"secret_access_key":"[REDACTED]"`,
			wantAbsent: []string{"tok-SECRET-123"},
		},
		{
			name:       "private_key redacted",
			body:       `{"private_key":"tok-SECRET-123"}`,
			wantSubstr: `"private_key":"[REDACTED]"`,
			wantAbsent: []string{"tok-SECRET-123"},
		},
		{
			name:       "bearer_token redacted",
			body:       `{"bearer_token":"tok-SECRET-123"}`,
			wantSubstr: `"bearer_token":"[REDACTED]"`,
			wantAbsent: []string{"tok-SECRET-123"},
		},
		{
			name:       "api_token redacted",
			body:       `{"api_token":"tok-SECRET-123"}`,
			wantSubstr: `"api_token":"[REDACTED]"`,
			wantAbsent: []string{"tok-SECRET-123"},
		},
		{
			name:       "auth_token redacted",
			body:       `{"auth_token":"tok-SECRET-123"}`,
			wantSubstr: `"auth_token":"[REDACTED]"`,
			wantAbsent: []string{"tok-SECRET-123"},
		},
		{
			name:       "access_key_secret redacted",
			body:       `{"access_key_secret":"tok-SECRET-123"}`,
			wantSubstr: `"access_key_secret":"[REDACTED]"`,
			wantAbsent: []string{"tok-SECRET-123"},
		},
		{
			name:       "raw token outside JSON",
			body:       `plain text containing tok-SECRET-1234567 raw`,
			wantSubstr: `[REDACTED]`,
			wantAbsent: []string{"tok-SECRET-1234567"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := string(client.redactSecrets([]byte(tt.body)))

			if !strings.Contains(got, tt.wantSubstr) {
				t.Errorf("redactSecrets(%q) = %q, want substring %q", tt.body, got, tt.wantSubstr)
			}
			for _, absent := range tt.wantAbsent {
				if strings.Contains(got, absent) {
					t.Errorf("redactSecrets(%q) = %q, must not contain %q", tt.body, got, absent)
				}
			}
		})
	}
}

func TestRedactSecretsShortBearerTokenLeavesBodyUntouched(t *testing.T) {
	client := New(Options{BearerToken: "t"})

	const body = `The selected service not found.`

	got := string(client.redactSecrets([]byte(body)))

	if got != body {
		t.Errorf("redactSecrets(%q) = %q, want body untouched for a 1-character bearer token", body, got)
	}
}

func TestRedactSecretsEscapedBearerToken(t *testing.T) {
	const token = `1234|abc/def+ghi=`
	client := New(Options{BearerToken: token})

	body := `{"callback_url":"https:\/\/x?t=1234|abc\/def+ghi="}`

	got := string(client.redactSecrets([]byte(body)))

	if strings.Contains(got, token) {
		t.Errorf("redactSecrets(%q) = %q, still contains raw token", body, got)
	}
	if strings.Contains(got, `1234|abc\/def+ghi=`) {
		t.Errorf("redactSecrets(%q) = %q, still contains escaped token", body, got)
	}
	if !strings.Contains(got, "[REDACTED]") {
		t.Errorf("redactSecrets(%q) = %q, want [REDACTED] present", body, got)
	}
}
