// Package acl provides ZCP Network ACL API operations.
package acl

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/zsoftly/zcp-cli/pkg/httpclient"
)

// NetworkACL represents a ZCP Network Access Control List. The live API
// returns id, name, and description; slug/status/vpcSlug are kept for older
// deployments.
type NetworkACL struct {
	ID          string `json:"id"`
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

// RuleCreateRequest holds parameters for creating a rule inside an ACL list
// (POST /vpcs/{vpc}/network-acl-list/{acl_list_id}/network-acl). Field names
// match the live API validation: protocol in tcp|udp|icmp|all|protocol_number;
// start/end port required for tcp/udp; icmp type/code required for icmp.
type RuleCreateRequest struct {
	Number         int    `json:"number,omitempty"`
	Description    string `json:"description,omitempty"`
	Protocol       string `json:"protocol"`
	ProtocolNumber string `json:"protocol_number,omitempty"`
	CIDRList       string `json:"cidr_list"`
	Action         string `json:"action,omitempty"`
	TrafficType    string `json:"traffic_type,omitempty"`
	ICMPType       *int   `json:"icmp_type,omitempty"`
	ICMPCode       *int   `json:"icmp_code,omitempty"`
	StartPort      *int   `json:"start_port,omitempty"`
	EndPort        *int   `json:"end_port,omitempty"`
}

// Rule represents a single ACL rule as returned by the live API.
type Rule struct {
	ID          string `json:"id"`
	Protocol    string `json:"protocol"`
	StartPort   string `json:"start_port"`
	EndPort     string `json:"end_port"`
	TrafficType string `json:"traffictype"`
	State       string `json:"state"`
	CIDRList    string `json:"cidrlist"`
	ACLID       string `json:"aclid"`
	ACLName     string `json:"aclname"`
	Number      int    `json:"number"`
	Action      string `json:"action"`
	Description string `json:"reason"`
}

// ReplaceACLRequest holds parameters for replacing an ACL on a network.
// The API expects the ACL's ID (UUID), not its name.
type ReplaceACLRequest struct {
	ACLID string `json:"acl_id"`
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

// ReplaceNetworkACL replaces the ACL on a network. aclID must be the ACL's
// ID (UUID) — use Resolve to translate a name first.
func (s *Service) ReplaceNetworkACL(ctx context.Context, networkSlug, aclID string) error {
	req := ReplaceACLRequest{ACLID: aclID}
	var env apiResponse
	if err := s.client.Post(ctx, "/networks/"+networkSlug+"/replace-acl-list", req, &env); err != nil {
		return fmt.Errorf("replacing ACL on network %s: %w", networkSlug, err)
	}
	return nil
}

// Delete removes an ACL list from a VPC by ACL ID.
func (s *Service) Delete(ctx context.Context, vpcSlug, aclID string) error {
	if err := s.client.Delete(ctx, "/vpcs/"+vpcSlug+"/network-acl-list/"+aclID, nil); err != nil {
		return fmt.Errorf("deleting ACL %s in VPC %s: %w", aclID, vpcSlug, err)
	}
	return nil
}

// ListRules returns the rules inside an ACL list.
func (s *Service) ListRules(ctx context.Context, vpcSlug, aclID string) ([]Rule, error) {
	var env apiResponse
	if err := s.client.Get(ctx, "/vpcs/"+vpcSlug+"/network-acl-list/"+aclID+"/network-acl", nil, &env); err != nil {
		return nil, fmt.Errorf("listing rules for ACL %s: %w", aclID, err)
	}
	var rules []Rule
	if err := json.Unmarshal(env.Data, &rules); err != nil {
		return nil, fmt.Errorf("decoding ACL rule list: %w", err)
	}
	return rules, nil
}

// CreateRule adds a rule to an ACL list.
func (s *Service) CreateRule(ctx context.Context, vpcSlug, aclID string, req RuleCreateRequest) error {
	var env apiResponse
	if err := s.client.Post(ctx, "/vpcs/"+vpcSlug+"/network-acl-list/"+aclID+"/network-acl", req, &env); err != nil {
		return fmt.Errorf("creating rule in ACL %s: %w", aclID, err)
	}
	return nil
}

// UpdateRule updates a rule in an ACL list in place (the rule ID is
// preserved). The request shape is identical to CreateRule.
func (s *Service) UpdateRule(ctx context.Context, vpcSlug, aclID, ruleID string, req RuleCreateRequest) error {
	var env apiResponse
	if err := s.client.Put(ctx, "/vpcs/"+vpcSlug+"/network-acl-list/"+aclID+"/network-acl/"+ruleID, nil, req, &env); err != nil {
		return fmt.Errorf("updating rule %s in ACL %s: %w", ruleID, aclID, err)
	}
	return nil
}

// DeleteRule removes a rule from an ACL list by rule ID.
func (s *Service) DeleteRule(ctx context.Context, vpcSlug, aclID, ruleID string) error {
	if err := s.client.Delete(ctx, "/vpcs/"+vpcSlug+"/network-acl-list/"+aclID+"/network-acl/"+ruleID, nil); err != nil {
		return fmt.Errorf("deleting rule %s in ACL %s: %w", ruleID, aclID, err)
	}
	return nil
}

// Resolve translates an ACL name (or ID) within a VPC to the ACL ID.
func (s *Service) Resolve(ctx context.Context, vpcSlug, nameOrID string) (string, error) {
	acls, err := s.List(ctx, vpcSlug)
	if err != nil {
		return "", err
	}
	for _, a := range acls {
		if a.ID == "" {
			continue // a match without an ID is unusable for acl_id requests
		}
		if a.ID == nameOrID || a.Name == nameOrID || a.Slug == nameOrID {
			return a.ID, nil
		}
	}
	return "", fmt.Errorf("ACL %q not found in VPC %q", nameOrID, vpcSlug)
}
