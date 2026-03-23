// Package kubernetes provides ZCP Kubernetes cluster API operations.
package kubernetes

import (
	"context"
	"fmt"
	"net/url"
	"strconv"

	"github.com/zsoftly/zcp-cli/internal/httpclient"
)

// Cluster represents a ZCP managed Kubernetes cluster.
type Cluster struct {
	UUID                   string `json:"uuid"`
	Name                   string `json:"name"`
	Description            string `json:"description"`
	State                  string `json:"state"`
	Size                   int    `json:"size"`
	ControlNodes           int    `json:"controlNodes"`
	NodeRootDiskSize       int    `json:"nodeRootDiskSize"`
	TransNetworkUUID       string `json:"transNetworkUuid"`
	ExternalLoadbalancerIP string `json:"externalLoadbalancerIpaddress"`
}

// Node represents a Kubernetes node (uses the instance response shape).
type Node struct {
	UUID      string `json:"uuid"`
	Name      string `json:"name"`
	State     string `json:"state"`
	Memory    string `json:"memory"`
	PrivateIP string `json:"instancePrivateIp"`
	ZoneUUID  string `json:"zoneUuid"`
	IsActive  bool   `json:"isActive"`
}

// Version represents a supported Kubernetes version.
type Version struct {
	UUID         string `json:"uuid"`
	Name         string `json:"name"`
	Description  string `json:"description"`
	IsActive     bool   `json:"isActive"`
	MinMemory    string `json:"minMemory"`
	MinCPUNumber string `json:"minCpuNumber"`
}

// CreateRequest holds parameters for creating a Kubernetes cluster.
type CreateRequest struct {
	Name                   string `json:"name"`
	ZoneUUID               string `json:"zoneUuid"`
	VersionUUID            string `json:"kubernetesSupportedVersionUuid"`
	ComputeOfferingUUID    string `json:"computeOfferingUuid"`
	TransNetworkUUID       string `json:"transNetworkUuid"`
	Size                   int64  `json:"size"`
	ControlNodes           int64  `json:"controlNodes"`
	SSHKeyName             string `json:"sshKeyName"`
	HAEnabled              bool   `json:"haEnabled"`
	NodeRootDiskSize       int64  `json:"nodeRootDiskSize,omitempty"`
	Description            string `json:"description,omitempty"`
	ExternalLoadbalancerIP string `json:"externalLoadbalancerIpaddress,omitempty"`
	DockerRegistryURL      string `json:"dockerRegistryUrl,omitempty"`
	DockerRegistryUsername string `json:"dockerRegistryUsername,omitempty"`
	DockerRegistryPassword string `json:"dockerRegistryPassword,omitempty"`
	DomainUUID             string `json:"domainUuid,omitempty"`
}

type listNodesResponse struct {
	Count                int    `json:"count"`
	ListInstanceResponse []Node `json:"listInstanceResponse"`
}

type listVersionsResponse struct {
	Count                 int       `json:"count"`
	ListKubernetesVersion []Version `json:"listKubernetesVersion"`
}

// Service provides Kubernetes API operations.
type Service struct {
	client *httpclient.Client
}

// NewService creates a new Kubernetes Service.
func NewService(client *httpclient.Client) *Service {
	return &Service{client: client}
}

// List returns clusters. clusterUUID is an optional filter.
// NOTE: The API returns a single cluster object (not a list), so we wrap it.
func (s *Service) List(ctx context.Context, clusterUUID string) ([]Cluster, error) {
	q := url.Values{}
	if clusterUUID != "" {
		q.Set("clusterUuid", clusterUUID)
	}
	// API returns a single cluster object, not an array
	var cluster Cluster
	if err := s.client.Get(ctx, "/restapi/kubernetes/listCluster", q, &cluster); err != nil {
		return nil, fmt.Errorf("listing kubernetes clusters: %w", err)
	}
	if cluster.UUID == "" {
		return []Cluster{}, nil
	}
	return []Cluster{cluster}, nil
}

// Create provisions a new Kubernetes cluster.
func (s *Service) Create(ctx context.Context, req CreateRequest) (*Cluster, error) {
	var cluster Cluster
	if err := s.client.Post(ctx, "/restapi/kubernetes/createKubernetes", req, &cluster); err != nil {
		return nil, fmt.Errorf("creating kubernetes cluster: %w", err)
	}
	return &cluster, nil
}

// Delete destroys a Kubernetes cluster.
func (s *Service) Delete(ctx context.Context, uuid string) error {
	q := url.Values{"uuid": {uuid}}
	if err := s.client.Delete(ctx, "/restapi/kubernetes/destroyKubernetes", q); err != nil {
		return fmt.Errorf("destroying kubernetes cluster %s: %w", uuid, err)
	}
	return nil
}

// Start starts a stopped Kubernetes cluster.
func (s *Service) Start(ctx context.Context, uuid string) (*Cluster, error) {
	q := url.Values{"uuid": {uuid}}
	var cluster Cluster
	if err := s.client.Put(ctx, "/restapi/kubernetes/startKubernetes", q, nil, &cluster); err != nil {
		return nil, fmt.Errorf("starting kubernetes cluster %s: %w", uuid, err)
	}
	return &cluster, nil
}

// Stop stops a running Kubernetes cluster.
func (s *Service) Stop(ctx context.Context, uuid string) (*Cluster, error) {
	q := url.Values{"uuid": {uuid}}
	var cluster Cluster
	if err := s.client.Put(ctx, "/restapi/kubernetes/stopKubernetes", q, nil, &cluster); err != nil {
		return nil, fmt.Errorf("stopping kubernetes cluster %s: %w", uuid, err)
	}
	return &cluster, nil
}

// Scale changes the worker node count for a cluster.
func (s *Service) Scale(ctx context.Context, uuid string, size int, autoscaling bool) (*Cluster, error) {
	q := url.Values{
		"uuid": {uuid},
		"size": {strconv.Itoa(size)},
	}
	if autoscaling {
		q.Set("autoscalingEnabled", "true")
	}
	var cluster Cluster
	if err := s.client.Put(ctx, "/restapi/kubernetes/scaleKubernetes", q, nil, &cluster); err != nil {
		return nil, fmt.Errorf("scaling kubernetes cluster %s: %w", uuid, err)
	}
	return &cluster, nil
}

// ListNodes returns the nodes in a Kubernetes cluster.
func (s *Service) ListNodes(ctx context.Context, clusterUUID string) ([]Node, error) {
	q := url.Values{"clusterUuid": {clusterUUID}}
	var resp listNodesResponse
	if err := s.client.Get(ctx, "/restapi/kubernetes/listNodes", q, &resp); err != nil {
		return nil, fmt.Errorf("listing nodes for cluster %s: %w", clusterUUID, err)
	}
	return resp.ListInstanceResponse, nil
}

// ListVersions returns available Kubernetes versions for a zone.
func (s *Service) ListVersions(ctx context.Context, zoneUUID string) ([]Version, error) {
	q := url.Values{"zoneUuid": {zoneUUID}}
	var resp listVersionsResponse
	if err := s.client.Get(ctx, "/restapi/costestimate/kubernetes-version-list", q, &resp); err != nil {
		return nil, fmt.Errorf("listing kubernetes versions: %w", err)
	}
	return resp.ListKubernetesVersion, nil
}
