// Package zone provides ZCP zone API operations.
package zone

import (
	"context"
	"fmt"
	"net/url"

	"github.com/zsoftly/zcp-cli/internal/httpclient"
)

// Zone represents a ZCP availability zone.
type Zone struct {
	UUID        string `json:"uuid"`
	Name        string `json:"name"`
	CountryName string `json:"countryName"`
	IsActive    bool   `json:"isActive"`
	ImageFlag   string `json:"imageFlag"`
}

type listZoneResponse struct {
	Count            int    `json:"count"`
	ListZoneResponse []Zone `json:"listZoneResponse"`
}

// Service provides zone API operations.
type Service struct {
	client *httpclient.Client
}

// NewService creates a new zone Service.
func NewService(client *httpclient.Client) *Service {
	return &Service{client: client}
}

// List returns all zones. If zoneUUID is non-empty, filters to that zone.
func (s *Service) List(ctx context.Context, zoneUUID string) ([]Zone, error) {
	q := url.Values{}
	if zoneUUID != "" {
		q.Set("uuid", zoneUUID)
	}

	var resp listZoneResponse
	if err := s.client.Get(ctx, "/restapi/zone/zonelist", q, &resp); err != nil {
		return nil, fmt.Errorf("listing zones: %w", err)
	}

	return resp.ListZoneResponse, nil
}
