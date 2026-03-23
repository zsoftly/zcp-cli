// Package firewall provides ZCP firewall rule API operations.
package firewall

import (
	"context"
	"fmt"
	"net/url"

	"github.com/zsoftly/zcp-cli/internal/httpclient"
)

// FirewallRule represents a ZCP firewall rule.
type FirewallRule struct {
	UUID          string `json:"uuid"`
	Status        string `json:"status"`
	IsActive      bool   `json:"isActive"`
	Protocol      string `json:"protocol"`
	StartPort     string `json:"startPort"`
	EndPort       string `json:"endPort"`
	CIDRList      string `json:"cidrList"`
	IPAddressUUID string `json:"ipAddressUuid"`
	ICMPType      string `json:"icmpType"`
	ICMPCode      string `json:"icmpCode"`
	ZoneUUID      string `json:"zoneUuid"`
}

// CreateRequest holds parameters for creating a firewall rule.
type CreateRequest struct {
	IPAddressUUID string `json:"ipAddressUuid"`
	Protocol      string `json:"protocol"`
	StartPort     string `json:"startPort,omitempty"`
	EndPort       string `json:"endPort,omitempty"`
	CIDRList      string `json:"cidrList,omitempty"`
	ICMPType      string `json:"icmpType,omitempty"`
	ICMPCode      string `json:"icmpCode,omitempty"`
}

type listFirewallRuleResponse struct {
	Count                    int            `json:"count"`
	ListFirewallRuleResponse []FirewallRule `json:"listFirewallRuleResponse"`
}

// Service provides firewall rule API operations.
type Service struct {
	client *httpclient.Client
}

// NewService creates a new firewall Service.
func NewService(client *httpclient.Client) *Service {
	return &Service{client: client}
}

// List returns firewall rules. zoneUUID is required; uuid and ipAddressUUID are optional filters.
func (s *Service) List(ctx context.Context, zoneUUID, uuid, ipAddressUUID string) ([]FirewallRule, error) {
	q := url.Values{"zoneUuid": {zoneUUID}}
	if uuid != "" {
		q.Set("uuid", uuid)
	}
	if ipAddressUUID != "" {
		q.Set("ipAddressUuid", ipAddressUUID)
	}
	var resp listFirewallRuleResponse
	if err := s.client.Get(ctx, "/restapi/firewallrule/firewallRuleList", q, &resp); err != nil {
		return nil, fmt.Errorf("listing firewall rules: %w", err)
	}
	return resp.ListFirewallRuleResponse, nil
}

// Create adds a new firewall rule.
func (s *Service) Create(ctx context.Context, req CreateRequest) (*FirewallRule, error) {
	var resp listFirewallRuleResponse
	if err := s.client.Post(ctx, "/restapi/firewallrule/createFirewallRule", req, &resp); err != nil {
		return nil, fmt.Errorf("creating firewall rule: %w", err)
	}
	if len(resp.ListFirewallRuleResponse) == 0 {
		return nil, fmt.Errorf("create firewall rule returned empty response")
	}
	return &resp.ListFirewallRuleResponse[0], nil
}

// Delete removes a firewall rule by UUID.
func (s *Service) Delete(ctx context.Context, uuid string) error {
	if err := s.client.Delete(ctx, "/restapi/firewallrule/deleteFirewallRule/"+uuid, nil); err != nil {
		return fmt.Errorf("deleting firewall rule %s: %w", uuid, err)
	}
	return nil
}
