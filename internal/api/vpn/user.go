// Package vpn provides ZCP VPN API operations for users and customer gateways.
package vpn

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/zsoftly/zcp-cli/internal/httpclient"
)

// User represents a ZCP VPN user.
type User struct {
	Slug     string `json:"slug"`
	UserName string `json:"userName"`
	Status   string `json:"status"`
}

// UserCreateRequest holds parameters for creating a VPN user.
type UserCreateRequest struct {
	Username      string `json:"username"`
	Password      string `json:"password"`
	CloudProvider string `json:"cloud_provider"`
	Region        string `json:"region"`
	Project       string `json:"project"`
}

// apiResponse is the STKCNSL response envelope.
type apiResponse struct {
	Status string          `json:"status"`
	Data   json.RawMessage `json:"data"`
}

// UserService provides VPN user API operations.
type UserService struct {
	client *httpclient.Client
}

// NewUserService creates a new UserService.
func NewUserService(client *httpclient.Client) *UserService {
	return &UserService{client: client}
}

// List returns all VPN users.
func (s *UserService) List(ctx context.Context) ([]User, error) {
	var env apiResponse
	if err := s.client.Get(ctx, "/vpn-users", nil, &env); err != nil {
		return nil, fmt.Errorf("listing VPN users: %w", err)
	}
	var users []User
	if err := json.Unmarshal(env.Data, &users); err != nil {
		return nil, fmt.Errorf("decoding VPN user list: %w", err)
	}
	return users, nil
}

// Create adds a new VPN user with the given request parameters.
func (s *UserService) Create(ctx context.Context, req UserCreateRequest) (*User, error) {
	var env apiResponse
	if err := s.client.Post(ctx, "/vpn-users", req, &env); err != nil {
		return nil, fmt.Errorf("creating VPN user: %w", err)
	}
	var u User
	if err := json.Unmarshal(env.Data, &u); err != nil {
		return nil, fmt.Errorf("decoding created VPN user: %w", err)
	}
	return &u, nil
}

// Delete removes a VPN user by slug.
func (s *UserService) Delete(ctx context.Context, slug string) error {
	if err := s.client.Delete(ctx, "/vpn-users/"+slug, nil); err != nil {
		return fmt.Errorf("deleting VPN user %q: %w", slug, err)
	}
	return nil
}
