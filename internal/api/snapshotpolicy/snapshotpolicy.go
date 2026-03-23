// Package snapshotpolicy provides ZCP snapshot policy API operations.
package snapshotpolicy

import (
	"context"
	"fmt"
	"net/url"

	"github.com/zsoftly/zcp-cli/internal/httpclient"
)

// SnapshotPolicy defines an automated snapshot schedule for a volume.
type SnapshotPolicy struct {
	UUID             string `json:"uuid"`
	Status           string `json:"status"`
	IsActive         bool   `json:"isActive"`
	VolumeUUID       string `json:"volumeUuid"`
	IntervalType     string `json:"intervalType"`
	ScheduleTime     string `json:"scheduleTime"`
	DayOfWeek        string `json:"dayOfWeek"`
	DayOfMonth       string `json:"dayOfMonth"`
	TimeZone         string `json:"timeZone"`
	MaximumSnapshots string `json:"maximumSnapshots"`
}

// CreateRequest holds parameters for creating a snapshot policy.
type CreateRequest struct {
	VolumeUUID       string `json:"volumeUuid"`
	IntervalType     string `json:"intervalType"`
	Timer            string `json:"timer"`
	DayOfWeek        string `json:"dayOfWeek,omitempty"`
	DayOfMonth       string `json:"dayOfMonth,omitempty"`
	TimeZone         string `json:"timeZone"`
	MaximumSnapshots string `json:"maximumSnapshots"`
}

type listSnapshotPoliciesResponse struct {
	Count                        int              `json:"count"`
	ListSnapShotPoliciesResponse []SnapshotPolicy `json:"listSnapShotPoliciesResponse"`
}

// Service provides snapshot policy API operations.
type Service struct {
	client *httpclient.Client
}

// NewService creates a new SnapshotPolicy Service.
func NewService(client *httpclient.Client) *Service {
	return &Service{client: client}
}

// List returns snapshot policies. volumeUUID and policyUUID are optional filters.
func (s *Service) List(ctx context.Context, volumeUUID, policyUUID string) ([]SnapshotPolicy, error) {
	q := url.Values{}
	if volumeUUID != "" {
		q.Set("volumeUuid", volumeUUID)
	}
	if policyUUID != "" {
		q.Set("uuid", policyUUID)
	}
	var resp listSnapshotPoliciesResponse
	if err := s.client.Get(ctx, "/restapi/snapshotPolicy/snapshotPolicyList", q, &resp); err != nil {
		return nil, fmt.Errorf("listing snapshot policies: %w", err)
	}
	return resp.ListSnapShotPoliciesResponse, nil
}

// Create creates a new snapshot policy for a volume.
func (s *Service) Create(ctx context.Context, req CreateRequest) (*SnapshotPolicy, error) {
	var resp listSnapshotPoliciesResponse
	if err := s.client.Post(ctx, "/restapi/snapshotPolicy/createSnapshotPolicy", req, &resp); err != nil {
		return nil, fmt.Errorf("creating snapshot policy: %w", err)
	}
	if len(resp.ListSnapShotPoliciesResponse) == 0 {
		return nil, fmt.Errorf("create snapshot policy returned empty response")
	}
	return &resp.ListSnapShotPoliciesResponse[0], nil
}

// Delete permanently removes a snapshot policy.
func (s *Service) Delete(ctx context.Context, uuid string) error {
	if err := s.client.Delete(ctx, "/restapi/snapshotPolicy/deleteSnapshotPolicy/"+uuid, nil); err != nil {
		return fmt.Errorf("deleting snapshot policy %s: %w", uuid, err)
	}
	return nil
}
