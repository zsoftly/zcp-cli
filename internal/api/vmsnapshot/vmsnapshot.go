// Package vmsnapshot provides ZCP VM snapshot API operations.
package vmsnapshot

import (
	"context"
	"fmt"
	"net/url"

	"github.com/zsoftly/zcp-cli/internal/httpclient"
)

// VMSnapshot represents a ZCP VM snapshot (whole-machine snapshot).
type VMSnapshot struct {
	UUID        string `json:"uuid"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Status      string `json:"status"`
	IsActive    bool   `json:"isActive"`
	IsCurrent   bool   `json:"isCurrent"`
	JobID       string `json:"jobId"`
	ZoneUUID    string `json:"zoneUuid"`
	DomainName  string `json:"domainName"`
	CreatedAt   string `json:"createdTimeStamp"`
}

// DeleteResponse is returned when deleting a VM snapshot.
type DeleteResponse struct {
	UUID   string `json:"uuid"`
	Status string `json:"status"`
}

// CreateRequest holds parameters for creating a VM snapshot.
type CreateRequest struct {
	Name               string `json:"name"`
	ZoneUUID           string `json:"zoneUuid"`
	VirtualMachineUUID string `json:"virtualmachineUuid"`
	Description        string `json:"description,omitempty"`
	SnapshotMemory     bool   `json:"snapshotMemory"`
}

type listVMSnapshotResponse struct {
	Count                  int          `json:"count"`
	ListVmSnapshotResponse []VMSnapshot `json:"listVmSnapshotResponse"`
}

// Service provides VM snapshot API operations.
type Service struct {
	client *httpclient.Client
}

// NewService creates a new VMSnapshot Service.
func NewService(client *httpclient.Client) *Service {
	return &Service{client: client}
}

// List returns VM snapshots. zoneUUID and snapshotUUID are optional filters.
func (s *Service) List(ctx context.Context, zoneUUID, snapshotUUID string) ([]VMSnapshot, error) {
	q := url.Values{}
	if zoneUUID != "" {
		q.Set("zoneUuid", zoneUUID)
	}
	if snapshotUUID != "" {
		q.Set("uuid", snapshotUUID)
	}
	var resp listVMSnapshotResponse
	if err := s.client.Get(ctx, "/restapi/vmsnapshot/vmsnapshotList", q, &resp); err != nil {
		return nil, fmt.Errorf("listing VM snapshots: %w", err)
	}
	return resp.ListVmSnapshotResponse, nil
}

// Create creates a new VM snapshot. Returns the snapshot including jobId for async tracking.
func (s *Service) Create(ctx context.Context, req CreateRequest) (*VMSnapshot, error) {
	var resp listVMSnapshotResponse
	if err := s.client.Post(ctx, "/restapi/vmsnapshot/createVmSnapshot", req, &resp); err != nil {
		return nil, fmt.Errorf("creating VM snapshot: %w", err)
	}
	if len(resp.ListVmSnapshotResponse) == 0 {
		return nil, fmt.Errorf("create VM snapshot returned empty response")
	}
	return &resp.ListVmSnapshotResponse[0], nil
}

// Delete permanently removes a VM snapshot.
func (s *Service) Delete(ctx context.Context, uuid string) (*DeleteResponse, error) {
	if err := s.client.Delete(ctx, "/restapi/vmsnapshot/deleteVmSnapshot/"+uuid, nil); err != nil {
		return nil, fmt.Errorf("deleting VM snapshot %s: %w", uuid, err)
	}
	return &DeleteResponse{UUID: uuid, Status: "deleted"}, nil
}

// Revert reverts an instance to a VM snapshot state (async — check jobId).
func (s *Service) Revert(ctx context.Context, uuid string) (*VMSnapshot, error) {
	q := url.Values{"uuid": {uuid}}
	var resp listVMSnapshotResponse
	if err := s.client.Get(ctx, "/restapi/vmsnapshot/revertToVmSnapshot", q, &resp); err != nil {
		return nil, fmt.Errorf("reverting to VM snapshot %s: %w", uuid, err)
	}
	if len(resp.ListVmSnapshotResponse) == 0 {
		return nil, fmt.Errorf("revert returned empty response")
	}
	return &resp.ListVmSnapshotResponse[0], nil
}
