// Package monitoring provides ZCP monitoring API operations (STKCNSL).
package monitoring

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/zsoftly/zcp-cli/internal/httpclient"
)

// envelope is the STKCNSL response wrapper: {"status":"Success","data":...}
type envelope struct {
	Status string          `json:"status"`
	Data   json.RawMessage `json:"data"`
}

// GlobalResource represents a single resource entry from the global monitoring overview.
type GlobalResource struct {
	Name       string  `json:"name"`
	Total      float64 `json:"total"`
	Used       float64 `json:"used"`
	Free       float64 `json:"free"`
	Unit       string  `json:"unit"`
	Percentage float64 `json:"percentage"`
}

// MetricPoint represents a single time-series data point for VM metrics.
type MetricPoint struct {
	Timestamp string  `json:"timestamp"`
	Value     float64 `json:"value"`
	Unit      string  `json:"unit"`
}

// DiskMetricPoint represents a single time-series data point for disk metrics
// that have separate read and write values.
type DiskMetricPoint struct {
	Timestamp string  `json:"timestamp"`
	Read      float64 `json:"read"`
	Write     float64 `json:"write"`
	Unit      string  `json:"unit"`
}

// NetworkMetricPoint represents a single time-series data point for network metrics
// that have separate incoming and outgoing values.
type NetworkMetricPoint struct {
	Timestamp string  `json:"timestamp"`
	Incoming  float64 `json:"incoming"`
	Outgoing  float64 `json:"outgoing"`
	Unit      string  `json:"unit"`
}

// Service provides monitoring API operations.
type Service struct {
	client *httpclient.Client
}

// NewService creates a new monitoring Service.
func NewService(client *httpclient.Client) *Service {
	return &Service{client: client}
}

// decodeEnvelope unmarshals the STKCNSL response envelope and returns the inner data.
func decodeEnvelope(raw json.RawMessage) (json.RawMessage, error) {
	var env envelope
	if err := json.Unmarshal(raw, &env); err != nil {
		return nil, fmt.Errorf("decoding response envelope: %w", err)
	}
	if env.Status != "Success" {
		return nil, fmt.Errorf("API returned status %q", env.Status)
	}
	return env.Data, nil
}

// Global returns the global resource monitoring overview.
func (s *Service) Global(ctx context.Context) ([]GlobalResource, error) {
	var raw json.RawMessage
	if err := s.client.Get(ctx, "/monitoring/global", url.Values{}, &raw); err != nil {
		return nil, fmt.Errorf("fetching global monitoring: %w", err)
	}
	data, err := decodeEnvelope(raw)
	if err != nil {
		return nil, err
	}
	var resources []GlobalResource
	if err := json.Unmarshal(data, &resources); err != nil {
		return nil, fmt.Errorf("decoding global resources: %w", err)
	}
	return resources, nil
}

// CPUUsage returns CPU usage metrics for a VM.
func (s *Service) CPUUsage(ctx context.Context, vmSlug string) ([]MetricPoint, error) {
	var raw json.RawMessage
	path := fmt.Sprintf("/monitoring/%s/cpu-usage", url.PathEscape(vmSlug))
	if err := s.client.Get(ctx, path, url.Values{}, &raw); err != nil {
		return nil, fmt.Errorf("fetching CPU usage for %s: %w", vmSlug, err)
	}
	data, err := decodeEnvelope(raw)
	if err != nil {
		return nil, err
	}
	var points []MetricPoint
	if err := json.Unmarshal(data, &points); err != nil {
		return nil, fmt.Errorf("decoding CPU usage: %w", err)
	}
	return points, nil
}

// MemoryUsage returns memory usage metrics for a VM.
func (s *Service) MemoryUsage(ctx context.Context, vmSlug string) ([]MetricPoint, error) {
	var raw json.RawMessage
	path := fmt.Sprintf("/monitoring/%s/memory-usage", url.PathEscape(vmSlug))
	if err := s.client.Get(ctx, path, url.Values{}, &raw); err != nil {
		return nil, fmt.Errorf("fetching memory usage for %s: %w", vmSlug, err)
	}
	data, err := decodeEnvelope(raw)
	if err != nil {
		return nil, err
	}
	var points []MetricPoint
	if err := json.Unmarshal(data, &points); err != nil {
		return nil, fmt.Errorf("decoding memory usage: %w", err)
	}
	return points, nil
}

// DiskReadWrite returns disk read/write metrics for a VM.
func (s *Service) DiskReadWrite(ctx context.Context, vmSlug string) ([]DiskMetricPoint, error) {
	var raw json.RawMessage
	path := fmt.Sprintf("/monitoring/%s/disk-read-write", url.PathEscape(vmSlug))
	if err := s.client.Get(ctx, path, url.Values{}, &raw); err != nil {
		return nil, fmt.Errorf("fetching disk read/write for %s: %w", vmSlug, err)
	}
	data, err := decodeEnvelope(raw)
	if err != nil {
		return nil, err
	}
	var points []DiskMetricPoint
	if err := json.Unmarshal(data, &points); err != nil {
		return nil, fmt.Errorf("decoding disk read/write: %w", err)
	}
	return points, nil
}

// DiskIOReadWrite returns disk IO read/write metrics for a VM.
func (s *Service) DiskIOReadWrite(ctx context.Context, vmSlug string) ([]DiskMetricPoint, error) {
	var raw json.RawMessage
	path := fmt.Sprintf("/monitoring/%s/disk-io-read-write", url.PathEscape(vmSlug))
	if err := s.client.Get(ctx, path, url.Values{}, &raw); err != nil {
		return nil, fmt.Errorf("fetching disk IO for %s: %w", vmSlug, err)
	}
	data, err := decodeEnvelope(raw)
	if err != nil {
		return nil, err
	}
	var points []DiskMetricPoint
	if err := json.Unmarshal(data, &points); err != nil {
		return nil, fmt.Errorf("decoding disk IO: %w", err)
	}
	return points, nil
}

// NetworkTraffic returns network traffic metrics for a VM.
func (s *Service) NetworkTraffic(ctx context.Context, vmSlug string) ([]NetworkMetricPoint, error) {
	var raw json.RawMessage
	path := fmt.Sprintf("/monitoring/%s/network-traffic", url.PathEscape(vmSlug))
	if err := s.client.Get(ctx, path, url.Values{}, &raw); err != nil {
		return nil, fmt.Errorf("fetching network traffic for %s: %w", vmSlug, err)
	}
	data, err := decodeEnvelope(raw)
	if err != nil {
		return nil, err
	}
	var points []NetworkMetricPoint
	if err := json.Unmarshal(data, &points); err != nil {
		return nil, fmt.Errorf("decoding network traffic: %w", err)
	}
	return points, nil
}

// Charts returns the monitoring charts data.
func (s *Service) Charts(ctx context.Context) (json.RawMessage, error) {
	var raw json.RawMessage
	if err := s.client.Get(ctx, "/monitoring/charts", url.Values{}, &raw); err != nil {
		return nil, fmt.Errorf("fetching monitoring charts: %w", err)
	}
	data, err := decodeEnvelope(raw)
	if err != nil {
		return nil, err
	}
	return data, nil
}
