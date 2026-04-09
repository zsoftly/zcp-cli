// Package autoscale provides ZCP VM Autoscale API operations (STKCNSL).
package autoscale

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/zsoftly/zcp-cli/internal/httpclient"
)

const basePath = "/autoscale"

// envelope is the STKCNSL standard response wrapper.
type envelope struct {
	Status string          `json:"status"`
	Data   json.RawMessage `json:"data"`
}

// AutoscaleGroup represents a VM autoscale group.
type AutoscaleGroup struct {
	Slug           string      `json:"slug"`
	Name           string      `json:"name"`
	State          string      `json:"state"`
	Plan           string      `json:"plan"`
	Template       string      `json:"template"`
	MinInstances   int         `json:"minInstances"`
	MaxInstances   int         `json:"maxInstances"`
	CurrentCount   int         `json:"currentCount"`
	CooldownPeriod int         `json:"cooldownPeriod"`
	ZoneSlug       string      `json:"zoneSlug"`
	NetworkSlug    string      `json:"networkSlug"`
	CreatedAt      string      `json:"createdAt"`
	UpdatedAt      string      `json:"updatedAt"`
	Policies       []Policy    `json:"policies,omitempty"`
	Conditions     []Condition `json:"conditions,omitempty"`
}

// Policy represents a scale-up policy for an autoscale group.
type Policy struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Metric      string `json:"metric"`
	Operator    string `json:"operator"`
	Threshold   int    `json:"threshold"`
	Duration    int    `json:"duration"`
	ScaleAmount int    `json:"scaleAmount"`
	Cooldown    int    `json:"cooldown"`
	CreatedAt   string `json:"createdAt"`
	UpdatedAt   string `json:"updatedAt"`
}

// Condition represents a scale-down condition for an autoscale group.
type Condition struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Metric      string `json:"metric"`
	Operator    string `json:"operator"`
	Threshold   int    `json:"threshold"`
	Duration    int    `json:"duration"`
	ScaleAmount int    `json:"scaleAmount"`
	Cooldown    int    `json:"cooldown"`
	CreatedAt   string `json:"createdAt"`
	UpdatedAt   string `json:"updatedAt"`
}

// CreateRequest holds parameters for creating an autoscale group.
type CreateRequest struct {
	Name           string `json:"name"`
	Plan           string `json:"plan"`
	Template       string `json:"template"`
	MinInstances   int    `json:"minInstances"`
	MaxInstances   int    `json:"maxInstances"`
	CooldownPeriod int    `json:"cooldownPeriod,omitempty"`
	ZoneSlug       string `json:"zoneSlug"`
	NetworkSlug    string `json:"networkSlug,omitempty"`
	CloudProvider  string `json:"cloud_provider"`
	Region         string `json:"region"`
	Project        string `json:"project"`
}

// ChangePlanRequest holds parameters for changing an autoscale group's plan.
type ChangePlanRequest struct {
	Plan string `json:"plan"`
}

// ChangeTemplateRequest holds parameters for changing an autoscale group's template.
type ChangeTemplateRequest struct {
	Template string `json:"template"`
}

// PolicyRequest holds parameters for creating or updating a scale-up policy.
type PolicyRequest struct {
	Name        string `json:"name"`
	Metric      string `json:"metric"`
	Operator    string `json:"operator"`
	Threshold   int    `json:"threshold"`
	Duration    int    `json:"duration"`
	ScaleAmount int    `json:"scaleAmount"`
	Cooldown    int    `json:"cooldown,omitempty"`
}

// ConditionRequest holds parameters for creating or updating a scale-down condition.
type ConditionRequest struct {
	Name        string `json:"name"`
	Metric      string `json:"metric"`
	Operator    string `json:"operator"`
	Threshold   int    `json:"threshold"`
	Duration    int    `json:"duration"`
	ScaleAmount int    `json:"scaleAmount"`
	Cooldown    int    `json:"cooldown,omitempty"`
}

// Service provides autoscale API operations.
type Service struct {
	client *httpclient.Client
}

// NewService creates a new autoscale Service.
func NewService(client *httpclient.Client) *Service {
	return &Service{client: client}
}

// decodeOne decodes a single object from the STKCNSL envelope data field.
func decodeOne[T any](raw json.RawMessage) (*T, error) {
	var v T
	if err := json.Unmarshal(raw, &v); err != nil {
		return nil, fmt.Errorf("decoding response data: %w", err)
	}
	return &v, nil
}

// decodeList decodes an array from the STKCNSL envelope data field.
func decodeList[T any](raw json.RawMessage) ([]T, error) {
	var v []T
	if err := json.Unmarshal(raw, &v); err != nil {
		return nil, fmt.Errorf("decoding response data: %w", err)
	}
	return v, nil
}

// List returns all autoscale groups.
func (s *Service) List(ctx context.Context) ([]AutoscaleGroup, error) {
	var env envelope
	if err := s.client.Get(ctx, basePath, nil, &env); err != nil {
		return nil, fmt.Errorf("listing autoscale groups: %w", err)
	}
	groups, err := decodeList[AutoscaleGroup](env.Data)
	if err != nil {
		return nil, fmt.Errorf("listing autoscale groups: %w", err)
	}
	return groups, nil
}

// Create provisions a new autoscale group.
func (s *Service) Create(ctx context.Context, req CreateRequest) (*AutoscaleGroup, error) {
	var env envelope
	if err := s.client.Post(ctx, basePath, req, &env); err != nil {
		return nil, fmt.Errorf("creating autoscale group: %w", err)
	}
	group, err := decodeOne[AutoscaleGroup](env.Data)
	if err != nil {
		return nil, fmt.Errorf("creating autoscale group: %w", err)
	}
	return group, nil
}

// ChangePlan changes the compute plan of an autoscale group.
func (s *Service) ChangePlan(ctx context.Context, slug, plan string) (*AutoscaleGroup, error) {
	path := fmt.Sprintf("%s/%s/change-plan", basePath, slug)
	var env envelope
	if err := s.client.Post(ctx, path, ChangePlanRequest{Plan: plan}, &env); err != nil {
		return nil, fmt.Errorf("changing plan for autoscale group %s: %w", slug, err)
	}
	group, err := decodeOne[AutoscaleGroup](env.Data)
	if err != nil {
		return nil, fmt.Errorf("changing plan for autoscale group %s: %w", slug, err)
	}
	return group, nil
}

// ChangeTemplate changes the template of an autoscale group.
func (s *Service) ChangeTemplate(ctx context.Context, slug, template string) (*AutoscaleGroup, error) {
	path := fmt.Sprintf("%s/%s/change-template", basePath, slug)
	var env envelope
	if err := s.client.Post(ctx, path, ChangeTemplateRequest{Template: template}, &env); err != nil {
		return nil, fmt.Errorf("changing template for autoscale group %s: %w", slug, err)
	}
	group, err := decodeOne[AutoscaleGroup](env.Data)
	if err != nil {
		return nil, fmt.Errorf("changing template for autoscale group %s: %w", slug, err)
	}
	return group, nil
}

// Enable enables an autoscale group.
func (s *Service) Enable(ctx context.Context, slug string) (*AutoscaleGroup, error) {
	path := fmt.Sprintf("%s/%s/enable", basePath, slug)
	var env envelope
	if err := s.client.Put(ctx, path, nil, nil, &env); err != nil {
		return nil, fmt.Errorf("enabling autoscale group %s: %w", slug, err)
	}
	group, err := decodeOne[AutoscaleGroup](env.Data)
	if err != nil {
		return nil, fmt.Errorf("enabling autoscale group %s: %w", slug, err)
	}
	return group, nil
}

// Disable disables an autoscale group.
func (s *Service) Disable(ctx context.Context, slug string) (*AutoscaleGroup, error) {
	path := fmt.Sprintf("%s/%s/disable", basePath, slug)
	var env envelope
	if err := s.client.Put(ctx, path, nil, nil, &env); err != nil {
		return nil, fmt.Errorf("disabling autoscale group %s: %w", slug, err)
	}
	group, err := decodeOne[AutoscaleGroup](env.Data)
	if err != nil {
		return nil, fmt.Errorf("disabling autoscale group %s: %w", slug, err)
	}
	return group, nil
}

// CreatePolicy creates a scale-up policy on an autoscale group.
func (s *Service) CreatePolicy(ctx context.Context, slug string, req PolicyRequest) (*Policy, error) {
	path := fmt.Sprintf("%s/%s/policies", basePath, slug)
	var env envelope
	if err := s.client.Post(ctx, path, req, &env); err != nil {
		return nil, fmt.Errorf("creating policy for autoscale group %s: %w", slug, err)
	}
	policy, err := decodeOne[Policy](env.Data)
	if err != nil {
		return nil, fmt.Errorf("creating policy for autoscale group %s: %w", slug, err)
	}
	return policy, nil
}

// UpdatePolicy updates a scale-up policy on an autoscale group.
func (s *Service) UpdatePolicy(ctx context.Context, slug string, policyID int, req PolicyRequest) (*Policy, error) {
	path := fmt.Sprintf("%s/%s/policies/%d", basePath, slug, policyID)
	var env envelope
	if err := s.client.Put(ctx, path, nil, req, &env); err != nil {
		return nil, fmt.Errorf("updating policy %d for autoscale group %s: %w", policyID, slug, err)
	}
	policy, err := decodeOne[Policy](env.Data)
	if err != nil {
		return nil, fmt.Errorf("updating policy %d for autoscale group %s: %w", policyID, slug, err)
	}
	return policy, nil
}

// DeletePolicy deletes a scale-up policy from an autoscale group.
func (s *Service) DeletePolicy(ctx context.Context, slug string, policyID int) error {
	path := fmt.Sprintf("%s/%s/policies/%d", basePath, slug, policyID)
	if err := s.client.Delete(ctx, path, nil); err != nil {
		return fmt.Errorf("deleting policy %d for autoscale group %s: %w", policyID, slug, err)
	}
	return nil
}

// CreateCondition creates a scale-down condition on an autoscale group.
func (s *Service) CreateCondition(ctx context.Context, slug string, req ConditionRequest) (*Condition, error) {
	path := fmt.Sprintf("%s/%s/conditions", basePath, slug)
	var env envelope
	if err := s.client.Post(ctx, path, req, &env); err != nil {
		return nil, fmt.Errorf("creating condition for autoscale group %s: %w", slug, err)
	}
	cond, err := decodeOne[Condition](env.Data)
	if err != nil {
		return nil, fmt.Errorf("creating condition for autoscale group %s: %w", slug, err)
	}
	return cond, nil
}

// UpdateCondition updates a scale-down condition on an autoscale group.
func (s *Service) UpdateCondition(ctx context.Context, slug string, conditionID int, req ConditionRequest) (*Condition, error) {
	path := fmt.Sprintf("%s/%s/conditions/%d", basePath, slug, conditionID)
	var env envelope
	if err := s.client.Put(ctx, path, nil, req, &env); err != nil {
		return nil, fmt.Errorf("updating condition %d for autoscale group %s: %w", conditionID, slug, err)
	}
	cond, err := decodeOne[Condition](env.Data)
	if err != nil {
		return nil, fmt.Errorf("updating condition %d for autoscale group %s: %w", conditionID, slug, err)
	}
	return cond, nil
}

// DeleteCondition deletes a scale-down condition from an autoscale group.
func (s *Service) DeleteCondition(ctx context.Context, slug string, conditionID int) error {
	path := fmt.Sprintf("%s/%s/conditions/%d", basePath, slug, conditionID)
	if err := s.client.Delete(ctx, path, nil); err != nil {
		return fmt.Errorf("deleting condition %d for autoscale group %s: %w", conditionID, slug, err)
	}
	return nil
}
