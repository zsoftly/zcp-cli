// Package vpc provides ZCP Virtual Private Cloud API operations.
package vpc

import (
	"context"
	"fmt"
	"net/url"

	"github.com/zsoftly/zcp-cli/internal/httpclient"
)

// VPC represents a ZCP Virtual Private Cloud.
type VPC struct {
	UUID        string `json:"uuid"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Status      string `json:"status"`
	IsActive    bool   `json:"isActive"`
	CIDR        string `json:"cIDR"`
	ZoneUUID    string `json:"zoneUuid"`
	ZoneName    string `json:"zoneName"`
	DomainName  string `json:"domainName"`
}

// CreateRequest holds parameters for creating a VPC.
type CreateRequest struct {
	Name                       string `json:"name"`
	ZoneUUID                   string `json:"zoneUuid"`
	VPCOfferingUUID            string `json:"vpcOfferingUuid"`
	CIDR                       string `json:"cIDR"`
	Description                string `json:"description"`
	NetworkDomain              string `json:"networkDomain,omitempty"`
	PublicLoadBalancerProvider string `json:"publicLoadBalancerProvider"`
}

// CreateNetworkRequest holds parameters for creating a VPC tier network.
type CreateNetworkRequest struct {
	Name                string `json:"name"`
	ZoneUUID            string `json:"zoneUuid"`
	NetworkOfferingUUID string `json:"networkOfferingUuid"`
	VPCUUID             string `json:"vpcUuid"`
	Gateway             string `json:"gateway"`
	Netmask             string `json:"netmask"`
	ACLUUID             string `json:"aclUuid,omitempty"`
}

// VPCNetwork represents a network tier inside a VPC.
type VPCNetwork struct {
	UUID                string `json:"uuid"`
	Name                string `json:"name"`
	IsActive            bool   `json:"isActive"`
	DomainName          string `json:"domainName"`
	CIDR                string `json:"cIDR"`
	Gateway             string `json:"gateway"`
	NetworkType         string `json:"networkType"`
	NetworkOfferingUUID string `json:"networkOfferingUuid"`
	ZoneUUID            string `json:"zoneUuid"`
	NetworkDomain       string `json:"networkDomain"`
	Status              string `json:"status"`
}

type listVpcNetworkResponse struct {
	Count               int          `json:"count"`
	ListNetworkResponse []VPCNetwork `json:"listNetworkResponse"`
}

// UpdateRequest holds parameters for updating a VPC.
type UpdateRequest struct {
	UUID        string `json:"uuid"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

type listVpcResponse struct {
	Count           int   `json:"count"`
	ListVpcResponse []VPC `json:"listVpcResponse"`
}

// Service provides VPC API operations.
type Service struct {
	client *httpclient.Client
}

// NewService creates a new VPC Service.
func NewService(client *httpclient.Client) *Service {
	return &Service{client: client}
}

// List returns VPCs in a zone. zoneUUID is required; uuid is an optional filter.
func (s *Service) List(ctx context.Context, zoneUUID, uuid string) ([]VPC, error) {
	q := url.Values{"zoneUuid": {zoneUUID}}
	if uuid != "" {
		q.Set("uuid", uuid)
	}
	var resp listVpcResponse
	if err := s.client.Get(ctx, "/restapi/vpc/vpcList", q, &resp); err != nil {
		return nil, fmt.Errorf("listing VPCs: %w", err)
	}
	return resp.ListVpcResponse, nil
}

// Get returns a single VPC by UUID using the dedicated vpcId endpoint.
func (s *Service) Get(ctx context.Context, uuid string) (*VPC, error) {
	q := url.Values{"uuid": {uuid}}
	var resp listVpcResponse
	if err := s.client.Get(ctx, "/restapi/vpc/vpcId", q, &resp); err != nil {
		return nil, fmt.Errorf("getting VPC %s: %w", uuid, err)
	}
	if len(resp.ListVpcResponse) == 0 {
		return nil, fmt.Errorf("VPC %q not found", uuid)
	}
	return &resp.ListVpcResponse[0], nil
}

// Create provisions a new VPC.
func (s *Service) Create(ctx context.Context, req CreateRequest) (*VPC, error) {
	var resp listVpcResponse
	if err := s.client.Post(ctx, "/restapi/vpc/createVpc", req, &resp); err != nil {
		return nil, fmt.Errorf("creating VPC: %w", err)
	}
	if len(resp.ListVpcResponse) == 0 {
		return nil, fmt.Errorf("create VPC returned empty response")
	}
	return &resp.ListVpcResponse[0], nil
}

// CreateNetwork creates a VPC tier network using the dedicated VPC endpoint.
func (s *Service) CreateNetwork(ctx context.Context, req CreateNetworkRequest) (*VPCNetwork, error) {
	var resp listVpcNetworkResponse
	if err := s.client.Post(ctx, "/restapi/vpc/createVpcNetwork", req, &resp); err != nil {
		return nil, fmt.Errorf("creating VPC network: %w", err)
	}
	if len(resp.ListNetworkResponse) == 0 {
		return nil, fmt.Errorf("create VPC network returned empty response")
	}
	return &resp.ListNetworkResponse[0], nil
}

// UpdateNetworkRequest holds parameters for updating a VPC tier network.
type UpdateNetworkRequest struct {
	UUID                string `json:"uuid"`
	Name                string `json:"name,omitempty"`
	Description         string `json:"description,omitempty"`
	NetworkOfferingUUID string `json:"networkOfferingUuid"`
	NetworkDomain       string `json:"networkDomain,omitempty"`
}

// UpdateNetwork modifies a VPC tier network.
func (s *Service) UpdateNetwork(ctx context.Context, req UpdateNetworkRequest) (*VPCNetwork, error) {
	var resp listVpcNetworkResponse
	if err := s.client.Put(ctx, "/restapi/vpc/updateVpcNetwork", nil, req, &resp); err != nil {
		return nil, fmt.Errorf("updating VPC network: %w", err)
	}
	if len(resp.ListNetworkResponse) == 0 {
		return nil, fmt.Errorf("update VPC network returned empty response")
	}
	return &resp.ListNetworkResponse[0], nil
}

// Update modifies a VPC's mutable attributes.
func (s *Service) Update(ctx context.Context, req UpdateRequest) (*VPC, error) {
	var resp listVpcResponse
	if err := s.client.Put(ctx, "/restapi/vpc/updateVpc", nil, req, &resp); err != nil {
		return nil, fmt.Errorf("updating VPC %s: %w", req.UUID, err)
	}
	if len(resp.ListVpcResponse) == 0 {
		return nil, fmt.Errorf("update VPC returned empty response")
	}
	return &resp.ListVpcResponse[0], nil
}

// Delete removes a VPC by UUID.
func (s *Service) Delete(ctx context.Context, uuid string) error {
	if err := s.client.Delete(ctx, "/restapi/vpc/deleteVpc/"+uuid, nil); err != nil {
		return fmt.Errorf("deleting VPC %s: %w", uuid, err)
	}
	return nil
}

// Restart restarts a VPC. cleanUp triggers cleanup; redundant enables redundant router.
func (s *Service) Restart(ctx context.Context, uuid string, cleanUp, redundant bool) (*VPC, error) {
	cleanUpStr := "false"
	if cleanUp {
		cleanUpStr = "true"
	}
	redundantStr := "false"
	if redundant {
		redundantStr = "true"
	}
	q := url.Values{
		"uuid":               {uuid},
		"cleanUpVPC":         {cleanUpStr},
		"redundantVpcRouter": {redundantStr},
	}
	var resp listVpcResponse
	if err := s.client.Get(ctx, "/restapi/vpc/restartVpc", q, &resp); err != nil {
		return nil, fmt.Errorf("restarting VPC %s: %w", uuid, err)
	}
	if len(resp.ListVpcResponse) == 0 {
		return nil, fmt.Errorf("restart VPC returned empty response")
	}
	return &resp.ListVpcResponse[0], nil
}
