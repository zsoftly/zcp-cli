// Package kubernetes provides ZCP Kubernetes cluster API operations.
package kubernetes

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/zsoftly/zcp-cli/internal/httpclient"
)

// Cluster represents a ZCP managed Kubernetes cluster from the STKCNSL API.
type Cluster struct {
	ID                   string          `json:"id"`
	Name                 string          `json:"name"`
	Slug                 string          `json:"slug"`
	Description          *string         `json:"description"`
	State                string          `json:"state"`
	UserID               string          `json:"user_id"`
	AccountID            string          `json:"account_id"`
	ProjectID            string          `json:"project_id"`
	RegionID             string          `json:"region_id"`
	CloudProviderID      string          `json:"cloud_provider_id"`
	CloudProviderSetupID string          `json:"cloud_provider_setup_id"`
	RequestStatus        bool            `json:"request_status"`
	Hostname             string          `json:"hostname"`
	PublicIP             *string         `json:"public_ip"`
	PrivateIP            *string         `json:"private_ip"`
	NodeSize             int             `json:"node_size"`
	ControlNodes         int             `json:"control_nodes"`
	Version              string          `json:"version"`
	EnableHA             bool            `json:"enable_ha"`
	CustomPlan           json.RawMessage `json:"custom_plan"`
	FrozenAt             *string         `json:"frozen_at"`
	SuspendedAt          *string         `json:"suspended_at"`
	TerminatedAt         *string         `json:"terminated_at"`
	CreatedAt            string          `json:"created_at"`
	UpdatedAt            string          `json:"updated_at"`
	DeletedAt            *string         `json:"deleted_at"`
	BillingCycle         *BillingCycle   `json:"billing_cycle,omitempty"`
	Region               *Region         `json:"region,omitempty"`
	Project              *Project        `json:"project,omitempty"`
	ServiceName          string          `json:"service_name"`
	ServiceDisplayName   string          `json:"service_display_name"`
	AllTimeConsumption   float64         `json:"all_time_consumption"`
}

// BillingCycle represents the billing cycle attached to a cluster.
type BillingCycle struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Slug     string `json:"slug"`
	Duration int    `json:"duration"`
	Unit     string `json:"unit"`
}

// Region represents the region attached to a cluster.
type Region struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Slug    string `json:"slug"`
	Country string `json:"country"`
}

// Project represents the project attached to a cluster.
type Project struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Slug string `json:"slug"`
}

// CreateRequest holds parameters for creating a Kubernetes cluster.
type CreateRequest struct {
	Name            string      `json:"name"`
	Version         string      `json:"version"`
	NodeSize        int         `json:"node_size"`
	ControlNodes    int         `json:"control_nodes"`
	CloudProvider   string      `json:"cloud_provider"`
	Region          string      `json:"region"`
	Project         string      `json:"project"`
	BillingCycle    string      `json:"billing_cycle"`
	EnableHA        bool        `json:"enable_ha"`
	Networks        []string    `json:"networks"`
	Plan            string      `json:"plan"`
	WithPoolCard    bool        `json:"with_pool_card"`
	IsCustomPlan    bool        `json:"is_custom_plan"`
	CustomPlan      interface{} `json:"custom_plan"`
	VirtualMachine  string      `json:"virtual_machine"`
	Coupon          *string     `json:"coupon"`
	StorageCategory string      `json:"storage_category"`
	SSHKey          string      `json:"ssh_key"`
	AuthMethod      string      `json:"authMethod"`
	Username        string      `json:"username"`
	Password        string      `json:"password"`
}

// UpgradeRequest holds parameters for upgrading (changing plan of) a Kubernetes cluster.
type UpgradeRequest struct {
	Plan         string      `json:"plan"`
	Slug         string      `json:"slug"`
	BillingCycle string      `json:"billing_cycle"`
	IsCustomPlan bool        `json:"is_custom_plan"`
	CustomPlan   interface{} `json:"custom_plan"`
}

// listResponse is the STKCNSL response envelope for paginated lists.
type listResponse struct {
	Status      string    `json:"status"`
	Message     string    `json:"message"`
	CurrentPage int       `json:"current_page"`
	Data        []Cluster `json:"data"`
	LastPage    int       `json:"last_page"`
	PerPage     int       `json:"per_page"`
	Total       int       `json:"total"`
}

// singleResponse is the STKCNSL response envelope for single-object responses.
type singleResponse struct {
	Status  string  `json:"status"`
	Message string  `json:"message"`
	Data    Cluster `json:"data"`
}

// messageResponse is the STKCNSL response envelope for action responses (start/stop/upgrade).
type messageResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

// Service provides Kubernetes API operations.
type Service struct {
	client *httpclient.Client
}

// NewService creates a new Kubernetes Service.
func NewService(client *httpclient.Client) *Service {
	return &Service{client: client}
}

// List returns all Kubernetes clusters.
func (s *Service) List(ctx context.Context) ([]Cluster, error) {
	var resp listResponse
	if err := s.client.Get(ctx, "/kubernetes-clusters", nil, &resp); err != nil {
		return nil, fmt.Errorf("listing kubernetes clusters: %w", err)
	}
	return resp.Data, nil
}

// Create provisions a new Kubernetes cluster.
func (s *Service) Create(ctx context.Context, req CreateRequest) (*Cluster, error) {
	var resp singleResponse
	if err := s.client.Post(ctx, "/kubernetes-clusters", req, &resp); err != nil {
		return nil, fmt.Errorf("creating kubernetes cluster: %w", err)
	}
	return &resp.Data, nil
}

// Start starts a stopped Kubernetes cluster.
func (s *Service) Start(ctx context.Context, slug string) error {
	var resp messageResponse
	if err := s.client.Put(ctx, fmt.Sprintf("/kubernetes-clusters/%s/start", slug), nil, nil, &resp); err != nil {
		return fmt.Errorf("starting kubernetes cluster %s: %w", slug, err)
	}
	return nil
}

// Stop stops a running Kubernetes cluster.
func (s *Service) Stop(ctx context.Context, slug string) error {
	var resp messageResponse
	if err := s.client.Put(ctx, fmt.Sprintf("/kubernetes-clusters/%s/stop", slug), nil, nil, &resp); err != nil {
		return fmt.Errorf("stopping kubernetes cluster %s: %w", slug, err)
	}
	return nil
}

// Upgrade changes the plan for a Kubernetes cluster.
func (s *Service) Upgrade(ctx context.Context, slug string, req UpgradeRequest) error {
	var resp messageResponse
	if err := s.client.Put(ctx, fmt.Sprintf("/kubernetes-clusters/%s/change-plan", slug), nil, req, &resp); err != nil {
		return fmt.Errorf("upgrading kubernetes cluster %s: %w", slug, err)
	}
	return nil
}
