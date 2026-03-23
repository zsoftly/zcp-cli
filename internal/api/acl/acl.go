// Package acl provides ZCP Network ACL API operations.
package acl

import (
	"context"
	"fmt"
	"net/url"

	"github.com/zsoftly/zcp-cli/internal/httpclient"
)

// NetworkACL represents a ZCP Network Access Control List.
type NetworkACL struct {
	UUID        string `json:"uuid"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Status      string `json:"status"`
	IsActive    bool   `json:"isActive"`
	ZoneUUID    string `json:"zoneUuid"`
	VPCUUID     string `json:"vpcUuid"`
}

// CreateRequest holds parameters for creating a Network ACL.
type CreateRequest struct {
	Name        string `json:"name"`
	VPCUUID     string `json:"vpcUuid"`
	Description string `json:"description,omitempty"`
}

type listNetworkAclListResponse struct {
	Count                      int          `json:"count"`
	ListNetworkAclListResponse []NetworkACL `json:"listNetworkAclListResponse"`
}

// Network represents a minimal network result returned by ACL operations.
type Network struct {
	UUID string `json:"uuid"`
	Name string `json:"name"`
}

type listNetworkResponse struct {
	Count               int       `json:"count"`
	ListNetworkResponse []Network `json:"listNetworkResponse"`
}

// Service provides Network ACL API operations.
type Service struct {
	client *httpclient.Client
}

// NewService creates a new ACL Service.
func NewService(client *httpclient.Client) *Service {
	return &Service{client: client}
}

// List returns network ACLs. zoneUUID is required; uuid and vpcUUID are optional filters.
func (s *Service) List(ctx context.Context, zoneUUID, uuid, vpcUUID string) ([]NetworkACL, error) {
	q := url.Values{"zoneUuid": {zoneUUID}}
	if uuid != "" {
		q.Set("uuid", uuid)
	}
	if vpcUUID != "" {
		q.Set("vpcUuid", vpcUUID)
	}
	var resp listNetworkAclListResponse
	if err := s.client.Get(ctx, "/restapi/networkacllist/networkAclList", q, &resp); err != nil {
		return nil, fmt.Errorf("listing network ACLs: %w", err)
	}
	return resp.ListNetworkAclListResponse, nil
}

// Create provisions a new Network ACL.
func (s *Service) Create(ctx context.Context, req CreateRequest) (*NetworkACL, error) {
	var resp listNetworkAclListResponse
	if err := s.client.Post(ctx, "/restapi/networkacllist/createNetworkAcl", req, &resp); err != nil {
		return nil, fmt.Errorf("creating network ACL: %w", err)
	}
	if len(resp.ListNetworkAclListResponse) == 0 {
		return nil, fmt.Errorf("create network ACL returned empty response")
	}
	return &resp.ListNetworkAclListResponse[0], nil
}

// Delete removes a Network ACL by UUID.
func (s *Service) Delete(ctx context.Context, uuid string) error {
	if err := s.client.Delete(ctx, "/restapi/networkacllist/deleteNetworkAcl/"+uuid, nil); err != nil {
		return fmt.Errorf("deleting network ACL %s: %w", uuid, err)
	}
	return nil
}

// ReplaceNetworkACL replaces the ACL on a network identified by networkUUID.
func (s *Service) ReplaceNetworkACL(ctx context.Context, networkUUID, aclUUID string) ([]Network, error) {
	q := url.Values{
		"uuid":    {networkUUID},
		"aclUuid": {aclUUID},
	}
	var resp listNetworkResponse
	if err := s.client.Get(ctx, "/restapi/network/replaceAcl", q, &resp); err != nil {
		return nil, fmt.Errorf("replacing ACL on network %s: %w", networkUUID, err)
	}
	return resp.ListNetworkResponse, nil
}
