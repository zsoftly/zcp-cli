// Package ipaddress provides ZCP public IP address API operations.
package ipaddress

import (
	"context"
	"fmt"
	"net/url"

	"github.com/zsoftly/zcp-cli/internal/httpclient"
)

// IPAddress represents a ZCP public IP address.
type IPAddress struct {
	UUID            string `json:"uuid"`
	PublicIPAddress string `json:"publicIpAddress"`
	State           string `json:"state"`
	IsActive        bool   `json:"isActive"`
	ZoneUUID        string `json:"zoneUuid"`
	ZoneName        string `json:"zoneName"`
	NetworkUUID     string `json:"networkUuid"`
	IsSourceNAT     bool   `json:"isSourcenat"`
	Status          string `json:"status"`
}

// StaticNATConfig holds the result of a static NAT enable/disable operation.
type StaticNATConfig struct {
	IPAddressUUID string `json:"ipAddressUuid"`
	VMUUID        string `json:"vmUuid"`
	VMName        string `json:"vmName"`
	NetworkUUID   string `json:"networkUuid"`
	IsActive      bool   `json:"isActive"`
	Status        string `json:"status"`
}

// EnableStaticNATRequest holds parameters for enabling static NAT.
type EnableStaticNATRequest struct {
	IPAddressUUID string `json:"ipAddressUuid"`
	VMUUID        string `json:"vmUuid"`
	NetworkUUID   string `json:"networkUuid"`
}

type listIPAddressResponse struct {
	Count                 int         `json:"count"`
	ListIpAddressResponse []IPAddress `json:"listIpAddressResponse"`
}

type enableStaticNATResponse struct {
	Count                       int               `json:"count"`
	KongAttachStaticNatResponse []StaticNATConfig `json:"kongAttachStaticNatResponse"`
}

// Service provides IP address API operations.
type Service struct {
	client *httpclient.Client
}

// NewService creates a new IPAddress Service.
func NewService(client *httpclient.Client) *Service {
	return &Service{client: client}
}

// List returns public IP addresses. zoneUUID is required; networkUUID is an optional filter.
func (s *Service) List(ctx context.Context, zoneUUID, networkUUID string) ([]IPAddress, error) {
	q := url.Values{"zoneUuid": {zoneUUID}}
	if networkUUID != "" {
		q.Set("networkUuid", networkUUID)
	}
	var resp listIPAddressResponse
	if err := s.client.Get(ctx, "/restapi/ipaddress/ipAddressList", q, &resp); err != nil {
		return nil, fmt.Errorf("listing IP addresses: %w", err)
	}
	return resp.ListIpAddressResponse, nil
}

// Acquire allocates a new public IP address for a network.
func (s *Service) Acquire(ctx context.Context, networkUUID, networkType string) (*IPAddress, error) {
	q := url.Values{
		"networkUuid": {networkUUID},
		"networkType": {networkType},
	}
	var resp listIPAddressResponse
	if err := s.client.Get(ctx, "/restapi/ipaddress/acquireIpAddress", q, &resp); err != nil {
		return nil, fmt.Errorf("acquiring IP address: %w", err)
	}
	if len(resp.ListIpAddressResponse) == 0 {
		return nil, fmt.Errorf("acquire IP address returned empty response")
	}
	return &resp.ListIpAddressResponse[0], nil
}

// Release deallocates a public IP address by UUID.
func (s *Service) Release(ctx context.Context, uuid string) error {
	if err := s.client.Delete(ctx, "/restapi/ipaddress/releaseIpAddress", url.Values{"uuid": {uuid}}); err != nil {
		return fmt.Errorf("releasing IP address %s: %w", uuid, err)
	}
	return nil
}

// EnableStaticNAT enables static NAT, associating a public IP with a VM.
func (s *Service) EnableStaticNAT(ctx context.Context, ipAddressUUID, vmUUID, networkUUID string) (*StaticNATConfig, error) {
	req := EnableStaticNATRequest{
		IPAddressUUID: ipAddressUUID,
		VMUUID:        vmUUID,
		NetworkUUID:   networkUUID,
	}
	var resp enableStaticNATResponse
	if err := s.client.Post(ctx, "/restapi/ipaddress/enableStaticNat", req, &resp); err != nil {
		return nil, fmt.Errorf("enabling static NAT for IP %s: %w", ipAddressUUID, err)
	}
	if len(resp.KongAttachStaticNatResponse) == 0 {
		return nil, fmt.Errorf("enable static NAT returned empty response")
	}
	return &resp.KongAttachStaticNatResponse[0], nil
}

// DisableStaticNAT removes the static NAT association for a public IP address.
func (s *Service) DisableStaticNAT(ctx context.Context, ipAddressUUID string) error {
	if err := s.client.Delete(ctx, "/restapi/ipaddress/disableStaticNat", url.Values{"ipAddressUuid": {ipAddressUUID}}); err != nil {
		return fmt.Errorf("disabling static NAT for IP %s: %w", ipAddressUUID, err)
	}
	return nil
}
