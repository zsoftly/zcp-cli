// Package marketplace provides ZCP marketplace app API operations.
package marketplace

import (
	"context"
	"fmt"
	"net/url"

	"github.com/zsoftly/zcp-cli/internal/httpclient"
)

// AppFile represents a file attached to a marketplace app.
type AppFile struct {
	ID               string `json:"id"`
	MarketplaceAppID string `json:"marketplace_app_id"`
	Name             string `json:"name"`
	CreatedAt        string `json:"created_at"`
	UpdatedAt        string `json:"updated_at"`
}

// App represents a marketplace application.
type App struct {
	ID                   string    `json:"id"`
	Name                 string    `json:"name"`
	Slug                 string    `json:"slug"`
	Category             string    `json:"category"`
	URL                  string    `json:"url"`
	ShortDescription     string    `json:"short_description"`
	AppMasterCategory    string    `json:"app_master_category"`
	IsFeatured           bool      `json:"is_featured"`
	DisplayOrder         *int      `json:"display_order"`
	StartupScript        *string   `json:"startup_script"`
	SortOrder            int       `json:"sort_order"`
	CreatedAt            string    `json:"created_at"`
	UpdatedAt            string    `json:"updated_at"`
	Icon                 string    `json:"icon"`
	DarkThemeLogo        string    `json:"dark_theme_logo"`
	TemplateID           string    `json:"template_id"`
	CloudProviderID      string    `json:"cloud_provider_id"`
	CloudProviderSetupID string    `json:"cloud_provider_setup_id"`
	RegionID             string    `json:"region_id"`
	Files                []AppFile `json:"files"`
}

// listAppsResponse wraps the marketplace apps response.
type listAppsResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
	Data    []App  `json:"data"`
}

// Service provides marketplace API operations.
type Service struct {
	client *httpclient.Client
}

// NewService creates a new marketplace Service.
func NewService(client *httpclient.Client) *Service {
	return &Service{client: client}
}

// ListApps returns all marketplace applications with optional filters.
func (s *Service) ListApps(ctx context.Context, region, include string) ([]App, error) {
	q := url.Values{}
	if region != "" {
		q.Set("filter[region]", region)
	}
	if include != "" {
		q.Set("include", include)
	}

	var resp listAppsResponse
	if err := s.client.Get(ctx, "/marketplace-apps", q, &resp); err != nil {
		return nil, fmt.Errorf("listing marketplace apps: %w", err)
	}
	return resp.Data, nil
}
