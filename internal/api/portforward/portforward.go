// Package portforward provides ZCP port forwarding rule API operations.
package portforward

import (
	"context"
	"fmt"

	"github.com/zsoftly/zcp-cli/internal/httpclient"
)

// PortForwardRule represents a ZCP port forwarding rule from the STKCNSL API.
type PortForwardRule struct {
	ID               string `json:"id"`
	RuleID           string `json:"rule_id"`
	Protocol         string `json:"protocol"`
	PublicStartPort  string `json:"public_start_port"`
	PublicEndPort    string `json:"public_end_port"`
	PrivateStartPort string `json:"private_start_port"`
	PrivateEndPort   string `json:"private_end_port"`
	VirtualMachine   string `json:"virtual_machine"`
	State            string `json:"state"`
	CreatedAt        string `json:"created_at"`
	UpdatedAt        string `json:"updated_at"`
}

// CreateRequest holds parameters for creating a port forwarding rule.
type CreateRequest struct {
	Protocol         string `json:"protocol"`
	PublicStartPort  string `json:"public_start_port"`
	PublicEndPort    string `json:"public_end_port,omitempty"`
	PrivateStartPort string `json:"private_start_port"`
	PrivateEndPort   string `json:"private_end_port,omitempty"`
	VirtualMachine   string `json:"virtual_machine"`
}

// listResponse is the STKCNSL envelope for port forwarding rule lists.
type listResponse struct {
	Status  string            `json:"status"`
	Message string            `json:"message"`
	Data    []PortForwardRule `json:"data"`
}

// singleResponse is the STKCNSL envelope for a single port forwarding rule response.
type singleResponse struct {
	Status  string          `json:"status"`
	Message string          `json:"message"`
	Data    PortForwardRule `json:"data"`
}

// Service provides port forwarding rule API operations.
type Service struct {
	client *httpclient.Client
}

// NewService creates a new portforward Service.
func NewService(client *httpclient.Client) *Service {
	return &Service{client: client}
}

// List returns port forwarding rules for a public IP address.
// ipSlug is the IP address slug (e.g. "1036521143").
func (s *Service) List(ctx context.Context, ipSlug string) ([]PortForwardRule, error) {
	var resp listResponse
	if err := s.client.Get(ctx, "/ipaddresses/"+ipSlug+"/port-forwarding-rules", nil, &resp); err != nil {
		return nil, fmt.Errorf("listing port forwarding rules for IP %s: %w", ipSlug, err)
	}
	return resp.Data, nil
}

// Create adds a new port forwarding rule on a public IP address.
// ipSlug is the IP address slug.
func (s *Service) Create(ctx context.Context, ipSlug string, req CreateRequest) (*PortForwardRule, error) {
	var resp singleResponse
	if err := s.client.Post(ctx, "/ipaddresses/"+ipSlug+"/port-forwarding-rules", req, &resp); err != nil {
		return nil, fmt.Errorf("creating port forwarding rule for IP %s: %w", ipSlug, err)
	}
	return &resp.Data, nil
}

// Delete removes a port forwarding rule by ID from a public IP address.
// ipSlug is the IP address slug; ruleID is the port forwarding rule ID.
func (s *Service) Delete(ctx context.Context, ipSlug, ruleID string) error {
	if err := s.client.Delete(ctx, "/ipaddresses/"+ipSlug+"/port-forwarding-rules/"+ruleID, nil); err != nil {
		return fmt.Errorf("deleting port forwarding rule %s for IP %s: %w", ruleID, ipSlug, err)
	}
	return nil
}
