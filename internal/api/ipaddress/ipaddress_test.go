package ipaddress_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/zsoftly/zcp-cli/internal/api/ipaddress"
	"github.com/zsoftly/zcp-cli/internal/httpclient"
)

func newClient(baseURL string) *httpclient.Client {
	return httpclient.New(httpclient.Options{
		BaseURL:   baseURL,
		APIKey:    "testkey",
		SecretKey: "testsecret",
		Timeout:   5 * time.Second,
	})
}

type listIPAddressResponse struct {
	Count                 int                   `json:"count"`
	ListIpAddressResponse []ipaddress.IPAddress `json:"listIpAddressResponse"`
}

type enableStaticNATResponse struct {
	Count                       int                         `json:"count"`
	KongAttachStaticNatResponse []ipaddress.StaticNATConfig `json:"kongAttachStaticNatResponse"`
}

func TestIPList(t *testing.T) {
	expected := []ipaddress.IPAddress{
		{UUID: "ip-1", PublicIPAddress: "203.0.113.1", ZoneUUID: "zone-1", State: "Allocated"},
		{UUID: "ip-2", PublicIPAddress: "203.0.113.2", ZoneUUID: "zone-1", State: "Allocated"},
	}

	var gotZone string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/restapi/ipaddress/ipAddressList" {
			http.NotFound(w, r)
			return
		}
		gotZone = r.URL.Query().Get("zoneUuid")
		if gotZone == "" {
			http.Error(w, "zoneUuid required", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(listIPAddressResponse{Count: len(expected), ListIpAddressResponse: expected})
	}))
	defer srv.Close()

	svc := ipaddress.NewService(newClient(srv.URL))
	ips, err := svc.List(context.Background(), "zone-1", "")
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(ips) != 2 {
		t.Fatalf("List() returned %d IPs, want 2", len(ips))
	}
	if gotZone != "zone-1" {
		t.Errorf("zoneUuid query param = %q, want %q", gotZone, "zone-1")
	}
	if ips[0].UUID != "ip-1" {
		t.Errorf("ips[0].UUID = %q, want %q", ips[0].UUID, "ip-1")
	}
}

func TestIPAcquire(t *testing.T) {
	acquired := ipaddress.IPAddress{
		UUID:            "ip-new",
		PublicIPAddress: "203.0.113.10",
		ZoneUUID:        "zone-1",
		NetworkUUID:     "net-1",
		State:           "Allocated",
	}

	var gotNetwork, gotNetworkType string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/restapi/ipaddress/acquireIpAddress" {
			http.NotFound(w, r)
			return
		}
		gotNetwork = r.URL.Query().Get("networkUuid")
		gotNetworkType = r.URL.Query().Get("networkType")
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(listIPAddressResponse{Count: 1, ListIpAddressResponse: []ipaddress.IPAddress{acquired}})
	}))
	defer srv.Close()

	svc := ipaddress.NewService(newClient(srv.URL))
	ip, err := svc.Acquire(context.Background(), "net-1", "Isolated")
	if err != nil {
		t.Fatalf("Acquire() error = %v", err)
	}
	if ip.UUID != "ip-new" {
		t.Errorf("ip.UUID = %q, want %q", ip.UUID, "ip-new")
	}
	if gotNetwork != "net-1" {
		t.Errorf("networkUuid query param = %q, want %q", gotNetwork, "net-1")
	}
	if gotNetworkType != "Isolated" {
		t.Errorf("networkType query param = %q, want %q", gotNetworkType, "Isolated")
	}
}

func TestIPRelease(t *testing.T) {
	var gotPath, gotMethod, gotUUID string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		gotUUID = r.URL.Query().Get("uuid")
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	svc := ipaddress.NewService(newClient(srv.URL))
	err := svc.Release(context.Background(), "ip-del-1")
	if err != nil {
		t.Fatalf("Release() error = %v", err)
	}
	if gotMethod != http.MethodDelete {
		t.Errorf("method = %q, want %q", gotMethod, http.MethodDelete)
	}
	if gotPath != "/restapi/ipaddress/releaseIpAddress" {
		t.Errorf("path = %q, want %q", gotPath, "/restapi/ipaddress/releaseIpAddress")
	}
	if gotUUID != "ip-del-1" {
		t.Errorf("uuid query param = %q, want %q", gotUUID, "ip-del-1")
	}
}

func TestIPEnableStaticNAT(t *testing.T) {
	natConfig := ipaddress.StaticNATConfig{
		IPAddressUUID: "ip-1",
		VMUUID:        "vm-1",
		NetworkUUID:   "net-1",
		IsActive:      true,
		Status:        "enabled",
	}

	var gotBody map[string]interface{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "expected POST", http.StatusMethodNotAllowed)
			return
		}
		if r.URL.Path != "/restapi/ipaddress/enableStaticNat" {
			http.NotFound(w, r)
			return
		}
		json.NewDecoder(r.Body).Decode(&gotBody)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(enableStaticNATResponse{Count: 1, KongAttachStaticNatResponse: []ipaddress.StaticNATConfig{natConfig}})
	}))
	defer srv.Close()

	svc := ipaddress.NewService(newClient(srv.URL))
	result, err := svc.EnableStaticNAT(context.Background(), "ip-1", "vm-1", "net-1")
	if err != nil {
		t.Fatalf("EnableStaticNAT() error = %v", err)
	}
	if result.IPAddressUUID != "ip-1" {
		t.Errorf("result.IPAddressUUID = %q, want %q", result.IPAddressUUID, "ip-1")
	}
	if gotBody["ipAddressUuid"] != "ip-1" {
		t.Errorf("body ipAddressUuid = %v, want %q", gotBody["ipAddressUuid"], "ip-1")
	}
	if gotBody["vmUuid"] != "vm-1" {
		t.Errorf("body vmUuid = %v, want %q", gotBody["vmUuid"], "vm-1")
	}
	if gotBody["networkUuid"] != "net-1" {
		t.Errorf("body networkUuid = %v, want %q", gotBody["networkUuid"], "net-1")
	}
}

func TestIPDisableStaticNAT(t *testing.T) {
	var gotPath, gotMethod, gotIPUUID string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		gotIPUUID = r.URL.Query().Get("ipAddressUuid")
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	svc := ipaddress.NewService(newClient(srv.URL))
	err := svc.DisableStaticNAT(context.Background(), "ip-1")
	if err != nil {
		t.Fatalf("DisableStaticNAT() error = %v", err)
	}
	if gotMethod != http.MethodDelete {
		t.Errorf("method = %q, want %q", gotMethod, http.MethodDelete)
	}
	if gotPath != "/restapi/ipaddress/disableStaticNat" {
		t.Errorf("path = %q, want %q", gotPath, "/restapi/ipaddress/disableStaticNat")
	}
	if gotIPUUID != "ip-1" {
		t.Errorf("ipAddressUuid query param = %q, want %q", gotIPUUID, "ip-1")
	}
}
