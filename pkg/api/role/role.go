// Package role provides ZCP role API operations.
//
// Roles are an account-level grouping of permissions that can be assigned to
// sub-users. They are addressed by SLUG (not UUID) on every single-resource
// route. Three roles are predefined — owner, service-administrator,
// service-viewer — and the API rejects update/delete on them.
package role

import (
	"context"
	"fmt"

	"github.com/zsoftly/zcp-cli/pkg/api/permission"
	"github.com/zsoftly/zcp-cli/pkg/httpclient"
)

// User is the trimmed user reference embedded in a role's users list.
type User struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

// Role represents a ZCP role. The list endpoint omits Permissions; only the
// single-role show (Get) populates it.
type Role struct {
	ID          string                  `json:"id"`
	Name        string                  `json:"name"`
	Slug        string                  `json:"slug"`
	Description string                  `json:"description"`
	Status      bool                    `json:"status"`
	Permissions []permission.Permission `json:"permissions,omitempty"`
	Users       []User                  `json:"users,omitempty"`
}

// CreateRequest holds parameters for creating a role. Permissions is a list of
// permission slugs (see the permission catalog).
type CreateRequest struct {
	Name        string   `json:"name"`
	Description string   `json:"description,omitempty"`
	Permissions []string `json:"permissions"`
}

// UpdateRequest holds parameters for updating a role. It represents the full
// desired state: Permissions REPLACE the role's existing set (it is not
// additive) and Description has NO omitempty on purpose — the API treats an
// absent description as "preserve" and an explicit "" as "clear" (verified live
// 2026-06-21), so omitting it would make clearing a description impossible.
// Callers must therefore send every field at its intended value.
type UpdateRequest struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Permissions []string `json:"permissions"`
}

// listResponse is the STKCNSL paginated response envelope (single page).
type listResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
	Data    []Role `json:"data"`
}

// singleResponse is the STKCNSL single-object response envelope.
type singleResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
	Data    Role   `json:"data"`
}

// Service provides role API operations.
type Service struct {
	client *httpclient.Client
}

// NewService creates a new role Service.
func NewService(client *httpclient.Client) *Service {
	return &Service{client: client}
}

// List returns all roles. The returned roles do not include their permissions;
// call Get for a role's full permission set.
func (s *Service) List(ctx context.Context) ([]Role, error) {
	var resp listResponse
	if err := s.client.Get(ctx, "/roles", nil, &resp); err != nil {
		return nil, fmt.Errorf("listing roles: %w", err)
	}
	return resp.Data, nil
}

// Get returns a single role by slug, including its permissions and assigned
// users. The API keys this route by slug; passing a UUID returns a 500.
func (s *Service) Get(ctx context.Context, slug string) (*Role, error) {
	var resp singleResponse
	if err := s.client.Get(ctx, "/roles/"+slug, nil, &resp); err != nil {
		return nil, fmt.Errorf("getting role %s: %w", slug, err)
	}
	return &resp.Data, nil
}

// Create provisions a new role with the given permission slugs.
func (s *Service) Create(ctx context.Context, req CreateRequest) (*Role, error) {
	var resp singleResponse
	if err := s.client.Post(ctx, "/roles", req, &resp); err != nil {
		return nil, fmt.Errorf("creating role: %w", err)
	}
	return &resp.Data, nil
}

// Update replaces a role's name, description, and permission set. The API
// rejects updates to predefined roles (owner/service-administrator/
// service-viewer) with a 403.
func (s *Service) Update(ctx context.Context, slug string, req UpdateRequest) (*Role, error) {
	var resp singleResponse
	if err := s.client.Put(ctx, "/roles/"+slug, nil, req, &resp); err != nil {
		return nil, fmt.Errorf("updating role %s: %w", slug, err)
	}
	return &resp.Data, nil
}

// Delete removes a role by slug.
func (s *Service) Delete(ctx context.Context, slug string) error {
	if err := s.client.Delete(ctx, "/roles/"+slug, nil); err != nil {
		return fmt.Errorf("deleting role %s: %w", slug, err)
	}
	return nil
}
