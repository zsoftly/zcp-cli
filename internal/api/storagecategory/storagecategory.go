// Package storagecategory provides ZCP storage category API operations (STKCNSL).
package storagecategory

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/zsoftly/zcp-cli/internal/httpclient"
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

// List returns all storage categories.
func (s *Service) List(ctx context.Context) ([]StorageCategory, error) {
	var env envelope
	if err := s.client.Get(ctx, "/storage-categories", nil, &env); err != nil {
		return nil, fmt.Errorf("listing storage categories: %w", err)
	}

	var categories []StorageCategory
	if err := json.Unmarshal(env.Data, &categories); err != nil {
		return nil, fmt.Errorf("decoding storage categories: %w", err)
	}

	return categories, nil
}
