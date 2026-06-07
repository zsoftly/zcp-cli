// Package kubernetes provides ZCP Kubernetes cluster API operations.
package kubernetes

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/zsoftly/zcp-cli/internal/httpclient"
)

// KubeconfigData is the nested object inside ClusterMeta.Config.
type KubeconfigData struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	ConfigData string `json:"configdata"`
}

// ClusterMeta holds the CloudStack-side details embedded in the API response.
type ClusterMeta struct {
	ControlNodes          string          `json:"control_nodes"`
	Size                  string          `json:"size"`
	KubernetesVersionName string          `json:"kubernetes_version_name"`
	IPAddress             string          `json:"ipaddress"`
	Endpoint              string          `json:"end_point"`
	State                 string          `json:"state"`
	Zone                  string          `json:"zone_name"`
	Config                *KubeconfigData `json:"config,omitempty"`
}

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
	Meta                 *ClusterMeta    `json:"meta,omitempty"`
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
	Name               string      `json:"name"`
	Version            string      `json:"version"`
	NodeSize           int         `json:"node_size"`
	WorkerNodeSize     int         `json:"worker_node_size"`
	ControlNodes       int         `json:"control_nodes"`
	CloudProvider      string      `json:"cloud_provider"`
	CloudProviderSetup string      `json:"cloud_provider_setup,omitempty"`
	Region             string      `json:"region"`
	Project            string      `json:"project"`
	BillingCycle       string      `json:"billing_cycle"`
	EnableHA           bool        `json:"enable_ha"`
	Networks           []string    `json:"networks"`
	Plan               string      `json:"plan"`
	WithPoolCard       bool        `json:"with_pool_card"`
	IsCustomPlan       bool        `json:"is_custom_plan"`
	CustomPlan         interface{} `json:"custom_plan"`
	VirtualMachine     string      `json:"virtual_machine"`
	Coupon             *string     `json:"coupon"`
	StorageCategory    string      `json:"storage_category"`
	SSHKey             string      `json:"ssh_key"`
	AuthMethod         string      `json:"authMethod"`
	Username           string      `json:"username"`
	Password           string      `json:"password"`
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

// Get returns a single Kubernetes cluster by slug.
func (s *Service) Get(ctx context.Context, slug string) (*Cluster, error) {
	var resp singleResponse
	if err := s.client.Get(ctx, "/kubernetes-clusters/"+slug, nil, &resp); err != nil {
		return nil, fmt.Errorf("getting kubernetes cluster %s: %w", slug, err)
	}
	return &resp.Data, nil
}

// ScaleRequest holds the worker node count for a scale operation.
type ScaleRequest struct {
	NodeSize int `json:"node_size"`
}

// Scale changes the number of worker nodes on a running cluster.
func (s *Service) Scale(ctx context.Context, slug string, nodeSize int) error {
	var resp messageResponse
	if err := s.client.Put(ctx, fmt.Sprintf("/kubernetes-clusters/%s/scale", slug), nil, ScaleRequest{NodeSize: nodeSize}, &resp); err != nil {
		return fmt.Errorf("scaling kubernetes cluster %s: %w", slug, err)
	}
	return nil
}

// GetKubeconfig returns the raw kubeconfig YAML for a cluster, or "" if not yet available.
func (s *Service) GetKubeconfig(ctx context.Context, slug string) (string, error) {
	var resp singleResponse
	if err := s.client.Get(ctx, "/kubernetes-clusters/"+slug, nil, &resp); err != nil {
		return "", fmt.Errorf("getting kubernetes cluster %s: %w", slug, err)
	}
	if resp.Data.Meta == nil || resp.Data.Meta.Config == nil {
		return "", fmt.Errorf("kubeconfig not available yet for cluster %s (state: %s)", slug, resp.Data.State)
	}
	return resp.Data.Meta.Config.ConfigData, nil
}

// Delete permanently deletes a Kubernetes cluster.
func (s *Service) Delete(ctx context.Context, slug string) error {
	if err := s.client.Delete(ctx, "/kubernetes-clusters/"+slug, nil); err != nil {
		return fmt.Errorf("deleting kubernetes cluster %s: %w", slug, err)
	}
	return nil
}
