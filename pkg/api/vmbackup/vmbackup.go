// Package vmbackup provides ZCP VM backup API operations (STKCNSL).
package vmbackup

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/zsoftly/zcp-cli/pkg/api/billing"
	"github.com/zsoftly/zcp-cli/pkg/api/response"
	"github.com/zsoftly/zcp-cli/pkg/httpclient"
)

// ServiceName is the service name the service-cancellation endpoint accepts
// for VM backup schedules. Verified live 2026-09-06: cancelling with
// "Virtual Machine Backup" is rejected ("The provided service is invalid."),
// but "Backups" succeeds (the response echoes back
// "service":"VirtualMachineBackup"). The project services summary
// (/projects/dashboard/{slug}/services) also counts VM backup schedules
// under the key "Backups".
const ServiceName = "Backups"

// ---------- Response envelope ----------

// Envelope wraps paginated STKCNSL responses.
type Envelope struct {
	Status      string          `json:"status"`
	Message     string          `json:"message"`
	Timezone    string          `json:"timezone"`
	CurrentPage int             `json:"current_page"`
	Data        json.RawMessage `json:"data"`
	LastPage    int             `json:"last_page"`
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

// VMRef is the virtual machine nested in a VM backup listing item.
type VMRef struct {
	ID   string `json:"id"`
	Slug string `json:"slug"`
	Name string `json:"name"`
}

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
	VirtualMachine       *VMRef  `json:"virtual_machine"`
	State                string  `json:"state"`
	Interval             string  `json:"interval"`
	At                   int     `json:"at"`
	ScheduledAt          string  `json:"scheduled_at"`
	ServiceName          string  `json:"service_name"`
	AllTimeConsumption   float64 `json:"all_time_consumption"`
	CreatedAt            string  `json:"created_at"`
	UpdatedAt            string  `json:"updated_at"`
	DeletedAt            *string `json:"deleted_at"`
}

// VMSlug returns the slug of the virtual machine this backup schedule
// belongs to. List responses nest it under "virtual_machine" with no
// top-level virtual_machine_id; fall back to VirtualMachineID so callers get
// a usable value either way.
func (v *VMBackup) VMSlug() string {
	if v.VirtualMachine != nil && v.VirtualMachine.Slug != "" {
		return v.VirtualMachine.Slug
	}
	return v.VirtualMachineID
}

// vmBackupAlias avoids infinite recursion when VMBackup.UnmarshalJSON
// re-decodes the payload through json.Unmarshal.
type vmBackupAlias VMBackup

// UnmarshalJSON decodes a VMBackup, tolerating the "at" field being either a
// JSON number or a quoted numeric string (list responses return "3"). A null
// or empty "at" decodes to 0.
func (v *VMBackup) UnmarshalJSON(data []byte) error {
	aux := struct {
		At json.RawMessage `json:"at"`
		*vmBackupAlias
	}{
		vmBackupAlias: (*vmBackupAlias)(v),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	at, err := response.ParseFlexInt(aux.At)
	if err != nil {
		return fmt.Errorf("vmbackup: field \"at\": %w", err)
	}
	v.At = at
	return nil
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

// maxListPages bounds the paginated List loop so a server that misreports
// last_page can't loop forever.
const maxListPages = 1000

// List returns all VM backups, walking every page of the listing.
func (s *Service) List(ctx context.Context, region, project string) ([]VMBackup, error) {
	var all []VMBackup
	page := 1
	for ; page <= maxListPages; page++ {
		q := url.Values{}
		if region != "" {
			q.Set("filter[region]", region)
		}
		if project != "" {
			q.Set("filter[project]", project)
		}
		if page > 1 {
			q.Set("page", fmt.Sprintf("%d", page))
		}

		var env Envelope
		if err := s.client.Get(ctx, "/virtual-machines/backups", q, &env); err != nil {
			return nil, fmt.Errorf("listing VM backups: %w", err)
		}
		var backups []VMBackup
		if err := json.Unmarshal(env.Data, &backups); err != nil {
			return nil, fmt.Errorf("decoding VM backups: %w", err)
		}
		all = append(all, backups...)

		// The loop counter, not the server-echoed current_page, drives
		// pagination: a server that ignores ?page and always echoes
		// current_page=1 would otherwise never advance past the first page.
		if len(backups) == 0 || env.LastPage <= 0 || page >= env.LastPage {
			return all, nil
		}
	}
	return nil, fmt.Errorf("listing VM backups: exceeded %d pages without reaching the last page", maxListPages)
}

// Create creates a new VM backup on the given VM slug.
func (s *Service) Create(ctx context.Context, vmSlug string, req CreateRequest) (*ActionResponse, error) {
	var resp ActionResponse
	if err := s.client.Post(ctx, "/virtual-machines/"+vmSlug+"/backups", req, &resp); err != nil {
		return nil, fmt.Errorf("creating VM backup on %s: %w", vmSlug, err)
	}
	return &resp, nil
}

// Delete requests deletion of a VM backup schedule.
//
// The route api/virtual-machines/backups/{slug} rejects DELETE ("Supported
// methods: PUT", verified live 2026-09-06), so a direct DELETE request can
// never succeed. Deletion instead goes through the unified service-
// cancellation workflow the CMP Web UI uses (the same one instance delete
// relies on): POST /billing/service-cancel-requests/{slug} with
// service_name "Backups" (verified live 2026-09-06).
func (s *Service) Delete(ctx context.Context, slug string) error {
	req := billing.CancelServiceRequest{
		ServiceName: ServiceName,
		Reason:      "not_needed_anymore",
		Type:        "Immediate",
		Status:      "Pending",
	}
	if err := billing.NewService(s.client).CancelService(ctx, slug, req); err != nil {
		return fmt.Errorf("deleting VM backup %s: %w", slug, err)
	}
	return nil
}
