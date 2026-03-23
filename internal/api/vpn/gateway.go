// Package vpn provides ZCP VPN API operations for gateways, customer gateways, connections, and users.
package vpn

import (
	"context"
	"fmt"
	"net/url"

	"github.com/zsoftly/zcp-cli/internal/httpclient"
)

// Gateway represents a ZCP VPN gateway.
type Gateway struct {
	UUID       string `json:"uuid"`
	PublicIP   string `json:"publicIpAddress"`
	DomainName string `json:"domainName"`
	ZoneUUID   string `json:"zoneUuid"`
	IsActive   bool   `json:"isActive"`
	VPCUUID    string `json:"vpcUuid"`
	Status     string `json:"status"`
}

type listVpnGatewayResponse struct {
	Count                  int       `json:"count"`
	ListVpnGatewayResponse []Gateway `json:"listVpnGatewayResponse"`
}

// GatewayService provides VPN gateway API operations.
type GatewayService struct {
	client *httpclient.Client
}

// NewGatewayService creates a new GatewayService.
func NewGatewayService(client *httpclient.Client) *GatewayService {
	return &GatewayService{client: client}
}

// List returns VPN gateways. zoneUUID is required; uuid and vpcUUID are optional filters.
func (s *GatewayService) List(ctx context.Context, zoneUUID, uuid, vpcUUID string) ([]Gateway, error) {
	q := url.Values{"zoneUuid": {zoneUUID}}
	if uuid != "" {
		q.Set("uuid", uuid)
	}
	if vpcUUID != "" {
		q.Set("vpcUuid", vpcUUID)
	}
	var resp listVpnGatewayResponse
	if err := s.client.Get(ctx, "/restapi/vpngateway/vpnGatewayList", q, &resp); err != nil {
		return nil, fmt.Errorf("listing VPN gateways: %w", err)
	}
	return resp.ListVpnGatewayResponse, nil
}

// Create provisions a new VPN gateway for the specified VPC.
func (s *GatewayService) Create(ctx context.Context, vpcUUID string) (*Gateway, error) {
	body := map[string]string{"vpcUuid": vpcUUID}
	var resp listVpnGatewayResponse
	if err := s.client.Post(ctx, "/restapi/vpngateway/addVpnGateway", body, &resp); err != nil {
		return nil, fmt.Errorf("creating VPN gateway: %w", err)
	}
	if len(resp.ListVpnGatewayResponse) == 0 {
		return nil, fmt.Errorf("create VPN gateway returned empty response")
	}
	return &resp.ListVpnGatewayResponse[0], nil
}

// Delete removes a VPN gateway by UUID.
func (s *GatewayService) Delete(ctx context.Context, uuid string) error {
	if err := s.client.Delete(ctx, "/restapi/vpngateway/deleteVpnGateway/"+uuid, nil); err != nil {
		return fmt.Errorf("deleting VPN gateway %s: %w", uuid, err)
	}
	return nil
}
