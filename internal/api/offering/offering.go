// Package offering provides ZCP compute, storage, network, and VPC offering API operations.
package offering

import (
	"context"
	"fmt"
	"net/url"

	"github.com/zsoftly/zcp-cli/internal/httpclient"
)

// ComputeOffering represents a compute offering (instance size).
type ComputeOffering struct {
	UUID        string `json:"uuid"`
	Name        string `json:"name"`
	DisplayText string `json:"displayText"`
	Memory      string `json:"memory"`
	CPUCores    string `json:"numberOfCores"`
	ClockSpeed  string `json:"clockSpeed"`
	StorageType string `json:"storageType"`
	IsActive    bool   `json:"isActive"`
}

// StorageOffering represents a disk/storage offering.
type StorageOffering struct {
	UUID         string `json:"uuid"`
	Name         string `json:"name"`
	Description  string `json:"description"`
	DiskSize     string `json:"diskSize"`
	StorageType  string `json:"storageType"`
	IsActive     bool   `json:"isActive"`
	IsCustomDisk bool   `json:"isCustomDisk"`
}

// NetworkOffering represents a network offering.
type NetworkOffering struct {
	UUID                 string `json:"uuid"`
	Name                 string `json:"name"`
	DisplayText          string `json:"displayText"`
	OfferName            string `json:"offerName"`
	Availability         string `json:"availability"`
	GuestIPType          string `json:"guestIpType"`
	IsActive             bool   `json:"isActive"`
	NetworkTrialOffering bool   `json:"networkTrialOffering"`
}

// VPCOffering represents a VPC offering.
type VPCOffering struct {
	UUID        string `json:"uuid"`
	Name        string `json:"name"`
	DisplayText string `json:"displayText"`
}

type listComputeOfferingResponse struct {
	Count                       int               `json:"count"`
	ListComputeOfferingResponse []ComputeOffering `json:"listComputeOfferingResponse"`
}

type listStorageOfferingResponse struct {
	Count                       int               `json:"count"`
	ListStorageOfferingResponse []StorageOffering `json:"listStorageOfferingResponse"`
}

type listNetworkOfferingResponse struct {
	Count                       int               `json:"count"`
	ListNetworkOfferingResponse []NetworkOffering `json:"listNetworkOfferingResponse"`
}

type listVpcOfferingResponse struct {
	Count                   int           `json:"count"`
	ListVpcOfferingResponse []VPCOffering `json:"listVpcOfferingResponse"`
}

// Service provides offering API operations.
type Service struct {
	client *httpclient.Client
}

// NewService creates a new offering Service.
func NewService(client *httpclient.Client) *Service {
	return &Service{client: client}
}

// ListCompute returns compute offerings. zoneUUID and offeringUUID are optional filters.
func (s *Service) ListCompute(ctx context.Context, zoneUUID, offeringUUID string) ([]ComputeOffering, error) {
	q := url.Values{}
	if zoneUUID != "" {
		q.Set("zoneUuid", zoneUUID)
	}
	if offeringUUID != "" {
		q.Set("uuid", offeringUUID)
	}
	var resp listComputeOfferingResponse
	if err := s.client.Get(ctx, "/restapi/compute/computeOfferingList", q, &resp); err != nil {
		return nil, fmt.Errorf("listing compute offerings: %w", err)
	}
	return resp.ListComputeOfferingResponse, nil
}

// ListStorage returns storage offerings. zoneUUID is an optional filter.
func (s *Service) ListStorage(ctx context.Context, zoneUUID string) ([]StorageOffering, error) {
	q := url.Values{}
	if zoneUUID != "" {
		q.Set("zoneUuid", zoneUUID)
	}
	var resp listStorageOfferingResponse
	if err := s.client.Get(ctx, "/restapi/storage/storageOfferingList", q, &resp); err != nil {
		return nil, fmt.Errorf("listing storage offerings: %w", err)
	}
	return resp.ListStorageOfferingResponse, nil
}

// ListNetwork returns network offerings. zoneUUID is an optional filter.
func (s *Service) ListNetwork(ctx context.Context, zoneUUID string) ([]NetworkOffering, error) {
	q := url.Values{}
	if zoneUUID != "" {
		q.Set("zoneUuid", zoneUUID)
	}
	var resp listNetworkOfferingResponse
	if err := s.client.Get(ctx, "/restapi/networkoffering/networkOfferingList", q, &resp); err != nil {
		return nil, fmt.Errorf("listing network offerings: %w", err)
	}
	return resp.ListNetworkOfferingResponse, nil
}

// ListVPC returns VPC offerings. zoneUUID is an optional filter.
func (s *Service) ListVPC(ctx context.Context, zoneUUID string) ([]VPCOffering, error) {
	q := url.Values{}
	if zoneUUID != "" {
		q.Set("zoneUuid", zoneUUID)
	}
	var resp listVpcOfferingResponse
	if err := s.client.Get(ctx, "/restapi/vpcoffering/vpcOfferingList", q, &resp); err != nil {
		return nil, fmt.Errorf("listing VPC offerings: %w", err)
	}
	return resp.ListVpcOfferingResponse, nil
}
