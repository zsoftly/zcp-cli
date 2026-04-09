// Package acl provides ZCP Network ACL API operations.
package acl

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/zsoftly/zcp-cli/internal/httpclient"
)

// NetworkACL represents a ZCP Network Access Control List.
type NetworkACL struct {
	Slug        string `json:"slug"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Status      string `json:"status"`
	VPCSlug     string `json:"vpcSlug"`
}

// ACLCreateRequest holds parameters for creating a Network ACL list.
type ACLCreateRequest struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	VPC         string `json:"vpc"`
}

// ACLRuleCreateRequest holds parameters for creating a rule inside an ACL.
type ACLRuleCreateRequest struct {
	Protocol    string `json:"protocol"`
	CIDRList    string `json:"cidrList,omitempty"`
	StartPort   int    `json:"startPort,omitempty"`
	EndPort     int    `json:"endPort,omitempty"`
	TrafficType string `json:"trafficType,omitempty"`
	Action      string `json:"action"`
	Number      int    `json:"number,omitempty"`
	ICMPCode    int    `json:"icmpCode,omitempty"`
	ICMPType    int    `json:"icmpType,omitempty"`
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

// ReplaceACLRequest holds parameters for replacing an ACL on a network.
type ReplaceACLRequest struct {
	ACLSlug string `json:"aclSlug"`
}

// apiResponse is the STKCNSL response envelope.
type apiResponse struct {
	Status string          `json:"status"`
	Data   json.RawMessage `json:"data"`
}

// Service provides Network ACL API operations.
type Service struct {
	client *httpclient.Client
}

// NewService creates a new ACL Service.
func NewService(client *httpclient.Client) *Service {
	return &Service{client: client}
}

// List returns network ACLs for a VPC by slug.
func (s *Service) List(ctx context.Context, vpcSlug string) ([]NetworkACL, error) {
	var env apiResponse
	if err := s.client.Get(ctx, "/vpcs/"+vpcSlug+"/network-acl-list", nil, &env); err != nil {
		return nil, fmt.Errorf("listing network ACLs for VPC %s: %w", vpcSlug, err)
	}
	var acls []NetworkACL
	if err := json.Unmarshal(env.Data, &acls); err != nil {
		return nil, fmt.Errorf("decoding network ACL list: %w", err)
	}
	return acls, nil
}

// Create creates a new ACL list in a VPC.
func (s *Service) Create(ctx context.Context, vpcSlug string, req ACLCreateRequest) error {
	var env apiResponse
	if err := s.client.Post(ctx, "/vpcs/"+vpcSlug+"/network-acl-list", req, &env); err != nil {
		return fmt.Errorf("creating ACL in VPC %s: %w", vpcSlug, err)
	}
	return nil
}

// ReplaceNetworkACL replaces the ACL on a network by slug.
func (s *Service) ReplaceNetworkACL(ctx context.Context, networkSlug, aclSlug string) error {
	req := ReplaceACLRequest{ACLSlug: aclSlug}
	var env apiResponse
	if err := s.client.Post(ctx, "/networks/"+networkSlug+"/replace-acl-list", req, &env); err != nil {
		return fmt.Errorf("replacing ACL on network %s: %w", networkSlug, err)
	}
	return nil
}
