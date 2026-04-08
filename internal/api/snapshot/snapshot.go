// Package snapshot provides ZCP block storage snapshot API operations
// targeting the STKCNSL API.
package snapshot

import (
	"context"
	"fmt"
	"net/url"

	"github.com/zsoftly/zcp-cli/internal/httpclient"
)

// Snapshot represents a STKCNSL block storage snapshot.
type Snapshot struct {
	ID                   string  `json:"id"`
	Name                 string  `json:"name"`
	Slug                 string  `json:"slug"`
	BlockstorageID       string  `json:"blockstorage_id"`
	UserID               string  `json:"user_id"`
	AccountID            string  `json:"account_id"`
	ProjectID            string  `json:"project_id"`
	RegionID             string  `json:"region_id"`
	CloudProviderID      string  `json:"cloud_provider_id"`
	CloudProviderSetupID string  `json:"cloud_provider_setup_id"`
	RequestStatus        bool    `json:"request_status"`
	ServiceName          string  `json:"service_name"`
	ServiceDisplayName   string  `json:"service_display_name"`
	AllTimeConsumption   float64 `json:"all_time_consumption"`
	HasContract          bool    `json:"has_contract"`
	FrozenAt             *string `json:"frozen_at"`
	SuspendedAt          *string `json:"suspended_at"`
	TerminatedAt         *string `json:"terminated_at"`
	CreatedAt            string  `json:"created_at"`
	UpdatedAt            string  `json:"updated_at"`
	DeletedAt            *string `json:"deleted_at"`
}

// listResponse is the STKCNSL paginated envelope for block storage snapshots.
type listResponse struct {
	Status      string     `json:"status"`
	Message     string     `json:"message"`
	CurrentPage int        `json:"current_page"`
	Data        []Snapshot `json:"data"`
	Total       int        `json:"total"`
}

// singleResponse is used when the API returns a single snapshot in `data`.
type singleResponse struct {
	Status  string   `json:"status"`
	Message string   `json:"message"`
	Data    Snapshot `json:"data"`
}

// CreateRequest holds parameters for creating a block storage snapshot.
type CreateRequest struct {
	Name          string `json:"name"`
	Plan          string `json:"plan"`
	Service       string `json:"service"`
	IsMemory      bool   `json:"is_memory"`
	Project       string `json:"project"`
	CloudProvider string `json:"cloud_provider"`
	Region        string `json:"region"`
	BillingCycle  string `json:"billing_cycle"`
	Coupon        string `json:"coupon,omitempty"`
}

// Service provides block storage snapshot API operations.
type Service struct {
	client *httpclient.Client
}

// NewService creates a new snapshot Service.
func NewService(client *httpclient.Client) *Service {
	return &Service{client: client}
}

// List returns block storage snapshots.
func (s *Service) List(ctx context.Context) ([]Snapshot, error) {
	q := url.Values{
		"include": {"blockstorage,region,cloud_provider,project"},
	}
	var resp listResponse
	if err := s.client.Get(ctx, "/blockstorages/snapshots", q, &resp); err != nil {
		return nil, fmt.Errorf("listing block storage snapshots: %w", err)
	}
	return resp.Data, nil
}

// Create creates a new block storage snapshot.
func (s *Service) Create(ctx context.Context, blockstorageSlug string, req CreateRequest) (*Snapshot, error) {
	var resp singleResponse
	path := fmt.Sprintf("/blockstorages/%s/snapshots", blockstorageSlug)
	if err := s.client.Post(ctx, path, req, &resp); err != nil {
		return nil, fmt.Errorf("creating block storage snapshot: %w", err)
	}
	return &resp.Data, nil
}

// Revert reverts a block storage volume to a snapshot state.
func (s *Service) Revert(ctx context.Context, blockstorageSlug, snapshotSlug string) (*Snapshot, error) {
	var resp singleResponse
	path := fmt.Sprintf("/blockstorages/%s/snapshots/%s/revert", blockstorageSlug, snapshotSlug)
	if err := s.client.Post(ctx, path, nil, &resp); err != nil {
		return nil, fmt.Errorf("reverting block storage snapshot %s: %w", snapshotSlug, err)
	}
	return &resp.Data, nil
}
