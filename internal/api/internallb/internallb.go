// Package internallb provides ZCP Internal Load Balancer API operations.
package internallb

import (
	"context"
	"fmt"
	"net/url"

	"github.com/zsoftly/zcp-cli/internal/httpclient"
)

// InternalLB represents a ZCP internal load balancer.
type InternalLB struct {
	UUID            string `json:"uuid"`
	Name            string `json:"name"`
	Description     string `json:"description"`
	Status          string `json:"status"`
	IsActive        bool   `json:"isActive"`
	Algorithm       string `json:"algorithm"`
	SourceIPAddress string `json:"sourceIpAddress"`
	SourcePort      string `json:"sourcePort"`
	InstancePort    string `json:"instancePort"`
	NetworkUUID     string `json:"networkUuid"`
	ZoneUUID        string `json:"zoneUuid"`
}

// CreateRequest holds parameters for creating an internal load balancer.
type CreateRequest struct {
	Name            string `json:"name"`
	NetworkUUID     string `json:"networkUuid"`
	SourcePort      string `json:"sourceport"`
	InstancePort    string `json:"instanceport"`
	Algorithm       string `json:"algorithm"`
	SourceIPAddress string `json:"sourceIpAddress"`
	Description     string `json:"description,omitempty"`
}

type listInternalLbResponse struct {
	Count                  int          `json:"count"`
	ListInternalLbResponse []InternalLB `json:"listInternalLbResponse"`
}

// Service provides internal load balancer API operations.
type Service struct {
	client *httpclient.Client
}

// NewService creates a new internal LB Service.
func NewService(client *httpclient.Client) *Service {
	return &Service{client: client}
}

// List returns internal load balancers. zoneUUID is required; uuid and networkUUID are optional filters.
func (s *Service) List(ctx context.Context, zoneUUID, uuid, networkUUID string) ([]InternalLB, error) {
	q := url.Values{"zoneUuid": {zoneUUID}}
	if uuid != "" {
		q.Set("uuid", uuid)
	}
	if networkUUID != "" {
		q.Set("networkUuid", networkUUID)
	}
	var resp listInternalLbResponse
	if err := s.client.Get(ctx, "/restapi/internallb/internalLbList", q, &resp); err != nil {
		return nil, fmt.Errorf("listing internal LBs: %w", err)
	}
	return resp.ListInternalLbResponse, nil
}

// Create provisions a new internal load balancer.
func (s *Service) Create(ctx context.Context, req CreateRequest) (*InternalLB, error) {
	var resp listInternalLbResponse
	if err := s.client.Post(ctx, "/restapi/internallb/createInternalLb", req, &resp); err != nil {
		return nil, fmt.Errorf("creating internal LB: %w", err)
	}
	if len(resp.ListInternalLbResponse) == 0 {
		return nil, fmt.Errorf("create internal LB returned empty response")
	}
	return &resp.ListInternalLbResponse[0], nil
}

// Delete removes an internal load balancer by UUID.
func (s *Service) Delete(ctx context.Context, uuid string) error {
	if err := s.client.Delete(ctx, "/restapi/internallb/deleteInternalLb/"+uuid, nil); err != nil {
		return fmt.Errorf("deleting internal LB %s: %w", uuid, err)
	}
	return nil
}
