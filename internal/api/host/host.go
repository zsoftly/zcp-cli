// Package host provides ZCP hypervisor host API operations (admin only).
package host

import (
	"context"
	"fmt"
	"net/url"

	"github.com/zsoftly/zcp-cli/internal/httpclient"
)

// Host represents a ZCP hypervisor host.
type Host struct {
	UUID                      string `json:"uuid"`
	Name                      string `json:"name"`
	Hypervisor                string `json:"hypervisor"`
	PodName                   string `json:"podName"`
	CPUCores                  string `json:"cpuCores"`
	CPUAllocated              string `json:"cpuAllocated"`
	CPUUsed                   string `json:"cpuUsed"`
	MemoryTotal               string `json:"memoryTotal"`
	MemoryAllocatedPercentage string `json:"memoryAllocatedPercentage"`
	MemoryUsedPercentage      string `json:"memoryUsedPercentage"`
	VMCount                   string `json:"vmCount"`
	IsActive                  bool   `json:"isActive"`
}

type listHostResponse struct {
	Count            int    `json:"count"`
	ListHostResponse []Host `json:"listHostResponse"`
}

// Service provides host API operations.
type Service struct {
	client *httpclient.Client
}

// NewService creates a new host Service.
func NewService(client *httpclient.Client) *Service { return &Service{client: client} }

// List returns hypervisor hosts. If uuid is non-empty, filters to that host.
func (s *Service) List(ctx context.Context, uuid string) ([]Host, error) {
	q := url.Values{}
	if uuid != "" {
		q.Set("uuid", uuid)
	}
	var resp listHostResponse
	if err := s.client.Get(ctx, "/restapi/host/hostList", q, &resp); err != nil {
		return nil, fmt.Errorf("listing hosts: %w", err)
	}
	return resp.ListHostResponse, nil
}
