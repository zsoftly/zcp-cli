// Package plan provides STKCNSL service plan listing operations.
package plan

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/zsoftly/zcp-cli/internal/httpclient"
)

// ServiceType identifies a STKCNSL service for plan lookups.
type ServiceType string

const (
	ServiceVM            ServiceType = "Virtual Machine"
	ServiceVirtualRouter ServiceType = "Virtual Router"
	ServiceBlockStorage  ServiceType = "Block Storage"
	ServiceLoadBalancer  ServiceType = "Load Balancer"
	ServiceKubernetes    ServiceType = "Kubernetes"
	ServiceIPAddress     ServiceType = "IP Address"
	ServiceVMSnapshot    ServiceType = "VM Snapshot"
	ServiceMyTemplate    ServiceType = "My Template"
	ServiceISO           ServiceType = "ISO"
	ServiceBackups       ServiceType = "Backups"
)

// Attribute holds the resource attributes embedded in a plan.
// Fields are decoded as json.RawMessage because shapes differ across service
// types; the typed helpers below extract what the CLI actually needs.
type Attribute struct {
	CPU                 json.Number `json:"cpu"`
	Memory              json.Number `json:"memory"`
	Storage             json.Number `json:"storage"`
	Size                json.Number `json:"size"`
	MemoryUnit          string      `json:"memory_unit"`
	StorageUnit         string      `json:"storage_unit"`
	FormattedMemory     string      `json:"formatted_memory"`
	FormattedStorage    string      `json:"formatted_storage"`
	FormattedSize       string      `json:"formatted_size"`
	FormattedCPU        json.Number `json:"formatted_cpu"`
	ComputeOfferingID   string      `json:"compute_offering_id"`
	DiskOfferingID      string      `json:"disk_offering_id"`
	NetworkRate         string      `json:"network_rate"`
	VPCOfferingID       string      `json:"vpc_offering_id"`
	FormattedMemoryUnit string      `json:"formatted_memory_unit"`
	FormattedSizeUnit   string      `json:"formatted_size_unit"`
}

// Tag holds optional marketing label data.
type Tag struct {
	Label string `json:"label"`
	Value string `json:"value"`
	Color string `json:"color"`
}

// BillingCycle represents a billing cycle (hourly, monthly, etc.).
type BillingCycle struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Slug      string `json:"slug"`
	Duration  int    `json:"duration"`
	Unit      string `json:"unit"`
	IsEnabled bool   `json:"is_enabled"`
	SortOrder int    `json:"sort_order"`
}

// Currency represents a currency.
type Currency struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Slug         string `json:"slug"`
	CurrencyName string `json:"currency_name"`
}

// Price represents a single price entry for a plan.
type Price struct {
	ID           string       `json:"id"`
	Amount       string       `json:"amount"`
	OTC          string       `json:"otc"`
	Currency     Currency     `json:"currency"`
	BillingCycle BillingCycle `json:"billing_cycle"`
}

// Plan represents a STKCNSL service plan.
type Plan struct {
	ID           string          `json:"id"`
	Name         string          `json:"name"`
	Slug         string          `json:"slug"`
	Attribute    Attribute       `json:"attribute"`
	Tag          json.RawMessage `json:"tag"` // can be object or empty array
	Status       bool            `json:"status"`
	IsCustom     bool            `json:"is_custom"`
	HourlyPrice  float64         `json:"hourly_price"`
	MonthlyPrice float64         `json:"monthly_price"`
	Prices       []Price         `json:"prices"`
	CreatedAt    string          `json:"created_at"`
	UpdatedAt    string          `json:"updated_at"`
}

// ParsedTag returns the tag label if present, or "-" if the tag field is an
// empty array or missing.
func (p *Plan) ParsedTag() string {
	if len(p.Tag) == 0 {
		return "-"
	}
	// Try object first
	var t Tag
	if err := json.Unmarshal(p.Tag, &t); err == nil && t.Label != "" {
		return t.Label
	}
	return "-"
}

// listResponse is the STKCNSL API envelope for plan list endpoints.
type listResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
	Data    []Plan `json:"data"`
}

// Service provides plan API operations.
type Service struct {
	client *httpclient.Client
}

// NewService creates a new plan Service.
func NewService(client *httpclient.Client) *Service {
	return &Service{client: client}
}

// List returns plans for the given service type.
func (s *Service) List(ctx context.Context, svc ServiceType) ([]Plan, error) {
	path := "/plans/service/" + url.PathEscape(string(svc))
	var resp listResponse
	if err := s.client.Get(ctx, path, nil, &resp); err != nil {
		return nil, fmt.Errorf("listing %s plans: %w", svc, err)
	}
	return resp.Data, nil
}
