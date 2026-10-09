// Package acl provides ZCP Network ACL API operations.
package acl

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"

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

// ruleListResponse is the paginated envelope returned by the ACL rule list API.
type ruleListResponse struct {
	Status      string `json:"status"`
	CurrentPage int    `json:"current_page"`
	Data        []Rule `json:"data"`
	Total       *int   `json:"total"`
}

// ListRulesOptions controls optional ACL rule pagination. Zero values retain
// ListRules' complete-list behavior.
type ListRulesOptions struct {
	MaxItems      int
	StartingToken string
	PageSize      int
	NoPaginate    bool
}

// ListRulesResult contains a page of ACL rules and an optional local token for
// fetching the next page.
type ListRulesResult struct {
	Rules     []Rule
	NextToken string
}

type listRulesToken struct {
	Version  int    `json:"v"`
	VPC      string `json:"vpc"`
	ACL      string `json:"acl"`
	Page     int    `json:"page"`
	Offset   int    `json:"offset"`
	Consumed int    `json:"consumed"`
	PageSize int    `json:"page_size"`
	Total    *int   `json:"total,omitempty"`
}

// Service provides Network ACL API operations.
type Service struct {
	client *httpclient.Client
}

// NewService creates a new ACL Service.
func NewService(client *httpclient.Client) *Service {
	return &Service{client: client}
}

// maxListPages bounds paginated ListRules loops so a server that reports an
// invalid total cannot cause an unbounded request loop.
const maxListPages = 1000

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

// ListRules returns all rules inside an ACL list.
func (s *Service) ListRules(ctx context.Context, vpcSlug, aclID string) ([]Rule, error) {
	result, err := s.ListRulesWithOptions(ctx, vpcSlug, aclID, ListRulesOptions{})
	if err != nil {
		return nil, err
	}
	return result.Rules, nil
}

// ListRulesWithOptions lists ACL rules with optional bounded pagination.
func (s *Service) ListRulesWithOptions(ctx context.Context, vpcSlug, aclID string, options ListRulesOptions) (ListRulesResult, error) {
	if options.MaxItems < 0 {
		return ListRulesResult{}, fmt.Errorf("listing rules for ACL %s: max items must not be negative", aclID)
	}
	if options.PageSize < 0 {
		return ListRulesResult{}, fmt.Errorf("listing rules for ACL %s: page size must not be negative", aclID)
	}
	if options.NoPaginate && (options.MaxItems != 0 || options.StartingToken != "" || options.PageSize != 0) {
		return ListRulesResult{}, fmt.Errorf("listing rules for ACL %s: no-paginate cannot be combined with pagination options", aclID)
	}

	path := "/vpcs/" + vpcSlug + "/network-acl-list/" + aclID + "/network-acl"
	page, offset, consumed := 1, 0, 0
	var reportedTotal *int
	if options.StartingToken != "" {
		token, err := decodeListRulesToken(options.StartingToken)
		if err != nil {
			return ListRulesResult{}, fmt.Errorf("listing rules for ACL %s: invalid starting token: %w", aclID, err)
		}
		if token.VPC != vpcSlug || token.ACL != aclID {
			return ListRulesResult{}, fmt.Errorf("listing rules for ACL %s: starting token is for a different ACL", aclID)
		}
		if options.PageSize != 0 && options.PageSize != token.PageSize {
			return ListRulesResult{}, fmt.Errorf("listing rules for ACL %s: page size must match the starting token", aclID)
		}
		page, offset, consumed = token.Page, token.Offset, token.Consumed
		options.PageSize, reportedTotal = token.PageSize, token.Total
	}

	if options.NoPaginate {
		resp, err := s.listRulesPage(ctx, path, 1, 0)
		if err != nil {
			return ListRulesResult{}, err
		}
		if _, err := validateRuleListResponse(aclID, 1, nil, resp); err != nil {
			return ListRulesResult{}, err
		}
		result := ListRulesResult{Rules: resp.Data}
		if resp.Total != nil && len(resp.Data) < *resp.Total {
			token, err := encodeListRulesToken(listRulesToken{
				Version: 1, VPC: vpcSlug, ACL: aclID, Page: 2,
				Consumed: len(resp.Data), Total: resp.Total,
			})
			if err != nil {
				return ListRulesResult{}, fmt.Errorf("listing rules for ACL %s: encoding continuation token: %w", aclID, err)
			}
			result.NextToken = token
		}
		return result, nil
	}

	if page < 1 || page > maxListPages || offset < 0 || consumed < 0 {
		return ListRulesResult{}, fmt.Errorf("listing rules for ACL %s: invalid starting token pagination state", aclID)
	}

	var rules []Rule
	for requests := 0; requests < maxListPages; requests++ {
		if page > maxListPages {
			return ListRulesResult{}, fmt.Errorf("listing rules for ACL %s: exceeded %d pages without reaching the reported total", aclID, maxListPages)
		}
		resp, err := s.listRulesPage(ctx, path, page, options.PageSize)
		if err != nil {
			return ListRulesResult{}, err
		}
		reportedTotal, err = validateRuleListResponse(aclID, page, reportedTotal, resp)
		if err != nil {
			return ListRulesResult{}, err
		}
		if offset > len(resp.Data) {
			return ListRulesResult{}, fmt.Errorf("listing rules for ACL %s: starting token offset %d exceeds page %d", aclID, offset, page)
		}
		if reportedTotal != nil && len(resp.Data)-offset > *reportedTotal-consumed {
			return ListRulesResult{}, fmt.Errorf("listing rules for ACL %s: page %d returned more rules than the reported total", aclID, page)
		}

		available := resp.Data[offset:]
		take := len(available)
		if options.MaxItems > 0 && take > options.MaxItems-len(rules) {
			take = options.MaxItems - len(rules)
		}
		if take > int(^uint(0)>>1)-consumed {
			return ListRulesResult{}, fmt.Errorf("listing rules for ACL %s: pagination count overflow", aclID)
		}
		rules = append(rules, available[:take]...)
		consumed += take
		nextOffset := offset + take

		hasMore := reportedTotal != nil && consumed < *reportedTotal
		if reportedTotal == nil && nextOffset < len(resp.Data) {
			hasMore = true
		}
		if !hasMore {
			return ListRulesResult{Rules: rules}, nil
		}
		if options.MaxItems > 0 && len(rules) == options.MaxItems {
			nextPage := page
			if nextOffset == len(resp.Data) {
				nextPage++
				nextOffset = 0
			}
			if nextPage > maxListPages {
				return ListRulesResult{}, fmt.Errorf("listing rules for ACL %s: exceeded %d pages without reaching the reported total", aclID, maxListPages)
			}
			token, err := encodeListRulesToken(listRulesToken{
				Version: 1, VPC: vpcSlug, ACL: aclID, Page: nextPage,
				Offset: nextOffset, Consumed: consumed, PageSize: options.PageSize, Total: reportedTotal,
			})
			if err != nil {
				return ListRulesResult{}, fmt.Errorf("listing rules for ACL %s: encoding continuation token: %w", aclID, err)
			}
			return ListRulesResult{Rules: rules, NextToken: token}, nil
		}

		page++
		offset = 0
	}
	return ListRulesResult{}, fmt.Errorf("listing rules for ACL %s: exceeded %d pages without reaching the reported total", aclID, maxListPages)
}

func (s *Service) listRulesPage(ctx context.Context, path string, page, pageSize int) (ruleListResponse, error) {
	q := url.Values{}
	if page > 1 {
		q.Set("page", strconv.Itoa(page))
	}
	if pageSize > 0 {
		q.Set("per_page", strconv.Itoa(pageSize))
	}
	var resp ruleListResponse
	if err := s.client.Get(ctx, path, q, &resp); err != nil {
		return ruleListResponse{}, fmt.Errorf("listing rules: %w", err)
	}
	return resp, nil
}

func validateRuleListResponse(aclID string, page int, reportedTotal *int, resp ruleListResponse) (*int, error) {
	if resp.CurrentPage < 0 {
		return nil, fmt.Errorf("listing rules for ACL %s: API returned invalid negative page %d", aclID, resp.CurrentPage)
	}
	if page > 1 && resp.CurrentPage == 0 {
		return nil, fmt.Errorf("listing rules for ACL %s: requested page %d but the API did not return pagination metadata", aclID, page)
	}
	if resp.CurrentPage > 0 && resp.CurrentPage != page {
		return nil, fmt.Errorf("listing rules for ACL %s: requested page %d but the API returned page %d", aclID, page, resp.CurrentPage)
	}
	if resp.Total != nil && *resp.Total < 0 {
		return nil, fmt.Errorf("listing rules for ACL %s: API returned invalid negative total %d", aclID, *resp.Total)
	}
	if reportedTotal == nil {
		reportedTotal = resp.Total
	} else if resp.Total == nil || *resp.Total != *reportedTotal {
		return nil, fmt.Errorf("listing rules for ACL %s: API changed reported total on page %d", aclID, page)
	}
	if reportedTotal == nil {
		return nil, nil
	}
	if *reportedTotal == 0 {
		if len(resp.Data) > 0 {
			return nil, fmt.Errorf("listing rules for ACL %s: page %d returned rules despite a total of zero", aclID, page)
		}
		return reportedTotal, nil
	}
	if len(resp.Data) > *reportedTotal {
		return nil, fmt.Errorf("listing rules for ACL %s: page %d returned more rules than the reported total", aclID, page)
	}
	if len(resp.Data) == 0 {
		return nil, fmt.Errorf("listing rules for ACL %s: page %d was empty before reaching reported total %d", aclID, page, *reportedTotal)
	}
	return reportedTotal, nil
}

func encodeListRulesToken(token listRulesToken) (string, error) {
	b, err := json.Marshal(token)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func decodeListRulesToken(value string) (listRulesToken, error) {
	if strings.TrimSpace(value) == "" {
		return listRulesToken{}, fmt.Errorf("token is empty")
	}
	b, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil {
		return listRulesToken{}, err
	}
	var token listRulesToken
	if err := json.Unmarshal(b, &token); err != nil {
		return listRulesToken{}, err
	}
	if token.Version != 1 || token.VPC == "" || token.ACL == "" || token.Page < 1 || token.Page > maxListPages || token.Offset < 0 || token.Consumed < 0 || token.Offset > token.Consumed || token.PageSize < 0 || (token.Total == nil && token.Page > 1) || (token.Total != nil && (*token.Total < 0 || token.Consumed > *token.Total)) {
		return listRulesToken{}, fmt.Errorf("token has invalid pagination state")
	}
	return token, nil
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
