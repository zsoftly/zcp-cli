// Package kubernetes provides ZCP Kubernetes cluster API operations.
package kubernetes

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/zsoftly/zcp-cli/pkg/httpclient"
)

// KubeconfigData is the nested object inside ClusterMeta.Config.
type KubeconfigData struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	ConfigData string `json:"configdata"`
}

// ClusterMeta holds the platform-side details embedded in the API response.
type ClusterMeta struct {
	ControlNodes          string          `json:"control_nodes"`
	Size                  string          `json:"size"`
	CPU                   string          `json:"cpu_number"`
	Memory                string          `json:"memory"`
	FormattedMemory       string          `json:"formated_memory"`
	AutoscalingEnabled    json.RawMessage `json:"autoscaling_enabled"`
	MinSize               json.RawMessage `json:"min_size"`
	MaxSize               json.RawMessage `json:"max_size"`
	KubernetesVersionName string          `json:"kubernetes_version_name"`
	KubernetesVersionID   string          `json:"kubernetes_version_id"`
	IPAddress             string          `json:"ipaddress"`
	Endpoint              string          `json:"end_point"`
	State                 string          `json:"state"`
	Zone                  string          `json:"zone_name"`
	Config                *KubeconfigData `json:"config,omitempty"`
}

// PlanAttribute describes the resources assigned by a Kubernetes node plan.
type PlanAttribute struct {
	FormattedCPU     json.RawMessage `json:"formatted_cpu"`
	CPU              json.RawMessage `json:"cpu"`
	FormattedMemory  string          `json:"formatted_memory"`
	Memory           json.RawMessage `json:"memory"`
	FormattedStorage string          `json:"formatted_storage"`
	Storage          json.RawMessage `json:"storage"`
}

// NodePlan describes a Kubernetes control-plane or worker plan returned with a cluster.
type NodePlan struct {
	Name      string         `json:"name"`
	Slug      string         `json:"slug"`
	Attribute *PlanAttribute `json:"attribute"`
}

// Offering describes the plans and custom resource values used by a Kubernetes cluster.
type Offering struct {
	BillingCycle                 *BillingCycle   `json:"billing_cycle,omitempty"`
	CPU                          json.RawMessage `json:"cpu"`
	Memory                       json.RawMessage `json:"memory"`
	Storage                      json.RawMessage `json:"storage"`
	FormattedMemory              string          `json:"formatted_memory"`
	FormattedStorage             string          `json:"formatted_storage"`
	MasterPlan                   *NodePlan       `json:"master_plan"`
	WorkerPlan                   *NodePlan       `json:"worker_plan"`
	MasterCustomCPU              json.RawMessage `json:"master_custom_cpu"`
	MasterCustomMemory           json.RawMessage `json:"master_custom_memory"`
	MasterCustomStorage          json.RawMessage `json:"master_custom_storage"`
	FormattedMasterCustomMemory  string          `json:"formatted_master_custom_memory"`
	FormattedMasterCustomStorage string          `json:"formatted_master_custom_storage"`
	WorkerCustomCPU              json.RawMessage `json:"worker_custom_cpu"`
	WorkerCustomMemory           json.RawMessage `json:"worker_custom_memory"`
	WorkerCustomStorage          json.RawMessage `json:"worker_custom_storage"`
	FormattedWorkerCustomMemory  string          `json:"formatted_worker_custom_memory"`
	FormattedWorkerCustomStorage string          `json:"formatted_worker_custom_storage"`
}

// Network holds the safe network identity returned with a cluster.
type Network struct {
	Name string `json:"name"`
	Slug string `json:"slug"`
}

// BlockStorage holds a volume identity and capacity returned with a cluster.
type BlockStorage struct {
	Name   string          `json:"name"`
	Slug   string          `json:"slug"`
	IsRoot bool            `json:"is_root"`
	Size   json.RawMessage `json:"size"`
}

// Autoscale marks a cluster with autoscaling enabled in the portal response.
type Autoscale struct{}

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
	WorkerNodeSize       int             `json:"worker_node_size"`
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
	Offering             *Offering       `json:"offering,omitempty"`
	Network              *Network        `json:"network,omitempty"`
	BlockStorages        []BlockStorage  `json:"blockstorages,omitempty"`
	Autoscale            *Autoscale      `json:"autoscale,omitempty"`
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
	Name               string   `json:"name"`
	Version            string   `json:"version"`
	NodeSize           int      `json:"node_size"`
	WorkerNodeSize     int      `json:"worker_node_size"`
	ControlNodes       int      `json:"control_nodes"`
	CloudProvider      string   `json:"cloud_provider"`
	CloudProviderSetup string   `json:"cloud_provider_setup,omitempty"`
	Region             string   `json:"region"`
	Project            string   `json:"project"`
	BillingCycle       string   `json:"billing_cycle"`
	EnableHA           bool     `json:"enable_ha"`
	EnableCSI          bool     `json:"enable_csi"`
	Networks           []string `json:"networks"`
	// Plan is retained for API compatibility with older Kubernetes cluster
	// creation flows. ZCP Kubernetes clusters use MasterPlan and
	// WorkerPlan instead.
	Plan                   string                  `json:"plan,omitempty"`
	MasterPlan             string                  `json:"master_plan,omitempty"`
	WorkerPlan             string                  `json:"worker_plan,omitempty"`
	BlockstoragePlan       string                  `json:"blockstorage_plan,omitempty"`
	BlockstorageCustomPlan *BlockstorageCustomPlan `json:"blockstorage_custom_plan,omitempty"`
	IsK8sCustomPlan        *bool                   `json:"is_k8s_custom_plan,omitempty"`
	MasterCustomPlan       json.RawMessage         `json:"master_custom_plan,omitempty"`
	IsK8sMasterCustomPlan  *bool                   `json:"is_k8s_master_custom_plan,omitempty"`
	WorkerCustomPlan       json.RawMessage         `json:"worker_custom_plan,omitempty"`
	IsK8sWorkerCustomPlan  *bool                   `json:"is_k8s_worker_custom_plan,omitempty"`
	WithPoolCard           bool                    `json:"with_pool_card"`
	IsCustomPlan           bool                    `json:"is_custom_plan"`
	CustomPlan             interface{}             `json:"custom_plan"`
	VirtualMachine         string                  `json:"virtual_machine"`
	Coupon                 *string                 `json:"coupon"`
	StorageCategory        string                  `json:"storage_category"`
	SSHKey                 string                  `json:"ssh_key"`
	AuthMethod             string                  `json:"authMethod"`
	Username               string                  `json:"username"`
	Password               string                  `json:"password"`
}

// BlockstorageCustomPlan holds the root-volume capacity in GB for a named storage tier.
type BlockstorageCustomPlan struct {
	Storage int `json:"storage"`
}

// UpgradeRequest holds parameters for upgrading (changing plan of) a Kubernetes cluster.
type UpgradeRequest struct {
	Plan         string      `json:"plan"`
	Slug         string      `json:"slug"`
	BillingCycle string      `json:"billing_cycle"`
	IsCustomPlan bool        `json:"is_custom_plan"`
	CustomPlan   interface{} `json:"custom_plan"`
}

// UpgradeVersionRequest holds the version slug for a Kubernetes version upgrade.
type UpgradeVersionRequest struct {
	Slug string `json:"slug"`
}

// KubernetesVersion represents an available Kubernetes version from the CMP catalog.
type KubernetesVersion struct {
	ID                         string `json:"id"`
	KubernetesClusterVersionID string `json:"kubernetes_cluster_version_id"`
	Name                       string `json:"name"`
	Slug                       string `json:"slug"`
	Version                    string `json:"version"`
	RegionID                   string `json:"region_id"`
}

// versionsListResponse is the STKCNSL response envelope for the versions list.
type versionsListResponse struct {
	Status      string              `json:"status"`
	Message     string              `json:"message"`
	CurrentPage int                 `json:"current_page"`
	Data        []KubernetesVersion `json:"data"`
	LastPage    int                 `json:"last_page"`
	PerPage     int                 `json:"per_page"`
	Total       int                 `json:"total"`
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
func (s *Service) List(ctx context.Context, region, project string) ([]Cluster, error) {
	var resp listResponse
	q := url.Values{}
	if region != "" {
		q.Set("filter[region]", region)
	}
	if project != "" {
		q.Set("filter[project]", project)
	}
	if err := s.client.Get(ctx, "/kubernetes-clusters", q, &resp); err != nil {
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

// UpgradeVersion upgrades the Kubernetes version of a cluster.
func (s *Service) UpgradeVersion(ctx context.Context, clusterSlug string, req UpgradeVersionRequest) error {
	var resp messageResponse
	if err := s.client.Post(ctx, fmt.Sprintf("/kubernetes-clusters/%s/version", clusterSlug), req, &resp); err != nil {
		return fmt.Errorf("upgrading kubernetes version for cluster %s: %w", clusterSlug, err)
	}
	return nil
}

// ListVersions returns all available Kubernetes versions from the CMP catalog.
func (s *Service) ListVersions(ctx context.Context) ([]KubernetesVersion, error) {
	var resp versionsListResponse
	if err := s.client.Get(ctx, "/kubernetes-clusters/versions", nil, &resp); err != nil {
		return nil, fmt.Errorf("listing kubernetes versions: %w", err)
	}
	return resp.Data, nil
}

// Get returns a single Kubernetes cluster by slug.
func (s *Service) Get(ctx context.Context, slug string) (*Cluster, error) {
	var resp singleResponse
	if err := s.client.Get(ctx, "/kubernetes-clusters/"+slug, nil, &resp); err != nil {
		return nil, fmt.Errorf("getting kubernetes cluster %s: %w", slug, err)
	}
	return &resp.Data, nil
}

// ScaleRequest holds the worker node count for a manual scale operation.
type ScaleRequest struct {
	NodeSize int `json:"node_size"`
}

type autoscaleRequest struct {
	NodeSize       *int `json:"node_size,omitempty"`
	Autoscale      *int `json:"autoscale,omitempty"`
	MinClusterSize *int `json:"min_cluster_size,omitempty"`
	MaxClusterSize *int `json:"max_cluster_size,omitempty"`
}

// Scale changes the number of worker nodes on a running cluster.
func (s *Service) Scale(ctx context.Context, slug string, nodeSize int) error {
	return s.scale(ctx, slug, ScaleRequest{NodeSize: nodeSize})
}

// EnableAutoscaling enables worker autoscaling with the supplied bounds.
func (s *Service) EnableAutoscaling(ctx context.Context, slug string, minWorkers, maxWorkers int) error {
	autoscale := 1
	return s.scale(ctx, slug, autoscaleRequest{
		Autoscale:      &autoscale,
		MinClusterSize: &minWorkers,
		MaxClusterSize: &maxWorkers,
	})
}

// DisableAutoscaling disables worker autoscaling and sets the worker count.
func (s *Service) DisableAutoscaling(ctx context.Context, slug string, workers int) error {
	autoscale := 0
	return s.scale(ctx, slug, autoscaleRequest{NodeSize: &workers, Autoscale: &autoscale})
}

func (s *Service) scale(ctx context.Context, slug string, req interface{}) error {
	var resp messageResponse
	if err := s.client.Put(ctx, fmt.Sprintf("/kubernetes-clusters/%s/scale", slug), nil, req, &resp); err != nil {
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
