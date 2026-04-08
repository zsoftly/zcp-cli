// Package backup provides ZCP block storage backup API operations
// targeting the STKCNSL API.
package backup

import (
	"context"
	"fmt"
	"net/url"

	"github.com/zsoftly/zcp-cli/internal/httpclient"
)

// Backup represents a STKCNSL block storage backup.
type Backup struct {
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
	Interval             string  `json:"interval"`
	At                   int     `json:"at"`
	Immediate            bool    `json:"immediate"`
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

// listResponse is the STKCNSL paginated envelope for block storage backups.
type listResponse struct {
	Status      string   `json:"status"`
	Message     string   `json:"message"`
	CurrentPage int      `json:"current_page"`
	Data        []Backup `json:"data"`
	Total       int      `json:"total"`
}

// singleResponse is used when the API returns a single backup in `data`.
type singleResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
	Data    Backup `json:"data"`
}

// CreateRequest holds parameters for creating a block storage backup.
type CreateRequest struct {
	Interval      string `json:"interval"`
	At            int    `json:"at"`
	Immediate     int    `json:"immediate"`
	CloudProvider string `json:"cloud_provider"`
	Region        string `json:"region"`
	BillingCycle  string `json:"billing_cycle"`
	Plan          string `json:"plan"`
	PseudoService string `json:"psudo_service"`
	Project       string `json:"project"`
}

// Service provides block storage backup API operations.
type Service struct {
	client *httpclient.Client
}

// NewService creates a new backup Service.
func NewService(client *httpclient.Client) *Service {
	return &Service{client: client}
}

// List returns block storage backups.
func (s *Service) List(ctx context.Context) ([]Backup, error) {
	q := url.Values{}
	var resp listResponse
	if err := s.client.Get(ctx, "/blockstorages/backups", q, &resp); err != nil {
		return nil, fmt.Errorf("listing block storage backups: %w", err)
	}
	return resp.Data, nil
}

// Create creates a new block storage backup.
func (s *Service) Create(ctx context.Context, blockstorageSlug string, req CreateRequest) (*Backup, error) {
	var resp singleResponse
	path := fmt.Sprintf("/blockstorages/%s/backups", blockstorageSlug)
	if err := s.client.Post(ctx, path, req, &resp); err != nil {
		return nil, fmt.Errorf("creating block storage backup: %w", err)
	}
	return &resp.Data, nil
}
