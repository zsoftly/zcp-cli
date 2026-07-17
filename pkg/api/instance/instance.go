// Package instance provides ZCP virtual machine API operations (STKCNSL).
package instance

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/zsoftly/zcp-cli/pkg/httpclient"
)

// ---------- Response envelope ----------

// Envelope wraps all paginated STKCNSL responses.
type Envelope struct {
	Status      string          `json:"status"`
	Message     string          `json:"message"`
	Timezone    string          `json:"timezone"`
	CurrentPage int             `json:"current_page"`
	LastPage    int             `json:"last_page"`
	Data        json.RawMessage `json:"data"`
	Total       int             `json:"total"`
}

// SingleEnvelope wraps single-object responses (show, create).
type SingleEnvelope struct {
	Status   string          `json:"status"`
	Message  string          `json:"message"`
	Timezone string          `json:"timezone"`
	Data     json.RawMessage `json:"data"`
}

// ActionResponse wraps simple action responses (start/stop/reboot/reset).
type ActionResponse struct {
	Status   string      `json:"status"`
	Message  string      `json:"message"`
	Timezone string      `json:"timezone"`
	Data     interface{} `json:"data"`
}

// ---------- Core types ----------

// VirtualMachine represents a STKCNSL virtual machine.
type VirtualMachine struct {
	ID                   string          `json:"id"`
	VMID                 string          `json:"vm_id"`
	Name                 string          `json:"name"`
	Slug                 string          `json:"slug"`
	Description          *string         `json:"description"`
	UserID               string          `json:"user_id"`
	AccountID            string          `json:"account_id"`
	ProjectID            string          `json:"project_id"`
	RegionID             string          `json:"region_id"`
	CloudProviderID      string          `json:"cloud_provider_id"`
	CloudProviderSetupID string          `json:"cloud_provider_setup_id"`
	RequestStatus        bool            `json:"request_status"`
	Hostname             string          `json:"hostname"`
	Username             string          `json:"username"`
	State                string          `json:"state"`
	IPAddresses          []IPAddresses   `json:"ipaddresses"`
	PublicIP             *string         `json:"public_ip"`
	PrivateIP            *string         `json:"private_ip"`
	FrozenAt             *string         `json:"frozen_at"`
	SuspendedAt          *string         `json:"suspended_at"`
	TerminatedAt         *string         `json:"terminated_at"`
	CreatedAt            string          `json:"created_at"`
	UpdatedAt            string          `json:"updated_at"`
	DeletedAt            *string         `json:"deleted_at"`
	IsVNF                bool            `json:"is_vnf"`
	ConsoleURL           *string         `json:"console_url"`
	Template             *VMTemplate     `json:"template"`
	Offering             *Offering       `json:"offering"`
	BillingCycle         *BillingCycle   `json:"billing_cycle"`
	Region               *Region         `json:"region"`
	CloudProvider        *CloudProvider  `json:"cloud_provider"`
	StorageSetting       *StorageSetting `json:"storage_setting"`
	Icon                 string          `json:"icon"`
	ServiceName          string          `json:"service_name"`
	ServiceDisplayName   string          `json:"service_display_name"`
	AllTimeConsumption   float64         `json:"all_time_consumption"`
	HasContract          bool            `json:"has_contract"`
	IsMetricsHidden      bool            `json:"is_metrics_hidden"`
	IsRestricted         bool            `json:"is_restricted"`
	HasAutoscale         bool            `json:"has_autoscale"`
	Networks             []VMNetwork     `json:"networks"`
}

// VMNetwork represents a network attached to a VM.
type VMNetwork struct {
	ID        string       `json:"id"`
	Name      string       `json:"name"`
	Slug      string       `json:"slug"`
	Type      string       `json:"type"`
	IsDefault bool         `json:"is_default"`
	Pivot     *VMNetworkIP `json:"pivot"`
}

// VMNetworkIP holds the IP assignment for a VM on a network.
type VMNetworkIP struct {
	IsDefault  bool   `json:"is_default"`
	MACAddress string `json:"macaddress"`
	IPAddress  string `json:"ipaddress"`
	Netmask    string `json:"netmask"`
}

// NetworkPrivateIP returns the private IP from the default network, falling back
// to the first network with an IP if no default is set, or "" if the VM has none.
// It returns "" (not a display placeholder) so callers such as `instance ssh` can
// detect "no IP" and fall through; the "-" placeholder is applied at display time.
func (vm *VirtualMachine) NetworkPrivateIP() string {
	var fallback string
	for _, n := range vm.Networks {
		if n.Pivot == nil || n.Pivot.IPAddress == "" {
			continue
		}
		if n.IsDefault || n.Pivot.IsDefault {
			return n.Pivot.IPAddress
		}
		if fallback == "" {
			fallback = n.Pivot.IPAddress
		}
	}
	return fallback
}

// GetPublicIPAddress returns the VM's public IP address, or "" if it has none.
// The VM's top-level public_ip is null even when a public IP is assigned, so the
// address is read from the ipaddresses list, selecting the entry the platform marks
// as a public IP (ip_type == "Public IP"). The per-entry "type" field is the IP
// version ("IPv4"/"IPv6"), NOT a public/private marker, so it must not be used here.
// Returns "" (not a display placeholder) for the same reason as NetworkPrivateIP.
func (vm *VirtualMachine) GetPublicIPAddress() string {
	for _, ip := range vm.IPAddresses {
		if ip.IPAddress != "" && ip.IPType == "Public IP" {
			return ip.IPAddress
		}
	}
	return ""
}

// VMTemplate represents the template/OS info on a VM.
type VMTemplate struct {
	ID              string           `json:"id"`
	TemplateID      string           `json:"template_id"`
	Name            string           `json:"name"`
	Slug            string           `json:"slug"`
	Type            string           `json:"type"`
	ImageType       string           `json:"image_type"`
	FileType        string           `json:"file_type"`
	PasswordEnabled bool             `json:"password_enabled"`
	IconURL         string           `json:"icon_url"`
	OperatingSystem *OperatingSystem `json:"operating_system"`
	OSVersion       *OSVersion       `json:"operating_system_version"`
	MarketPlaceApp  *json.RawMessage `json:"market_place_app"`
}

// OperatingSystem describes the OS family.
type OperatingSystem struct {
	ID                string `json:"id"`
	Name              string `json:"name"`
	Slug              string `json:"slug"`
	VMDefaultUsername string `json:"vm_default_username"`
	Family            string `json:"family"`
}

// OSVersion describes a specific OS version.
type OSVersion struct {
	ID          string `json:"id"`
	Version     string `json:"version"`
	PricingType string `json:"pricing_type"`
}

// IPAddresses represents an IP address entry attached to a VM.
type IPAddresses struct {
	ID        string `json:"id"`
	IPAddress string `json:"ipaddress"`
	Type      string `json:"type"`    // IP version, e.g. "IPv4" — NOT a public/private marker
	IPType    string `json:"ip_type"` // platform label, e.g. "Public IP"
}

// BillingCycle represents a billing period.
type BillingCycle struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Slug     string `json:"slug"`
	Duration int    `json:"duration"`
	Unit     string `json:"unit"`
}
type Offering struct {
	ID           string        `json:"id"`
	BillingCycle *BillingCycle `json:"billing_cycle"`
}

// Region represents a cloud region.
type Region struct {
	ID          string `json:"id"`
	RegionID    string `json:"region_id"`
	Name        string `json:"name"`
	Slug        string `json:"slug"`
	Country     string `json:"country"`
	CountryCode string `json:"country_code"`
}

// CloudProvider represents the cloud provider.
type CloudProvider struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	DisplayName string `json:"display_name"`
	Slug        string `json:"slug"`
}

// StorageSetting represents the storage configuration.
type StorageSetting struct {
	ID              string           `json:"id"`
	Name            string           `json:"name"`
	Slug            string           `json:"slug"`
	StorageCategory *StorageCategory `json:"storage_category"`
}

// StorageCategory represents the type of storage (SSD, NVMe, etc.).
type StorageCategory struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Slug string `json:"slug"`
}

// ActivityLog represents a VM activity log entry.
type ActivityLog struct {
	ID          json.Number `json:"id"`
	Category    string      `json:"category"`
	Action      string      `json:"action"`
	Status      string      `json:"status"`
	Error       string      `json:"error"`
	Description string      `json:"description"`
	CreatedAt   string      `json:"created_at"`
	UpdatedAt   string      `json:"updated_at"`
	Project     string      `json:"project"`
}

// Addon represents a VM addon.
type Addon struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Slug        string `json:"slug"`
	Description string `json:"description"`
	Status      bool   `json:"status"`
	CreatedAt   string `json:"created_at"`
}

// ---------- Request types ----------

// CreateRequest holds parameters for creating a VM via STKCNSL.
type CreateRequest struct {
	Name                 string      `json:"name"`
	CloudProvider        string      `json:"cloud_provider"`
	Project              string      `json:"project"`
	Region               string      `json:"region"`
	BootSource           string      `json:"boot_source"`
	Server               string      `json:"server,omitempty"`
	Template             string      `json:"template"`
	IsPublic             bool        `json:"is_public"`
	NetworkType          string      `json:"network_type"`
	Networks             []string    `json:"networks"`
	BillingCycle         string      `json:"billing_cycle"`
	SSHKey               *string     `json:"ssh_key"`
	AuthMethod           string      `json:"authMethod,omitempty"`
	Plan                 string      `json:"plan"`
	CustomPlan           *CustomPlan `json:"custom_plan"`
	OSFamily             string      `json:"os_family,omitempty"`
	TemplateType         string      `json:"template_type,omitempty"`
	Hostname             string      `json:"hostname"`
	Username             string      `json:"username,omitempty"`
	Password             *string     `json:"password"`
	Coupon               *string     `json:"coupon"`
	Addons               []string    `json:"addons"`
	UserData             *string     `json:"user_data"`
	StorageCategory      string      `json:"storage_category,omitempty"`
	ComputeCategory      string      `json:"compute_category,omitempty"`
	BlockstoragePlan     string      `json:"blockstorage_plan,omitempty"`
	NetworkPlan          string      `json:"network_plan,omitempty"`
	IsVNF                bool        `json:"is_vnf"`
	IsVMPasswordRequired bool        `json:"is_vm_password_required"`
	IsVMSSHRequired      bool        `json:"is_vm_ssh_required"`
	IsFreeTrial          bool        `json:"is_free_trial_plan"`
}

// CustomPlan allows specifying custom CPU/memory/storage when using a custom plan.
type CustomPlan struct {
	Storage string `json:"storage,omitempty"`
	CPU     string `json:"cpu,omitempty"`
	Memory  string `json:"memory,omitempty"`
}

// ChangeLabelRequest holds parameters for changing a VM hostname.
type ChangeLabelRequest struct {
	Name     string `json:"name"`
	Hostname string `json:"hostname"`
}

// ChangePasswordRequest holds parameters for changing a VM password.
type ChangePasswordRequest struct {
	Password string `json:"password"`
	VM       string `json:"vm"`
}

// ChangePlanRequest holds parameters for changing a VM plan.
type ChangePlanRequest struct {
	Plan         string `json:"plan"`
	Slug         string `json:"slug"`
	VM           string `json:"vm"`
	BillingCycle string `json:"billing_cycle"`
}

// ChangeTemplateRequest holds parameters for changing a VM OS template.
type ChangeTemplateRequest struct {
	Template string `json:"template"`
}

// ChangeStartupScriptRequest holds parameters for changing a VM startup script.
type ChangeStartupScriptRequest struct {
	UserData string `json:"user_data"`
}

// AddNetworkRequest holds parameters for adding a network to a VM.
type AddNetworkRequest struct {
	Network string `json:"network"`
}

// TagRequest holds parameters for creating or deleting a tag on a VM.
type TagRequest struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// PurchaseAddonRequest holds parameters for purchasing a VM addon.
type PurchaseAddonRequest struct {
	VirtualMachine string       `json:"virtual_machine"`
	OSFamily       string       `json:"os_family,omitempty"`
	Project        string       `json:"project"`
	Region         string       `json:"region"`
	CloudProvider  string       `json:"cloud_provider"`
	Addons         []AddonInput `json:"addons"`
	Service        string       `json:"service"`
	BillingCycle   string       `json:"billing_cycle"`
	Plan           string       `json:"plan,omitempty"`
	Coupon         *string      `json:"coupon"`
}

// AddonInput describes a single addon to purchase.
type AddonInput struct {
	Category string `json:"category"`
	ID       string `json:"id"`
	Slug     string `json:"slug"`
	Quantity int    `json:"quantity"`
}

// ---------- Service ----------

// Service provides virtual machine API operations.
type Service struct {
	client *httpclient.Client
}

// NewService creates a new instance Service.
func NewService(client *httpclient.Client) *Service {
	return &Service{client: client}
}

// List returns virtual machines scoped to region and project. VMs are
// region- and project-specific; empty values are omitted from the filter.
func (s *Service) List(ctx context.Context, region, project string) ([]VirtualMachine, error) {
	// The endpoint is paginated; walk every page so callers (and reference
	// resolution) see the full set rather than just the first page.
	var all []VirtualMachine
	for page := 1; ; page++ {
		q := url.Values{
			"include": {"networks,ipaddresses,offering"},
		}
		if region != "" {
			q.Set("filter[region]", region)
		}
		if project != "" {
			q.Set("filter[project]", project)
		}
		q.Set("page", fmt.Sprintf("%d", page))

		var env Envelope
		if err := s.client.Get(ctx, "/virtual-machines", q, &env); err != nil {
			return nil, fmt.Errorf("listing virtual machines: %w", err)
		}
		if env.Status != "Success" {
			return nil, fmt.Errorf("listing virtual machines: %s", env.Message)
		}
		var vms []VirtualMachine
		if err := json.Unmarshal(env.Data, &vms); err != nil {
			return nil, fmt.Errorf("decoding virtual machines: %w", err)
		}
		all = append(all, vms...)

		// Stop once the server reports the last page, once we've collected the
		// reported total, or when a page comes back empty (defensive guard
		// against a missing/zero total that would otherwise loop forever).
		if len(vms) == 0 {
			break
		}
		if env.LastPage > 0 && env.CurrentPage >= env.LastPage {
			break
		}
		if env.LastPage == 0 && (env.Total == 0 || len(all) >= env.Total) {
			break
		}
	}
	return all, nil
}

// Get returns a single virtual machine by slug.
func (s *Service) Get(ctx context.Context, slug string) (*VirtualMachine, error) {
	var env SingleEnvelope
	if err := s.client.Get(ctx, "/virtual-machines/"+slug, nil, &env); err != nil {
		return nil, fmt.Errorf("getting virtual machine %s: %w", slug, err)
	}
	if env.Status != "Success" {
		return nil, fmt.Errorf("getting virtual machine %s: %s", slug, env.Message)
	}
	var vm VirtualMachine
	if err := json.Unmarshal(env.Data, &vm); err != nil {
		return nil, fmt.Errorf("decoding virtual machine: %w", err)
	}
	return &vm, nil
}

// Create provisions a new virtual machine.
func (s *Service) Create(ctx context.Context, req CreateRequest) (*VirtualMachine, error) {
	var env SingleEnvelope
	if err := s.client.Post(ctx, "/virtual-machines", req, &env); err != nil {
		return nil, fmt.Errorf("creating virtual machine: %w", err)
	}
	if env.Status != "Success" {
		return nil, fmt.Errorf("creating virtual machine: %s", env.Message)
	}
	var vm VirtualMachine
	if err := json.Unmarshal(env.Data, &vm); err != nil {
		return nil, fmt.Errorf("decoding virtual machine: %w", err)
	}
	return &vm, nil
}

// Start starts a stopped virtual machine.
func (s *Service) Start(ctx context.Context, slug string) (*ActionResponse, error) {
	var resp ActionResponse
	if err := s.client.Put(ctx, "/virtual-machines/"+slug+"/start", nil, nil, &resp); err != nil {
		return nil, fmt.Errorf("starting virtual machine %s: %w", slug, err)
	}
	return &resp, nil
}

// Stop stops a running virtual machine.
func (s *Service) Stop(ctx context.Context, slug string) (*ActionResponse, error) {
	var resp ActionResponse
	if err := s.client.Put(ctx, "/virtual-machines/"+slug+"/stop", nil, nil, &resp); err != nil {
		return nil, fmt.Errorf("stopping virtual machine %s: %w", slug, err)
	}
	return &resp, nil
}

// Reboot reboots a virtual machine.
func (s *Service) Reboot(ctx context.Context, slug string) (*ActionResponse, error) {
	var resp ActionResponse
	if err := s.client.Put(ctx, "/virtual-machines/"+slug+"/reboot", nil, nil, &resp); err != nil {
		return nil, fmt.Errorf("rebooting virtual machine %s: %w", slug, err)
	}
	return &resp, nil
}

// Reset resets a virtual machine.
func (s *Service) Reset(ctx context.Context, slug string) (*ActionResponse, error) {
	var resp ActionResponse
	if err := s.client.Put(ctx, "/virtual-machines/"+slug+"/reset", nil, nil, &resp); err != nil {
		return nil, fmt.Errorf("resetting virtual machine %s: %w", slug, err)
	}
	return &resp, nil
}

// ActivityLogs returns activity logs for a virtual machine.
func (s *Service) ActivityLogs(ctx context.Context, slug string) ([]ActivityLog, error) {
	var env Envelope
	if err := s.client.Get(ctx, "/loggers/service/VirtualMachine/"+slug, nil, &env); err != nil {
		return nil, fmt.Errorf("getting activity logs for %s: %w", slug, err)
	}
	var logs []ActivityLog
	if err := json.Unmarshal(env.Data, &logs); err != nil {
		return nil, fmt.Errorf("decoding activity logs: %w", err)
	}
	return logs, nil
}

// CreateTag creates a tag on a virtual machine.
func (s *Service) CreateTag(ctx context.Context, slug string, req TagRequest) (*ActionResponse, error) {
	var resp ActionResponse
	if err := s.client.Post(ctx, "/virtual-machines/"+slug+"/tags", req, &resp); err != nil {
		return nil, fmt.Errorf("creating tag on %s: %w", slug, err)
	}
	return &resp, nil
}

// DeleteTag deletes a tag from a virtual machine.
func (s *Service) DeleteTag(ctx context.Context, slug string, key string) error {
	q := url.Values{"key": {key}}
	if err := s.client.Delete(ctx, "/virtual-machines/"+slug+"/tags", q); err != nil {
		return fmt.Errorf("deleting tag from %s: %w", slug, err)
	}
	return nil
}

// ChangeHostname changes the hostname/label of a virtual machine.
func (s *Service) ChangeHostname(ctx context.Context, slug string, req ChangeLabelRequest) (*ActionResponse, error) {
	var resp ActionResponse
	if err := s.client.Post(ctx, "/virtual-machines/"+slug+"/change-label", req, &resp); err != nil {
		return nil, fmt.Errorf("changing hostname for %s: %w", slug, err)
	}
	return &resp, nil
}

// ChangePassword resets the password of a virtual machine.
func (s *Service) ChangePassword(ctx context.Context, slug string, req ChangePasswordRequest) (*ActionResponse, error) {
	var resp ActionResponse
	if err := s.client.Post(ctx, "/virtual-machines/"+slug+"/change-password", req, &resp); err != nil {
		return nil, fmt.Errorf("changing password for %s: %w", slug, err)
	}
	return &resp, nil
}

// ChangePlan changes the plan of a virtual machine.
func (s *Service) ChangePlan(ctx context.Context, slug string, req ChangePlanRequest) (*ActionResponse, error) {
	var resp ActionResponse
	if err := s.client.Post(ctx, "/virtual-machines/"+slug+"/change-plan", req, &resp); err != nil {
		return nil, fmt.Errorf("changing plan for %s: %w", slug, err)
	}
	return &resp, nil
}

// ChangeOS changes the OS template of a virtual machine.
func (s *Service) ChangeOS(ctx context.Context, slug string, req ChangeTemplateRequest) (*ActionResponse, error) {
	var resp ActionResponse
	if err := s.client.Post(ctx, "/virtual-machines/"+slug+"/change-template", req, &resp); err != nil {
		return nil, fmt.Errorf("changing OS for %s: %w", slug, err)
	}
	return &resp, nil
}

// ChangeStartupScript changes the startup script of a virtual machine.
func (s *Service) ChangeStartupScript(ctx context.Context, slug string, req ChangeStartupScriptRequest) (*ActionResponse, error) {
	var resp ActionResponse
	if err := s.client.Post(ctx, "/virtual-machines/"+slug+"/change-startup-script", req, &resp); err != nil {
		return nil, fmt.Errorf("changing startup script for %s: %w", slug, err)
	}
	return &resp, nil
}

// AddNetwork adds a network to a virtual machine.
func (s *Service) AddNetwork(ctx context.Context, slug string, req AddNetworkRequest) (*ActionResponse, error) {
	var resp ActionResponse
	if err := s.client.Post(ctx, "/virtual-machines/"+slug+"/add-network", req, &resp); err != nil {
		return nil, fmt.Errorf("adding network to %s: %w", slug, err)
	}
	return &resp, nil
}

// ListAddons returns addons for a virtual machine.
func (s *Service) ListAddons(ctx context.Context, slug string) ([]Addon, error) {
	var env Envelope
	if err := s.client.Get(ctx, "/virtual-machines/"+slug+"/addons", nil, &env); err != nil {
		return nil, fmt.Errorf("listing addons for %s: %w", slug, err)
	}
	var addons []Addon
	if err := json.Unmarshal(env.Data, &addons); err != nil {
		return nil, fmt.Errorf("decoding addons: %w", err)
	}
	return addons, nil
}

// PurchaseAddon purchases an addon for a virtual machine.
func (s *Service) PurchaseAddon(ctx context.Context, req PurchaseAddonRequest) (*ActionResponse, error) {
	var resp ActionResponse
	if err := s.client.Post(ctx, "/virtual-machines/addons", req, &resp); err != nil {
		return nil, fmt.Errorf("purchasing addon: %w", err)
	}
	return &resp, nil
}

// Delete performs a direct destroy of a virtual machine via DELETE /virtual-machines/{slug}.
// If expunge is true, ?expunge=true is sent to force immediate purge from the hypervisor
// rather than leaving the VM in a soft-deleted/Destroyed state pending CloudStack expunge.
//
// NOTE: this endpoint does NOT release the VM's auto-assigned public IP — the delete_public_ip
// query param is accepted but ignored by the CMP, leaving the IP Allocated/billable. To delete
// a VM AND release its public IP (as the CMP Web UI does), use the service-cancellation workflow
// instead: billing.Service.CancelService with delete_public_ip set (see 'zcp instance delete').
func (s *Service) Delete(ctx context.Context, slug string, expunge, deletePublicIP bool) error {
	q := url.Values{}
	if expunge {
		q.Set("expunge", "true")
	}
	if deletePublicIP {
		q.Set("delete_public_ip", "true")
	}
	if len(q) == 0 {
		q = nil
	}
	if err := s.client.Delete(ctx, "/virtual-machines/"+slug, q); err != nil {
		return fmt.Errorf("deleting virtual machine %s: %w", slug, err)
	}
	return nil
}

// VMMeta is the live, hypervisor-synced view of a VM from GET /virtual-machines/{slug}/meta.
// This endpoint performs a real-time reconcile against the underlying platform (CloudStack/APC)
// and updates the stored state before returning, so it reports the true state even when the
// cached list/show endpoints lag — a Running VM can otherwise stay "Starting" in list/show
// until the platform's own reconciliation catches up.
type VMMeta struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	State string `json:"state"`
}

// Meta returns the live, hypervisor-synced view of a VM (see VMMeta). Prefer this over Get when
// you need the authoritative current lifecycle state (for example when polling for Running),
// because Get/List can report a stale state for several minutes after a state change.
func (s *Service) Meta(ctx context.Context, slug string) (*VMMeta, error) {
	var env SingleEnvelope
	if err := s.client.Get(ctx, "/virtual-machines/"+slug+"/meta", nil, &env); err != nil {
		return nil, fmt.Errorf("getting virtual machine meta %s: %w", slug, err)
	}
	if env.Status != "Success" {
		return nil, fmt.Errorf("getting virtual machine meta %s: %s", slug, env.Message)
	}
	var meta VMMeta
	if err := json.Unmarshal(env.Data, &meta); err != nil {
		return nil, fmt.Errorf("decoding virtual machine meta: %w", err)
	}
	return &meta, nil
}

// WaitForState polls the VM until it reaches one of the target states or the context is cancelled.
//
// It polls the meta endpoint rather than Get: /meta forces a live reconcile with the hypervisor
// and reports the true state, whereas the cached Get/List can keep reporting "Starting" long
// after the VM is actually Running (the CMP's background reconciliation is unreliable). Calling
// meta also reconciles the stored state, so the follow-up Get returns a consistent object.
func (s *Service) WaitForState(ctx context.Context, slug string, targetStates []string, pollInterval time.Duration) (*VirtualMachine, error) {
	if pollInterval == 0 {
		pollInterval = 5 * time.Second
	}
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-ticker.C:
			meta, err := s.Meta(ctx, slug)
			if err != nil {
				return nil, err
			}
			for _, target := range targetStates {
				if strings.EqualFold(meta.State, target) {
					vm, err := s.Get(ctx, slug)
					if err != nil {
						return nil, err
					}
					// meta is authoritative; surface it even if Get is momentarily stale.
					vm.State = meta.State
					return vm, nil
				}
			}
		}
	}
}

// StringVal safely dereferences a string pointer for display.
func StringVal(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
