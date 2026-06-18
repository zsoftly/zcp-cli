package objectstorage

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestBucketPublicPolicy(t *testing.T) {
	read := bucketPublicPolicy("my-bucket", false)

	// public-read grants GetObject but never write/delete.
	if !strings.Contains(read, `"s3:GetObject"`) {
		t.Errorf("public-read policy missing s3:GetObject: %s", read)
	}
	if strings.Contains(read, "s3:PutObject") || strings.Contains(read, "s3:DeleteObject") {
		t.Errorf("public-read policy must not grant write/delete: %s", read)
	}
	if !strings.Contains(read, `arn:aws:s3:::my-bucket/*`) || !strings.Contains(read, `arn:aws:s3:::my-bucket"`) {
		t.Errorf("policy missing expected bucket/object resources: %s", read)
	}

	// public-read-write additionally grants write + delete.
	rw := bucketPublicPolicy("my-bucket", true)
	for _, a := range []string{"s3:GetObject", "s3:PutObject", "s3:DeleteObject"} {
		if !strings.Contains(rw, a) {
			t.Errorf("public-read-write policy missing %s: %s", a, rw)
		}
	}

	// Both must be valid JSON (we marshal rather than interpolate raw).
	for _, p := range []string{read, rw} {
		var v map[string]any
		if err := json.Unmarshal([]byte(p), &v); err != nil {
			t.Errorf("policy is not valid JSON (%v): %s", err, p)
		}
	}
}
