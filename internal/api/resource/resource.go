// Package resource provides ZCP available resource API operations.
package resource

import (
	"context"
	"fmt"
	"net/url"

	"github.com/zsoftly/zcp-cli/internal/httpclient"
)

// AvailableResource represents a domain-scoped resource limit and usage.
type AvailableResource struct {
	ResourceType   string `json:"resourceType"`
	AvailableLimit string `json:"availableLimit"`
	UsedLimit      string `json:"usedLimit"`
	MaximumLimit   string `json:"maximumLimit"`
}

type listAvailableResourceResponse struct {
	Count                     int                 `json:"count"`
	KongUserAvailableResource []AvailableResource `json:"kongUserAvailableResource"`
}

// Service provides resource API operations.
type Service struct {
	client *httpclient.Client
}

// NewService creates a new resource Service.
func NewService(client *httpclient.Client) *Service {
	return &Service{client: client}
}

// ListAvailable returns all available resource limits for the authenticated domain.
func (s *Service) ListAvailable(ctx context.Context) ([]AvailableResource, error) {
	var resp listAvailableResourceResponse
	if err := s.client.Get(ctx, "/restapi/availableResource/getAvailableResourceByDomain", url.Values{}, &resp); err != nil {
		return nil, fmt.Errorf("listing available resources: %w", err)
	}
	return resp.KongUserAvailableResource, nil
}
