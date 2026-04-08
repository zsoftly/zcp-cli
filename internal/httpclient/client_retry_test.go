package httpclient_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync/atomic"
	"testing"
	"time"

	"github.com/zsoftly/zcp-cli/internal/httpclient"
)

// newFastClient returns a client pointing at srv with a short timeout and
// MaxRetries set to max so tests don't wait for real backoff durations.
// Tests that need to exercise backoff timing override Timeout themselves.
func newTestClient(srv *httptest.Server, maxRetries int) *httpclient.Client {
	return httpclient.New(httpclient.Options{
		BaseURL:     srv.URL,
		BearerToken: "tok",
		Timeout:     5 * time.Second,
		MaxRetries:  maxRetries,
	})
}

// TestRetryGET500EventuallySucceeds verifies that a GET retries on 500 and
// succeeds once the server starts returning 200.
func TestRetryGET500EventuallySucceeds(t *testing.T) {
	var requestCount int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&requestCount, 1)
		if n < 3 {
			// First two attempts fail with 500.
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"message":"server error"}`))
			return
		}
		// Third attempt succeeds.
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok"}`))
	}))
	defer srv.Close()

	// Use MaxRetries=3 so it will retry enough times, but set a very short
	// client timeout to avoid waiting for real backoff. We override with a
	// custom server that responds immediately, so backoff sleeps dominate —
	// keep MaxRetries small to keep test fast.
	client := httpclient.New(httpclient.Options{
		BaseURL:     srv.URL,
		BearerToken: "tok",
		Timeout:     5 * time.Second,
		MaxRetries:  3,
	})

	var result map[string]string
	err := client.Get(context.Background(), "/test", url.Values{}, &result)
	if err != nil {
		t.Fatalf("Get() expected success after retries, got error: %v", err)
	}

	got := atomic.LoadInt32(&requestCount)
	if got != 3 {
		t.Errorf("request count = %d, want 3 (1 initial + 2 retries)", got)
	}
	if result["status"] != "ok" {
		t.Errorf("result[status] = %q, want %q", result["status"], "ok")
	}
}

// TestRetryGET429 verifies that a GET retries on 429 Too Many Requests.
func TestRetryGET429(t *testing.T) {
	var requestCount int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&requestCount, 1)
		if n == 1 {
			w.WriteHeader(http.StatusTooManyRequests)
			w.Write([]byte(`{"message":"rate limited"}`))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok"}`))
	}))
	defer srv.Close()

	client := httpclient.New(httpclient.Options{
		BaseURL:     srv.URL,
		BearerToken: "tok",
		Timeout:     5 * time.Second,
		MaxRetries:  3,
	})

	var result map[string]string
	err := client.Get(context.Background(), "/test", url.Values{}, &result)
	if err != nil {
		t.Fatalf("Get() expected success after 429 retry, got error: %v", err)
	}

	got := atomic.LoadInt32(&requestCount)
	if got != 2 {
		t.Errorf("request count = %d, want 2 (1 initial + 1 retry)", got)
	}
}

// TestPOSTDoesNotRetryOn500 verifies that POST requests are never retried.
func TestPOSTDoesNotRetryOn500(t *testing.T) {
	var requestCount int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&requestCount, 1)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"message":"server error"}`))
	}))
	defer srv.Close()

	client := httpclient.New(httpclient.Options{
		BaseURL:     srv.URL,
		BearerToken: "tok",
		Timeout:     5 * time.Second,
		MaxRetries:  3,
	})

	var result map[string]string
	err := client.Post(context.Background(), "/test", map[string]string{"key": "val"}, &result)
	if err == nil {
		t.Fatal("Post() expected error for 500, got nil")
	}

	got := atomic.LoadInt32(&requestCount)
	if got != 1 {
		t.Errorf("request count = %d, want 1 (POST must not retry)", got)
	}
}

// TestRetryGETExceedsMaxRetries verifies that after exhausting all retries the
// last error is returned.
func TestRetryGETExceedsMaxRetries(t *testing.T) {
	var requestCount int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&requestCount, 1)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"message":"always failing"}`))
	}))
	defer srv.Close()

	client := httpclient.New(httpclient.Options{
		BaseURL:     srv.URL,
		BearerToken: "tok",
		Timeout:     5 * time.Second,
		MaxRetries:  2, // 1 initial + 2 retries = 3 total requests
	})

	err := client.Get(context.Background(), "/test", url.Values{}, nil)
	if err == nil {
		t.Fatal("Get() expected error after exhausting retries, got nil")
	}

	got := atomic.LoadInt32(&requestCount)
	if got != 3 {
		t.Errorf("request count = %d, want 3 (1 initial + 2 retries)", got)
	}
}

// TestRetryContextCancellationStopsLoop verifies that context cancellation
// interrupts the retry backoff and returns ctx.Err().
func TestRetryContextCancellationStopsLoop(t *testing.T) {
	var requestCount int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&requestCount, 1)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"message":"server error"}`))
	}))
	defer srv.Close()

	client := httpclient.New(httpclient.Options{
		BaseURL:     srv.URL,
		BearerToken: "tok",
		Timeout:     5 * time.Second,
		MaxRetries:  3,
	})

	// Cancel immediately after first request fires. The retry backoff (1s)
	// will be interrupted by ctx cancellation.
	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan error, 1)
	go func() {
		done <- client.Get(ctx, "/test", url.Values{}, nil)
	}()

	// Wait briefly for the first request to complete, then cancel.
	time.Sleep(50 * time.Millisecond)
	cancel()

	select {
	case err := <-done:
		if err == nil {
			t.Fatal("Get() expected error after context cancellation, got nil")
		}
	case <-time.After(3 * time.Second):
		t.Fatal("Get() did not return after context cancellation within 3s")
	}
}

// TestRetryGET404DoesNotRetry verifies that a non-retryable 404 is returned
// immediately without additional attempts.
func TestRetryGET404DoesNotRetry(t *testing.T) {
	var requestCount int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&requestCount, 1)
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"message":"not found"}`))
	}))
	defer srv.Close()

	client := httpclient.New(httpclient.Options{
		BaseURL:     srv.URL,
		BearerToken: "tok",
		Timeout:     5 * time.Second,
		MaxRetries:  3,
	})

	err := client.Get(context.Background(), "/missing", url.Values{}, nil)
	if err == nil {
		t.Fatal("Get() expected error for 404, got nil")
	}

	got := atomic.LoadInt32(&requestCount)
	if got != 1 {
		t.Errorf("request count = %d, want 1 (404 must not retry)", got)
	}
}
