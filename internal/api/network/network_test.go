package network_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/zsoftly/zcp-cli/internal/api/network"
	"github.com/zsoftly/zcp-cli/internal/httpclient"
)

// helpers

func newClient(baseURL string) *httpclient.Client {
	return httpclient.New(httpclient.Options{
		BaseURL:   baseURL,
		APIKey:    "testkey",
		SecretKey: "testsecret",
		Timeout:   5 * time.Second,
	})
}

type listNetworkResponse struct {
	Count               int               `json:"count"`
	ListNetworkResponse []network.Network `json:"listNetworkResponse"`
}

func makeNetwork(uuid, name, networkType string) network.Network {
	return network.Network{
		UUID:        uuid,
		Name:        name,
		Status:      "Implemented",
		IsActive:    true,
		NetworkType: networkType,
		Gateway:     "10.0.0.1",
		CIDR:        "10.0.0.0/24",
		ZoneUUID:    "zone-uuid-1",
	}
}

// TestNetworkList verifies URL path, required zoneUuid param, and response parsing.
func TestNetworkList(t *testing.T) {
	networks := []network.Network{
		makeNetwork("net-1", "web-network", "Isolated"),
		makeNetwork("net-2", "db-network", "Isolated"),
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/restapi/network/networkList" {
			http.NotFound(w, r)
			return
		}
		zoneUUID := r.URL.Query().Get("zoneUuid")
		if zoneUUID == "" {
			http.Error(w, "zoneUuid required", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(listNetworkResponse{
			Count:               len(networks),
			ListNetworkResponse: networks,
		})
	}))
	defer srv.Close()

	svc := network.NewService(newClient(srv.URL))

	result, err := svc.List(context.Background(), "zone-uuid-1", "")
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(result) != 2 {
		t.Fatalf("List() returned %d networks, want 2", len(result))
	}
	if result[0].UUID != "net-1" {
		t.Errorf("result[0].UUID = %q, want %q", result[0].UUID, "net-1")
	}
	if result[1].Name != "db-network" {
		t.Errorf("result[1].Name = %q, want %q", result[1].Name, "db-network")
	}
}

// TestNetworkCreate verifies POST body and response parsing.
func TestNetworkCreate(t *testing.T) {
	created := makeNetwork("new-net-1", "my-network", "Isolated")

	var gotBody map[string]interface{}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "expected POST", http.StatusMethodNotAllowed)
			return
		}
		if r.URL.Path != "/restapi/network/createNetwork" {
			http.NotFound(w, r)
			return
		}
		json.NewDecoder(r.Body).Decode(&gotBody)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(listNetworkResponse{
			Count:               1,
			ListNetworkResponse: []network.Network{created},
		})
	}))
	defer srv.Close()

	svc := network.NewService(newClient(srv.URL))

	req := network.CreateRequest{
		Name:                "my-network",
		ZoneUUID:            "zone-1",
		NetworkOfferingUUID: "offering-1",
	}

	net, err := svc.Create(context.Background(), req)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if net.UUID != "new-net-1" {
		t.Errorf("net.UUID = %q, want %q", net.UUID, "new-net-1")
	}
	if gotBody["name"] != "my-network" {
		t.Errorf("body[name] = %v, want %q", gotBody["name"], "my-network")
	}
	if gotBody["zoneUuid"] != "zone-1" {
		t.Errorf("body[zoneUuid] = %v, want %q", gotBody["zoneUuid"], "zone-1")
	}
	if gotBody["networkOfferingUuid"] != "offering-1" {
		t.Errorf("body[networkOfferingUuid] = %v, want %q", gotBody["networkOfferingUuid"], "offering-1")
	}
}

// TestNetworkDelete verifies DELETE path includes uuid.
func TestNetworkDelete(t *testing.T) {
	var gotPath string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			http.Error(w, "expected DELETE", http.StatusMethodNotAllowed)
			return
		}
		gotPath = r.URL.Path
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	svc := network.NewService(newClient(srv.URL))

	err := svc.Delete(context.Background(), "net-del-1")
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if gotPath != "/restapi/network/deleteNetwork/net-del-1" {
		t.Errorf("path = %q, want %q", gotPath, "/restapi/network/deleteNetwork/net-del-1")
	}
}

// TestNetworkGet verifies uuid param is sent and a single result is returned.
func TestNetworkGet(t *testing.T) {
	expected := makeNetwork("net-99", "target-network", "Isolated")

	var gotUUID string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/restapi/network/networkId" {
			http.NotFound(w, r)
			return
		}
		gotUUID = r.URL.Query().Get("uuid")
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(listNetworkResponse{
			Count:               1,
			ListNetworkResponse: []network.Network{expected},
		})
	}))
	defer srv.Close()

	svc := network.NewService(newClient(srv.URL))

	net, err := svc.Get(context.Background(), "zone-1", "net-99")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if gotUUID != "net-99" {
		t.Errorf("uuid query param = %q, want %q", gotUUID, "net-99")
	}
	if net.UUID != "net-99" {
		t.Errorf("net.UUID = %q, want %q", net.UUID, "net-99")
	}
	if net.Name != "target-network" {
		t.Errorf("net.Name = %q, want %q", net.Name, "target-network")
	}
}
