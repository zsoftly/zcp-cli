// Package resource provides ZCP available resource API operations.
package resource

import (
	"context"
	"fmt"
	"net/url"
	"strconv"

	"github.com/zsoftly/zcp-cli/internal/httpclient"
)

// AvailableResource represents a domain-scoped resource limit and usage.
type AvailableResource struct {
	ResourceType   string `json:"resourceType"`
	AvailableLimit string `json:"availableLimit"`
	UsedLimit      string `json:"usedLimit"`
	MaximumLimit   string `json:"maximumLimit"`
}

// ResourceQuota represents a resource quota limit entry from the quota API.
type ResourceQuota struct {
	QuotaType      string `json:"quotaType"`
	UnitType       string `json:"unitType"`
	AvailableLimit int64  `json:"availableLimit"`
	UsedLimit      int64  `json:"usedLimit"`
	MaximumLimit   int64  `json:"maximumLimit"`
	DomainUUID     string `json:"domainUuid"`
}

type listAvailableResourceResponse struct {
	Count                     int                 `json:"count"`
	KongUserAvailableResource []AvailableResource `json:"kongUserAvailableResource"`
}

type listResourceQuotaResponse struct {
	Count                     int             `json:"count"`
	ListResourceQuotaResponse []ResourceQuota `json:"listResourceQuotaResponse"`
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
// It first tries the resource-quota endpoint, then falls back to the legacy available-resource endpoint.
func (s *Service) ListAvailable(ctx context.Context) ([]AvailableResource, error) {
	// Try the resource-quota endpoint first (more reliable).
	quotas, err := s.ListQuota(ctx, "")
	if err == nil && len(quotas) > 0 {
		// Convert ResourceQuota to AvailableResource for backward compatibility.
		resources := make([]AvailableResource, 0, len(quotas))
		for _, q := range quotas {
			label := q.QuotaType
			if q.UnitType != "" {
				label += " (" + q.UnitType + ")"
			}
			resources = append(resources, AvailableResource{
				ResourceType:   label,
				AvailableLimit: strconv.FormatInt(q.AvailableLimit, 10),
				UsedLimit:      strconv.FormatInt(q.UsedLimit, 10),
				MaximumLimit:   strconv.FormatInt(q.MaximumLimit, 10),
			})
		}
		return resources, nil
	}

	// Fall back to the legacy endpoint.
	var resp listAvailableResourceResponse
	if err := s.client.Get(ctx, "/restapi/availableResource/getAvailableResourceByDomain", url.Values{}, &resp); err != nil {
		return nil, fmt.Errorf("listing available resources: %w", err)
	}
	return resp.KongUserAvailableResource, nil
}

// ListQuota returns resource quota limits. domainUUID is an optional filter.
func (s *Service) ListQuota(ctx context.Context, domainUUID string) ([]ResourceQuota, error) {
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
