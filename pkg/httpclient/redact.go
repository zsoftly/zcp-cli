package httpclient

import (
	"encoding/json"
	"regexp"
	"strings"
)

// secretFieldPattern matches JSON object fields whose key is one of a known
// set of credential-bearing field names, capturing the key/colon/whitespace
// prefix separately from the value so the value alone can be replaced.
//
// The value alternative matches a JSON string (including one with escaped
// characters such as \" inside it), a JSON number (including exponent
// forms such as 1e10), the literals true/false, or null.
var secretFieldPattern = regexp.MustCompile(
	`(?i)("(?:access_key_token|access_token|refresh_token|secret_key|api_key|token|password|secret|client_secret|secret_access_key|private_key|bearer_token|api_token|auth_token|access_key_secret)"\s*:\s*)("(?:\\.|[^"\\])*"|-?\d+(?:\.\d+)?(?:[eE][+-]?\d+)?|true|false|null)`,
)

// redactSecrets returns a copy of body with the values of known
// credential-bearing JSON fields (access_key_token, token, access_token,
// refresh_token, password, secret_key, secret, api_key, client_secret,
// secret_access_key, private_key, bearer_token, api_token, auth_token,
// access_key_secret) replaced with "[REDACTED]". Matching is
// case-insensitive on the field name and anchored on the quoted key, so
// fields like "tokenizer" are left untouched. A null value is left as null
// since there is nothing to redact.
//
// If the client's configured bearer token is non-empty, at least 16
// characters long, and appears anywhere else in body, it is also replaced
// with [REDACTED] as a second line of defense. The minimum length guards
// against short tokens (e.g. in local dev/test setups) matching unrelated
// substrings of the response body. Both the raw token and its JSON-encoded
// forms (quoted, and with "/" escaped as "\/" the way Laravel's JSON
// encoder does) are replaced, so a token embedded in a JSON string survives
// escaping and is still caught.
func (c *Client) redactSecrets(body []byte) []byte {
	redacted := secretFieldPattern.ReplaceAllFunc(body, func(match []byte) []byte {
		sub := secretFieldPattern.FindSubmatch(match)
		if len(sub) != 3 {
			return match
		}
		if string(sub[2]) == "null" {
			return match
		}
		return append(append([]byte{}, sub[1]...), []byte(`"[REDACTED]"`)...)
	})

	const minTokenLen = 16
	if token := c.opts.BearerToken; len(token) >= minTokenLen {
		out := string(redacted)
		out = strings.ReplaceAll(out, token, "[REDACTED]")

		if encoded, err := json.Marshal(token); err == nil && len(encoded) >= 2 {
			quoted := string(encoded[1 : len(encoded)-1])
			out = strings.ReplaceAll(out, quoted, "[REDACTED]")
			out = strings.ReplaceAll(out, strings.ReplaceAll(quoted, "/", `\/`), "[REDACTED]")
		}

		redacted = []byte(out)
	}

	return redacted
}
