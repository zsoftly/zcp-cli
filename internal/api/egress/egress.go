// Package egress provides ZCP egress rule API operations.
//
// In the STKCNSL API, egress rules are nested under networks:
//
//	GET    /networks/{SLUG}/egress-firewall-rules
//	POST   /networks/{SLUG}/egress-firewall-rules
//	DELETE /networks/{SLUG}/egress-firewall-rules/{ID}
//
// This package delegates to the network package's egress methods but preserves
// the Service/NewService pattern for backward compatibility with the commands layer.
package egress

import (
	"context"
	"fmt"

	"github.com/zsoftly/zcp-cli/internal/httpclient"
)

// EgressRule represents a ZCP egress firewall rule.
type EgressRule struct {
	ID        string `json:"id"`
	Protocol  string `json:"protocol"`
	StartPort string `json:"start_port"`
	EndPort   string `json:"end_port"`
	CIDR      string `json:"cidr"`
	ICMPType  string `json:"icmp_type"`
	ICMPCode  string `json:"icmp_code"`
	Status    string `json:"status"`
}

// CreateRequest holds parameters for creating an egress rule.
type CreateRequest struct {
	NetworkSlug string `json:"-"`
	Protocol    string `json:"protocol"`
	StartPort   string `json:"start_port,omitempty"`
	EndPort     string `json:"end_port,omitempty"`
	CIDR        string `json:"cidr,omitempty"`
	ICMPType    string `json:"icmp_type,omitempty"`
	ICMPCode    string `json:"icmp_code,omitempty"`
}

type listEgressRuleResponse struct {
	Status string       `json:"status"`
	Data   []EgressRule `json:"data"`
}

type singleEgressRuleResponse struct {
	Status string     `json:"status"`
	Data   EgressRule `json:"data"`
}

// Service provides egress rule API operations.
type Service struct {
	client *httpclient.Client
}

// NewService creates a new egress Service.
func NewService(client *httpclient.Client) *Service {
	return &Service{client: client}
}

// List returns egress rules for a network identified by slug.
func (s *Service) List(ctx context.Context, networkSlug string) ([]EgressRule, error) {
	var resp listEgressRuleResponse
	if err := s.client.Get(ctx, "/networks/"+networkSlug+"/egress-firewall-rules", nil, &resp); err != nil {
		return nil, fmt.Errorf("listing egress rules for network %s: %w", networkSlug, err)
	}
	return resp.Data, nil
}

// Create adds a new egress rule to a network.
func (s *Service) Create(ctx context.Context, req CreateRequest) (*EgressRule, error) {
	var resp singleEgressRuleResponse
	if err := s.client.Post(ctx, "/networks/"+req.NetworkSlug+"/egress-firewall-rules", req, &resp); err != nil {
		return nil, fmt.Errorf("creating egress rule for network %s: %w", req.NetworkSlug, err)
	}
	return &resp.Data, nil
}

// Delete removes an egress rule by ID from the given network.
func (s *Service) Delete(ctx context.Context, networkSlug string, ruleID string) error {
	path := fmt.Sprintf("/networks/%s/egress-firewall-rules/%s", networkSlug, ruleID)
	if err := s.client.Delete(ctx, path, nil); err != nil {
		return fmt.Errorf("deleting egress rule %s for network %s: %w", ruleID, networkSlug, err)
	}
	return nil
}
