// Package permission provides ZCP permission catalog API operations.
//
// Permissions are an account-level, read-only catalog: each entry is a fine
// grained capability (e.g. "virtual-machine-manage") that can be assigned to a
// role. The catalog is not region- or project-scoped.
package permission

import (
	"context"
	"fmt"

	"github.com/zsoftly/zcp-cli/pkg/httpclient"
)

// Permission represents a single assignable capability in the ZCP catalog.
type Permission struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Slug        string `json:"slug"`
	Description string `json:"description"`
	Guard       string `json:"guard"`
	Status      bool   `json:"status"`
	Category    string `json:"category"`
}

// listResponse is the STKCNSL paginated response envelope. The endpoint returns
// the full catalog on a single page (per_page = -1), so no pagination walk is
// needed.
type listResponse struct {
	Status  string       `json:"status"`
	Message string       `json:"message"`
	Data    []Permission `json:"data"`
}

// Service provides permission catalog API operations.
type Service struct {
	client *httpclient.Client
}

// NewService creates a new permission Service.
func NewService(client *httpclient.Client) *Service {
	return &Service{client: client}
}

// List returns the full permission catalog.
func (s *Service) List(ctx context.Context) ([]Permission, error) {
	var resp listResponse
	if err := s.client.Get(ctx, "/permissions", nil, &resp); err != nil {
		return nil, fmt.Errorf("listing permissions: %w", err)
	}
	return resp.Data, nil
}
