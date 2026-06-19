// Package network provides ZCP network API operations.
package network

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"github.com/zsoftly/zcp-cli/pkg/httpclient"
)

// Network represents a ZCP network (isolated or VPC subnet).
type Network struct {
	ID           string `json:"id"`
	Slug         string `json:"slug"`
	Name         string `json:"name"`
	Status       bool   `json:"status"`
	NetworkType  string `json:"type"`
	Gateway      string `json:"gateway"`
	CIDR         string `json:"cidr"`
	Netmask      string `json:"netmask"`
	DNS1         string `json:"dns1"`
	DNS2         string `json:"dns2"`
	ZoneSlug     string `json:"zone_slug"`
	ZoneName     string `json:"zone_name"`
	Category     string `json:"category"`
	Description  string `json:"description"`
	IsDefault    bool   `json:"is_default"`
	VPC          string `json:"vpc"`
	BillingCycle string `json:"billing_cycle"`
}

// UnmarshalJSON provides backward-compatible decoding for Network.
// The live API returns status as bool and network type under "type"; older
// deployments may return status as string ("Active"/"Inactive") and/or use
// "network_type" as the key.
func (n *Network) UnmarshalJSON(b []byte) error {
	type networkRaw struct {
		ID             string          `json:"id"`
		Slug           string          `json:"slug"`
		Name           string          `json:"name"`
		Status         json.RawMessage `json:"status"`
		NetworkType    json.RawMessage `json:"type"`
		NetworkTypeAlt json.RawMessage `json:"network_type"`
		Gateway        string          `json:"gateway"`
		CIDR           string          `json:"cidr"`
		Netmask        string          `json:"netmask"`
		DNS1           string          `json:"dns1"`
		DNS2           string          `json:"dns2"`
		ZoneSlug       string          `json:"zone_slug"`
		ZoneName       string          `json:"zone_name"`
		Category       string          `json:"category"`
		Description    string          `json:"description"`
		IsDefault      json.RawMessage `json:"is_default"`
		VPC            string          `json:"vpc"`
		BillingCycle   string          `json:"billing_cycle"`
	}
	var raw networkRaw
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	n.ID = raw.ID
	n.Slug = raw.Slug
	n.Name = raw.Name
	n.Gateway = raw.Gateway
	n.CIDR = raw.CIDR
	n.Netmask = raw.Netmask
	n.DNS1 = raw.DNS1
	n.DNS2 = raw.DNS2
	n.ZoneSlug = raw.ZoneSlug
	n.ZoneName = raw.ZoneName
	n.Category = raw.Category
	n.Description = raw.Description
	n.VPC = raw.VPC
	n.BillingCycle = raw.BillingCycle

	// The create endpoint returns is_default as 0/1; list returns true/false.
	if len(raw.IsDefault) > 0 {
		d := strings.Trim(string(raw.IsDefault), `"`)
		n.IsDefault = d == "true" || d == "1"
	}

	if len(raw.Status) > 0 {
		s := strings.Trim(string(raw.Status), `"`)
		n.Status = s == "true" || strings.EqualFold(s, "active") || s == "1"
	}

	typeRaw := raw.NetworkType
	if len(typeRaw) == 0 {
		typeRaw = raw.NetworkTypeAlt
	}
	if len(typeRaw) > 0 {
		if err := json.Unmarshal(typeRaw, &n.NetworkType); err != nil {
			return fmt.Errorf("decoding network type: %w", err)
		}
	}

	return nil
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
	Name          string `json:"name"`
	CategorySlug  string `json:"category_slug,omitempty"`
	ZoneSlug      string `json:"zone_slug,omitempty"`
	Gateway       string `json:"gateway,omitempty"`
	Netmask       string `json:"netmask,omitempty"`
	Description   string `json:"description,omitempty"`
	CloudProvider string `json:"cloud_provider"`
	Region        string `json:"region"`
	Project       string `json:"project"`
	VPC           string `json:"vpc,omitempty"`
	BillingCycle  string `json:"billing_cycle,omitempty"`
	Type          string `json:"type,omitempty"`
	NetworkPlan   string `json:"network_plan,omitempty"`
}

// Detail holds the provider-side state of a network as returned by
// GET /networks/{slug}. The interesting fields live under "meta", which is
// the raw CloudStack network view.
type Detail struct {
	ID        string `json:"id"`
	Slug      string `json:"slug"`
	Name      string `json:"name"`
	NetworkID string `json:"network_id"`
	Meta      struct {
		CIDR     string `json:"cidr"`
		Netmask  string `json:"netmask"`
		Gateway  string `json:"gateway"`
		Type     string `json:"type"`
		State    string `json:"state"`
		VPCID    string `json:"vpc_id"`
		ACLName  string `json:"acl_name"`
		ZoneName string `json:"zone_name"`
	} `json:"meta"`
}

// UpdateRequest holds parameters for updating a network.
type UpdateRequest struct {
	Name        string  `json:"name,omitempty"`
	Description *string `json:"description,omitempty"`
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
func (s *Service) List(ctx context.Context, region, project string) ([]Network, error) {
	var resp listNetworkResponse
	q := url.Values{}
	if region != "" {
		q.Set("filter[region]", region)
	}
	if project != "" {
		q.Set("filter[project]", project)
	}
	if err := s.client.Get(ctx, "/networks", q, &resp); err != nil {
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

// Get returns a single network by slug within the optional region/project scope.
func (s *Service) Get(ctx context.Context, slug, region, project string) (*Network, error) {
	networks, err := s.List(ctx, region, project)
	if err != nil {
		return nil, err
	}
	for i := range networks {
		if networks[i].Slug == slug {
			return &networks[i], nil
		}
	}
	return nil, fmt.Errorf("network %q not found", slug)
}

// GetDetail returns the provider-side detail of a network from
// GET /networks/{slug}, including its CIDR, state, and VPC membership.
func (s *Service) GetDetail(ctx context.Context, slug string) (*Detail, error) {
	type detailResponse struct {
		Status string `json:"status"`
		Data   Detail `json:"data"`
	}
	var resp detailResponse
	if err := s.client.Get(ctx, "/networks/"+slug, nil, &resp); err != nil {
		return nil, fmt.Errorf("getting network %s: %w", slug, err)
	}
	if resp.Data.Slug == "" {
		return nil, fmt.Errorf("network %q not found", slug)
	}
	return &resp.Data, nil
}

// Update modifies a network's mutable attributes.
func (s *Service) Update(ctx context.Context, slug string, req UpdateRequest) (*Network, error) {
	type rawEnv struct {
		Status string          `json:"status"`
		Data   json.RawMessage `json:"data"`
	}
	var env rawEnv
	if err := s.client.Put(ctx, "/networks/"+slug, nil, req, &env); err != nil {
		return nil, fmt.Errorf("updating network %s: %w", slug, err)
	}
	// The Update API may return data:null or data:[null] — fall back to GET.
	raw := string(env.Data)
	if len(env.Data) > 0 && raw != "null" && raw != "[null]" {
		var n Network
		if err := json.Unmarshal(env.Data, &n); err == nil && n.Slug != "" {
			return &n, nil
		}
	}
	return s.Get(ctx, slug, "", "")
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

// Delete removes an isolated network. The network must have no VMs attached.
// Its SOURCE-NAT IP is released automatically by CloudStack on deletion.
func (s *Service) Delete(ctx context.Context, slug string) error {
	if err := s.client.Delete(ctx, "/networks/"+slug, nil); err != nil {
		return fmt.Errorf("deleting network %s: %w", slug, err)
	}
	return nil
}
