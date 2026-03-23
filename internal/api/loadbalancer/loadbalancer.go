// Package loadbalancer provides ZCP Load Balancer Rule API operations.
package loadbalancer

import (
	"context"
	"fmt"
	"net/url"

	"github.com/zsoftly/zcp-cli/internal/httpclient"
)

// Rule represents a ZCP load balancer rule.
type Rule struct {
	UUID           string `json:"uuid"`
	Name           string `json:"name"`
	Status         string `json:"status"`
	State          string `json:"state"`
	IsActive       bool   `json:"isActive"`
	Algorithm      string `json:"algorithm"`
	PublicPort     string `json:"publicPort"`
	PrivatePort    string `json:"privatePort"`
	IPAddressUUID  string `json:"ipAddressUuid"`
	StickinessName string `json:"stickinessName"`
	ZoneUUID       string `json:"zoneUuid"`
}

// CreateRequest holds parameters for creating a load balancer rule.
type CreateRequest struct {
	Name                 string `json:"name"`
	PublicIPUUID         string `json:"publicIpUuid"`
	PublicPort           string `json:"publicport"`
	PrivatePort          string `json:"privateport"`
	Algorithm            string `json:"algorithm"`
	NetworkUUID          string `json:"networkUuid,omitempty"`
	StickinessPolicyUUID string `json:"stickinessPolicyUuid,omitempty"`
}

// UpdateRequest holds parameters for updating a load balancer rule.
type UpdateRequest struct {
	UUID      string `json:"uuid"`
	Name      string `json:"name"`
	Algorithm string `json:"algorithm"`
}

type listLoadBalancerRuleResponse struct {
	Count                        int    `json:"count"`
	ListLoadBalancerRuleResponse []Rule `json:"listLoadBalancerRuleResponse"`
}

// Service provides load balancer rule API operations.
type Service struct {
	client *httpclient.Client
}

// NewService creates a new load balancer Service.
func NewService(client *httpclient.Client) *Service {
	return &Service{client: client}
}

// List returns load balancer rules in a zone. zoneUUID is required; uuid and ipAddressUUID are optional filters.
func (s *Service) List(ctx context.Context, zoneUUID, uuid, ipAddressUUID string) ([]Rule, error) {
	q := url.Values{"zoneUuid": {zoneUUID}}
	if uuid != "" {
		q.Set("uuid", uuid)
	}
	if ipAddressUUID != "" {
		q.Set("ipAddressUuid", ipAddressUUID)
	}
	var resp listLoadBalancerRuleResponse
	if err := s.client.Get(ctx, "/restapi/loadbalancerrule/loadBalancerRuleList", q, &resp); err != nil {
		return nil, fmt.Errorf("listing load balancer rules: %w", err)
	}
	return resp.ListLoadBalancerRuleResponse, nil
}

// Create provisions a new load balancer rule.
func (s *Service) Create(ctx context.Context, req CreateRequest) (*Rule, error) {
	var resp listLoadBalancerRuleResponse
	if err := s.client.Post(ctx, "/restapi/loadbalancerrule/createLoadBalancerRule", req, &resp); err != nil {
		return nil, fmt.Errorf("creating load balancer rule: %w", err)
	}
	if len(resp.ListLoadBalancerRuleResponse) == 0 {
		return nil, fmt.Errorf("create load balancer rule returned empty response")
	}
	return &resp.ListLoadBalancerRuleResponse[0], nil
}

// Update modifies a load balancer rule's mutable attributes.
func (s *Service) Update(ctx context.Context, req UpdateRequest) (*Rule, error) {
	var resp listLoadBalancerRuleResponse
	if err := s.client.Put(ctx, "/restapi/loadbalancerrule/updateLoadBalancerRule", nil, req, &resp); err != nil {
		return nil, fmt.Errorf("updating load balancer rule %s: %w", req.UUID, err)
	}
	if len(resp.ListLoadBalancerRuleResponse) == 0 {
		return nil, fmt.Errorf("update load balancer rule returned empty response")
	}
	return &resp.ListLoadBalancerRuleResponse[0], nil
}

// Delete removes a load balancer rule by UUID.
func (s *Service) Delete(ctx context.Context, uuid string) error {
	if err := s.client.Delete(ctx, "/restapi/loadbalancerrule/deleteLoadBalancerRule/"+uuid, nil); err != nil {
		return fmt.Errorf("deleting load balancer rule %s: %w", uuid, err)
	}
	return nil
}
