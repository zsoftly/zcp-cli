// Package network provides ZCP network API operations.
package network

import (
	"context"
	"fmt"

	"github.com/zsoftly/zcp-cli/internal/httpclient"
)

// Network represents a ZCP isolated network.
type Network struct {
	ID          string `json:"id"`
	Slug        string `json:"slug"`
	Name        string `json:"name"`
	Status      string `json:"status"`
	NetworkType string `json:"network_type"`
	Gateway     string `json:"gateway"`
	CIDR        string `json:"cidr"`
	Netmask     string `json:"netmask"`
	DNS1        string `json:"dns1"`
	DNS2        string `json:"dns2"`
	ZoneSlug    string `json:"zone_slug"`
	ZoneName    string `json:"zone_name"`
	Category    string `json:"category"`
	Description string `json:"description"`
	IsDefault   bool   `json:"is_default"`
}

// Category represents a network category (offering).
type Category struct {
	ID          string `json:"id"`
	Slug        string `json:"slug"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

// EgressRule represents a network egress firewall rule.
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

// CreateRequest holds parameters for creating a network.
type CreateRequest struct {
	Name         string `json:"name"`
	CategorySlug string `json:"category_slug"`
	ZoneSlug     string `json:"zone_slug,omitempty"`
	Gateway      string `json:"gateway,omitempty"`
	Netmask      string `json:"netmask,omitempty"`
	Description  string `json:"description,omitempty"`
}

// UpdateRequest holds parameters for updating a network.
type UpdateRequest struct {
	Name        string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`
}

// CreateEgressRuleRequest holds parameters for creating an egress rule.
type CreateEgressRuleRequest struct {
	Protocol  string `json:"protocol"`
	StartPort string `json:"start_port,omitempty"`
	EndPort   string `json:"end_port,omitempty"`
	CIDR      string `json:"cidr,omitempty"`
	ICMPType  string `json:"icmp_type,omitempty"`
	ICMPCode  string `json:"icmp_code,omitempty"`
}

// apiResponse is the STKCNSL standard envelope.
type apiResponse struct {
	Status string `json:"status"`
	Data   any    `json:"-"`
}

type listNetworkResponse struct {
	Status string    `json:"status"`
	Data   []Network `json:"data"`
}

type singleNetworkResponse struct {
	Status string  `json:"status"`
	Data   Network `json:"data"`
}

type listCategoryResponse struct {
	Status string     `json:"status"`
	Data   []Category `json:"data"`
}

type listEgressRuleResponse struct {
	Status string       `json:"status"`
	Data   []EgressRule `json:"data"`
}

type singleEgressRuleResponse struct {
	Status string     `json:"status"`
	Data   EgressRule `json:"data"`
}

// Service provides network API operations.
type Service struct {
	client *httpclient.Client
}

// NewService creates a new network Service.
func NewService(client *httpclient.Client) *Service {
	return &Service{client: client}
}

// List returns all isolated networks.
func (s *Service) List(ctx context.Context) ([]Network, error) {
	var resp listNetworkResponse
	if err := s.client.Get(ctx, "/networks", nil, &resp); err != nil {
		return nil, fmt.Errorf("listing networks: %w", err)
	}
	return resp.Data, nil
}

// Create provisions a new isolated network.
func (s *Service) Create(ctx context.Context, req CreateRequest) (*Network, error) {
	var resp singleNetworkResponse
	if err := s.client.Post(ctx, "/networks", req, &resp); err != nil {
		return nil, fmt.Errorf("creating network: %w", err)
	}
	return &resp.Data, nil
}

// Update modifies a network's mutable attributes.
func (s *Service) Update(ctx context.Context, slug string, req UpdateRequest) (*Network, error) {
	var resp singleNetworkResponse
	if err := s.client.Put(ctx, "/networks/"+slug, nil, req, &resp); err != nil {
		return nil, fmt.Errorf("updating network %s: %w", slug, err)
	}
	return &resp.Data, nil
}

// ListCategories returns available network categories (offerings).
func (s *Service) ListCategories(ctx context.Context) ([]Category, error) {
	var resp listCategoryResponse
	if err := s.client.Get(ctx, "/network/categories", nil, &resp); err != nil {
		return nil, fmt.Errorf("listing network categories: %w", err)
	}
	return resp.Data, nil
}

// ListEgressRules returns egress firewall rules for a network.
func (s *Service) ListEgressRules(ctx context.Context, networkSlug string) ([]EgressRule, error) {
	var resp listEgressRuleResponse
	if err := s.client.Get(ctx, "/networks/"+networkSlug+"/egress-firewall-rules", nil, &resp); err != nil {
		return nil, fmt.Errorf("listing egress rules for network %s: %w", networkSlug, err)
	}
	return resp.Data, nil
}

// CreateEgressRule adds an egress firewall rule to a network.
func (s *Service) CreateEgressRule(ctx context.Context, networkSlug string, req CreateEgressRuleRequest) (*EgressRule, error) {
	var resp singleEgressRuleResponse
	if err := s.client.Post(ctx, "/networks/"+networkSlug+"/egress-firewall-rules", req, &resp); err != nil {
		return nil, fmt.Errorf("creating egress rule for network %s: %w", networkSlug, err)
	}
	return &resp.Data, nil
}

// DeleteEgressRule removes an egress firewall rule from a network.
func (s *Service) DeleteEgressRule(ctx context.Context, networkSlug string, ruleID string) error {
	path := fmt.Sprintf("/networks/%s/egress-firewall-rules/%s", networkSlug, ruleID)
	if err := s.client.Delete(ctx, path, nil); err != nil {
		return fmt.Errorf("deleting egress rule %s for network %s: %w", ruleID, networkSlug, err)
	}
	return nil
}
