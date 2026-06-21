// Package subuser provides ZCP sub-user API operations.
//
// Sub-users are the additional users under an account (the account owner is not
// included in the list). They are account-level — not region/project-scoped —
// and addressed by UUID on the update/delete routes. The API exposes no
// single-user GET (the list is the only read), no working server-side filters,
// and no dedicated block/unblock route — blocking is done by PUT with
// is_blocked set.
package subuser

import (
	"context"
	"fmt"

	"github.com/zsoftly/zcp-cli/pkg/httpclient"
)

// Role is the trimmed role reference embedded in a sub-user.
type Role struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Slug string `json:"slug"`
}

// Project is the trimmed project reference embedded in a sub-user.
type Project struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Slug string `json:"slug"`
}

// SubUser represents a ZCP sub-user.
type SubUser struct {
	ID               string    `json:"id"`
	Name             string    `json:"name"`
	Email            string    `json:"email"`
	UserType         string    `json:"user_type"`
	RegisteredBy     string    `json:"registered_by"`
	RegistrationType string    `json:"registration_type"`
	IsBlocked        bool      `json:"is_blocked"`
	Reason           string    `json:"reason"`
	UserStatus       string    `json:"user_status"`
	ParentID         string    `json:"parent_id"`
	Role             *Role     `json:"role"`
	Projects         []Project `json:"projects"`
	CreatedAt        string    `json:"created_at"`
	UpdatedAt        string    `json:"updated_at"`
	LastLogin        *string   `json:"last_login"`
}

// ProjectSlugs returns the slugs of the sub-user's assigned projects, suitable
// for echoing back into an update request (which requires the projects field).
func (u *SubUser) ProjectSlugs() []string {
	slugs := make([]string, 0, len(u.Projects))
	for _, p := range u.Projects {
		slugs = append(slugs, p.Slug)
	}
	return slugs
}

// RoleSlug returns the sub-user's role slug, or "" if unset.
func (u *SubUser) RoleSlug() string {
	if u.Role == nil {
		return ""
	}
	return u.Role.Slug
}

// CreateRequest holds parameters for creating a sub-user. Required by the API:
// Name, Email, Password, Role (a role slug), and Projects (project slugs). The
// email must be a valid company email address and the password must be at least
// 8 characters with upper, lower, digit, and special character.
type CreateRequest struct {
	Name           string   `json:"name"`
	Email          string   `json:"email"`
	Password       string   `json:"password"`
	Role           string   `json:"role"`
	Projects       []string `json:"projects"`
	AuthUser       string   `json:"auth_user,omitempty"`
	IsUserPassword bool     `json:"is_user_password"`
	IsPartner      bool     `json:"is_partner"`
	IsBlocked      bool     `json:"is_blocked"`
}

// UpdateRequest holds parameters for updating a sub-user. The API requires Email
// and Projects on every update, so callers that only mean to change one field
// must echo the current Email/Projects back (see the command layer, which loads
// the current user first).
type UpdateRequest struct {
	Name     string   `json:"name,omitempty"`
	Email    string   `json:"email"`
	Role     string   `json:"role,omitempty"`
	Projects []string `json:"projects"`
	// IsPartner is a pointer (omitempty) and is sent ONLY when the caller
	// explicitly changes it. This is forced, not a preference: the sub-user read
	// model exposes no partner flag at all — GET /users, the POST/PUT response,
	// and ?include= all omit is_partner (verified live 2026-06-21; the user
	// object only carries id, name, email, account, user_type, registered_by,
	// registration_type, is_blocked, reason, user_status, login_attempt_status,
	// parent_id, subsidiary_id, role, projects, timestamps, last_login). Because
	// we cannot read the current value back, we cannot echo it to preserve it on
	// a PUT; sending it only on an explicit --partner change is the safest option
	// (sending it unconditionally would risk clearing it). Preserving partner
	// status across updates requires the API to surface is_partner in the read
	// model first.
	IsPartner *bool `json:"is_partner,omitempty"`
	IsBlocked bool  `json:"is_blocked"`
}

// listResponse is the STKCNSL paginated response envelope (single page).
type listResponse struct {
	Status  string    `json:"status"`
	Message string    `json:"message"`
	Data    []SubUser `json:"data"`
}

// singleResponse is the STKCNSL single-object response envelope.
type singleResponse struct {
	Status  string  `json:"status"`
	Message string  `json:"message"`
	Data    SubUser `json:"data"`
}

// Service provides sub-user API operations.
type Service struct {
	client *httpclient.Client
}

// NewService creates a new sub-user Service.
func NewService(client *httpclient.Client) *Service {
	return &Service{client: client}
}

// List returns all sub-users under the account.
func (s *Service) List(ctx context.Context) ([]SubUser, error) {
	var resp listResponse
	if err := s.client.Get(ctx, "/users", nil, &resp); err != nil {
		return nil, fmt.Errorf("listing sub-users: %w", err)
	}
	return resp.Data, nil
}

// Create provisions a new sub-user.
func (s *Service) Create(ctx context.Context, req CreateRequest) (*SubUser, error) {
	var resp singleResponse
	if err := s.client.Post(ctx, "/users", req, &resp); err != nil {
		return nil, fmt.Errorf("creating sub-user: %w", err)
	}
	return &resp.Data, nil
}

// Update modifies a sub-user by ID (UUID). Email and Projects must be set on req.
func (s *Service) Update(ctx context.Context, id string, req UpdateRequest) (*SubUser, error) {
	var resp singleResponse
	if err := s.client.Put(ctx, "/users/"+id, nil, req, &resp); err != nil {
		return nil, fmt.Errorf("updating sub-user %s: %w", id, err)
	}
	return &resp.Data, nil
}

// Delete removes a sub-user by ID (UUID).
func (s *Service) Delete(ctx context.Context, id string) error {
	if err := s.client.Delete(ctx, "/users/"+id, nil); err != nil {
		return fmt.Errorf("deleting sub-user %s: %w", id, err)
	}
	return nil
}
