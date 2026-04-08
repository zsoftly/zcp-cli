// Package ipaddress provides ZCP public IP address API operations.
package ipaddress

import (
	"context"
	"fmt"
	"net/url"

	"github.com/zsoftly/zcp-cli/internal/httpclient"
)

// IPAddress represents a ZCP public IP address from the STKCNSL API.
type IPAddress struct {
	ID                 string  `json:"id"`
	IPID               string  `json:"ip_id"`
	IPAddress          string  `json:"ipaddress"`
	Type               string  `json:"type"`
	NetworkID          string  `json:"network_id"`
	VirtualMachineID   string  `json:"virtual_machine_id"`
	VPCID              string  `json:"vpc_id"`
	Strategy           string  `json:"strategy"`
	Name               string  `json:"name"`
	Slug               string  `json:"slug"`
	Description        string  `json:"description"`
	UserID             string  `json:"user_id"`
	AccountID          string  `json:"account_id"`
	ProjectID          string  `json:"project_id"`
	RegionID           string  `json:"region_id"`
	RequestStatus      bool    `json:"request_status"`
	IsManualAcquire    bool    `json:"is_manual_acquire"`
	VirtualMachineName string  `json:"virtual_machine_name"`
	ServiceName        string  `json:"service_name"`
	AllTimeConsumption float64 `json:"all_time_consumption"`
	CreatedAt          string  `json:"created_at"`
	UpdatedAt          string  `json:"updated_at"`
	DeletedAt          string  `json:"deleted_at"`
	FrozenAt           string  `json:"frozen_at"`
	SuspendedAt        string  `json:"suspended_at"`
	TerminatedAt       string  `json:"terminated_at"`
}

// CreateRequest holds parameters for allocating a new public IP address.
type CreateRequest struct {
	VPC          string `json:"vpc,omitempty"`
	Network      string `json:"network,omitempty"`
	Plan         string `json:"plan"`
	BillingCycle string `json:"billing_cycle"`
}

// StaticNATRequest holds parameters for enabling static NAT.
type StaticNATRequest struct {
	VirtualMachine string `json:"virtual_machine"`
}

// RemoteAccessVPN represents a remote access VPN entry on an IP address.
type RemoteAccessVPN struct {
	ID        string `json:"id"`
	PublicIP  string `json:"public_ip"`
	State     string `json:"state"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

// listResponse is the STKCNSL envelope for paginated IP address lists.
type listResponse struct {
	Status  string      `json:"status"`
	Message string      `json:"message"`
	Data    []IPAddress `json:"data"`
}

// singleResponse is the STKCNSL envelope for single IP address responses.
type singleResponse struct {
	Status  string    `json:"status"`
	Message string    `json:"message"`
	Data    IPAddress `json:"data"`
}

// vpnListResponse is the STKCNSL envelope for remote access VPN lists.
type vpnListResponse struct {
	Status  string            `json:"status"`
	Message string            `json:"message"`
	Data    []RemoteAccessVPN `json:"data"`
}

// vpnSingleResponse is the STKCNSL envelope for a single VPN response.
type vpnSingleResponse struct {
	Status  string          `json:"status"`
	Message string          `json:"message"`
	Data    RemoteAccessVPN `json:"data"`
}

// Service provides IP address API operations.
type Service struct {
	client *httpclient.Client
}

// NewService creates a new IPAddress Service.
func NewService(client *httpclient.Client) *Service {
	return &Service{client: client}
}

// List returns public IP addresses. Optional filters: vpcSlug.
func (s *Service) List(ctx context.Context, vpcSlug string) ([]IPAddress, error) {
	q := url.Values{}
	if vpcSlug != "" {
		q.Set("filter[vpc]", vpcSlug)
	}
	var resp listResponse
	if err := s.client.Get(ctx, "/ipaddresses", q, &resp); err != nil {
		return nil, fmt.Errorf("listing IP addresses: %w", err)
	}
	return resp.Data, nil
}

// Allocate creates (allocates) a new public IP address.
func (s *Service) Allocate(ctx context.Context, req CreateRequest) (*IPAddress, error) {
	var resp singleResponse
	if err := s.client.Post(ctx, "/ipaddresses", req, &resp); err != nil {
		return nil, fmt.Errorf("allocating IP address: %w", err)
	}
	return &resp.Data, nil
}

// EnableStaticNAT enables static NAT, associating a public IP with a VM.
// ipSlug is the IP address slug (e.g. "1036521143").
// vmSlug is the virtual machine slug.
func (s *Service) EnableStaticNAT(ctx context.Context, ipSlug, vmSlug string) (*IPAddress, error) {
	body := StaticNATRequest{VirtualMachine: vmSlug}
	var resp singleResponse
	if err := s.client.Post(ctx, "/ipaddresses/"+ipSlug+"/static-nat", body, &resp); err != nil {
		return nil, fmt.Errorf("enabling static NAT for IP %s: %w", ipSlug, err)
	}
	return &resp.Data, nil
}

// ListRemoteAccessVPNs returns remote access VPNs for a public IP address.
func (s *Service) ListRemoteAccessVPNs(ctx context.Context, ipSlug string) ([]RemoteAccessVPN, error) {
	var resp vpnListResponse
	if err := s.client.Get(ctx, "/ipaddresses/"+ipSlug+"/remote-access-vpns", nil, &resp); err != nil {
		return nil, fmt.Errorf("listing remote access VPNs for IP %s: %w", ipSlug, err)
	}
	return resp.Data, nil
}

// EnableRemoteAccessVPN enables remote access VPN on a public IP address.
func (s *Service) EnableRemoteAccessVPN(ctx context.Context, ipSlug string) (*RemoteAccessVPN, error) {
	var resp vpnSingleResponse
	if err := s.client.Post(ctx, "/ipaddresses/"+ipSlug+"/remote-access-vpns", nil, &resp); err != nil {
		return nil, fmt.Errorf("enabling remote access VPN for IP %s: %w", ipSlug, err)
	}
	return &resp.Data, nil
}

// DisableRemoteAccessVPN disables a remote access VPN on a public IP address.
func (s *Service) DisableRemoteAccessVPN(ctx context.Context, ipSlug, vpnID string) error {
	if err := s.client.Delete(ctx, "/ipaddresses/"+ipSlug+"/remote-access-vpns/"+vpnID, nil); err != nil {
		return fmt.Errorf("disabling remote access VPN %s for IP %s: %w", vpnID, ipSlug, err)
	}
	return nil
}
