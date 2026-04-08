// Package region provides ZCP region API operations (STKCNSL).
package region

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/zsoftly/zcp-cli/internal/httpclient"
)

// CloudProvider is the embedded cloud provider within a region.
type CloudProvider struct {
	ID                 string   `json:"id"`
	ServerID           string   `json:"server_id"`
	Name               string   `json:"name"`
	DisplayName        string   `json:"display_name"`
	Slug               string   `json:"slug"`
	Description        string   `json:"description"`
	IsMultiRegionSetup bool     `json:"is_multi_region_setup"`
	Status             bool     `json:"status"`
	Services           []string `json:"services"`
}

// CloudProviderSetup is the setup configuration nested in a region.
type CloudProviderSetup struct {
	ID              string `json:"id"`
	Name            string `json:"name"`
	Slug            string `json:"slug"`
	CloudProviderID string `json:"cloud_provider_id"`
	CredentialOf    string `json:"credential_of"`
	Version         string `json:"version"`
	Monitoring      string `json:"monitoring"`
	Timezone        string `json:"timezone"`
	Status          bool   `json:"status"`
}

// Region represents a STKCNSL region.
type Region struct {
	ID                    string              `json:"id"`
	Name                  string              `json:"name"`
	Slug                  string              `json:"slug"`
	CloudProvider         *CloudProvider      `json:"cloud_provider"`
	CloudProviderSetup    *CloudProviderSetup `json:"cloud_provider_setup"`
	Country               string              `json:"country"`
	CountryCode           string              `json:"country_code"`
	ContinentCode         string              `json:"continent_code"`
	ContinentName         string              `json:"continent_name"`
	Description           string              `json:"description"`
	Status                bool                `json:"status"`
	CPUSpeed              int                 `json:"cpu_speed"`
	IsComingSoon          bool                `json:"is_coming_soon"`
	ConsoleProxyIPAddress string              `json:"console_proxy_ip_address"`
	ConsoleProxyDomain    string              `json:"console_proxy_domain"`
	CreatedAt             string              `json:"created_at"`
	UpdatedAt             string              `json:"updated_at"`
}

// envelope is the STKCNSL response wrapper.
type envelope struct {
	Status  string          `json:"status"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

// Service provides region API operations.
type Service struct {
	client *httpclient.Client
}

// NewService creates a new region Service.
func NewService(client *httpclient.Client) *Service {
	return &Service{client: client}
}

// List returns all regions.
func (s *Service) List(ctx context.Context) ([]Region, error) {
	var env envelope
	if err := s.client.Get(ctx, "/regions", nil, &env); err != nil {
		return nil, fmt.Errorf("listing regions: %w", err)
	}

	var regions []Region
	if err := json.Unmarshal(env.Data, &regions); err != nil {
		return nil, fmt.Errorf("decoding regions: %w", err)
	}

	return regions, nil
}
