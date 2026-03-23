package vpn

import (
	"context"
	"fmt"
	"net/url"

	"github.com/zsoftly/zcp-cli/internal/httpclient"
)

// CustomerGateway represents a ZCP VPN customer gateway.
type CustomerGateway struct {
	UUID            string `json:"uuid"`
	IsActive        bool   `json:"isActive"`
	IKEPolicy       string `json:"ikepolicy"`
	ESPLifetime     string `json:"esplifetime"`
	IKELifetime     string `json:"ikelifetime"`
	IPSecPSK        string `json:"ipsecPresharedKey"`
	IKEVersion      string `json:"ikeVersion"`
	CIDRList        string `json:"cidrList"`
	ForceEncap      bool   `json:"forceencap"`
	SplitConnection bool   `json:"splitConnection"`
}

// CustomerGatewayRequest holds parameters for creating or updating a VPN customer gateway.
type CustomerGatewayRequest struct {
	Name            string `json:"name"`
	Gateway         string `json:"gateway"`
	CIDRList        string `json:"cidrlist"`
	IPSecPSK        string `json:"ipsecpsk"`
	IKEPolicy       string `json:"ikepolicy"`
	ESPPolicy       string `json:"esppolicy"`
	IKELifetime     string `json:"ikelifetime"`
	ESPLifetime     string `json:"esplifetime"`
	IKEEncryption   string `json:"ikeEncryption"`
	IKEHash         string `json:"ikeHash"`
	IKEVersion      string `json:"ikeVersion,omitempty"`
	ESPEncryption   string `json:"espEncryption"`
	ESPHash         string `json:"espHash"`
	ForceEncap      bool   `json:"forceencap"`
	SplitConnection bool   `json:"splitConnection"`
	DPD             bool   `json:"dpd"`
}

// CustomerGatewayUpdateRequest holds the UUID plus all create fields for an update.
type CustomerGatewayUpdateRequest struct {
	UUID string `json:"uuid"`
	CustomerGatewayRequest
}

type listVpnCustomerGatewayResponse struct {
	Count                          int               `json:"count"`
	ListVpnCustomerGatewayResponse []CustomerGateway `json:"listVpnCustomerGatewayResponse"`
}

// CustomerGatewayService provides VPN customer gateway API operations.
type CustomerGatewayService struct {
	client *httpclient.Client
}

// NewCustomerGatewayService creates a new CustomerGatewayService.
func NewCustomerGatewayService(client *httpclient.Client) *CustomerGatewayService {
	return &CustomerGatewayService{client: client}
}

// List returns VPN customer gateways. uuid is an optional filter.
func (s *CustomerGatewayService) List(ctx context.Context, uuid string) ([]CustomerGateway, error) {
	var q url.Values
	if uuid != "" {
		q = url.Values{"uuid": {uuid}}
	}
	var resp listVpnCustomerGatewayResponse
	if err := s.client.Get(ctx, "/restapi/vpncustomergateway/vpnCustomerGatewayList", q, &resp); err != nil {
		return nil, fmt.Errorf("listing VPN customer gateways: %w", err)
	}
	return resp.ListVpnCustomerGatewayResponse, nil
}

// Create provisions a new VPN customer gateway.
func (s *CustomerGatewayService) Create(ctx context.Context, req CustomerGatewayRequest) (*CustomerGateway, error) {
	var resp listVpnCustomerGatewayResponse
	if err := s.client.Post(ctx, "/restapi/vpncustomergateway/addVpnCustomerGateway", req, &resp); err != nil {
		return nil, fmt.Errorf("creating VPN customer gateway: %w", err)
	}
	if len(resp.ListVpnCustomerGatewayResponse) == 0 {
		return nil, fmt.Errorf("create VPN customer gateway returned empty response")
	}
	return &resp.ListVpnCustomerGatewayResponse[0], nil
}

// Update modifies a VPN customer gateway.
func (s *CustomerGatewayService) Update(ctx context.Context, req CustomerGatewayUpdateRequest) (*CustomerGateway, error) {
	var resp listVpnCustomerGatewayResponse
	if err := s.client.Put(ctx, "/restapi/vpncustomergateway/updateVpnCustomerGateway", nil, req, &resp); err != nil {
		return nil, fmt.Errorf("updating VPN customer gateway %s: %w", req.UUID, err)
	}
	if len(resp.ListVpnCustomerGatewayResponse) == 0 {
		return nil, fmt.Errorf("update VPN customer gateway returned empty response")
	}
	return &resp.ListVpnCustomerGatewayResponse[0], nil
}

// Delete removes a VPN customer gateway by UUID.
func (s *CustomerGatewayService) Delete(ctx context.Context, uuid string) error {
	if err := s.client.Delete(ctx, "/restapi/vpncustomergateway/deleteVpnCustomerGateway/"+uuid, nil); err != nil {
		return fmt.Errorf("deleting VPN customer gateway %s: %w", uuid, err)
	}
	return nil
}
