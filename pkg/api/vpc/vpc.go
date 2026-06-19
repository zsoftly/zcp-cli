// Package vpc provides ZCP Virtual Private Cloud API operations.
package vpc

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"

	"github.com/zsoftly/zcp-cli/pkg/httpclient"
)

// ErrNotFound is returned by Get when no VPC matches the slug. Callers can
// distinguish a confirmed missing VPC from transport or server errors with
// errors.Is(err, ErrNotFound).
var ErrNotFound = errors.New("VPC not found")

// VPC represents a ZCP Virtual Private Cloud.
type VPC struct {
	Slug        string `json:"slug"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Status      string `json:"status"`
	CIDR        string `json:"cidr"`
	ZoneName    string `json:"zoneName"`
	DomainName  string `json:"domainName"`
}

// CreateRequest holds parameters for creating a VPC.
type CreateRequest struct {
	Name            string `json:"name"`
	CloudProvider   string `json:"cloud_provider"`
	Region          string `json:"region"`
	Project         string `json:"project"`
	Type            string `json:"type"`
	BillingCycle    string `json:"billing_cycle"`
	CIDR            string `json:"cidr"`
	Size            string `json:"size"`
	Plan            string `json:"plan"`
	StorageCategory string `json:"storage_category"`
	Description     string `json:"description,omitempty"`
	Coupon          string `json:"coupon,omitempty"`
}

// UpdateRequest holds parameters for updating a VPC.
type UpdateRequest struct {
	Name        string  `json:"name"`
	Description *string `json:"description,omitempty"`
}

// NetworkACL represents a network ACL inside a VPC. The live API returns
// id, name, and description; slug/status are kept for older deployments.
type NetworkACL struct {
	ID          string `json:"id"`
	Slug        string `json:"slug"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Status      string `json:"status"`
}

// ACLListCreateRequest holds parameters for creating a Network ACL list.
type ACLListCreateRequest struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	VPC         string `json:"vpc"`
}

// ACLRule represents a single ACL rule.
type ACLRule struct {
	Slug        string `json:"slug"`
	Protocol    string `json:"protocol"`
	CIDRList    string `json:"cidrList"`
	StartPort   int    `json:"startPort"`
	EndPort     int    `json:"endPort"`
	TrafficType string `json:"trafficType"`
	Action      string `json:"action"`
	Number      int    `json:"number"`
}

// VPNGateway represents a VPN gateway attached to a VPC.
type VPNGateway struct {
	ID       string `json:"id"`
	PublicIP string `json:"public_ip"`
	VPCID    string `json:"vpc_id"`
	VPCName  string `json:"vpc_name"`
}

// VPNGatewayCreateRequest holds parameters for creating a VPN gateway.
type VPNGatewayCreateRequest struct {
	// Intentionally empty — the VPC slug is in the URL path.
}

// apiResponse is the STKCNSL response envelope.
type apiResponse struct {
	Status string          `json:"status"`
	Data   json.RawMessage `json:"data"`
}

// Service provides VPC API operations.
type Service struct {
	client *httpclient.Client
}

// NewService creates a new VPC Service.
func NewService(client *httpclient.Client) *Service {
	return &Service{client: client}
}

// List returns all VPCs. zoneSlug is an optional filter.
func (s *Service) List(ctx context.Context, zoneSlug, region, project string) ([]VPC, error) {
	q := url.Values{}
	if zoneSlug != "" {
		q.Set("zoneSlug", zoneSlug)
	}
	if region != "" {
		q.Set("filter[region]", region)
	}
	if project != "" {
		q.Set("filter[project]", project)
	}
	var env apiResponse
	if err := s.client.Get(ctx, "/vpcs", q, &env); err != nil {
		return nil, fmt.Errorf("listing VPCs: %w", err)
	}
	var vpcs []VPC
	if err := json.Unmarshal(env.Data, &vpcs); err != nil {
		return nil, fmt.Errorf("decoding VPC list: %w", err)
	}
	return vpcs, nil
}

// vpcDetail is the GET /vpcs/{slug} response shape. The provider-side state
// (CIDR, state, zone) lives under "meta", which is the raw CloudStack view;
// the list endpoint does not include it.
type vpcDetail struct {
	Slug        string `json:"slug"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Meta        struct {
		CIDR          string `json:"cidr"`
		State         string `json:"state"`
		ZoneName      string `json:"zone_name"`
		NetworkDomain string `json:"network_domain"`
	} `json:"meta"`
}

// Get returns a single VPC by slug from GET /vpcs/{slug}, including its
// CIDR, state, and zone. Falls back to filtering the list endpoint when the
// detail response cannot be decoded.
func (s *Service) Get(ctx context.Context, slug string) (*VPC, error) {
	var env apiResponse
	if err := s.client.Get(ctx, "/vpcs/"+url.PathEscape(slug), nil, &env); err == nil {
		var d vpcDetail
		if jerr := json.Unmarshal(env.Data, &d); jerr == nil && d.Slug != "" {
			return &VPC{
				Slug:        d.Slug,
				Name:        d.Name,
				Description: d.Description,
				Status:      d.Meta.State,
				CIDR:        d.Meta.CIDR,
				ZoneName:    d.Meta.ZoneName,
				DomainName:  d.Meta.NetworkDomain,
			}, nil
		}
	}
	vpcs, err := s.List(ctx, "", "", "")
	if err != nil {
		return nil, err
	}
	for i := range vpcs {
		if vpcs[i].Slug == slug {
			return &vpcs[i], nil
		}
	}
	return nil, fmt.Errorf("VPC %q: %w", slug, ErrNotFound)
}

// Create provisions a new VPC.
func (s *Service) Create(ctx context.Context, req CreateRequest) (*VPC, error) {
	var env apiResponse
	if err := s.client.Post(ctx, "/vpcs", req, &env); err != nil {
		return nil, fmt.Errorf("creating VPC: %w", err)
	}
	var v VPC
	if err := json.Unmarshal(env.Data, &v); err != nil {
		return nil, fmt.Errorf("decoding created VPC: %w", err)
	}
	// The create response omits provider-side state (CIDR, status, zone) —
	// fetch the detail view, keeping the create result if it isn't ready yet.
	if v.Slug != "" && (v.CIDR == "" || v.Status == "") {
		if full, err := s.Get(ctx, v.Slug); err == nil {
			return full, nil
		}
	}
	return &v, nil
}

// Update modifies a VPC's mutable attributes.
func (s *Service) Update(ctx context.Context, slug string, req UpdateRequest) (*VPC, error) {
	var env apiResponse
	if err := s.client.Put(ctx, "/vpcs/"+slug, nil, req, &env); err != nil {
		return nil, fmt.Errorf("updating VPC %s: %w", slug, err)
	}
	// The Update API may return data:null — fall back to GET.
	if len(env.Data) > 0 && string(env.Data) != "null" {
		var v VPC
		if err := json.Unmarshal(env.Data, &v); err == nil && v.Slug != "" {
			return &v, nil
		}
	}
	return s.Get(ctx, slug)
}

// Delete removes a VPC by slug.
func (s *Service) Delete(ctx context.Context, slug string) error {
	if err := s.client.Delete(ctx, "/vpcs/"+slug, nil); err != nil {
		return fmt.Errorf("deleting VPC %s: %w", slug, err)
	}
	return nil
}

// Restart restarts a VPC by slug.
func (s *Service) Restart(ctx context.Context, slug string) (*VPC, error) {
	var env apiResponse
	if err := s.client.Get(ctx, "/vpcs/"+slug+"/restart", nil, &env); err != nil {
		return nil, fmt.Errorf("restarting VPC %s: %w", slug, err)
	}
	var v VPC
	if err := json.Unmarshal(env.Data, &v); err != nil {
		return nil, fmt.Errorf("decoding restarted VPC: %w", err)
	}
	return &v, nil
}

// ListACLs returns the network ACLs for a VPC.
func (s *Service) ListACLs(ctx context.Context, vpcSlug string) ([]NetworkACL, error) {
	var env apiResponse
	if err := s.client.Get(ctx, "/vpcs/"+vpcSlug+"/network-acl-list", nil, &env); err != nil {
		return nil, fmt.Errorf("listing ACLs for VPC %s: %w", vpcSlug, err)
	}
	var acls []NetworkACL
	if err := json.Unmarshal(env.Data, &acls); err != nil {
		return nil, fmt.Errorf("decoding ACL list: %w", err)
	}
	return acls, nil
}

// CreateACL creates a new ACL list in a VPC.
func (s *Service) CreateACL(ctx context.Context, vpcSlug string, req ACLListCreateRequest) error {
	var env apiResponse
	if err := s.client.Post(ctx, "/vpcs/"+vpcSlug+"/network-acl-list", req, &env); err != nil {
		return fmt.Errorf("creating ACL in VPC %s: %w", vpcSlug, err)
	}
	return nil
}

// ReplaceNetworkACL replaces the ACL on a network by slug.
func (s *Service) ReplaceNetworkACL(ctx context.Context, networkSlug string, req map[string]string) error {
	var env apiResponse
	if err := s.client.Post(ctx, "/networks/"+networkSlug+"/replace-acl-list", req, &env); err != nil {
		return fmt.Errorf("replacing ACL on network %s: %w", networkSlug, err)
	}
	return nil
}

// ListVPNGateways returns VPN gateways for a VPC.
func (s *Service) ListVPNGateways(ctx context.Context, vpcSlug string) ([]VPNGateway, error) {
	var env apiResponse
	if err := s.client.Get(ctx, "/vpcs/"+vpcSlug+"/vpn-gateways", nil, &env); err != nil {
		return nil, fmt.Errorf("listing VPN gateways for VPC %s: %w", vpcSlug, err)
	}
	var gateways []VPNGateway
	if err := json.Unmarshal(env.Data, &gateways); err != nil {
		return nil, fmt.Errorf("decoding VPN gateway list: %w", err)
	}
	return gateways, nil
}

// CreateVPNGateway creates a VPN gateway for a VPC.
func (s *Service) CreateVPNGateway(ctx context.Context, vpcSlug string) (*VPNGateway, error) {
	// Best-effort pre-create snapshot. If this fails, existing stays empty and the
	// diff below treats every post-create gateway as new (acceptable degraded mode).
	before, _ := s.ListVPNGateways(ctx, vpcSlug)
	existing := make(map[string]bool, len(before))
	for _, gw := range before {
		existing[gw.ID] = true
	}

	var env apiResponse
	if err := s.client.Post(ctx, "/vpcs/"+vpcSlug+"/vpn-gateways", VPNGatewayCreateRequest{}, &env); err != nil {
		return nil, fmt.Errorf("creating VPN gateway for VPC %s: %w", vpcSlug, err)
	}
	// The Create API may return data:null — diff pre/post lists to find the new gateway.
	var gw VPNGateway
	if len(env.Data) > 0 && string(env.Data) != "null" {
		if err := json.Unmarshal(env.Data, &gw); err == nil && gw.ID != "" {
			return &gw, nil
		}
	}
	after, err := s.ListVPNGateways(ctx, vpcSlug)
	if err != nil {
		return nil, fmt.Errorf("creating VPN gateway for VPC %s: post-create list: %w", vpcSlug, err)
	}
	for i := range after {
		if !existing[after[i].ID] {
			return &after[i], nil
		}
	}
	return nil, fmt.Errorf("creating VPN gateway for VPC %s: new gateway not found after create", vpcSlug)
}

// DeleteVPNGateway deletes a VPN gateway from a VPC.
func (s *Service) DeleteVPNGateway(ctx context.Context, vpcSlug, gatewayID string) error {
	if err := s.client.Delete(ctx, "/vpcs/"+vpcSlug+"/vpn-gateways/"+gatewayID, nil); err != nil {
		return fmt.Errorf("deleting VPN gateway %s from VPC %s: %w", gatewayID, vpcSlug, err)
	}
	return nil
}
