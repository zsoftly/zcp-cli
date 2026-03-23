// Package quota provides ZCP resource quota API operations.
package quota

import (
	"context"
	"fmt"
	"net/url"

	"github.com/zsoftly/zcp-cli/internal/httpclient"
)

// ResourceQuota represents a domain resource limit.
type ResourceQuota struct {
	UnitType       string `json:"unitType"`
	QuotaType      string `json:"quotaType"`
	AvailableLimit string `json:"availableLimit"`
	DomainUUID     string `json:"domainUuid"`
	UsedLimit      string `json:"usedLimit"`
	MaximumLimit   string `json:"maximumLimit"`
}

type listResourceQuotaResponse struct {
	Count                     int             `json:"count"`
	ListResourceQuotaResponse []ResourceQuota `json:"listResourceQuotaResponse"`
}

// Service provides quota API operations.
type Service struct {
	client *httpclient.Client
}

// NewService creates a new quota Service.
func NewService(client *httpclient.Client) *Service { return &Service{client: client} }

// List returns resource quotas. If domainUUID is non-empty, filters to that domain.
func (s *Service) List(ctx context.Context, domainUUID string) ([]ResourceQuota, error) {
	q := url.Values{}
	if domainUUID != "" {
		q.Set("domainUuid", domainUUID)
	}
	var resp listResourceQuotaResponse
	if err := s.client.Get(ctx, "/restapi/resource-quota/get-resource-limit", q, &resp); err != nil {
		return nil, fmt.Errorf("listing resource quotas: %w", err)
	}
	return resp.ListResourceQuotaResponse, nil
}
