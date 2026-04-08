package waiters_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/zsoftly/zcp-cli/internal/api/waiters"
	"github.com/zsoftly/zcp-cli/internal/httpclient"
)

type statusResponse struct {
	JobID        string `json:"jobId"`
	ResourceID   string `json:"resourceId"`
	ResourceType string `json:"resourceType"`
	Status       string `json:"status"`
	ErrorMessage string `json:"errorMessage"`
	ErrorCode    int    `json:"errorCode"`
}

func newTestClient(srv *httptest.Server) *httpclient.Client {
	return httpclient.New(httpclient.Options{
		BaseURL:     srv.URL,
		BearerToken: "test-token",
		Timeout:     5 * time.Second,
	})
}

func TestWaiterCompletesOnFirstPoll(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(statusResponse{
			JobID:      "job-1",
			ResourceID: "res-1",
			Status:     waiters.JobStatusComplete,
		})
	}))
	defer srv.Close()

	client := newTestClient(srv)
	var progressBuf bytes.Buffer
	w := waiters.New(client,
		waiters.WithPollInterval(50*time.Millisecond),
		waiters.WithWaitTimeout(5*time.Second),
		waiters.WithProgressWriter(&progressBuf),
	)

	status, err := w.Wait(context.Background(), "job-1")
	if err != nil {
		t.Fatalf("Wait() error = %v", err)
	}
	if status.Status != waiters.JobStatusComplete {
		t.Errorf("Status = %q, want %q", status.Status, waiters.JobStatusComplete)
	}
}

func TestWaiterReturnsErrorOnFailure(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(statusResponse{
			JobID:        "job-fail",
			Status:       waiters.JobStatusFailed,
			ErrorMessage: "disk quota exceeded",
		})
	}))
	defer srv.Close()

	client := newTestClient(srv)
	w := waiters.New(client,
		waiters.WithPollInterval(50*time.Millisecond),
		waiters.WithWaitTimeout(5*time.Second),
	)

	_, err := w.Wait(context.Background(), "job-fail")
	if err == nil {
		t.Fatal("expected error for failed job, got nil")
	}
}

func TestWaiterContextCancellation(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(statusResponse{
			JobID:  "job-pending",
			Status: waiters.JobStatusPending,
		})
	}))
	defer srv.Close()

	client := newTestClient(srv)
	w := waiters.New(client,
		waiters.WithPollInterval(50*time.Millisecond),
		waiters.WithWaitTimeout(30*time.Second),
	)

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	_, err := w.Wait(ctx, "job-pending")
	if err == nil {
		t.Fatal("expected error from context cancellation, got nil")
	}
}
