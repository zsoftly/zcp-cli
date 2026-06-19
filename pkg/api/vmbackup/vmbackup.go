// Package vmbackup provides ZCP VM backup API operations (STKCNSL).
package vmbackup

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/zsoftly/zcp-cli/pkg/httpclient"
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

// ActionResponse wraps simple action responses.
type ActionResponse struct {
	Status   string      `json:"status"`
	Message  string      `json:"message"`
	Timezone string      `json:"timezone"`
	Data     interface{} `json:"data"`
}

// ---------- Types ----------

// VMBackup represents a STKCNSL VM backup.
type VMBackup struct {
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

// CreateRequest holds parameters for creating a VM backup.
type CreateRequest struct {
	Interval      string  `json:"interval"`
	At            int     `json:"at"`
	Immediate     int     `json:"immediate"`
	CloudProvider string  `json:"cloud_provider"`
	Region        string  `json:"region"`
	BillingCycle  string  `json:"billing_cycle"`
	Plan          string  `json:"plan"`
	PseudoService string  `json:"psudo_service"`
	Project       string  `json:"project"`
	IsVMSnapshot  bool    `json:"is_vm_snapshot"`
	Coupon        *string `json:"coupon"`
}

// ---------- Service ----------

// Service provides VM backup API operations.
type Service struct {
	client *httpclient.Client
}

// NewService creates a new VMBackup Service.
func NewService(client *httpclient.Client) *Service {
	return &Service{client: client}
}

// List returns all VM backups.
func (s *Service) List(ctx context.Context, region, project string) ([]VMBackup, error) {
	var env Envelope
	q := url.Values{}
	if region != "" {
		q.Set("filter[region]", region)
	}
	if project != "" {
		q.Set("filter[project]", project)
	}
	if err := s.client.Get(ctx, "/virtual-machines/backups", q, &env); err != nil {
		return nil, fmt.Errorf("listing VM backups: %w", err)
	}
	var backups []VMBackup
	if err := json.Unmarshal(env.Data, &backups); err != nil {
		return nil, fmt.Errorf("decoding VM backups: %w", err)
	}
	return backups, nil
}

// Create creates a new VM backup on the given VM slug.
func (s *Service) Create(ctx context.Context, vmSlug string, req CreateRequest) (*ActionResponse, error) {
	var resp ActionResponse
	if err := s.client.Post(ctx, "/virtual-machines/"+vmSlug+"/backups", req, &resp); err != nil {
		return nil, fmt.Errorf("creating VM backup on %s: %w", vmSlug, err)
	}
	return &resp, nil
}

// Delete permanently deletes a VM backup.
func (s *Service) Delete(ctx context.Context, slug string) error {
	if err := s.client.Delete(ctx, "/virtual-machines/backups/"+slug, nil); err != nil {
		return fmt.Errorf("deleting VM backup %s: %w", slug, err)
	}
	return nil
}
