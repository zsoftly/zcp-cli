// Package securitygroup provides ZCP security group API operations.
package securitygroup

import (
	"context"
	"fmt"
	"net/url"

	"github.com/zsoftly/zcp-cli/internal/httpclient"
)

// FirewallRule is a security group inbound rule.
type FirewallRule struct {
	UUID      string `json:"uuid"`
	Protocol  string `json:"protocol"`
	StartPort string `json:"startPort"`
	EndPort   string `json:"endPort"`
	CIDRList  string `json:"cidrList"`
	ICMPType  string `json:"icmpType"`
	ICMPCode  string `json:"icmpCode"`
}

// EgressRule is a security group outbound rule.
type EgressRule struct {
	UUID      string `json:"uuid"`
	Protocol  string `json:"protocol"`
	StartPort string `json:"startPort"`
	EndPort   string `json:"endPort"`
	ICMPType  string `json:"icmpType"`
	ICMPCode  string `json:"icmpCode"`
}

// SecurityGroup represents a ZCP security group with embedded rules.
type SecurityGroup struct {
	UUID          string         `json:"uuid"`
	Name          string         `json:"name"`
	Description   string         `json:"description"`
	IsActive      bool           `json:"isActive"`
	Status        string         `json:"status"`
	FirewallRules []FirewallRule `json:"securityGroupFirewallRule"`
	EgressRules   []EgressRule   `json:"securityGroupEgressRule"`
}

// CreateGroupRequest holds parameters for creating a security group.
type CreateGroupRequest struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

// CreateFirewallRuleRequest holds parameters for an inbound rule.
type CreateFirewallRuleRequest struct {
	SecurityGroupUUID string `json:"securityGroupUuid"`
	Protocol          string `json:"protocol"`
	StartPort         string `json:"startPort,omitempty"`
	EndPort           string `json:"endPort,omitempty"`
	CIDRList          string `json:"cidrList,omitempty"`
	ICMPType          string `json:"icmpType,omitempty"`
	ICMPCode          string `json:"icmpCode,omitempty"`
}

// CreateEgressRuleRequest holds parameters for an outbound rule.
type CreateEgressRuleRequest struct {
	SecurityGroupUUID string `json:"securityGroupUuid"`
	Protocol          string `json:"protocol"`
	StartPort         string `json:"startPort,omitempty"`
	EndPort           string `json:"endPort,omitempty"`
	ICMPType          string `json:"icmpType,omitempty"`
	ICMPCode          string `json:"icmpCode,omitempty"`
}

type listSecurityGroupResponse struct {
	Count                     int             `json:"count"`
	ListSecurityGroupResponse []SecurityGroup `json:"listSecurityGroupResponse"`
}

// Service provides security group API operations.
type Service struct {
	client *httpclient.Client
}

// NewService creates a new security group Service.
func NewService(client *httpclient.Client) *Service { return &Service{client: client} }

// List returns security groups. uuid is an optional filter.
func (s *Service) List(ctx context.Context, uuid string) ([]SecurityGroup, error) {
	q := url.Values{}
	if uuid != "" {
		q.Set("uuid", uuid)
	}
	var resp listSecurityGroupResponse
	if err := s.client.Get(ctx, "/restapi/securitygroup/securityList", q, &resp); err != nil {
		return nil, fmt.Errorf("listing security groups: %w", err)
	}
	return resp.ListSecurityGroupResponse, nil
}

// Get returns a single security group by UUID.
func (s *Service) Get(ctx context.Context, uuid string) (*SecurityGroup, error) {
	groups, err := s.List(ctx, uuid)
	if err != nil {
		return nil, err
	}
	if len(groups) == 0 {
		return nil, fmt.Errorf("security group %q not found", uuid)
	}
	return &groups[0], nil
}

// Create creates a new security group.
func (s *Service) Create(ctx context.Context, req CreateGroupRequest) (*SecurityGroup, error) {
	var resp listSecurityGroupResponse
	if err := s.client.Post(ctx, "/restapi/securitygroup/createSecurityGroup", req, &resp); err != nil {
		return nil, fmt.Errorf("creating security group: %w", err)
	}
	if len(resp.ListSecurityGroupResponse) == 0 {
		return nil, fmt.Errorf("create security group returned empty response")
	}
	return &resp.ListSecurityGroupResponse[0], nil
}

// Delete removes a security group by UUID.
func (s *Service) Delete(ctx context.Context, uuid string) error {
	if err := s.client.Delete(ctx, "/restapi/securitygroup/deleteSecurityGroup/"+uuid, nil); err != nil {
		return fmt.Errorf("deleting security group %s: %w", uuid, err)
	}
	return nil
}

// CreateFirewallRule adds an inbound rule to a security group.
func (s *Service) CreateFirewallRule(ctx context.Context, req CreateFirewallRuleRequest) (*SecurityGroup, error) {
	var resp listSecurityGroupResponse
	if err := s.client.Post(ctx, "/restapi/securitygroup/createSecurityGroupFirewallRule", req, &resp); err != nil {
		return nil, fmt.Errorf("creating security group firewall rule: %w", err)
	}
	if len(resp.ListSecurityGroupResponse) == 0 {
		return nil, fmt.Errorf("create firewall rule returned empty response")
	}
	return &resp.ListSecurityGroupResponse[0], nil
}

// CreateEgressRule adds an outbound rule to a security group.
func (s *Service) CreateEgressRule(ctx context.Context, req CreateEgressRuleRequest) (*SecurityGroup, error) {
	var resp listSecurityGroupResponse
	if err := s.client.Post(ctx, "/restapi/securitygroup/createSecurityGroupEgressRule", req, &resp); err != nil {
		return nil, fmt.Errorf("creating security group egress rule: %w", err)
	}
	if len(resp.ListSecurityGroupResponse) == 0 {
		return nil, fmt.Errorf("create egress rule returned empty response")
	}
	return &resp.ListSecurityGroupResponse[0], nil
}

// DeleteRule deletes an inbound or egress rule from a security group.
// ruleType must be "firewall" or "egress".
func (s *Service) DeleteRule(ctx context.Context, sgUUID, ruleType, ruleUUID string) error {
	q := url.Values{
		"securityGroupUuid": {sgUUID},
		"ruleType":          {ruleType},
		"ruleUuid":          {ruleUUID},
	}
	if err := s.client.Delete(ctx, "/restapi/securitygroup/deleteSecurityGroupRule", q); err != nil {
		return fmt.Errorf("deleting security group rule: %w", err)
	}
	return nil
}
