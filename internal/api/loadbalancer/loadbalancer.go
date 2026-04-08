// Package loadbalancer provides ZCP Load Balancer API operations (STKCNSL).
package loadbalancer

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/zsoftly/zcp-cli/internal/httpclient"
)

// LoadBalancer represents a ZCP load balancer from the STKCNSL API.
type LoadBalancer struct {
	ID                   string         `json:"id"`
	Name                 string         `json:"name"`
	Slug                 string         `json:"slug"`
	State                string         `json:"state"`
	UserID               string         `json:"user_id"`
	AccountID            string         `json:"account_id"`
	ProjectID            string         `json:"project_id"`
	RegionID             string         `json:"region_id"`
	CloudProviderID      string         `json:"cloud_provider_id"`
	CloudProviderSetupID string         `json:"cloud_provider_setup_id"`
	RequestStatus        bool           `json:"request_status"`
	CreatedAt            string         `json:"created_at"`
	UpdatedAt            string         `json:"updated_at"`
	DeletedAt            *string        `json:"deleted_at"`
	FrozenAt             *string        `json:"frozen_at"`
	SuspendedAt          *string        `json:"suspended_at"`
	TerminatedAt         *string        `json:"terminated_at"`
	Rules                []Rule         `json:"load_balancer_rules,omitempty"`
	IPAddress            *IPAddress     `json:"ipaddress,omitempty"`
	Region               *Region        `json:"region,omitempty"`
	Project              *Project       `json:"project,omitempty"`
	CloudProvider        *CloudProvider `json:"cloud_provider,omitempty"`
	ServiceName          string         `json:"service_name"`
	ServiceDisplayName   string         `json:"service_display_name"`
	AllTimeConsumption   float64        `json:"all_time_consumption"`
}

// Rule represents a load balancer rule.
type Rule struct {
	ID                  string `json:"id"`
	LoadBalancerID      string `json:"load_balancer_id"`
	Name                string `json:"name"`
	PublicPort          string `json:"public_port"`
	PrivatePort         string `json:"private_port"`
	Protocol            string `json:"protocol"`
	Algorithm           string `json:"algorithm"`
	StickyMethod        string `json:"sticky_method"`
	EnableTLSProtocol   bool   `json:"enable_tls_protocol"`
	EnableProxyProtocol bool   `json:"enable_proxy_protocol"`
	CreatedAt           string `json:"created_at"`
	UpdatedAt           string `json:"updated_at"`
}

// IPAddress is a nested IP address on a load balancer.
type IPAddress struct {
	ID        string `json:"id"`
	IPAddress string `json:"ip_address"`
	Slug      string `json:"slug"`
}

// Region is a nested region on a load balancer.
type Region struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Slug    string `json:"slug"`
	Country string `json:"country"`
}

// Project is a nested project on a load balancer.
type Project struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Slug string `json:"slug"`
}

// CloudProvider is a nested cloud provider on a load balancer.
type CloudProvider struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	DisplayName string `json:"display_name"`
	Slug        string `json:"slug"`
}

// CreateRuleSpec describes a single rule to include when creating a load balancer.
type CreateRuleSpec struct {
	Name                string         `json:"name"`
	PublicPort          string         `json:"public_port"`
	PrivatePort         string         `json:"private_port"`
	Protocol            string         `json:"protocol"`
	Algorithm           string         `json:"algorithm"`
	StickyMethod        string         `json:"sticky_method,omitempty"`
	EnableTLSProtocol   bool           `json:"enable_tls_protocol"`
	EnableProxyProtocol bool           `json:"enable_proxy_protocol"`
	VirtualMachines     []VMAttachment `json:"virtual_machines"`
}

// VMAttachment identifies a VM to attach to a load balancer rule.
type VMAttachment struct {
	Slug      string `json:"slug"`
	IPAddress string `json:"ipaddress,omitempty"`
}

// CreateRequest holds parameters for creating a load balancer.
type CreateRequest struct {
	Name          string           `json:"name"`
	CloudProvider string           `json:"cloud_provider"`
	Project       string           `json:"project"`
	Region        string           `json:"region"`
	Network       string           `json:"network"`
	Plan          string           `json:"plan"`
	BillingCycle  string           `json:"billing_cycle"`
	AcquireNewIP  bool             `json:"aquire_new_ip"`
	IPAddress     *string          `json:"ipaddress"`
	Rules         []CreateRuleSpec `json:"rules"`
	IsVMSnapshot  bool             `json:"is_vm_snapshot"`
	Coupon        *string          `json:"coupon"`
}

// CreateRuleRequest holds parameters for adding rules to an existing load balancer.
type CreateRuleRequest struct {
	Rules []CreateRuleSpec `json:"rules"`
}

// AttachVMRequest holds parameters for attaching VMs to a load balancer rule.
type AttachVMRequest struct {
	VirtualMachines []string `json:"virtual_machines"`
	CloudProvider   string   `json:"cloud_provider"`
	Region          string   `json:"region"`
	Project         string   `json:"project"`
}

// listResponse is the paginated envelope returned by GET /load-balancers.
type listResponse struct {
	Status      string          `json:"status"`
	Message     string          `json:"message"`
	CurrentPage int             `json:"current_page"`
	Data        json.RawMessage `json:"data"`
	Total       int             `json:"total"`
}

// singleResponse is the envelope for create/mutate operations.
type singleResponse struct {
	Status  string          `json:"status"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

// Service provides load balancer API operations.
type Service struct {
	client *httpclient.Client
}

// NewService creates a new load balancer Service.
func NewService(client *httpclient.Client) *Service {
	return &Service{client: client}
}

// List returns all load balancers. The include parameter requests nested relations.
func (s *Service) List(ctx context.Context) ([]LoadBalancer, error) {
	q := url.Values{
		"include": {"project,cloud_provider,ipaddress,region,load_balancer_rules,offering"},
	}
	var resp listResponse
	if err := s.client.Get(ctx, "/load-balancers", q, &resp); err != nil {
		return nil, fmt.Errorf("listing load balancers: %w", err)
	}
	var lbs []LoadBalancer
	if err := json.Unmarshal(resp.Data, &lbs); err != nil {
		return nil, fmt.Errorf("decoding load balancers: %w", err)
	}
	return lbs, nil
}

// Create provisions a new load balancer.
func (s *Service) Create(ctx context.Context, req CreateRequest) (*LoadBalancer, error) {
	var resp singleResponse
	if err := s.client.Post(ctx, "/load-balancers", req, &resp); err != nil {
		return nil, fmt.Errorf("creating load balancer: %w", err)
	}
	var lb LoadBalancer
	if err := json.Unmarshal(resp.Data, &lb); err != nil {
		return nil, fmt.Errorf("decoding load balancer: %w", err)
	}
	return &lb, nil
}

// CreateRule adds rules to an existing load balancer.
func (s *Service) CreateRule(ctx context.Context, lbSlug string, req CreateRuleRequest) error {
	var resp singleResponse
	if err := s.client.Post(ctx, "/load-balancers/"+lbSlug+"/load-balancer-rules", req, &resp); err != nil {
		return fmt.Errorf("creating rule on load balancer %s: %w", lbSlug, err)
	}
	return nil
}

// AttachVM attaches VMs to a load balancer rule.
func (s *Service) AttachVM(ctx context.Context, lbSlug, ruleID string, req AttachVMRequest) error {
	path := fmt.Sprintf("/load-balancers/%s/load-balancer-rules/%s/attach", lbSlug, ruleID)
	var resp singleResponse
	if err := s.client.Post(ctx, path, req, &resp); err != nil {
		return fmt.Errorf("attaching VMs to rule %s on load balancer %s: %w", ruleID, lbSlug, err)
	}
	return nil
}
