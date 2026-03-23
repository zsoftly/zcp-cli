package vpn

import (
	"context"
	"fmt"
	"net/url"

	"github.com/zsoftly/zcp-cli/internal/httpclient"
)

// Connection represents a ZCP VPN connection.
type Connection struct {
	UUID                string `json:"uuid"`
	State               string `json:"state"`
	IsActive            bool   `json:"isActive"`
	IKEPolicy           string `json:"ikePolicy"`
	ESPPolicy           string `json:"espPolicy"`
	IPSecPSK            string `json:"ipsecPresharedKey"`
	PublicIP            string `json:"publicIpAddress"`
	ZoneUUID            string `json:"zoneUuid"`
	CustomerGatewayUUID string `json:"customerGatewayUuid"`
	VPNGatewayUUID      string `json:"vpnGatewayUuid"`
}

// ConnectionCreateRequest holds parameters for creating a VPN connection.
type ConnectionCreateRequest struct {
	VPCUUID             string `json:"vpcUuid"`
	CustomerGatewayUUID string `json:"customerGatewayUuid"`
	Passive             bool   `json:"passive"`
}

type listVpnConnectionResponse struct {
	Count                     int          `json:"count"`
	ListVpnConnectionResponse []Connection `json:"listVpnConnectionResponse"`
}

// ConnectionService provides VPN connection API operations.
type ConnectionService struct {
	client *httpclient.Client
}

// NewConnectionService creates a new ConnectionService.
func NewConnectionService(client *httpclient.Client) *ConnectionService {
	return &ConnectionService{client: client}
}

// List returns VPN connections. zoneUUID is required; uuid and vpcUUID are optional filters.
func (s *ConnectionService) List(ctx context.Context, zoneUUID, uuid, vpcUUID string) ([]Connection, error) {
	q := url.Values{"zoneUuid": {zoneUUID}}
	if uuid != "" {
		q.Set("uuid", uuid)
	}
	if vpcUUID != "" {
		q.Set("vpcUuid", vpcUUID)
	}
	var resp listVpnConnectionResponse
	if err := s.client.Get(ctx, "/restapi/vpnconnection/vpnConnectionList", q, &resp); err != nil {
		return nil, fmt.Errorf("listing VPN connections: %w", err)
	}
	return resp.ListVpnConnectionResponse, nil
}

// Create establishes a new VPN connection.
func (s *ConnectionService) Create(ctx context.Context, req ConnectionCreateRequest) (*Connection, error) {
	var resp listVpnConnectionResponse
	if err := s.client.Post(ctx, "/restapi/vpnconnection/addVpnConnection", req, &resp); err != nil {
		return nil, fmt.Errorf("creating VPN connection: %w", err)
	}
	if len(resp.ListVpnConnectionResponse) == 0 {
		return nil, fmt.Errorf("create VPN connection returned empty response")
	}
	return &resp.ListVpnConnectionResponse[0], nil
}

// Reset resets a VPN connection by UUID using a PUT with no body.
func (s *ConnectionService) Reset(ctx context.Context, uuid string) (*Connection, error) {
	var resp listVpnConnectionResponse
	if err := s.client.Put(ctx, "/restapi/vpnconnection/resetVpnConnection/"+uuid, nil, nil, &resp); err != nil {
		return nil, fmt.Errorf("resetting VPN connection %s: %w", uuid, err)
	}
	if len(resp.ListVpnConnectionResponse) == 0 {
		return nil, fmt.Errorf("reset VPN connection returned empty response")
	}
	return &resp.ListVpnConnectionResponse[0], nil
}

// Delete removes a VPN connection by UUID.
func (s *ConnectionService) Delete(ctx context.Context, uuid string) error {
	if err := s.client.Delete(ctx, "/restapi/vpnconnection/deleteVpnConnection/"+uuid, nil); err != nil {
		return fmt.Errorf("deleting VPN connection %s: %w", uuid, err)
	}
	return nil
}
