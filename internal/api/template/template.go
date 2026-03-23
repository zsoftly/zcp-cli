// Package template provides ZCP template API operations.
package template

import (
	"context"
	"fmt"
	"net/url"

	"github.com/zsoftly/zcp-cli/internal/httpclient"
)

// Template represents a ZCP VM template.
type Template struct {
	UUID           string `json:"uuid"`
	Name           string `json:"name"`
	Description    string `json:"description"`
	Format         string `json:"format"`
	OsCategoryName string `json:"osCategoryName"`
	ZoneName       string `json:"zoneName"`
	ZoneUUID       string `json:"zoneUuid"`
	TemplateCost   string `json:"templateCost"`
	IsActive       string `json:"isActive"`
}

type listTemplateResponse struct {
	Count                int        `json:"count"`
	ListTemplateResponse []Template `json:"listTemplateResponse"`
}

// Service provides template API operations.
type Service struct {
	client *httpclient.Client
}

// NewService creates a new template Service.
func NewService(client *httpclient.Client) *Service {
	return &Service{client: client}
}

// List returns all templates. zoneUUID and templateUUID are optional filters.
func (s *Service) List(ctx context.Context, zoneUUID, templateUUID string) ([]Template, error) {
	q := url.Values{}
	if zoneUUID != "" {
		q.Set("zoneUuid", zoneUUID)
	}
	if templateUUID != "" {
		q.Set("uuid", templateUUID)
	}

	var resp listTemplateResponse
	if err := s.client.Get(ctx, "/restapi/template/templateList", q, &resp); err != nil {
		return nil, fmt.Errorf("listing templates: %w", err)
	}
	return resp.ListTemplateResponse, nil
}
