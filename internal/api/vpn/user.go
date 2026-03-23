package vpn

import (
	"context"
	"fmt"
	"net/url"

	"github.com/zsoftly/zcp-cli/internal/httpclient"
)

// User represents a ZCP VPN user.
type User struct {
	UUID       string `json:"uuid"`
	UserName   string `json:"userName"`
	IsActive   bool   `json:"isActive"`
	DomainUUID string `json:"domainUuid"`
	Status     string `json:"status"`
}

type listVpnUserResponse struct {
	Count               int    `json:"count"`
	ListVpnUserResponse []User `json:"listVpnUserResponse"`
}

// UserService provides VPN user API operations.
type UserService struct {
	client *httpclient.Client
}

// NewUserService creates a new UserService.
func NewUserService(client *httpclient.Client) *UserService {
	return &UserService{client: client}
}

// List returns VPN users. uuid is an optional filter.
func (s *UserService) List(ctx context.Context, uuid string) ([]User, error) {
	var q url.Values
	if uuid != "" {
		q = url.Values{"uuid": {uuid}}
	}
	var resp listVpnUserResponse
	if err := s.client.Get(ctx, "/restapi/vpnuser/vpnUserlist", q, &resp); err != nil {
		return nil, fmt.Errorf("listing VPN users: %w", err)
	}
	return resp.ListVpnUserResponse, nil
}

// Create adds a new VPN user with the given username and password.
func (s *UserService) Create(ctx context.Context, username, password string) (*User, error) {
	body := map[string]string{
		"username": username,
		"password": password,
	}
	var resp listVpnUserResponse
	if err := s.client.Post(ctx, "/restapi/vpnuser/addVpnUser", body, &resp); err != nil {
		return nil, fmt.Errorf("creating VPN user: %w", err)
	}
	if len(resp.ListVpnUserResponse) == 0 {
		return nil, fmt.Errorf("create VPN user returned empty response")
	}
	return &resp.ListVpnUserResponse[0], nil
}

// Delete removes a VPN user by username (not UUID) via query param.
func (s *UserService) Delete(ctx context.Context, username string) error {
	q := url.Values{"userName": {username}}
	if err := s.client.Delete(ctx, "/restapi/vpnuser/deleteVpnUser", q); err != nil {
		return fmt.Errorf("deleting VPN user %q: %w", username, err)
	}
	return nil
}
