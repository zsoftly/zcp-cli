// Package cloudprovider provides ZCP cloud provider API operations (STKCNSL).
package cloudprovider

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/zsoftly/zcp-cli/internal/httpclient"
)

// CloudProvider represents a STKCNSL cloud provider.
type CloudProvider struct {
	ID                 string   `json:"id"`
	ServerID           string   `json:"server_id"`
	Name               string   `json:"name"`
	DisplayName        string   `json:"display_name"`
	Slug               string   `json:"slug"`
	Description        string   `json:"description"`
	IsMultiRegionSetup bool     `json:"is_multi_region_setup"`
	Status             bool     `json:"status"`
	CreatedAt          string   `json:"created_at"`
	UpdatedAt          string   `json:"updated_at"`
	SortOrder          *int     `json:"sort_order"`
	Icon               string   `json:"icon"`
	Services           []string `json:"services"`
}

// envelope is the STKCNSL response wrapper.
type envelope struct {
	Status  string          `json:"status"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

// Service provides cloud provider API operations.
type Service struct {
	client *httpclient.Client
}

// NewService creates a new cloud provider Service.
func NewService(client *httpclient.Client) *Service {
	return &Service{client: client}
}

// List returns all cloud providers.
func (s *Service) List(ctx context.Context) ([]CloudProvider, error) {
	var env envelope
	if err := s.client.Get(ctx, "/cloud-providers", nil, &env); err != nil {
		return nil, fmt.Errorf("listing cloud providers: %w", err)
	}

	var providers []CloudProvider
	if err := json.Unmarshal(env.Data, &providers); err != nil {
		return nil, fmt.Errorf("decoding cloud providers: %w", err)
	}

	return providers, nil
}
