// Package storagecategory provides ZCP storage category API operations (STKCNSL).
package storagecategory

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/zsoftly/zcp-cli/pkg/httpclient"
)

// StorageCategory represents a STKCNSL storage category
// (e.g. SSD Storage, NVMe, HDD Storage).
type StorageCategory struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Slug      string `json:"slug"`
	Status    bool   `json:"status"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

// envelope is the STKCNSL response wrapper.
type envelope struct {
	Status  string          `json:"status"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

// Service provides storage category API operations.
type Service struct {
	client *httpclient.Client
}

// NewService creates a new storage category Service.
func NewService(client *httpclient.Client) *Service {
	return &Service{client: client}
}

// List returns storage categories scoped to regionSlug. Categories are
// region-specific (e.g. YUL exposes pro-nvme, YOW exposes nvme), so an empty
// regionSlug returns all regions — used only for internal id→slug lookups, not
// user-facing listings. The server honors filter[region]=<slug>.
func (s *Service) List(ctx context.Context, regionSlug string) ([]StorageCategory, error) {
	q := url.Values{}
	if regionSlug != "" {
		q.Set("filter[region]", regionSlug)
	}
	var env envelope
	if err := s.client.Get(ctx, "/storage-categories", q, &env); err != nil {
		return nil, fmt.Errorf("listing storage categories: %w", err)
	}

	var categories []StorageCategory
	if err := json.Unmarshal(env.Data, &categories); err != nil {
		return nil, fmt.Errorf("decoding storage categories: %w", err)
	}

	return categories, nil
}
