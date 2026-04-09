// Package virtualrouter provides ZCP virtual router API operations.
package virtualrouter

import (
	"context"
	"fmt"

	"github.com/zsoftly/zcp-cli/internal/httpclient"
)

// VirtualRouter represents a ZCP virtual router.
type VirtualRouter struct {
	ID          string `json:"id"`
	Slug        string `json:"slug"`
	Name        string `json:"name"`
	Status      string `json:"status"`
	State       string `json:"state"`
	Gateway     string `json:"gateway"`
	PublicIP    string `json:"public_ip"`
	GuestIP     string `json:"guest_ip"`
	NetworkSlug string `json:"network_slug"`
	ZoneSlug    string `json:"zone_slug"`
	ZoneName    string `json:"zone_name"`
	Role        string `json:"role"`
	Redundant   bool   `json:"is_redundant"`
	Version     string `json:"version"`
}

// CreateRequest holds parameters for creating a virtual router.
type CreateRequest struct {
	Name          string `json:"vr_name"`
	NetworkSlug   string `json:"network_slug"`
	PlanSlug      string `json:"plan,omitempty"`
	CloudProvider string `json:"cloud_provider"`
	Region        string `json:"region"`
	Project       string `json:"project"`
}

type listVirtualRouterResponse struct {
	Status string          `json:"status"`
	Data   []VirtualRouter `json:"data"`
}

type singleVirtualRouterResponse struct {
	Status string        `json:"status"`
	Data   VirtualRouter `json:"data"`
}

// Service provides virtual router API operations.
type Service struct {
	client *httpclient.Client
}

// NewService creates a new virtual router Service.
func NewService(client *httpclient.Client) *Service {
	return &Service{client: client}
}

// List returns all virtual routers.
func (s *Service) List(ctx context.Context) ([]VirtualRouter, error) {
	var resp listVirtualRouterResponse
	if err := s.client.Get(ctx, "/virtual-routers", nil, &resp); err != nil {
		return nil, fmt.Errorf("listing virtual routers: %w", err)
	}
	return resp.Data, nil
}

// Create provisions a new virtual router.
func (s *Service) Create(ctx context.Context, req CreateRequest) (*VirtualRouter, error) {
	var resp singleVirtualRouterResponse
	if err := s.client.Post(ctx, "/virtual-routers", req, &resp); err != nil {
		return nil, fmt.Errorf("creating virtual router: %w", err)
	}
	return &resp.Data, nil
}

// Reboot restarts a virtual router by slug.
func (s *Service) Reboot(ctx context.Context, slug string) (*VirtualRouter, error) {
	var resp singleVirtualRouterResponse
	if err := s.client.Get(ctx, "/virtual-routers/"+slug+"/reboot", nil, &resp); err != nil {
		return nil, fmt.Errorf("rebooting virtual router %s: %w", slug, err)
	}
	return &resp.Data, nil
}
