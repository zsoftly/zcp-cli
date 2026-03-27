// Package volume provides ZCP volume API operations.
package volume

import (
	"context"
	"fmt"
	"net/url"
	"strconv"

	"github.com/zsoftly/zcp-cli/internal/httpclient"
)

// Volume represents a ZCP data volume.
type Volume struct {
	UUID                string `json:"uuid"`
	Name                string `json:"name"`
	Status              string `json:"status"`
	IsActive            bool   `json:"isActive"`
	VolumeType          string `json:"volumeType"`
	StorageDiskSize     string `json:"storageDiskSize"`
	StorageOfferingUUID string `json:"storageOfferingUuid"`
	StorageOfferingName string `json:"storageOfferingName"`
	VMInstanceUUID      string `json:"vmUuid"`
	VMInstanceName      string `json:"vmInstanceName"`
	ZoneUUID            string `json:"zoneUuid"`
	DomainName          string `json:"domainName"`
	JobID               string `json:"jobId"`
	CreatedAt           int64  `json:"createdTimeStamp"`
	ErrorMessage        string `json:"errorMessage"`
	IsShrink            bool   `json:"isShrink"`
}

// DeleteResponse is returned when deleting a volume.
type DeleteResponse struct {
	UUID   string `json:"uuid"`
	Status string `json:"status"`
}

// CreateRequest holds parameters for creating a volume.
type CreateRequest struct {
	Name                string `json:"name"`
	ZoneUUID            string `json:"zoneUuid"`
	StorageOfferingUUID string `json:"storageOfferingUuid"`
	DiskSize            int    `json:"diskSize,omitempty"`
}

type listVolumeResponse struct {
	Count              int      `json:"count"`
	ListVolumeResponse []Volume `json:"listVolumeResponse"`
}

// Service provides volume API operations.
type Service struct {
	client *httpclient.Client
}

// NewService creates a new volume Service.
func NewService(client *httpclient.Client) *Service {
	return &Service{client: client}
}

// List returns volumes. zoneUUID is required. vmUUID and volumeUUID are optional filters.
func (s *Service) List(ctx context.Context, zoneUUID, vmUUID, volumeUUID string) ([]Volume, error) {
	q := url.Values{"zoneUuid": {zoneUUID}}
	if vmUUID != "" {
		q.Set("vmUuid", vmUUID)
	}
	if volumeUUID != "" {
		q.Set("uuid", volumeUUID)
	}
	var resp listVolumeResponse
	if err := s.client.Get(ctx, "/restapi/volume/volumeList", q, &resp); err != nil {
		return nil, fmt.Errorf("listing volumes: %w", err)
	}
	return resp.ListVolumeResponse, nil
}

// Create creates a new data volume. Returns the volume (may include jobId for async tracking).
func (s *Service) Create(ctx context.Context, req CreateRequest) (*Volume, error) {
	var resp listVolumeResponse
	if err := s.client.Post(ctx, "/restapi/volume/createVolume", req, &resp); err != nil {
		return nil, fmt.Errorf("creating volume: %w", err)
	}
	if len(resp.ListVolumeResponse) == 0 {
		return nil, fmt.Errorf("create volume returned empty response")
	}
	return &resp.ListVolumeResponse[0], nil
}

// Attach attaches a volume to an instance.
func (s *Service) Attach(ctx context.Context, volumeUUID, instanceUUID string) (*Volume, error) {
	q := url.Values{"uuid": {volumeUUID}, "instanceUuid": {instanceUUID}}
	var resp listVolumeResponse
	if err := s.client.Get(ctx, "/restapi/volume/attachVolume", q, &resp); err != nil {
		return nil, fmt.Errorf("attaching volume %s to instance %s: %w", volumeUUID, instanceUUID, err)
	}
	if len(resp.ListVolumeResponse) == 0 {
		return nil, fmt.Errorf("attach volume returned empty response")
	}
	return &resp.ListVolumeResponse[0], nil
}

// Detach detaches a volume from its instance.
func (s *Service) Detach(ctx context.Context, volumeUUID string) (*Volume, error) {
	q := url.Values{"uuid": {volumeUUID}}
	var resp listVolumeResponse
	if err := s.client.Get(ctx, "/restapi/volume/detachVolume", q, &resp); err != nil {
		return nil, fmt.Errorf("detaching volume %s: %w", volumeUUID, err)
	}
	if len(resp.ListVolumeResponse) == 0 {
		return nil, fmt.Errorf("detach volume returned empty response")
	}
	return &resp.ListVolumeResponse[0], nil
}

// Delete deletes a volume permanently.
func (s *Service) Delete(ctx context.Context, uuid string) (*DeleteResponse, error) {
	if err := s.client.Delete(ctx, "/restapi/volume/deleteVolume/"+uuid, nil); err != nil {
		return nil, fmt.Errorf("deleting volume %s: %w", uuid, err)
	}
	return &DeleteResponse{UUID: uuid, Status: "deleted"}, nil
}

// Resize changes a volume's storage offering or disk size.
func (s *Service) Resize(ctx context.Context, uuid, storageOfferingUUID string, diskSize int, isShrink bool) (*Volume, error) {
	q := url.Values{"uuid": {uuid}, "storageOfferingUuid": {storageOfferingUUID}}
	if diskSize > 0 {
		q.Set("diskSize", strconv.Itoa(diskSize))
	}
	if isShrink {
		q.Set("isShrink", "true")
	}
	var resp listVolumeResponse
	if err := s.client.Get(ctx, "/restapi/volume/resizeVolume", q, &resp); err != nil {
		return nil, fmt.Errorf("resizing volume %s: %w", uuid, err)
	}
	if len(resp.ListVolumeResponse) == 0 {
		return nil, fmt.Errorf("resize volume returned empty response")
	}
	return &resp.ListVolumeResponse[0], nil
}
