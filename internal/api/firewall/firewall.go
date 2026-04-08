// Package firewall provides ZCP firewall rule API operations.
package firewall

import (
	"context"
	"fmt"

	"github.com/zsoftly/zcp-cli/internal/httpclient"
)

// FirewallRule represents a ZCP firewall rule from the STKCNSL API.
type FirewallRule struct {
	ID                  string      `json:"id"`
	RuleID              string      `json:"rule_id"`
	Protocol            string      `json:"protocol"`
	StartPort           interface{} `json:"start_port"`
	EndPort             interface{} `json:"end_port"`
	CIDRList            string      `json:"cidr_list"`
	DestinationCIDRList string      `json:"destination_cidr_list"`
	ICMPType            string      `json:"icmp_type"`
	ICMPCode            string      `json:"icmp_code"`
	State               string      `json:"state"`
	CreatedAt           string      `json:"created_at"`
	UpdatedAt           string      `json:"updated_at"`
}

// CreateRequest holds parameters for creating a firewall rule.
type CreateRequest struct {
	Protocol            string      `json:"protocol"`
	CIDRList            string      `json:"cidr_list,omitempty"`
	DestinationCIDRList string      `json:"destination_cidr_list,omitempty"`
	StartPort           interface{} `json:"start_port,omitempty"`
	EndPort             interface{} `json:"end_port,omitempty"`
}

// listResponse is the STKCNSL envelope for firewall rule lists.
type listResponse struct {
	Status  string         `json:"status"`
	Message string         `json:"message"`
	Data    []FirewallRule `json:"data"`
}

// singleResponse is the STKCNSL envelope for a single firewall rule response.
type singleResponse struct {
	Status  string       `json:"status"`
	Message string       `json:"message"`
	Data    FirewallRule `json:"data"`
}

// Service provides firewall rule API operations.
type Service struct {
	client *httpclient.Client
}

// NewService creates a new firewall Service.
func NewService(client *httpclient.Client) *Service {
	return &Service{client: client}
}

// List returns firewall rules for a public IP address.
// ipSlug is the IP address slug (e.g. "1036521143").
func (s *Service) List(ctx context.Context, ipSlug string) ([]FirewallRule, error) {
	var resp listResponse
	if err := s.client.Get(ctx, "/ipaddresses/"+ipSlug+"/firewall-rules", nil, &resp); err != nil {
		return nil, fmt.Errorf("listing firewall rules for IP %s: %w", ipSlug, err)
	}
	return resp.Data, nil
}

// Create adds a new firewall rule on a public IP address.
// ipSlug is the IP address slug.
func (s *Service) Create(ctx context.Context, ipSlug string, req CreateRequest) (*FirewallRule, error) {
	var resp singleResponse
	if err := s.client.Post(ctx, "/ipaddresses/"+ipSlug+"/firewall-rules", req, &resp); err != nil {
		return nil, fmt.Errorf("creating firewall rule for IP %s: %w", ipSlug, err)
	}
	return &resp.Data, nil
}

// Delete removes a firewall rule by ID from a public IP address.
// ipSlug is the IP address slug; ruleID is the firewall rule ID.
func (s *Service) Delete(ctx context.Context, ipSlug, ruleID string) error {
	if err := s.client.Delete(ctx, "/ipaddresses/"+ipSlug+"/firewall-rules/"+ruleID, nil); err != nil {
		return fmt.Errorf("deleting firewall rule %s for IP %s: %w", ruleID, ipSlug, err)
	}
	return nil
}
