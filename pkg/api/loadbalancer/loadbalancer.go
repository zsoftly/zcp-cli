// Package loadbalancer provides ZCP Load Balancer API operations (STKCNSL).
package loadbalancer

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"

	"github.com/zsoftly/zcp-cli/pkg/httpclient"
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
	LastPage    int             `json:"last_page"`
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
func (s *Service) List(ctx context.Context, region, project string) ([]LoadBalancer, error) {
	// The endpoint is paginated (Laravel-style ?page=N with last_page). Fetch every page so
	// callers that resolve a specific LB — e.g. `loadbalancer delete --release-ip` — don't
	// miss one that lands on a later page. maxListPages caps the loop against a server that
	// misreports last_page. A single-page (or non-paginated) response returns after page 1.
	var all []LoadBalancer
	for page := 1; page <= maxListPages; page++ {
		q := url.Values{
			"include": {"project,cloud_provider,ipaddress,region,load_balancer_rules,offering"},
		}
		if region != "" {
			q.Set("filter[region]", region)
		}
		if project != "" {
			q.Set("filter[project]", project)
		}
		if page > 1 {
			q.Set("page", strconv.Itoa(page))
		}
		var resp listResponse
		if err := s.client.Get(ctx, "/load-balancers", q, &resp); err != nil {
			return nil, fmt.Errorf("listing load balancers: %w", err)
		}
		var lbs []LoadBalancer
		if err := json.Unmarshal(resp.Data, &lbs); err != nil {
			return nil, fmt.Errorf("decoding load balancers: %w", err)
		}
		all = append(all, lbs...)
		if len(lbs) == 0 || resp.LastPage <= page {
			break
		}
	}
	return all, nil
}

// maxListPages bounds paginated List loops so a server that misreports last_page can't
// spin forever. Set well above any realistic page count.
const maxListPages = 1000

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

// DetachVMRequest holds parameters for detaching a VM from a load balancer rule.
type DetachVMRequest struct {
	VirtualMachine string `json:"virtual_machine"`
}

// Delete performs a direct delete of a load balancer via DELETE /load-balancers/{slug}.
//
// NOTE: this endpoint does NOT release the load balancer's public IP — it is left
// Allocated/billable. Neither does the service-cancellation workflow ('zcp loadbalancer
// delete'): unlike a VM's ephemeral auto-assigned IP, an LB's public IP is a separate,
// reusable resource, so it must be released explicitly via ipaddress.Service.Release
// (exposed as the CLI's --release-ip flag) — and only for a STATIC IP the LB owns, never
// a network source-NAT IP.
func (s *Service) Delete(ctx context.Context, slug string) error {
	if err := s.client.Delete(ctx, "/load-balancers/"+slug, nil); err != nil {
		return fmt.Errorf("deleting load balancer %s: %w", slug, err)
	}
	return nil
}

// DeleteRule removes a rule from a load balancer.
func (s *Service) DeleteRule(ctx context.Context, lbSlug, ruleID string) error {
	path := fmt.Sprintf("/load-balancers/%s/load-balancer-rules/%s", lbSlug, ruleID)
	if err := s.client.Delete(ctx, path, nil); err != nil {
		return fmt.Errorf("deleting rule %s from load balancer %s: %w", ruleID, lbSlug, err)
	}
	return nil
}

// DetachVM detaches a VM from a load balancer rule.
func (s *Service) DetachVM(ctx context.Context, lbSlug, ruleID, vmSlug string) error {
	path := fmt.Sprintf("/load-balancers/%s/load-balancer-rules/%s/detach", lbSlug, ruleID)
	if err := s.client.Post(ctx, path, DetachVMRequest{VirtualMachine: vmSlug}, nil); err != nil {
		return fmt.Errorf("detaching VM %s from rule %s on load balancer %s: %w", vmSlug, ruleID, lbSlug, err)
	}
	return nil
}
