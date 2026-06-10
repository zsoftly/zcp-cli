package vpn

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/zsoftly/zcp-cli/pkg/httpclient"
)

// CustomerGateway represents a ZCP VPN customer gateway.
type CustomerGateway struct {
	Slug               string `json:"slug"`
	Name               string `json:"name"`
	Gateway            string `json:"gateway"`
	IKEPolicy          string `json:"ike_policy"`
	ESPPolicy          string `json:"esp_policy"`
	ESPLifetime        string `json:"esp_lifetime"`
	IKELifetime        string `json:"ike_lifetime"`
	IPSecPSK           string `json:"ipsecpsk"`
	IKEVersion         string `json:"ike_version"`
	CIDRList           string `json:"cidr_list"`
	ForceEncapsulation bool   `json:"forceencap"`
	SplitConnections   string `json:"split_connections"`
	DeadPeerDetection  bool   `json:"dpd"`
}

// CustomerGatewayRequest holds parameters for creating or updating a VPN customer gateway.
type CustomerGatewayRequest struct {
	Name               string `json:"name"`
	Gateway            string `json:"gateway"`
	CIDRList           string `json:"cidr_list"`
	IPSecPSK           string `json:"ipsec_preshared_key"`
	IKEPolicy          string `json:"ike_policy"`
	ESPPolicy          string `json:"esp_policy"`
	IKELifetime        string `json:"ike_lifetime,omitempty"`
	ESPLifetime        string `json:"esp_lifetime,omitempty"`
	IKEEncryption      string `json:"ike_encryption,omitempty"`
	IKEHash            string `json:"ike_hash,omitempty"`
	IKEVersion         string `json:"ike_version,omitempty"`
	IKEDH              string `json:"ike_dh,omitempty"`
	ESPEncryption      string `json:"esp_encryption,omitempty"`
	ESPHash            string `json:"esp_hash,omitempty"`
	ESPDH              string `json:"esp_dh,omitempty"`
	ESPPFS             string `json:"esp_pfs,omitempty"`
	ForceEncapsulation bool   `json:"force_encapsulation"`
	SplitConnections   bool   `json:"split_connections"`
	DeadPeerDetection  bool   `json:"dead_peer_detection"`
	CloudProvider      string `json:"cloud_provider,omitempty"`
	Region             string `json:"region,omitempty"`
	Project            string `json:"project,omitempty"`
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

// Get returns a single VPN customer gateway by slug.
func (s *CustomerGatewayService) Get(ctx context.Context, slug string) (*CustomerGateway, error) {
	var env apiResponse
	if err := s.client.Get(ctx, "/vpn-customer-gateways/"+slug, nil, &env); err != nil {
		return nil, fmt.Errorf("getting VPN customer gateway %s: %w", slug, err)
	}
	if len(env.Data) == 0 || string(env.Data) == "null" {
		return nil, fmt.Errorf("VPN customer gateway %q not found or still provisioning", slug)
	}
	var cg CustomerGateway
	if err := json.Unmarshal(env.Data, &cg); err != nil {
		return nil, fmt.Errorf("decoding VPN customer gateway: %w", err)
	}
	if cg.Slug == "" {
		cg.Slug = slug
	}
	return &cg, nil
}

// Create provisions a new VPN customer gateway.
func (s *CustomerGatewayService) Create(ctx context.Context, req CustomerGatewayRequest) (*CustomerGateway, error) {
	var env apiResponse
	if err := s.client.Post(ctx, "/vpn-customer-gateways", req, &env); err != nil {
		return nil, fmt.Errorf("creating VPN customer gateway: %w", err)
	}
	// The Create API returns a metadata-only response (no VPN config fields).
	// Try to fetch the full config; if the gateway is still provisioning, return partial data.
	var partial CustomerGateway
	if len(env.Data) > 0 && string(env.Data) != "null" {
		_ = json.Unmarshal(env.Data, &partial)
	}
	if partial.Slug != "" {
		if full, err := s.Get(ctx, partial.Slug); err == nil && full.Gateway != "" {
			return full, nil
		}
	}
	if partial.Slug == "" {
		return nil, fmt.Errorf("creating VPN customer gateway: no slug in response")
	}
	return &partial, nil
}

// Update modifies a VPN customer gateway.
func (s *CustomerGatewayService) Update(ctx context.Context, slug string, req CustomerGatewayRequest) (*CustomerGateway, error) {
	var env apiResponse
	if err := s.client.Put(ctx, "/vpn-customer-gateways/"+slug, nil, req, &env); err != nil {
		return nil, fmt.Errorf("updating VPN customer gateway %s: %w", slug, err)
	}
	// The Update API may return partial/null data — fall back to Get.
	raw := string(env.Data)
	if len(env.Data) > 0 && raw != "null" && raw != "[null]" {
		var cg CustomerGateway
		if err := json.Unmarshal(env.Data, &cg); err == nil && cg.Slug != "" {
			return &cg, nil
		}
	}
	return s.Get(ctx, slug)
}

// Delete removes a VPN customer gateway by slug.
func (s *CustomerGatewayService) Delete(ctx context.Context, slug string) error {
	if err := s.client.Delete(ctx, "/vpn-customer-gateways/"+slug, nil); err != nil {
		return fmt.Errorf("deleting VPN customer gateway %s: %w", slug, err)
	}
	return nil
}
