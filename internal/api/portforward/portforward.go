// Package portforward provides ZCP port forwarding rule API operations.
package portforward

import (
	"context"
	"fmt"
	"net/url"

	"github.com/zsoftly/zcp-cli/internal/httpclient"
)

// PortForwardRule represents a ZCP port forwarding rule.
type PortForwardRule struct {
	UUID               string `json:"uuid"`
	Status             string `json:"status"`
	IsActive           bool   `json:"isActive"`
	Protocol           string `json:"protocol"`
	PublicPort         string `json:"publicPort"`
	PublicEndPort      string `json:"publicEndPort"`
	PrivatePort        string `json:"privatePort"`
	PrivateEndPort     string `json:"privateEndPort"`
	IPAddressUUID      string `json:"ipAddressUuid"`
	VirtualMachineName string `json:"virtualMachineName"`
	ZoneUUID           string `json:"zoneUuid"`
}

// CreateRequest holds parameters for creating a port forwarding rule.
type CreateRequest struct {
	IPAddressUUID      string `json:"ipAddressUuid"`
	Protocol           string `json:"protocol"`
	PublicPort         string `json:"publicPort"`
	PublicEndPort      string `json:"publicEndPort,omitempty"`
	PrivatePort        string `json:"privatePort"`
	PrivateEndPort     string `json:"privateEndPort,omitempty"`
	VirtualMachineUUID string `json:"virtualmachineUuid"`
	NetworkUUID        string `json:"networkUuid,omitempty"`
}

type listPortForwardingResponse struct {
	Count                      int               `json:"count"`
	ListPortForwardingResponse []PortForwardRule `json:"listPortForwardingResponse"`
}

// Service provides port forwarding rule API operations.
type Service struct {
	client *httpclient.Client
}

// NewService creates a new portforward Service.
func NewService(client *httpclient.Client) *Service {
	return &Service{client: client}
}

// List returns port forwarding rules. zoneUUID is required; uuid, vmUUID, and ipAddressUUID are optional filters.
func (s *Service) List(ctx context.Context, zoneUUID, uuid, vmUUID, ipAddressUUID string) ([]PortForwardRule, error) {
	q := url.Values{"zoneUuid": {zoneUUID}}
	if uuid != "" {
		q.Set("uuid", uuid)
	}
	if vmUUID != "" {
		q.Set("vmUuid", vmUUID)
	}
	if ipAddressUUID != "" {
		q.Set("ipAddressUuid", ipAddressUUID)
	}
	var resp listPortForwardingResponse
	if err := s.client.Get(ctx, "/restapi/portforwardingrule/portForwardingRuleList", q, &resp); err != nil {
		return nil, fmt.Errorf("listing port forwarding rules: %w", err)
	}
	return resp.ListPortForwardingResponse, nil
}

// Create adds a new port forwarding rule.
func (s *Service) Create(ctx context.Context, req CreateRequest) (*PortForwardRule, error) {
	var resp listPortForwardingResponse
	if err := s.client.Post(ctx, "/restapi/portforwardingrule/createPortForwardingRule", req, &resp); err != nil {
		return nil, fmt.Errorf("creating port forwarding rule: %w", err)
	}
	if len(resp.ListPortForwardingResponse) == 0 {
		return nil, fmt.Errorf("create port forwarding rule returned empty response")
	}
	return &resp.ListPortForwardingResponse[0], nil
}

// Delete removes a port forwarding rule by UUID.
func (s *Service) Delete(ctx context.Context, uuid string) error {
	if err := s.client.Delete(ctx, "/restapi/portforwardingrule/deletePortForwardingRule/"+uuid, nil); err != nil {
		return fmt.Errorf("deleting port forwarding rule %s: %w", uuid, err)
	}
	return nil
}
