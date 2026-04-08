// Package dashboard provides STKCNSL dashboard and service cancellation API operations.
package dashboard

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/zsoftly/zcp-cli/internal/httpclient"
)

// envelope is the standard STKCNSL response wrapper.
// All STKCNSL endpoints return {"status": "...", "data": ...}.
type envelope struct {
	Status string          `json:"status"`
	Data   json.RawMessage `json:"data"`
}

// ServiceCounts holds the per-service resource counts returned by the
// analytics/account/services/counts endpoint.
type ServiceCounts struct {
	Instance     int `json:"instance"`
	Kubernetes   int `json:"kubernetes"`
	Volume       int `json:"volume"`
	Snapshot     int `json:"snapshot"`
	Network      int `json:"network"`
	VPC          int `json:"vpc"`
	PublicIP     int `json:"publicIp"`
	Firewall     int `json:"firewall"`
	LoadBalancer int `json:"loadBalancer"`
	VPN          int `json:"vpn"`
	SSHKey       int `json:"sshKey"`
	Template     int `json:"template"`
}

// CancelResponse holds the result of a service cancellation request.
type CancelResponse struct {
	Message string `json:"message"`
}

// Service provides dashboard API operations against the STKCNSL API.
type Service struct {
	client *httpclient.Client
}

// NewService creates a new dashboard Service.
func NewService(client *httpclient.Client) *Service {
	return &Service{client: client}
}

// GetServiceCounts returns a summary of active resource counts for the account.
func (s *Service) GetServiceCounts(ctx context.Context) (*ServiceCounts, error) {
	var env envelope
	if err := s.client.Get(ctx, "/analytics/account/services/counts", url.Values{}, &env); err != nil {
		return nil, fmt.Errorf("getting service counts: %w", err)
	}

	if env.Status != "Success" {
		return nil, fmt.Errorf("unexpected response status: %s", env.Status)
	}

	var counts ServiceCounts
	if err := json.Unmarshal(env.Data, &counts); err != nil {
		return nil, fmt.Errorf("decoding service counts: %w", err)
	}
	return &counts, nil
}

// CancelServiceRequest holds the request body for service cancellation.
type CancelServiceRequest struct {
	Reason string `json:"reason"`
}

// CancelService submits a cancellation request for the given service slug.
func (s *Service) CancelService(ctx context.Context, serviceSlug, reason string) (*CancelResponse, error) {
	if serviceSlug == "" {
		return nil, fmt.Errorf("service slug is required")
	}

	path := fmt.Sprintf("/billing/service-cancel-requests/%s", url.PathEscape(serviceSlug))
	body := CancelServiceRequest{Reason: reason}

	var env envelope
	if err := s.client.Post(ctx, path, body, &env); err != nil {
		return nil, fmt.Errorf("cancelling service %s: %w", serviceSlug, err)
	}

	if env.Status != "Success" {
		return nil, fmt.Errorf("unexpected response status: %s", env.Status)
	}

	var resp CancelResponse
	if err := json.Unmarshal(env.Data, &resp); err != nil {
		return nil, fmt.Errorf("decoding cancel response: %w", err)
	}
	return &resp, nil
}
