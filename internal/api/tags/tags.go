// Package tags provides ZCP resource tag API operations.
package tags

import (
	"context"
	"fmt"
	"net/url"

	"github.com/zsoftly/zcp-cli/internal/httpclient"
)

// Tag represents a ZCP resource tag.
type Tag struct {
	UUID         string `json:"uuid"`
	IsActive     bool   `json:"isActive"`
	Key          string `json:"key"`
	Value        string `json:"value"`
	ResourceUUID string `json:"resourceUuid"`
	ResourceType string `json:"resourceType"`
}

// CreateRequest holds parameters for creating a resource tag.
type CreateRequest struct {
	Key          string `json:"key"`
	Value        string `json:"value"`
	ResourceUUID string `json:"resourceUuid"`
}

type listTagsResponse struct {
	Count                  int   `json:"count"`
	KongCreateTagsResponse []Tag `json:"kongCreateTagsResponse"`
}

// Service provides resource tag API operations.
type Service struct {
	client *httpclient.Client
}

// NewService creates a new tags Service.
func NewService(client *httpclient.Client) *Service {
	return &Service{client: client}
}

// List returns resource tags. All parameters are optional filters.
func (s *Service) List(ctx context.Context, resourceUUID, resourceType string) ([]Tag, error) {
	q := url.Values{}
	if resourceUUID != "" {
		q.Set("resourceUuid", resourceUUID)
	}
	if resourceType != "" {
		q.Set("resourceType", resourceType)
	}
	var resp listTagsResponse
	if err := s.client.Get(ctx, "/restapi/resourcetags/resourceTagsList", q, &resp); err != nil {
		return nil, fmt.Errorf("listing tags: %w", err)
	}
	return resp.KongCreateTagsResponse, nil
}

// Create adds a new resource tag.
// resourceType and zoneUUID are sent as query params (unusual API design);
// key, value, and resourceUuid are sent as body fields.
func (s *Service) Create(ctx context.Context, resourceType, zoneUUID string, req CreateRequest) (*Tag, error) {
	path := fmt.Sprintf("/restapi/resourcetags/createTags?resourceType=%s&zoneUuid=%s",
		url.QueryEscape(resourceType), url.QueryEscape(zoneUUID))
	var resp listTagsResponse
	if err := s.client.Post(ctx, path, req, &resp); err != nil {
		return nil, fmt.Errorf("creating tag: %w", err)
	}
	if len(resp.KongCreateTagsResponse) == 0 {
		return nil, fmt.Errorf("create tag returned empty response")
	}
	return &resp.KongCreateTagsResponse[0], nil
}

// Delete removes a resource tag by UUID.
func (s *Service) Delete(ctx context.Context, uuid string) error {
	if err := s.client.Delete(ctx, "/restapi/resourcetags/deleteResourceTag/"+uuid, nil); err != nil {
		return fmt.Errorf("deleting tag %s: %w", uuid, err)
	}
	return nil
}
