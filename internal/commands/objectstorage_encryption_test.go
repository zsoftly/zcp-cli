package commands

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// ─── object-storage bucket encryption ───────────────────────────────────────
//
// Verified live 2026-09-07 (issue #54): the region's Ceph RADOS Gateway
// accepts PutBucketEncryption (SSE-S3) and then rejects every PutObject on
// that bucket with a 400 InvalidArgument error until encryption is removed.
// Per-request SSE-S3, SSE-KMS, and SSE-C are rejected too, because the
// gateway has no encryption key backend configured. 'enable' therefore fails
// fast, before building a client or contacting anything, while 'status' and
// 'disable' keep working so a user can inspect or clear an existing setting.
//
// 'bucket encryption enable/status' take only positional args
// (<storage-slug> <bucket-slug>); they do not define --project/--region
// flags, so this test does not pass them. buildClientAndPrinter also does
// not require project or region: TestMain (see main_test.go) already sets
// ZCP_API_URL to an unroutable address and ZCP_BEARER_TOKEN, which is enough
// for the command to reach RunE.

func TestBucketEncryptionEnableRefusesToRun(t *testing.T) {
	// Some earlier tests in this package unset ZCP_BEARER_TOKEN when they
	// finish (see commands_test.go), so set it here rather than relying on
	// TestMain's value surviving to this point.
	t.Setenv("ZCP_BEARER_TOKEN", "test-token")

	requests := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	_, _, err := execCmd(t, NewObjectStorageCmd(), "bucket", "encryption", "enable", "store", "bucket",
		"--api-url", srv.URL)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "not supported") {
		t.Errorf("error = %q, want containing %q", err, "not supported")
	}
	if requests != 0 {
		t.Errorf("expected 'enable' to make no HTTP requests, got %d", requests)
	}
}

func TestBucketEncryptionStatusStillAttemptsRequest(t *testing.T) {
	t.Setenv("ZCP_BEARER_TOKEN", "test-token")

	requests := 0
	// 400 is non-retryable (unlike 500/429), so the test does not have to
	// wait out the client's retry backoff.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer srv.Close()

	_, _, err := execCmd(t, NewObjectStorageCmd(), "bucket", "encryption", "status", "store", "bucket",
		"--api-url", srv.URL)
	if err == nil {
		t.Fatal("expected an error from the stub API, got nil")
	}
	if strings.Contains(err.Error(), "not supported") {
		t.Errorf("'status' must not be blocked like 'enable', got: %v", err)
	}
	if requests == 0 {
		t.Error("expected 'status' to attempt at least one HTTP request")
	}
}
