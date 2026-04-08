package vpn

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/zsoftly/zcp-cli/internal/httpclient"
)

// CustomerGateway represents a ZCP VPN customer gateway.
type CustomerGateway struct {
	Slug            string `json:"slug"`
	Name            string `json:"name"`
	Gateway         string `json:"gateway"`
	IKEPolicy       string `json:"ikepolicy"`
	ESPPolicy       string `json:"esppolicy"`
	ESPLifetime     string `json:"esplifetime"`
	IKELifetime     string `json:"ikelifetime"`
	IPSecPSK        string `json:"ipsecPresharedKey"`
	IKEVersion      string `json:"ikeVersion"`
	CIDRList        string `json:"cidrList"`
	ForceEncap      bool   `json:"forceencap"`
	SplitConnection bool   `json:"splitConnection"`
	DPD             bool   `json:"dpd"`
}

// CustomerGatewayRequest holds parameters for creating a VPN customer gateway.
type CustomerGatewayRequest struct {
	Name            string `json:"name"`
	Gateway         string `json:"gateway"`
	CIDRList        string `json:"cidrlist"`
	IPSecPSK        string `json:"ipsecpsk"`
	IKEPolicy       string `json:"ikepolicy"`
	ESPPolicy       string `json:"esppolicy"`
	IKELifetime     string `json:"ikelifetime,omitempty"`
	ESPLifetime     string `json:"esplifetime,omitempty"`
	IKEEncryption   string `json:"ikeEncryption,omitempty"`
	IKEHash         string `json:"ikeHash,omitempty"`
	IKEVersion      string `json:"ikeVersion,omitempty"`
	ESPEncryption   string `json:"espEncryption,omitempty"`
	ESPHash         string `json:"espHash,omitempty"`
	ForceEncap      bool   `json:"forceencap"`
	SplitConnection bool   `json:"splitConnection"`
	DPD             bool   `json:"dpd"`
}

// CustomerGatewayService provides VPN customer gateway API operations.
type CustomerGatewayService struct {
	client *httpclient.Client
}

// NewCustomerGatewayService creates a new CustomerGatewayService.
func NewCustomerGatewayService(client *httpclient.Client) *CustomerGatewayService {
	return &CustomerGatewayService{client: client}
}

// List returns all VPN customer gateways.
func (s *CustomerGatewayService) List(ctx context.Context) ([]CustomerGateway, error) {
	var env apiResponse
	if err := s.client.Get(ctx, "/vpn-customer-gateways", nil, &env); err != nil {
		return nil, fmt.Errorf("listing VPN customer gateways: %w", err)
	}
	var cgs []CustomerGateway
	if err := json.Unmarshal(env.Data, &cgs); err != nil {
		return nil, fmt.Errorf("decoding VPN customer gateway list: %w", err)
	}
	return cgs, nil
}

// Create provisions a new VPN customer gateway.
func (s *CustomerGatewayService) Create(ctx context.Context, req CustomerGatewayRequest) (*CustomerGateway, error) {
	var env apiResponse
	if err := s.client.Post(ctx, "/vpn-customer-gateways", req, &env); err != nil {
		return nil, fmt.Errorf("creating VPN customer gateway: %w", err)
	}
	var cg CustomerGateway
	if err := json.Unmarshal(env.Data, &cg); err != nil {
		return nil, fmt.Errorf("decoding created VPN customer gateway: %w", err)
	}
	return &cg, nil
}

// Update modifies a VPN customer gateway.
func (s *CustomerGatewayService) Update(ctx context.Context, slug string, req CustomerGatewayRequest) (*CustomerGateway, error) {
	var env apiResponse
	if err := s.client.Put(ctx, "/vpn-customer-gateways/"+slug, nil, req, &env); err != nil {
		return nil, fmt.Errorf("updating VPN customer gateway %s: %w", slug, err)
	}
	var cg CustomerGateway
	if err := json.Unmarshal(env.Data, &cg); err != nil {
		return nil, fmt.Errorf("decoding updated VPN customer gateway: %w", err)
	}
	return &cg, nil
}

// Delete removes a VPN customer gateway by slug.
func (s *CustomerGatewayService) Delete(ctx context.Context, slug string) error {
	if err := s.client.Delete(ctx, "/vpn-customer-gateways/"+slug, nil); err != nil {
		return fmt.Errorf("deleting VPN customer gateway %s: %w", slug, err)
	}
	return nil
}
