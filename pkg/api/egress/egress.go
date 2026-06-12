// Package egress provides ZCP egress rule API operations.
//
// In the STKCNSL API, egress rules are nested under networks:
//
//	GET    /networks/{SLUG}/egress-firewall-rules
//	POST   /networks/{SLUG}/egress-firewall-rules
//	DELETE /networks/{SLUG}/egress-firewall-rules/{ID}
//
// This package delegates to the network package's egress methods but preserves
// the Service/NewService pattern for backward compatibility with the commands layer.
package egress

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/zsoftly/zcp-cli/pkg/httpclient"
)

// EgressRule represents a ZCP egress firewall rule.
type EgressRule struct {
	ID        string `json:"id"`
	Protocol  string `json:"protocol"`
	StartPort string `json:"start_port"`
	EndPort   string `json:"end_port"`
	CIDR      string `json:"cidr"`
	DestCIDR  string `json:"destcidr_list"`
	ICMPType  string `json:"icmp_type"`
	ICMPCode  string `json:"icmp_code"`
	Status    string `json:"status"`
}

// flexString trims quotes from a raw JSON scalar — the live API returns ports
// and ICMP fields as numbers (80) while older deployments use strings ("80").
func flexString(raw json.RawMessage) string {
	v := strings.Trim(strings.TrimSpace(string(raw)), `"`)
	if v == "null" {
		return ""
	}
	return v
}

// UnmarshalJSON tolerates number-or-string port/ICMP fields and pulls the
// rule state from the provider's _original block (the top level has none).
func (r *EgressRule) UnmarshalJSON(b []byte) error {
	type raw struct {
		ID        string          `json:"id"`
		Protocol  string          `json:"protocol"`
		StartPort json.RawMessage `json:"start_port"`
		EndPort   json.RawMessage `json:"end_port"`
		CIDR      string          `json:"cidr"`
		DestCIDR  string          `json:"destcidr_list"`
		ICMPType  json.RawMessage `json:"icmp_type"`
		ICMPCode  json.RawMessage `json:"icmp_code"`
		Status    string          `json:"status"`
		Original  struct {
			State string `json:"state"`
		} `json:"_original"`
	}
	var v raw
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}
	r.ID = v.ID
	r.Protocol = v.Protocol
	r.StartPort = flexString(v.StartPort)
	r.EndPort = flexString(v.EndPort)
	r.CIDR = v.CIDR
	r.DestCIDR = v.DestCIDR
	r.ICMPType = flexString(v.ICMPType)
	r.ICMPCode = flexString(v.ICMPCode)
	r.Status = v.Status
	if r.Status == "" {
		r.Status = v.Original.State
	}
	return nil
}

// CreateRequest holds parameters for creating an egress rule.
type CreateRequest struct {
	NetworkSlug string `json:"-"`
	Protocol    string `json:"protocol"`
	StartPort   string `json:"start_port,omitempty"`
	EndPort     string `json:"end_port,omitempty"`
	CIDR        string `json:"cidr,omitempty"`
	ICMPType    string `json:"icmp_type,omitempty"`
	ICMPCode    string `json:"icmp_code,omitempty"`
}

type listEgressRuleResponse struct {
	Status string       `json:"status"`
	Data   []EgressRule `json:"data"`
}

type singleEgressRuleResponse struct {
	Status string     `json:"status"`
	Data   EgressRule `json:"data"`
}

// Service provides egress rule API operations.
type Service struct {
	client *httpclient.Client
}

// NewService creates a new egress Service.
func NewService(client *httpclient.Client) *Service {
	return &Service{client: client}
}

// List returns egress rules for a network identified by slug.
func (s *Service) List(ctx context.Context, networkSlug string) ([]EgressRule, error) {
	var resp listEgressRuleResponse
	if err := s.client.Get(ctx, "/networks/"+networkSlug+"/egress-firewall-rules", nil, &resp); err != nil {
		return nil, fmt.Errorf("listing egress rules for network %s: %w", networkSlug, err)
	}
	return resp.Data, nil
}

// Create adds a new egress rule to a network. The create endpoint returns
// data:null — fall back to the list and return the rule matching the request.
func (s *Service) Create(ctx context.Context, req CreateRequest) (*EgressRule, error) {
	var resp singleEgressRuleResponse
	if err := s.client.Post(ctx, "/networks/"+req.NetworkSlug+"/egress-firewall-rules", req, &resp); err != nil {
		return nil, fmt.Errorf("creating egress rule for network %s: %w", req.NetworkSlug, err)
	}
	if resp.Data.ID != "" {
		return &resp.Data, nil
	}
	rules, lerr := s.List(ctx, req.NetworkSlug)
	if lerr != nil {
		return nil, fmt.Errorf("egress rule for network %s was created, but fetching it back failed: %w", req.NetworkSlug, lerr)
	}
	for i := range rules {
		r := &rules[i]
		if r.Protocol != req.Protocol || r.StartPort != req.StartPort || r.EndPort != req.EndPort {
			continue
		}
		// The API echoes the requested CIDR in destcidr_list; top-level cidr
		// is the network's source CIDR and must not be compared here.
		if req.CIDR != "" && r.DestCIDR != req.CIDR {
			continue
		}
		return r, nil
	}
	return nil, fmt.Errorf("egress rule for network %s was created, but is not yet visible in the rule list — check with: zcp egress list --network %s", req.NetworkSlug, req.NetworkSlug)
}

// Delete removes an egress rule by ID from the given network.
func (s *Service) Delete(ctx context.Context, networkSlug string, ruleID string) error {
	path := fmt.Sprintf("/networks/%s/egress-firewall-rules/%s", networkSlug, ruleID)
	if err := s.client.Delete(ctx, path, nil); err != nil {
		return fmt.Errorf("deleting egress rule %s for network %s: %w", ruleID, networkSlug, err)
	}
	return nil
}
