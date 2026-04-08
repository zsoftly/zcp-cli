// Package project provides ZCP Project API operations (STKCNSL).
package project

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/zsoftly/zcp-cli/internal/httpclient"
)

// envelope is the standard STKCNSL response wrapper.
type envelope struct {
	Status string          `json:"status"`
	Data   json.RawMessage `json:"data"`
}

// Project represents a ZCP project.
type Project struct {
	ID          string      `json:"id"`
	Name        string      `json:"name"`
	Slug        string      `json:"slug"`
	Description string      `json:"description"`
	IconID      string      `json:"icon_id"`
	Purpose     string      `json:"purpose"`
	Status      interface{} `json:"status"`
	CreatedAt   string      `json:"created_at"`
	UpdatedAt   string      `json:"updated_at"`
}

// CreateRequest holds parameters for creating a project.
type CreateRequest struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Icon        string `json:"icon,omitempty"`
	Purpose     string `json:"purpose,omitempty"`
	Status      int    `json:"status"`
}

// UpdateRequest holds parameters for updating a project.
type UpdateRequest struct {
	Name        string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`
	IconID      string `json:"icon_id,omitempty"`
	Purpose     string `json:"purpose,omitempty"`
}

// Icon represents a project icon.
type Icon struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	URL  string `json:"url"`
}

// User represents a user assigned to a project.
type User struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
	Role  string `json:"role"`
}

// AddUserRequest holds parameters for adding a user to a project.
type AddUserRequest struct {
	Email string `json:"email"`
	Role  string `json:"role,omitempty"`
}

// DashboardService represents a service entry on the project dashboard.
type DashboardService struct {
	Name   string `json:"name"`
	Type   string `json:"type"`
	Status string `json:"status"`
	Count  int    `json:"count"`
}

// Service provides Project API operations.
type Service struct {
	client *httpclient.Client
}

// NewService creates a new Project Service.
func NewService(client *httpclient.Client) *Service {
	return &Service{client: client}
}

// decode unwraps the STKCNSL envelope and unmarshals data into dst.
func decode(raw json.RawMessage, dst interface{}) error {
	var env envelope
	if err := json.Unmarshal(raw, &env); err != nil {
		// Not wrapped — try direct decode.
		return json.Unmarshal(raw, dst)
	}
	if env.Status != "" && env.Data != nil {
		return json.Unmarshal(env.Data, dst)
	}
	// Fallback: raw is the data itself.
	return json.Unmarshal(raw, dst)
}

// List returns all projects.
func (s *Service) List(ctx context.Context) ([]Project, error) {
	var raw json.RawMessage
	if err := s.client.Get(ctx, "/projects", nil, &raw); err != nil {
		return nil, fmt.Errorf("listing projects: %w", err)
	}
	var projects []Project
	if err := decode(raw, &projects); err != nil {
		return nil, fmt.Errorf("decoding projects: %w", err)
	}
	return projects, nil
}

// Create provisions a new project.
func (s *Service) Create(ctx context.Context, req CreateRequest) (*Project, error) {
	var raw json.RawMessage
	if err := s.client.Post(ctx, "/projects", req, &raw); err != nil {
		return nil, fmt.Errorf("creating project: %w", err)
	}
	var p Project
	if err := decode(raw, &p); err != nil {
		return nil, fmt.Errorf("decoding project: %w", err)
	}
	return &p, nil
}

// Update modifies an existing project identified by slug.
func (s *Service) Update(ctx context.Context, slug string, req UpdateRequest) (*Project, error) {
	var raw json.RawMessage
	if err := s.client.Put(ctx, "/projects/"+slug, nil, req, &raw); err != nil {
		return nil, fmt.Errorf("updating project %s: %w", slug, err)
	}
	var p Project
	if err := decode(raw, &p); err != nil {
		return nil, fmt.Errorf("decoding project: %w", err)
	}
	return &p, nil
}

// Dashboard returns services for a project's dashboard.
func (s *Service) Dashboard(ctx context.Context, slug string) ([]DashboardService, error) {
	var raw json.RawMessage
	if err := s.client.Get(ctx, "/projects/dashboard/"+slug+"/services", nil, &raw); err != nil {
		return nil, fmt.Errorf("getting project dashboard %s: %w", slug, err)
	}
	var services []DashboardService
	if err := decode(raw, &services); err != nil {
		return nil, fmt.Errorf("decoding dashboard services: %w", err)
	}
	return services, nil
}

// ListIcons returns all available project icons.
func (s *Service) ListIcons(ctx context.Context) ([]Icon, error) {
	var raw json.RawMessage
	if err := s.client.Get(ctx, "/project-icons", nil, &raw); err != nil {
		return nil, fmt.Errorf("listing project icons: %w", err)
	}
	var icons []Icon
	if err := decode(raw, &icons); err != nil {
		return nil, fmt.Errorf("decoding project icons: %w", err)
	}
	return icons, nil
}

// ListUsers returns users assigned to a project.
func (s *Service) ListUsers(ctx context.Context, slug string) ([]User, error) {
	var raw json.RawMessage
	if err := s.client.Get(ctx, "/projects/"+slug+"/users", nil, &raw); err != nil {
		return nil, fmt.Errorf("listing users for project %s: %w", slug, err)
	}
	var users []User
	if err := decode(raw, &users); err != nil {
		return nil, fmt.Errorf("decoding project users: %w", err)
	}
	return users, nil
}

// AddUser adds a user to a project by slug.
func (s *Service) AddUser(ctx context.Context, slug string, req AddUserRequest) (*User, error) {
	var raw json.RawMessage
	if err := s.client.Post(ctx, "/projects/"+slug+"/users", req, &raw); err != nil {
		return nil, fmt.Errorf("adding user to project %s: %w", slug, err)
	}
	var u User
	if err := decode(raw, &u); err != nil {
		return nil, fmt.Errorf("decoding project user: %w", err)
	}
	return &u, nil
}

// Delete removes a project by slug.
func (s *Service) Delete(ctx context.Context, slug string) error {
	if err := s.client.Delete(ctx, "/projects/"+slug, nil); err != nil {
		return fmt.Errorf("deleting project %s: %w", slug, err)
	}
	return nil
}
