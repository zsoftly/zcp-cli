// Package egress provides ZCP egress rule API operations.
package egress

import (
	"context"
	"fmt"
	"net/url"

	"github.com/zsoftly/zcp-cli/internal/httpclient"
)

// EgressRule represents a ZCP egress firewall rule.
type EgressRule struct {
	UUID        string `json:"uuid"`
	Status      string `json:"status"`
	IsActive    bool   `json:"isActive"`
	Protocol    string `json:"protocol"`
	StartPort   string `json:"startPort"`
	EndPort     string `json:"endPort"`
	NetworkUUID string `json:"networkUuid"`
	CIDRList    string `json:"cidrList"`
	ICMPType    string `json:"icmpType"`
	ICMPCode    string `json:"icmpCode"`
	ZoneUUID    string `json:"zoneUuid"`
}

// CreateRequest holds parameters for creating an egress rule.
type CreateRequest struct {
	NetworkUUID string `json:"networkUuid"`
	Protocol    string `json:"protocol"`
	StartPort   string `json:"startPort,omitempty"`
	EndPort     string `json:"endPort,omitempty"`
	CIDRList    string `json:"cidrList,omitempty"`
	ICMPType    string `json:"icmpType,omitempty"`
	ICMPCode    string `json:"icmpCode,omitempty"`
}

type listEgressRuleResponse struct {
	Count                  int          `json:"count"`
	ListEgressRuleResponse []EgressRule `json:"listEgressRuleResponse"`
}

// Service provides egress rule API operations.
type Service struct {
	client *httpclient.Client
}

// NewService creates a new egress Service.
func NewService(client *httpclient.Client) *Service {
	return &Service{client: client}
}

// List returns egress rules. zoneUUID is required; uuid and networkUUID are optional filters.
func (s *Service) List(ctx context.Context, zoneUUID, uuid, networkUUID string) ([]EgressRule, error) {
	q := url.Values{"zoneUuid": {zoneUUID}}
	if uuid != "" {
		q.Set("uuid", uuid)
	}
	if networkUUID != "" {
		q.Set("networkUuid", networkUUID)
	}
	var resp listEgressRuleResponse
	if err := s.client.Get(ctx, "/restapi/egressrule/egressRuleList", q, &resp); err != nil {
		return nil, fmt.Errorf("listing egress rules: %w", err)
	}
	return resp.ListEgressRuleResponse, nil
}

// Create adds a new egress rule.
func (s *Service) Create(ctx context.Context, req CreateRequest) (*EgressRule, error) {
	var resp listEgressRuleResponse
	if err := s.client.Post(ctx, "/restapi/egressrule/createEgressRule", req, &resp); err != nil {
		return nil, fmt.Errorf("creating egress rule: %w", err)
	}
	if len(resp.ListEgressRuleResponse) == 0 {
		return nil, fmt.Errorf("create egress rule returned empty response")
	}
	return &resp.ListEgressRuleResponse[0], nil
}

// Delete removes an egress rule by UUID.
func (s *Service) Delete(ctx context.Context, uuid string) error {
	if err := s.client.Delete(ctx, "/restapi/egressrule/deleteEgressRule/"+uuid, nil); err != nil {
		return fmt.Errorf("deleting egress rule %s: %w", uuid, err)
	}
	return nil
}
