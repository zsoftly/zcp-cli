// Package vpc provides ZCP Virtual Private Cloud API operations.
package vpc

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/zsoftly/zcp-cli/internal/httpclient"
)

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
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

// NetworkACL represents a network ACL inside a VPC.
type NetworkACL struct {
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
	Slug     string `json:"slug"`
	PublicIP string `json:"publicIpAddress"`
	VPCUUID  string `json:"vpcUuid"`
	VPCSlug  string `json:"vpcSlug"`
	ZoneName string `json:"zoneName"`
	Status   string `json:"status"`
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
func (s *Service) List(ctx context.Context, zoneSlug string) ([]VPC, error) {
	var q url.Values
	if zoneSlug != "" {
		q = url.Values{"zoneSlug": {zoneSlug}}
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

// Get returns a single VPC by slug.
func (s *Service) Get(ctx context.Context, slug string) (*VPC, error) {
	vpcs, err := s.List(ctx, "")
	if err != nil {
		return nil, err
	}
	for i := range vpcs {
		if vpcs[i].Slug == slug {
			return &vpcs[i], nil
		}
	}
	return nil, fmt.Errorf("VPC %q not found", slug)
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
	return &v, nil
}

// Update modifies a VPC's mutable attributes.
func (s *Service) Update(ctx context.Context, slug string, req UpdateRequest) (*VPC, error) {
	var env apiResponse
	if err := s.client.Put(ctx, "/vpcs/"+slug, nil, req, &env); err != nil {
		return nil, fmt.Errorf("updating VPC %s: %w", slug, err)
	}
	var v VPC
	if err := json.Unmarshal(env.Data, &v); err != nil {
		return nil, fmt.Errorf("decoding updated VPC: %w", err)
	}
	return &v, nil
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
	var env apiResponse
	if err := s.client.Post(ctx, "/vpcs/"+vpcSlug+"/vpn-gateways", VPNGatewayCreateRequest{}, &env); err != nil {
		return nil, fmt.Errorf("creating VPN gateway for VPC %s: %w", vpcSlug, err)
	}
	var gw VPNGateway
	if err := json.Unmarshal(env.Data, &gw); err != nil {
		return nil, fmt.Errorf("decoding created VPN gateway: %w", err)
	}
	return &gw, nil
}

// DeleteVPNGateway deletes a VPN gateway from a VPC.
func (s *Service) DeleteVPNGateway(ctx context.Context, vpcSlug, gatewayID string) error {
	if err := s.client.Delete(ctx, "/vpcs/"+vpcSlug+"/vpn-gateways/"+gatewayID, nil); err != nil {
		return fmt.Errorf("deleting VPN gateway %s from VPC %s: %w", gatewayID, vpcSlug, err)
	}
	return nil
}
