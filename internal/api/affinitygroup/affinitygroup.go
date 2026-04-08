// Package affinitygroup provides ZCP affinity group API operations.
package affinitygroup

import (
	"context"
	"fmt"

	"github.com/zsoftly/zcp-cli/internal/httpclient"
)

// AffinityGroup represents a ZCP affinity group.
type AffinityGroup struct {
	ID                   string         `json:"id"`
	Name                 string         `json:"name"`
	Slug                 string         `json:"slug"`
	Description          string         `json:"description"`
	Type                 string         `json:"type"`
	State                string         `json:"state"`
	Status               string         `json:"status"`
	AffinityGroupID      string         `json:"affinity_group_id"`
	RegionID             string         `json:"region_id"`
	CloudProviderID      string         `json:"cloud_provider_id"`
	CloudProviderSetupID string         `json:"cloud_provider_setup_id"`
	ProjectID            string         `json:"project_id"`
	AccountID            string         `json:"account_id"`
	CreatedAt            string         `json:"created_at"`
	UpdatedAt            string         `json:"updated_at"`
	Region               *Region        `json:"region,omitempty"`
	Project              *Project       `json:"project,omitempty"`
	CloudProvider        *CloudProvider `json:"cloud_provider,omitempty"`
}

// Region represents the region of an affinity group.
type Region struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Slug string `json:"slug"`
}

// Project represents the project of an affinity group.
type Project struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Slug string `json:"slug"`
}

// CloudProvider represents the cloud provider of an affinity group.
type CloudProvider struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	DisplayName string `json:"display_name"`
	Slug        string `json:"slug"`
}

// CreateRequest holds parameters for creating an affinity group.
type CreateRequest struct {
	Name          string `json:"name"`
	Type          string `json:"type"`
	Description   string `json:"description,omitempty"`
	Project       string `json:"project"`
	Region        string `json:"region"`
	CloudProvider string `json:"cloud_provider"`
}

// listResponse is the STKCNSL paginated response envelope.
type listResponse struct {
	Status  string          `json:"status"`
	Message string          `json:"message"`
	Data    []AffinityGroup `json:"data"`
	Total   int             `json:"total"`
}

// singleResponse is the STKCNSL single-object response envelope.
type singleResponse struct {
	Status  string        `json:"status"`
	Message string        `json:"message"`
	Data    AffinityGroup `json:"data"`
}

// Service provides affinity group API operations.
type Service struct {
	client *httpclient.Client
}

// NewService creates a new affinity group Service.
func NewService(client *httpclient.Client) *Service {
	return &Service{client: client}
}

// List returns all affinity groups.
func (s *Service) List(ctx context.Context) ([]AffinityGroup, error) {
	var resp listResponse
	if err := s.client.Get(ctx, "/affinity-groups", nil, &resp); err != nil {
		return nil, fmt.Errorf("listing affinity groups: %w", err)
	}
	return resp.Data, nil
}

// Create provisions a new affinity group.
func (s *Service) Create(ctx context.Context, req CreateRequest) (*AffinityGroup, error) {
	var resp singleResponse
	if err := s.client.Post(ctx, "/affinity-groups", req, &resp); err != nil {
		return nil, fmt.Errorf("creating affinity group: %w", err)
	}
	return &resp.Data, nil
}

// Delete removes an affinity group by slug.
func (s *Service) Delete(ctx context.Context, slug string) error {
	if err := s.client.Delete(ctx, "/affinity-groups/"+slug, nil); err != nil {
		return fmt.Errorf("deleting affinity group %s: %w", slug, err)
	}
	return nil
}
