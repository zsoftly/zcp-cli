// Package template provides ZCP template API operations.
package template

import (
	"context"
	"fmt"
	"net/url"

	"github.com/zsoftly/zcp-cli/internal/httpclient"
)

// Template represents a ZCP VM template from the public catalog.
type Template struct {
	ID                       string                  `json:"id"`
	TemplateID               string                  `json:"template_id"`
	AccountID                *string                 `json:"account_id"`
	CloudProviderID          string                  `json:"cloud_provider_id"`
	CloudProviderSetupID     string                  `json:"cloud_provider_setup_id"`
	RegionID                 string                  `json:"region_id"`
	Type                     string                  `json:"type"`
	Name                     string                  `json:"name"`
	Slug                     string                  `json:"slug"`
	OperatingSystemID        string                  `json:"operating_system_id"`
	OperatingSystemVersionID string                  `json:"operating_system_version_id"`
	Status                   bool                    `json:"status"`
	PasswordEnabled          bool                    `json:"password_enabled"`
	SortOrder                int                     `json:"sort_order"`
	CreatedAt                string                  `json:"created_at"`
	UpdatedAt                string                  `json:"updated_at"`
	StartupScript            *string                 `json:"startup_script"`
	FileType                 string                  `json:"file_type"`
	ImageType                string                  `json:"image_type"`
	PasswordMethod           string                  `json:"password_method"`
	EnableResetPassword      bool                    `json:"enable_reset_password"`
	IconURL                  string                  `json:"icon_url"`
	OperatingSystemVersion   *OperatingSystemVersion `json:"operating_system_version,omitempty"`
	OperatingSystem          *OperatingSystem        `json:"operating_system,omitempty"`
}

// OperatingSystem represents the OS of a template.
type OperatingSystem struct {
	ID                string `json:"id"`
	Name              string `json:"name"`
	Slug              string `json:"slug"`
	VMDefaultUsername string `json:"vm_default_username"`
	Family            string `json:"family"`
	SortOrder         int    `json:"sort_order"`
	Icon              string `json:"icon"`
}

// OperatingSystemVersion represents the OS version of a template.
type OperatingSystemVersion struct {
	ID                string `json:"id"`
	OperatingSystemID string `json:"operating_system_id"`
	Version           string `json:"version"`
	PricingType       string `json:"pricing_type"`
}

// AccountTemplate represents a user-created template (account template).
type AccountTemplate struct {
	ID                   string         `json:"id"`
	Name                 string         `json:"name"`
	Slug                 string         `json:"slug"`
	Description          string         `json:"description"`
	TemplateID           string         `json:"template_id"`
	AccountID            string         `json:"account_id"`
	CloudProviderID      string         `json:"cloud_provider_id"`
	CloudProviderSetupID string         `json:"cloud_provider_setup_id"`
	RegionID             string         `json:"region_id"`
	ProjectID            string         `json:"project_id"`
	State                string         `json:"state"`
	Status               string         `json:"status"`
	ImageType            string         `json:"image_type"`
	FileType             string         `json:"file_type"`
	Format               string         `json:"format"`
	PasswordEnabled      bool           `json:"password_enabled"`
	URL                  string         `json:"url"`
	CreatedAt            string         `json:"created_at"`
	UpdatedAt            string         `json:"updated_at"`
	Region               *Region        `json:"region,omitempty"`
	Project              *Project       `json:"project,omitempty"`
	CloudProvider        *CloudProvider `json:"cloud_provider,omitempty"`
}

// Region represents the region of a template resource.
type Region struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Slug string `json:"slug"`
}

// Project represents the project of a template resource.
type Project struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Slug string `json:"slug"`
}

// CloudProvider represents the cloud provider of a template resource.
type CloudProvider struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	DisplayName string `json:"display_name"`
	Slug        string `json:"slug"`
}

// CreateAccountTemplateRequest holds parameters for creating an account template.
type CreateAccountTemplateRequest struct {
	Name                   string `json:"name"`
	Description            string `json:"description,omitempty"`
	URL                    string `json:"url,omitempty"`
	CloudProvider          string `json:"cloud_provider"`
	Region                 string `json:"region"`
	Project                string `json:"project"`
	OSTypeID               string `json:"os_type_id"`
	ImageType              string `json:"image_type"`
	OperatingSystem        string `json:"operating_system"`
	OperatingSystemVersion string `json:"operating_system_version"`
	PasswordEnabled        bool   `json:"password_enabled"`
	BillingCycle           string `json:"billing_cycle"`
	Plan                   string `json:"plan,omitempty"`
	Format                 string `json:"format,omitempty"`
	RootDiskController     string `json:"root_disk_controller,omitempty"`
	TemplateType           string `json:"template_type,omitempty"`
	IsFeatured             bool   `json:"is_featured,omitempty"`
	RequiresHVM            bool   `json:"requires_hvm,omitempty"`
	IsDynamicallyScalable  bool   `json:"is_dynamically_scalable,omitempty"`
	IsUploadFromLocal      bool   `json:"is_upload_from_local"`
	Coupon                 string `json:"coupon,omitempty"`
	VirtualMachine         string `json:"virtual_machine,omitempty"`
}

// listTemplateResponse is the STKCNSL paginated response envelope for templates.
type listTemplateResponse struct {
	Status  string     `json:"status"`
	Message string     `json:"message"`
	Data    []Template `json:"data"`
	Total   int        `json:"total"`
}

// listAccountTemplateResponse is the STKCNSL paginated response envelope for account templates.
type listAccountTemplateResponse struct {
	Status  string            `json:"status"`
	Message string            `json:"message"`
	Data    []AccountTemplate `json:"data"`
	Total   int               `json:"total"`
}

// singleAccountTemplateResponse is the STKCNSL single-object response envelope.
type singleAccountTemplateResponse struct {
	Status  string          `json:"status"`
	Message string          `json:"message"`
	Data    AccountTemplate `json:"data"`
}

// Service provides template API operations.
type Service struct {
	client *httpclient.Client
}

// NewService creates a new template Service.
func NewService(client *httpclient.Client) *Service {
	return &Service{client: client}
}

// List returns all public templates. regionSlug is an optional filter.
func (s *Service) List(ctx context.Context, regionSlug string) ([]Template, error) {
	q := url.Values{}
	q.Set("include", "operating_system,operating_system_version,region,cloud_provider")
	if regionSlug != "" {
		q.Set("filter[region]", regionSlug)
	}
	var resp listTemplateResponse
	if err := s.client.Get(ctx, "/templates", q, &resp); err != nil {
		return nil, fmt.Errorf("listing templates: %w", err)
	}
	return resp.Data, nil
}

// ListAccount returns templates owned by the authenticated account.
func (s *Service) ListAccount(ctx context.Context) ([]AccountTemplate, error) {
	q := url.Values{}
	q.Set("include", "region,cloud_provider,template,project")
	var resp listAccountTemplateResponse
	if err := s.client.Get(ctx, "/account/templates", q, &resp); err != nil {
		return nil, fmt.Errorf("listing account templates: %w", err)
	}
	return resp.Data, nil
}

// CreateAccount creates a new account template.
func (s *Service) CreateAccount(ctx context.Context, req CreateAccountTemplateRequest) (*AccountTemplate, error) {
	var resp singleAccountTemplateResponse
	if err := s.client.Post(ctx, "/account/templates", req, &resp); err != nil {
		return nil, fmt.Errorf("creating account template: %w", err)
	}
	return &resp.Data, nil
}

// DeleteAccount removes an account template by slug.
func (s *Service) DeleteAccount(ctx context.Context, slug string) error {
	if err := s.client.Delete(ctx, "/account/templates/"+slug, nil); err != nil {
		return fmt.Errorf("deleting account template %s: %w", slug, err)
	}
	return nil
}
