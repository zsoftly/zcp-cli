// Package iso provides ZCP ISO image API operations.
package iso

import (
	"context"
	"fmt"
	"net/url"

	"github.com/zsoftly/zcp-cli/internal/httpclient"
)

// ISO represents a ZCP ISO image.
type ISO struct {
	ID                   string           `json:"id"`
	Name                 string           `json:"name"`
	Slug                 string           `json:"slug"`
	Description          string           `json:"description"`
	ISOURL               string           `json:"url"`
	State                string           `json:"state"`
	Status               string           `json:"status"`
	PasswordEnabled      bool             `json:"password_enabled"`
	IsExtractable        bool             `json:"is_extractable"`
	IsBootable           bool             `json:"is_bootable"`
	ImageType            string           `json:"image_type"`
	FileType             string           `json:"file_type"`
	RegionID             string           `json:"region_id"`
	CloudProviderID      string           `json:"cloud_provider_id"`
	CloudProviderSetupID string           `json:"cloud_provider_setup_id"`
	ProjectID            string           `json:"project_id"`
	TemplateID           string           `json:"template_id"`
	AccountID            string           `json:"account_id"`
	OperatingSystemID    string           `json:"operating_system_id"`
	CreatedAt            string           `json:"created_at"`
	UpdatedAt            string           `json:"updated_at"`
	Region               *Region          `json:"region,omitempty"`
	Project              *Project         `json:"project,omitempty"`
	CloudProvider        *CloudProvider   `json:"cloud_provider,omitempty"`
	OperatingSystem      *OperatingSystem `json:"operating_system,omitempty"`
}

// Region represents the region of an ISO.
type Region struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Slug string `json:"slug"`
}

// Project represents the project of an ISO.
type Project struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Slug string `json:"slug"`
}

// CloudProvider represents the cloud provider of an ISO.
type CloudProvider struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	DisplayName string `json:"display_name"`
	Slug        string `json:"slug"`
}

// OperatingSystem represents the OS associated with an ISO.
type OperatingSystem struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Slug   string `json:"slug"`
	Family string `json:"family"`
}

// CreateRequest holds parameters for creating an ISO.
type CreateRequest struct {
	Name                   string `json:"name"`
	Description            string `json:"description,omitempty"`
	URL                    string `json:"url"`
	CloudProvider          string `json:"cloud_provider"`
	Project                string `json:"project"`
	Region                 string `json:"region"`
	OSTypeID               string `json:"os_type_id"`
	ImageType              string `json:"image_type"`
	OperatingSystem        string `json:"operating_system"`
	OperatingSystemVersion string `json:"operating_system_version"`
	BillingCycle           string `json:"billing_cycle"`
	PasswordEnabled        bool   `json:"password_enabled"`
	IsExtractable          bool   `json:"is_extractable"`
	IsBootable             bool   `json:"is_bootable"`
	IsUploadFromLocal      bool   `json:"is_upload_from_local"`
	Service                string `json:"service"`
	Coupon                 string `json:"coupon,omitempty"`
}

// UpdateRequest holds parameters for updating ISO permissions.
type UpdateRequest struct {
	PasswordEnabled bool `json:"password_enabled"`
	IsExtractable   bool `json:"is_extractable"`
	IsBootable      bool `json:"is_bootable"`
}

// listResponse is the STKCNSL paginated response envelope.
type listResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
	Data    []ISO  `json:"data"`
	Total   int    `json:"total"`
}

// singleResponse is the STKCNSL single-object response envelope.
type singleResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
	Data    ISO    `json:"data"`
}

// Service provides ISO API operations.
type Service struct {
	client *httpclient.Client
}

// NewService creates a new ISO Service.
func NewService(client *httpclient.Client) *Service {
	return &Service{client: client}
}

// List returns all ISO images.
func (s *Service) List(ctx context.Context) ([]ISO, error) {
	q := url.Values{}
	q.Set("include", "project,region,cloud_provider")
	var resp listResponse
	if err := s.client.Get(ctx, "/isos", q, &resp); err != nil {
		return nil, fmt.Errorf("listing ISOs: %w", err)
	}
	return resp.Data, nil
}

// Create registers a new ISO image.
func (s *Service) Create(ctx context.Context, req CreateRequest) (*ISO, error) {
	var resp singleResponse
	if err := s.client.Post(ctx, "/isos", req, &resp); err != nil {
		return nil, fmt.Errorf("creating ISO: %w", err)
	}
	return &resp.Data, nil
}

// Update modifies ISO permissions (password, extractable, bootable).
func (s *Service) Update(ctx context.Context, slug string, req UpdateRequest) error {
	if err := s.client.Put(ctx, "/isos/"+slug+"/iso-permission", nil, req, nil); err != nil {
		return fmt.Errorf("updating ISO %s: %w", slug, err)
	}
	return nil
}

// Delete removes an ISO image by slug.
func (s *Service) Delete(ctx context.Context, slug string) error {
	if err := s.client.Delete(ctx, "/isos/"+slug, nil); err != nil {
		return fmt.Errorf("deleting ISO %s: %w", slug, err)
	}
	return nil
}
