// Package network provides ZCP network API operations.
package network

import (
	"context"
	"fmt"
	"net/url"

	"github.com/zsoftly/zcp-cli/internal/httpclient"
)

// Network represents a ZCP virtual network.
type Network struct {
	UUID                string `json:"uuid"`
	Name                string `json:"name"`
	Status              string `json:"status"`
	IsActive            bool   `json:"isActive"`
	NetworkType         string `json:"networkType"`
	Gateway             string `json:"gateway"`
	CIDR                string `json:"getcIDR"`
	ZoneUUID            string `json:"zoneUuid"`
	DomainName          string `json:"domainName"`
	NetworkOfferingUUID string `json:"networkOfferingUuid"`
	NetworkACLList      string `json:"networkAclList"`
	CleanUpNetwork      bool   `json:"cleanUpNetwork"`
	NetworkDomain       string `json:"networkDomain"`
}

// CreateRequest holds parameters for creating a network.
type CreateRequest struct {
	Name                string `json:"name"`
	ZoneUUID            string `json:"zoneUuid"`
	NetworkOfferingUUID string `json:"networkOfferingUuid"`
	VirtualMachineUUID  string `json:"virtualmachineUuid,omitempty"`
	IsPublic            bool   `json:"isPublic,omitempty"`
}

// UpdateRequest holds parameters for updating a network.
type UpdateRequest struct {
	UUID          string `json:"uuid"`
	Name          string `json:"name"`
	Description   string `json:"description,omitempty"`
	CIDR          string `json:"getcIDR,omitempty"`
	NetworkDomain string `json:"networkDomain,omitempty"`
}

type listNetworkResponse struct {
	Count               int       `json:"count"`
	ListNetworkResponse []Network `json:"listNetworkResponse"`
}

// Service provides network API operations.
type Service struct {
	client *httpclient.Client
}

// NewService creates a new network Service.
func NewService(client *httpclient.Client) *Service {
	return &Service{client: client}
}

// List returns networks in a zone. zoneUUID is required; networkUUID is an optional filter.
func (s *Service) List(ctx context.Context, zoneUUID, networkUUID string) ([]Network, error) {
	q := url.Values{"zoneUuid": {zoneUUID}}
	if networkUUID != "" {
		q.Set("uuid", networkUUID)
	}
	var resp listNetworkResponse
	if err := s.client.Get(ctx, "/restapi/network/networkList", q, &resp); err != nil {
		return nil, fmt.Errorf("listing networks: %w", err)
	}
	return resp.ListNetworkResponse, nil
}

// Get returns a single network by UUID using the dedicated networkId endpoint.
func (s *Service) Get(ctx context.Context, zoneUUID, uuid string) (*Network, error) {
	q := url.Values{"uuid": {uuid}}
	var resp listNetworkResponse
	if err := s.client.Get(ctx, "/restapi/network/networkId", q, &resp); err != nil {
		return nil, fmt.Errorf("getting network %s: %w", uuid, err)
	}
	if len(resp.ListNetworkResponse) == 0 {
		return nil, fmt.Errorf("network %q not found", uuid)
	}
	return &resp.ListNetworkResponse[0], nil
}

// Create provisions a new network.
func (s *Service) Create(ctx context.Context, req CreateRequest) (*Network, error) {
	var resp listNetworkResponse
	if err := s.client.Post(ctx, "/restapi/network/createNetwork", req, &resp); err != nil {
		return nil, fmt.Errorf("creating network: %w", err)
	}
	if len(resp.ListNetworkResponse) == 0 {
		return nil, fmt.Errorf("create network returned empty response")
	}
	return &resp.ListNetworkResponse[0], nil
}

// Delete removes a network by UUID.
func (s *Service) Delete(ctx context.Context, uuid string) error {
	if err := s.client.Delete(ctx, "/restapi/network/deleteNetwork/"+uuid, nil); err != nil {
		return fmt.Errorf("deleting network %s: %w", uuid, err)
	}
	return nil
}

// Update modifies a network's mutable attributes.
// NOTE: The API defines this as PUT but we use Post since httpclient has no Put method.
func (s *Service) Update(ctx context.Context, req UpdateRequest) (*Network, error) {
	var resp listNetworkResponse
	if err := s.client.Post(ctx, "/restapi/network/updateNetwork", req, &resp); err != nil {
		return nil, fmt.Errorf("updating network %s: %w", req.UUID, err)
	}
	if len(resp.ListNetworkResponse) == 0 {
		return nil, fmt.Errorf("update network returned empty response")
	}
	return &resp.ListNetworkResponse[0], nil
}

// Restart restarts a network. cleanUp triggers cleanup of stale resources.
func (s *Service) Restart(ctx context.Context, uuid string, cleanUp bool) (*Network, error) {
	cleanUpStr := "false"
	if cleanUp {
		cleanUpStr = "true"
	}
	q := url.Values{"uuid": {uuid}, "cleanUpNetwork": {cleanUpStr}}
	var resp listNetworkResponse
	if err := s.client.Get(ctx, "/restapi/network/restartNetwork", q, &resp); err != nil {
		return nil, fmt.Errorf("restarting network %s: %w", uuid, err)
	}
	if len(resp.ListNetworkResponse) == 0 {
		return nil, fmt.Errorf("restart network returned empty response")
	}
	return &resp.ListNetworkResponse[0], nil
}
