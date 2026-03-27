// Package instance provides ZCP instance (VM) API operations.
package instance

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/zsoftly/zcp-cli/internal/httpclient"
)

// Instance represents a ZCP virtual machine.
type Instance struct {
	UUID                string `json:"uuid"`
	Name                string `json:"name"`
	DisplayName         string `json:"displayName"`
	Description         string `json:"description"`
	State               string `json:"state"`
	IsActive            bool   `json:"isActive"`
	Memory              string `json:"memory"`
	TemplateName        string `json:"templateName"`
	TemplateUUID        string `json:"templateUuid"`
	ComputeOfferingUUID string `json:"computeOfferingUuid"`
	StorageOfferingUUID string `json:"storageOfferingUuid"`
	NetworkName         string `json:"networkName"`
	NetworkUUID         string `json:"networkUuid"`
	PrivateIP           string `json:"instancePrivateIp"`
	ZoneUUID            string `json:"zoneUuid"`
	SSHKeyUUID          string `json:"sshUuid"`
	OwnerName           string `json:"instanceOwnerName"`
	RootDiskSize        int64  `json:"rootDiskSize"`
	VolumeSize          string `json:"volumeSize"`
	DiskSize            int64  `json:"diskSize"`
	CPUCore             string `json:"cpuCore"`
	Status              string `json:"status"`
}

// Network represents an attached network on an instance.
type Network struct {
	UUID           string `json:"uuid"`
	Name           string `json:"name"`
	Type           string `json:"type"`
	PrivateIP      string `json:"privateIp"`
	PublicIP       string `json:"publicIp"`
	Gateway        string `json:"gateway"`
	Netmask        string `json:"netmask"`
	DefaultNetwork bool   `json:"defaultNetwork"`
}

// Status holds the current state of an instance.
type Status struct {
	UUID   string `json:"uuid"`
	Status string `json:"status"`
}

// Password holds an instance console/OS password.
type Password struct {
	UUID     string `json:"uuid"`
	Password string `json:"password"`
}

// CreateRequest holds parameters for creating an instance.
type CreateRequest struct {
	Name                string `json:"name"`
	ZoneUUID            string `json:"zoneUuid"`
	TemplateUUID        string `json:"templateUuid"`
	ComputeOfferingUUID string `json:"computeOfferingUuid"`
	NetworkUUID         string `json:"networkUuid"`
	StorageOfferingUUID string `json:"storageOfferingUuid,omitempty"`
	DiskSize            int    `json:"diskSize,omitempty"`
	RootDiskSize        int    `json:"rootDiskSize,omitempty"`
	SSHKeyName          string `json:"sshKeyName,omitempty"`
	SecurityGroupName   string `json:"securitygroupName,omitempty"`
	HypervisorName      string `json:"hypervisorName,omitempty"`
	Memory              string `json:"memory,omitempty"`
	CPUCore             string `json:"cpuCore,omitempty"`
}

type listInstanceResponse struct {
	Count                int        `json:"count"`
	ListInstanceResponse []Instance `json:"listInstanceResponse"`
}

type listInstanceNetworkResponse struct {
	Count                        int       `json:"count"`
	KongInstanceNetworkResponses []Network `json:"kongInstanceNetworkResponses"`
}

type instanceStatusResponse struct {
	UUID   string `json:"uuid"`
	Status string `json:"status"`
}

type listInstancePasswordResponse struct {
	Count                        int        `json:"count"`
	ListInstancePasswordResponse []Password `json:"listInstancePasswordResponse"`
}

// Service provides instance API operations.
type Service struct {
	client *httpclient.Client
}

// NewService creates a new instance Service.
func NewService(client *httpclient.Client) *Service {
	return &Service{client: client}
}

// List returns instances. zoneUUID is required. vmUUID is an optional filter.
func (s *Service) List(ctx context.Context, zoneUUID, vmUUID string) ([]Instance, error) {
	q := url.Values{"zoneUuid": {zoneUUID}}
	if vmUUID != "" {
		q.Set("vmUuid", vmUUID)
	}
	var resp listInstanceResponse
	if err := s.client.Get(ctx, "/restapi/instance/instanceList", q, &resp); err != nil {
		return nil, fmt.Errorf("listing instances: %w", err)
	}
	return resp.ListInstanceResponse, nil
}

// Get returns a single instance by UUID.
func (s *Service) Get(ctx context.Context, zoneUUID, vmUUID string) (*Instance, error) {
	instances, err := s.List(ctx, zoneUUID, vmUUID)
	if err != nil {
		return nil, err
	}
	if len(instances) == 0 {
		return nil, fmt.Errorf("instance %q not found in zone %q", vmUUID, zoneUUID)
	}
	return &instances[0], nil
}

// Create provisions a new instance.
func (s *Service) Create(ctx context.Context, req CreateRequest) (*Instance, error) {
	var resp listInstanceResponse
	if err := s.client.Post(ctx, "/restapi/instance/createInstance", req, &resp); err != nil {
		return nil, fmt.Errorf("creating instance: %w", err)
	}
	if len(resp.ListInstanceResponse) == 0 {
		return nil, fmt.Errorf("create instance returned empty response")
	}
	return &resp.ListInstanceResponse[0], nil
}

// Start starts a stopped instance. Returns the updated instance.
func (s *Service) Start(ctx context.Context, uuid string) (*Instance, error) {
	q := url.Values{"uuid": {uuid}}
	var resp listInstanceResponse
	if err := s.client.Get(ctx, "/restapi/instance/startInstance", q, &resp); err != nil {
		return nil, fmt.Errorf("starting instance %s: %w", uuid, err)
	}
	if len(resp.ListInstanceResponse) == 0 {
		return nil, fmt.Errorf("start returned empty response")
	}
	return &resp.ListInstanceResponse[0], nil
}

// Stop stops a running instance. forceStop bypasses graceful shutdown.
func (s *Service) Stop(ctx context.Context, uuid string, forceStop bool) (*Instance, error) {
	forceStr := "false"
	if forceStop {
		forceStr = "true"
	}
	q := url.Values{"uuid": {uuid}, "forceStop": {forceStr}}
	var resp listInstanceResponse
	if err := s.client.Get(ctx, "/restapi/instance/stopInstance", q, &resp); err != nil {
		return nil, fmt.Errorf("stopping instance %s: %w", uuid, err)
	}
	if len(resp.ListInstanceResponse) == 0 {
		return nil, fmt.Errorf("stop returned empty response")
	}
	return &resp.ListInstanceResponse[0], nil
}

// Destroy deletes an instance. expunge permanently removes the VM.
func (s *Service) Destroy(ctx context.Context, uuid string, expunge bool) error {
	expungeStr := "false"
	if expunge {
		expungeStr = "true"
	}
	q := url.Values{"uuid": {uuid}, "expunge": {expungeStr}}
	var resp listInstanceResponse
	if err := s.client.Get(ctx, "/restapi/instance/destroyInstance", q, &resp); err != nil {
		return fmt.Errorf("destroying instance %s: %w", uuid, err)
	}
	return nil
}

// Recover recovers an instance from an error state.
func (s *Service) Recover(ctx context.Context, uuid string) (*Instance, error) {
	q := url.Values{"uuid": {uuid}}
	var resp listInstanceResponse
	if err := s.client.Get(ctx, "/restapi/instance/recoverVm", q, &resp); err != nil {
		return nil, fmt.Errorf("recovering instance %s: %w", uuid, err)
	}
	if len(resp.ListInstanceResponse) == 0 {
		return nil, fmt.Errorf("recover returned empty response")
	}
	return &resp.ListInstanceResponse[0], nil
}

// Resize changes the compute offering for an instance (requires it to be stopped).
func (s *Service) Resize(ctx context.Context, uuid, offeringUUID, cpuCore, memory string) (*Instance, error) {
	q := url.Values{"uuid": {uuid}, "offeringUuid": {offeringUUID}}
	if cpuCore != "" {
		q.Set("cpuCore", cpuCore)
	}
	if memory != "" {
		q.Set("memory", memory)
	}
	var resp listInstanceResponse
	if err := s.client.Get(ctx, "/restapi/instance/resizeVm", q, &resp); err != nil {
		return nil, fmt.Errorf("resizing instance %s: %w", uuid, err)
	}
	if len(resp.ListInstanceResponse) == 0 {
		return nil, fmt.Errorf("resize returned empty response")
	}
	return &resp.ListInstanceResponse[0], nil
}

// GetStatus returns the current operational status of an instance.
func (s *Service) GetStatus(ctx context.Context, uuid string) (*Status, error) {
	q := url.Values{"uuid": {uuid}}
	var resp instanceStatusResponse
	if err := s.client.Get(ctx, "/restapi/instance/vmStatus", q, &resp); err != nil {
		return nil, fmt.Errorf("getting status for instance %s: %w", uuid, err)
	}
	return &Status{UUID: resp.UUID, Status: resp.Status}, nil
}

// WaitForState polls the instance status until it reaches one of the target states or the context is cancelled.
// targetStates should be the expected terminal state(s), e.g. "Running", "Stopped", "Destroyed".
// pollInterval controls how often to poll; pass 0 for the default (3s).
func (s *Service) WaitForState(ctx context.Context, uuid string, targetStates []string, pollInterval time.Duration) (*Status, error) {
	if pollInterval == 0 {
		pollInterval = 3 * time.Second
	}
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-ticker.C:
			status, err := s.GetStatus(ctx, uuid)
			if err != nil {
				return nil, err
			}
			for _, target := range targetStates {
				if strings.EqualFold(status.Status, target) {
					return status, nil
				}
			}
		}
	}
}

// ListNetworks returns the networks attached to an instance.
func (s *Service) ListNetworks(ctx context.Context, uuid string) ([]Network, error) {
	q := url.Values{"uuid": {uuid}}
	var resp listInstanceNetworkResponse
	if err := s.client.Get(ctx, "/restapi/instance/instanceNetworkList", q, &resp); err != nil {
		return nil, fmt.Errorf("listing networks for instance %s: %w", uuid, err)
	}
	return resp.KongInstanceNetworkResponses, nil
}

// ListPasswords returns OS/console passwords for instances. zoneUUID required; vmUUID optional.
func (s *Service) ListPasswords(ctx context.Context, zoneUUID, vmUUID string) ([]Password, error) {
	q := url.Values{"zoneUuid": {zoneUUID}}
	if vmUUID != "" {
		q.Set("uuid", vmUUID)
	}
	var resp listInstancePasswordResponse
	if err := s.client.Get(ctx, "/restapi/instance/instancePasswordList", q, &resp); err != nil {
		return nil, fmt.Errorf("listing passwords: %w", err)
	}
	return resp.ListInstancePasswordResponse, nil
}

// Rename updates an instance's display name.
func (s *Service) Rename(ctx context.Context, uuid, displayName string) (*Instance, error) {
	body := map[string]string{"uuid": uuid, "displayName": displayName}
	var resp listInstanceResponse
	if err := s.client.Put(ctx, "/restapi/instance/updateInstanceName", nil, body, &resp); err != nil {
		return nil, fmt.Errorf("renaming instance %s: %w", uuid, err)
	}
	if len(resp.ListInstanceResponse) == 0 {
		return nil, fmt.Errorf("rename returned empty response")
	}
	return &resp.ListInstanceResponse[0], nil
}
