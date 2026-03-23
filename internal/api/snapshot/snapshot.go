// Package snapshot provides ZCP volume snapshot API operations.
package snapshot

import (
	"context"
	"fmt"
	"net/url"

	"github.com/zsoftly/zcp-cli/internal/httpclient"
)

// Snapshot represents a ZCP volume snapshot.
type Snapshot struct {
	UUID         string `json:"uuid"`
	Name         string `json:"name"`
	Status       string `json:"status"`
	IsActive     bool   `json:"isActive"`
	VolumeUUID   string `json:"volumeUuid"`
	SnapshotType string `json:"snapshotType"`
	DomainName   string `json:"domainName"`
	ZoneUUID     string `json:"zoneUuid"`
	SnapshotTime string `json:"snapshotTime"`
}

// CreateRequest holds parameters for creating a snapshot.
type CreateRequest struct {
	Name       string `json:"name"`
	VolumeUUID string `json:"volumeUuid"`
	ZoneUUID   string `json:"zoneUuid"`
}

type listSnapshotResponse struct {
	Count                int        `json:"count"`
	ListSnapShotResponse []Snapshot `json:"listSnapShotResponse"`
}

// Service provides snapshot API operations.
type Service struct {
	client *httpclient.Client
}

// NewService creates a new snapshot Service.
func NewService(client *httpclient.Client) *Service {
	return &Service{client: client}
}

// List returns snapshots. zoneUUID and snapshotUUID are optional filters.
func (s *Service) List(ctx context.Context, zoneUUID, snapshotUUID string) ([]Snapshot, error) {
	q := url.Values{}
	if zoneUUID != "" {
		q.Set("zoneUuid", zoneUUID)
	}
	if snapshotUUID != "" {
		q.Set("uuid", snapshotUUID)
	}
	var resp listSnapshotResponse
	if err := s.client.Get(ctx, "/restapi/snapshot/snapshotList", q, &resp); err != nil {
		return nil, fmt.Errorf("listing snapshots: %w", err)
	}
	return resp.ListSnapShotResponse, nil
}

// Create creates a new snapshot of the given volume.
func (s *Service) Create(ctx context.Context, req CreateRequest) (*Snapshot, error) {
	var resp listSnapshotResponse
	if err := s.client.Post(ctx, "/restapi/snapshot/createSnapshot", req, &resp); err != nil {
		return nil, fmt.Errorf("creating snapshot: %w", err)
	}
	if len(resp.ListSnapShotResponse) == 0 {
		return nil, fmt.Errorf("create snapshot returned empty response")
	}
	return &resp.ListSnapShotResponse[0], nil
}

// Delete permanently removes a snapshot.
func (s *Service) Delete(ctx context.Context, uuid string) error {
	if err := s.client.Delete(ctx, "/restapi/snapshot/deleteSnapshot/"+uuid, nil); err != nil {
		return fmt.Errorf("deleting snapshot %s: %w", uuid, err)
	}
	return nil
}
