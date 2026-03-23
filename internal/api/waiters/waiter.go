// Package waiters provides async operation polling for ZCP API jobs.
package waiters

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"time"

	"github.com/zsoftly/zcp-cli/internal/httpclient"
)

const (
	// DefaultPollInterval is how often the waiter polls for status.
	DefaultPollInterval = 3 * time.Second
	// DefaultWaitTimeout is the maximum time to wait for a job.
	DefaultWaitTimeout = 10 * time.Minute
	// JobStatusComplete means the async job finished successfully.
	JobStatusComplete = "COMPLETE"
	// JobStatusFailed means the async job failed.
	JobStatusFailed = "FAILED"
	// JobStatusPending means the async job is still in progress.
	JobStatusPending = "PENDING"
)

// ResourceStatus holds the result of a polled async job.
type ResourceStatus struct {
	JobID        string `json:"jobId"`
	ResourceID   string `json:"resourceId"`
	ResourceType string `json:"resourceType"`
	Status       string `json:"status"`
	ErrorMessage string `json:"errorMessage"`
	ErrorCode    int    `json:"errorCode"`
}

// Waiter polls the ZCP async job status endpoint.
type Waiter struct {
	client       *httpclient.Client
	pollInterval time.Duration
	waitTimeout  time.Duration
	progressOut  io.Writer
}

// Option configures a Waiter.
type Option func(*Waiter)

// WithPollInterval sets the polling interval.
func WithPollInterval(d time.Duration) Option {
	return func(w *Waiter) { w.pollInterval = d }
}

// WithWaitTimeout sets the maximum wait duration.
func WithWaitTimeout(d time.Duration) Option {
	return func(w *Waiter) { w.waitTimeout = d }
}

// WithProgressWriter sets where progress dots are written (e.g. os.Stderr).
func WithProgressWriter(w io.Writer) Option {
	return func(wt *Waiter) { wt.progressOut = w }
}

// New creates a new Waiter using the given client.
func New(client *httpclient.Client, opts ...Option) *Waiter {
	w := &Waiter{
		client:       client,
		pollInterval: DefaultPollInterval,
		waitTimeout:  DefaultWaitTimeout,
		progressOut:  io.Discard,
	}
	for _, opt := range opts {
		opt(w)
	}
	return w
}

// Wait polls until the job reaches a terminal state (COMPLETE or FAILED).
// Returns the final ResourceStatus or an error if the context is cancelled or
// the job failed.
func (w *Waiter) Wait(ctx context.Context, jobID string) (*ResourceStatus, error) {
	deadline := time.Now().Add(w.waitTimeout)
	ticker := time.NewTicker(w.pollInterval)
	defer ticker.Stop()

	fmt.Fprintf(w.progressOut, "Waiting for job %s", jobID)

	for {
		select {
		case <-ctx.Done():
			fmt.Fprintln(w.progressOut)
			return nil, ctx.Err()
		case t := <-ticker.C:
			if t.After(deadline) {
				fmt.Fprintln(w.progressOut)
				return nil, fmt.Errorf("timed out waiting for job %s after %s", jobID, w.waitTimeout)
			}

			status, err := w.getStatus(ctx, jobID)
			if err != nil {
				fmt.Fprintln(w.progressOut)
				return nil, fmt.Errorf("polling job %s: %w", jobID, err)
			}

			fmt.Fprintf(w.progressOut, ".")

			switch status.Status {
			case JobStatusComplete:
				fmt.Fprintln(w.progressOut, " done")
				return status, nil
			case JobStatusFailed:
				fmt.Fprintln(w.progressOut, " failed")
				msg := status.ErrorMessage
				if msg == "" {
					msg = fmt.Sprintf("job failed with error code %d", status.ErrorCode)
				}
				return status, fmt.Errorf("job %s failed: %s", jobID, msg)
			}
			// Still pending — continue polling
		}
	}
}

// GetStatus fetches the current status of a job without waiting.
func (w *Waiter) GetStatus(ctx context.Context, jobID string) (*ResourceStatus, error) {
	return w.getStatus(ctx, jobID)
}

type resourceStatusResponse struct {
	JobID        string `json:"jobId"`
	ResourceID   string `json:"resourceId"`
	ErrorMessage string `json:"errorMessage"`
	ErrorCode    int    `json:"errorCode"`
	ResourceType string `json:"resourceType"`
	Status       string `json:"status"`
}

func (w *Waiter) getStatus(ctx context.Context, jobID string) (*ResourceStatus, error) {
	q := url.Values{"jobId": {jobID}}
	var resp resourceStatusResponse
	if err := w.client.Get(ctx, "/restapi/asyncjob/resourceStatus", q, &resp); err != nil {
		return nil, err
	}
	return &ResourceStatus{
		JobID:        resp.JobID,
		ResourceID:   resp.ResourceID,
		ResourceType: resp.ResourceType,
		Status:       resp.Status,
		ErrorMessage: resp.ErrorMessage,
		ErrorCode:    resp.ErrorCode,
	}, nil
}

// ParseJobResponse extracts a jobId from a generic API response body.
// Many ZCP async endpoints return {"jobId": "..."} on success.
func ParseJobResponse(body []byte) (string, error) {
	var result struct {
		JobID string `json:"jobId"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("parsing job response: %w", err)
	}
	if result.JobID == "" {
		return "", fmt.Errorf("no jobId in response")
	}
	return result.JobID, nil
}
