// Package vmsnapshot provides ZCP VM snapshot API operations (STKCNSL).
package vmsnapshot

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/zsoftly/zcp-cli/internal/httpclient"
)

// ---------- Response envelope ----------

// Envelope wraps paginated STKCNSL responses.
type Envelope struct {
	Status      string          `json:"status"`
	Message     string          `json:"message"`
	Timezone    string          `json:"timezone"`
	CurrentPage int             `json:"current_page"`
	Data        json.RawMessage `json:"data"`
	Total       int             `json:"total"`
}

// ActionResponse wraps simple action responses (revert).
type ActionResponse struct {
	Status   string      `json:"status"`
	Message  string      `json:"message"`
	Timezone string      `json:"timezone"`
	Data     interface{} `json:"data"`
}

// ---------- Types ----------

// VMSnapshot represents a STKCNSL VM snapshot.
type VMSnapshot struct {
	ID                   string  `json:"id"`
	Name                 string  `json:"name"`
	Slug                 string  `json:"slug"`
	Description          *string `json:"description"`
	UserID               string  `json:"user_id"`
	AccountID            string  `json:"account_id"`
	ProjectID            string  `json:"project_id"`
	RegionID             string  `json:"region_id"`
	CloudProviderID      string  `json:"cloud_provider_id"`
	CloudProviderSetupID string  `json:"cloud_provider_setup_id"`
	VirtualMachineID     string  `json:"virtual_machine_id"`
	State                string  `json:"state"`
	ServiceName          string  `json:"service_name"`
	AllTimeConsumption   float64 `json:"all_time_consumption"`
	CreatedAt            string  `json:"created_at"`
	UpdatedAt            string  `json:"updated_at"`
	DeletedAt            *string `json:"deleted_at"`
}

// ---------- Request types ----------

// CreateRequest holds parameters for creating a VM snapshot.
type CreateRequest struct {
	Name          string  `json:"name"`
	BillingCycle  string  `json:"billing_cycle"`
	Plan          string  `json:"plan"`
	IsMemory      bool    `json:"is_memory"`
	IsVMSnapshot  bool    `json:"is_vm_snapshot"`
	Project       string  `json:"project"`
	CloudProvider string  `json:"cloud_provider"`
	Region        string  `json:"region"`
	Service       string  `json:"service"`
	Coupon        *string `json:"coupon"`
}

// ---------- Service ----------

// Service provides VM snapshot API operations.
type Service struct {
	client *httpclient.Client
}

// NewService creates a new VMSnapshot Service.
func NewService(client *httpclient.Client) *Service {
	return &Service{client: client}
}

// List returns all VM snapshots.
func (s *Service) List(ctx context.Context) ([]VMSnapshot, error) {
	var env Envelope
	if err := s.client.Get(ctx, "/virtual-machines/snapshots", nil, &env); err != nil {
		return nil, fmt.Errorf("listing VM snapshots: %w", err)
	}
	var snapshots []VMSnapshot
	if err := json.Unmarshal(env.Data, &snapshots); err != nil {
		return nil, fmt.Errorf("decoding VM snapshots: %w", err)
	}
	return snapshots, nil
}

// Create creates a new VM snapshot on the given VM slug.
func (s *Service) Create(ctx context.Context, vmSlug string, req CreateRequest) (*ActionResponse, error) {
	var resp ActionResponse
	if err := s.client.Post(ctx, "/virtual-machines/"+vmSlug+"/snapshots", req, &resp); err != nil {
		return nil, fmt.Errorf("creating VM snapshot on %s: %w", vmSlug, err)
	}
	return &resp, nil
}

// Delete permanently removes a VM snapshot by slug.
func (s *Service) Delete(ctx context.Context, snapshotSlug string) error {
	if err := s.client.Delete(ctx, "/virtual-machines/snapshots/"+snapshotSlug, nil); err != nil {
		return fmt.Errorf("deleting VM snapshot %s: %w", snapshotSlug, err)
	}
	return nil
}

// Revert reverts an instance to a VM snapshot state.
func (s *Service) Revert(ctx context.Context, snapshotSlug string) (*ActionResponse, error) {
	var resp ActionResponse
	if err := s.client.Post(ctx, "/virtual-machines/snapshots/"+snapshotSlug+"/revert", nil, &resp); err != nil {
		return nil, fmt.Errorf("reverting VM snapshot %s: %w", snapshotSlug, err)
	}
	return &resp, nil
}
