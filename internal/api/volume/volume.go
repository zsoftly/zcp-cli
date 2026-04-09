// Package volume provides ZCP block storage (volume) API operations
// targeting the STKCNSL API.
package volume

import (
	"context"
	"fmt"
	"net/url"

	"github.com/zsoftly/zcp-cli/internal/httpclient"
)

// StorageSetting represents the storage configuration for a block storage volume.
type StorageSetting struct {
	ID                string `json:"id"`
	Name              string `json:"name"`
	Slug              string `json:"slug"`
	StorageCategoryID string `json:"storage_category_id"`
	RegionID          string `json:"region_id"`
}

// Region represents the region where the volume is deployed.
type Region struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Slug    string `json:"slug"`
	Country string `json:"country"`
}

// CloudProvider represents the cloud provider for the volume.
type CloudProvider struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	DisplayName string `json:"display_name"`
	Slug        string `json:"slug"`
}

// BillingCycle represents a billing cycle attached to an offering.
type BillingCycle struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Slug string `json:"slug"`
}

// Offering represents the billing/plan offering on a volume.
type Offering struct {
	ID            string        `json:"id"`
	Size          interface{}   `json:"size"`
	Price         string        `json:"price"`
	BillingStatus bool          `json:"billing_status"`
	RenewAt       string        `json:"renew_at"`
	BillingCycle  *BillingCycle `json:"billing_cycle"`
}

// Volume represents a STKCNSL block storage volume.
type Volume struct {
	ID                    string          `json:"id"`
	BlockstorageID        string          `json:"blockstorage_id"`
	VirtualMachineID      string          `json:"virtual_machine_id"`
	Size                  interface{}     `json:"size"`
	Name                  string          `json:"name"`
	Slug                  string          `json:"slug"`
	Description           *string         `json:"description"`
	UserID                string          `json:"user_id"`
	AccountID             string          `json:"account_id"`
	ProjectID             string          `json:"project_id"`
	RegionID              string          `json:"region_id"`
	CloudProviderID       string          `json:"cloud_provider_id"`
	CloudProviderSetupID  string          `json:"cloud_provider_setup_id"`
	RequestStatus         bool            `json:"request_status"`
	VolumeType            string          `json:"volume_type"`
	Bootable              bool            `json:"bootable"`
	IsRoot                bool            `json:"is_root"`
	IsSnapshotHidden      bool            `json:"is_snapshot_hidden"`
	IsRestricted          bool            `json:"is_restricted"`
	IsResizeEnable        bool            `json:"is_resize_enable"`
	ServiceName           string          `json:"service_name"`
	ServiceDisplayName    string          `json:"service_display_name"`
	AllTimeConsumption    float64         `json:"all_time_consumption"`
	HasContract           bool            `json:"has_contract"`
	IsServiceTrialExpired bool            `json:"is_service_trial_expired"`
	FrozenAt              *string         `json:"frozen_at"`
	SuspendedAt           *string         `json:"suspended_at"`
	TerminatedAt          *string         `json:"terminated_at"`
	CreatedAt             string          `json:"created_at"`
	UpdatedAt             string          `json:"updated_at"`
	DeletedAt             *string         `json:"deleted_at"`
	StorageSetting        *StorageSetting `json:"storage_setting"`
	CloudProvider         *CloudProvider  `json:"cloud_provider"`
	Region                *Region         `json:"region"`
	Offering              *Offering       `json:"offering"`
}

// listResponse is the STKCNSL paginated envelope for block storage volumes.
type listResponse struct {
	Status      string   `json:"status"`
	Message     string   `json:"message"`
	CurrentPage int      `json:"current_page"`
	Data        []Volume `json:"data"`
	Total       int      `json:"total"`
}

// singleResponse is used when the API returns a single volume in `data`.
type singleResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
	Data    Volume `json:"data"`
}

// CreateRequest holds parameters for creating a block storage volume.
type CreateRequest struct {
	Name            string `json:"name"`
	Project         string `json:"project"`
	CloudProvider   string `json:"cloud_provider"`
	Region          string `json:"region"`
	BillingCycle    string `json:"billing_cycle"`
	StorageCategory string `json:"storage_category"`
	Plan            string `json:"plan"`
	IsCustomPlan    bool   `json:"is_custom_plan"`
	CustomPlan      string `json:"custom_plan,omitempty"`
	VirtualMachine  string `json:"virtual_machine,omitempty"`
	Coupon          string `json:"coupon,omitempty"`
	IsFreeTrial     bool   `json:"is_free_trial_plan"`
}

// AttachRequest holds parameters for attaching a volume to a VM.
type AttachRequest struct {
	VirtualMachine string `json:"virtual_machine"`
}

// Service provides block storage API operations.
type Service struct {
	client *httpclient.Client
}

// NewService creates a new volume Service.
func NewService(client *httpclient.Client) *Service {
	return &Service{client: client}
}

// List returns block storage volumes. Use the include parameter to embed related resources.
func (s *Service) List(ctx context.Context) ([]Volume, error) {
	q := url.Values{
		"include": {"cloud_provider,region,virtual_machine,project,snapshots,offering"},
	}
	var resp listResponse
	if err := s.client.Get(ctx, "/blockstorages", q, &resp); err != nil {
		return nil, fmt.Errorf("listing block storages: %w", err)
	}
	return resp.Data, nil
}

// Create creates a new block storage volume.
func (s *Service) Create(ctx context.Context, req CreateRequest) (*Volume, error) {
	var resp singleResponse
	if err := s.client.Post(ctx, "/blockstorages", req, &resp); err != nil {
		return nil, fmt.Errorf("creating block storage: %w", err)
	}
	return &resp.Data, nil
}

// Attach attaches a volume to a virtual machine.
func (s *Service) Attach(ctx context.Context, volumeSlug, vmSlug string) (*Volume, error) {
	body := AttachRequest{VirtualMachine: vmSlug}
	var resp singleResponse
	path := fmt.Sprintf("/blockstorages/%s/attach", volumeSlug)
	if err := s.client.Post(ctx, path, body, &resp); err != nil {
		return nil, fmt.Errorf("attaching block storage %s to VM %s: %w", volumeSlug, vmSlug, err)
	}
	return &resp.Data, nil
}

// Detach detaches a volume from its virtual machine.
func (s *Service) Detach(ctx context.Context, volumeSlug string) (*Volume, error) {
	var resp singleResponse
	path := fmt.Sprintf("/blockstorages/%s/detach", volumeSlug)
	if err := s.client.Post(ctx, path, nil, &resp); err != nil {
		return nil, fmt.Errorf("detaching block storage %s: %w", volumeSlug, err)
	}
	return &resp.Data, nil
}
