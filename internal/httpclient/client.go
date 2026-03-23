// Package httpclient provides the shared HTTP client used by all ZCP API service packages.
package httpclient

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/zsoftly/zcp-cli/internal/api/apierrors"
	"github.com/zsoftly/zcp-cli/internal/version"
)

// Options configures a Client.
type Options struct {
	BaseURL   string
	APIKey    string
	SecretKey string
	Timeout   time.Duration
	Debug     bool
	// DebugOut is where debug output is written (defaults to os.Stderr in New).
	DebugOut io.Writer
	// MaxRetries is the number of times to retry GET requests on transient failures.
	// Default is 3. Set to 0 to disable retries.
	MaxRetries int
	// RetryGETs controls whether GET requests are retried on transient failures.
	// Default is true.
	RetryGETs bool
}

// Client is a ZCP API HTTP client that injects auth headers and handles errors.
type Client struct {
	opts       Options
	httpClient *http.Client
}

// New creates a new Client with the given options.
// BaseURL, APIKey, and SecretKey are required.
func New(opts Options) *Client {
	if opts.Timeout == 0 {
		opts.Timeout = 30 * time.Second
	}
	if opts.DebugOut == nil {
		// lazy import to avoid circular; we use a standard writer
		opts.DebugOut = io.Discard
	}
	if opts.MaxRetries == 0 {
		opts.MaxRetries = 3
	}
	// RetryGETs defaults to true; zero value (false) means "not set, use default".
	opts.RetryGETs = true
	return &Client{
		opts: opts,
		httpClient: &http.Client{
			Timeout: opts.Timeout,
		},
	}
}

// SetDebugOut sets where debug logs go (typically os.Stderr).
func (c *Client) SetDebugOut(w io.Writer) {
	c.opts.DebugOut = w
}

// Get performs a GET request, decoding the JSON response body into result.
func (c *Client) Get(ctx context.Context, path string, query url.Values, result interface{}) error {
	return c.do(ctx, http.MethodGet, path, query, nil, result)
}

// Post performs a POST request with a JSON body, decoding the response into result.
func (c *Client) Post(ctx context.Context, path string, body interface{}, result interface{}) error {
	return c.do(ctx, http.MethodPost, path, nil, body, result)
}

// Delete performs a DELETE request.
func (c *Client) Delete(ctx context.Context, path string, query url.Values) error {
	return c.do(ctx, http.MethodDelete, path, query, nil, nil)
}

// Put performs a PUT request. query params are optional (pass nil if not needed).
// body is optional (pass nil for query-only PUTs like start/stop operations).
// result is optional (pass nil to discard response body).
func (c *Client) Put(ctx context.Context, path string, query url.Values, body interface{}, result interface{}) error {
	return c.do(ctx, http.MethodPut, path, query, body, result)
}

// isRetryable reports whether a response status code or network error warrants a retry.
func isRetryable(statusCode int, err error) bool {
	if err != nil {
		// Network errors (connection refused, timeout, etc.) are retryable.
		return true
	}
	// 429 Too Many Requests and 5xx server errors are retryable.
	return statusCode == 429 || (statusCode >= 500 && statusCode <= 599)
}

// do executes the HTTP request, retrying GET requests on transient failures.
func (c *Client) do(ctx context.Context, method, path string, query url.Values, body interface{}, result interface{}) error {
	maxAttempts := 1
	if method == http.MethodGet && c.opts.RetryGETs && c.opts.MaxRetries > 0 {
		maxAttempts = c.opts.MaxRetries + 1
	}

	var lastErr error
	for attempt := 0; attempt < maxAttempts; attempt++ {
		if attempt > 0 {
			// Exponential backoff: 1s, 2s, 4s (capped at 4s).
			backoff := time.Duration(1<<uint(attempt-1)) * time.Second
			if backoff > 4*time.Second {
				backoff = 4 * time.Second
			}
			if c.opts.Debug {
				fmt.Fprintf(c.opts.DebugOut, "[DEBUG] retry %d/%d for %s %s after %s (last error: %v)\n",
					attempt, c.opts.MaxRetries, method, path, backoff, lastErr)
			}
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(backoff):
			}
		}

		err := c.doOnce(ctx, method, path, query, body, result)
		if err == nil {
			return nil
		}

		// Determine whether this error is retryable.
		var apiErr *apierrors.APIError
		if errors.As(err, &apiErr) {
			if !isRetryable(apiErr.StatusCode, nil) {
				return err
			}
		}
		// Network/other errors are treated as retryable.
		lastErr = err
	}
	return lastErr
}

// doOnce performs a single HTTP request without any retry logic.
func (c *Client) doOnce(ctx context.Context, method, path string, query url.Values, body interface{}, result interface{}) error {
	// Build URL
	base := strings.TrimRight(c.opts.BaseURL, "/")
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	fullURL := base + path
	if len(query) > 0 {
		fullURL += "?" + query.Encode()
	}

	// Encode body
	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("encoding request body: %w", err)
		}
		bodyReader = bytes.NewReader(data)
	}

	req, err := http.NewRequestWithContext(ctx, method, fullURL, bodyReader)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}

	// Auth headers
	req.Header.Set("apikey", c.opts.APIKey)
	req.Header.Set("secretkey", c.opts.SecretKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "zcp-cli/"+version.Version)

	if c.opts.Debug {
		fmt.Fprintf(c.opts.DebugOut, "[DEBUG] %s %s\n", method, fullURL)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("reading response body: %w", err)
	}

	if c.opts.Debug {
		fmt.Fprintf(c.opts.DebugOut, "[DEBUG] %s %s -> %d\n", method, fullURL, resp.StatusCode)
		if len(respBody) > 0 && len(respBody) < 4096 {
			fmt.Fprintf(c.opts.DebugOut, "[DEBUG] response: %s\n", string(respBody))
		}
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return apierrors.ParseResponse(resp.StatusCode, respBody)
	}

	if result != nil && len(respBody) > 0 {
		if err := json.Unmarshal(respBody, result); err != nil {
			return fmt.Errorf("decoding response: %w", err)
		}
	}

	return nil
}
