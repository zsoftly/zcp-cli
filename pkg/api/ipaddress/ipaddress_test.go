package ipaddress_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/zsoftly/zcp-cli/pkg/api/ipaddress"
	"github.com/zsoftly/zcp-cli/pkg/httpclient"
)

func newClient(baseURL string) *httpclient.Client {
	return httpclient.New(httpclient.Options{
		BaseURL:     baseURL,
		BearerToken: "test-token",
		Timeout:     5 * time.Second,
	})
}

type listResponse struct {
	Status  string                `json:"status"`
	Message string                `json:"message"`
	Data    []ipaddress.IPAddress `json:"data"`
}

type singleResponse struct {
	Status  string              `json:"status"`
	Message string              `json:"message"`
	Data    ipaddress.IPAddress `json:"data"`
}

type vpnListResponse struct {
	Status  string                      `json:"status"`
	Message string                      `json:"message"`
	Data    []ipaddress.RemoteAccessVPN `json:"data"`
}

type vpnSingleResponse struct {
	Status  string                    `json:"status"`
	Message string                    `json:"message"`
	Data    ipaddress.RemoteAccessVPN `json:"data"`
}

func TestIPList(t *testing.T) {
	expected := []ipaddress.IPAddress{
		{ID: "id-1", Slug: "1030011", IPAddress: "103.0.0.11", Strategy: "SOURCE-NAT"},
		{ID: "id-2", Slug: "1030012", IPAddress: "103.0.0.12", Strategy: "STATIC-NAT"},
	}

	var gotVPC string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/ipaddresses" {
			http.NotFound(w, r)
			return
		}
		gotVPC = r.URL.Query().Get("filter[vpc]")
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(listResponse{Status: "Success", Data: expected})
	}))
	defer srv.Close()

	svc := ipaddress.NewService(newClient(srv.URL))
	ips, err := svc.List(context.Background(), "my-vpc", "", "")
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(ips) != 2 {
		t.Fatalf("List() returned %d IPs, want 2", len(ips))
	}
	if gotVPC != "my-vpc" {
		t.Errorf("filter[vpc] query param = %q, want %q", gotVPC, "my-vpc")
	}
	if ips[0].ID != "id-1" {
		t.Errorf("ips[0].ID = %q, want %q", ips[0].ID, "id-1")
	}
	if ips[0].Strategy != "SOURCE-NAT" {
		t.Errorf("ips[0].Strategy = %q, want %q", ips[0].Strategy, "SOURCE-NAT")
	}
}

func TestIPListNoFilter(t *testing.T) {
	expected := []ipaddress.IPAddress{
		{ID: "id-1", Slug: "1030011", IPAddress: "103.0.0.11"},
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/ipaddresses" {
			http.NotFound(w, r)
			return
		}
		if r.URL.Query().Get("filter[vpc]") != "" {
			t.Error("expected no filter[vpc] query param")
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(listResponse{Status: "Success", Data: expected})
	}))
	defer srv.Close()

	svc := ipaddress.NewService(newClient(srv.URL))
	ips, err := svc.List(context.Background(), "", "", "")
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(ips) != 1 {
		t.Fatalf("List() returned %d IPs, want 1", len(ips))
	}
}

func TestIPAllocate(t *testing.T) {
	allocated := ipaddress.IPAddress{
		ID:        "id-new",
		Slug:      "10300113",
		IPAddress: "103.0.0.113",
		RegionID:  "region-1",
	}

	var gotBody map[string]interface{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "expected POST", http.StatusMethodNotAllowed)
			return
		}
		if r.URL.Path != "/ipaddresses" {
			http.NotFound(w, r)
			return
		}
		json.NewDecoder(r.Body).Decode(&gotBody)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(singleResponse{Status: "Success", Data: allocated})
	}))
	defer srv.Close()

	svc := ipaddress.NewService(newClient(srv.URL))
	ip, err := svc.Allocate(context.Background(), ipaddress.CreateRequest{
		Plan:         "ip-plan",
		BillingCycle: "hourly",
		Network:      "my-network",
	})
	if err != nil {
		t.Fatalf("Allocate() error = %v", err)
	}
	if ip.ID != "id-new" {
		t.Errorf("ip.ID = %q, want %q", ip.ID, "id-new")
	}
	if gotBody["plan"] != "ip-plan" {
		t.Errorf("body plan = %v, want %q", gotBody["plan"], "ip-plan")
	}
	if gotBody["billing_cycle"] != "hourly" {
		t.Errorf("body billing_cycle = %v, want %q", gotBody["billing_cycle"], "hourly")
	}
	if gotBody["network"] != "my-network" {
		t.Errorf("body network = %v, want %q", gotBody["network"], "my-network")
	}
}

func TestIPEnableStaticNAT(t *testing.T) {
	natResult := ipaddress.IPAddress{
		ID:                 "id-1",
		Slug:               "1030011",
		IPAddress:          "103.0.0.11",
		Strategy:           "STATIC-NAT",
		VirtualMachineName: "my-vm",
	}

	var gotBody map[string]interface{}
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "expected POST", http.StatusMethodNotAllowed)
			return
		}
		gotPath = r.URL.Path
		json.NewDecoder(r.Body).Decode(&gotBody)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(singleResponse{Status: "Success", Data: natResult})
	}))
	defer srv.Close()

	svc := ipaddress.NewService(newClient(srv.URL))
	result, err := svc.EnableStaticNAT(context.Background(), "1030011", "my-vm", "my-network")
	if err != nil {
		t.Fatalf("EnableStaticNAT() error = %v", err)
	}
	if gotPath != "/ipaddresses/1030011/static-nat" {
		t.Errorf("path = %q, want %q", gotPath, "/ipaddresses/1030011/static-nat")
	}
	if result.Status != "Success" {
		t.Errorf("result.Status = %q, want %q", result.Status, "Success")
	}
	if gotBody["virtual_machine"] != "my-vm" {
		t.Errorf("body virtual_machine = %v, want %q", gotBody["virtual_machine"], "my-vm")
	}
	if gotBody["network"] != "my-network" {
		t.Errorf("body network = %v, want %q", gotBody["network"], "my-network")
	}
}

func TestIPListRemoteAccessVPNs(t *testing.T) {
	expected := []ipaddress.RemoteAccessVPN{
		{ID: "vpn-1", PublicIP: "103.0.0.11", State: "Running"},
	}

	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(vpnListResponse{Status: "Success", Data: expected})
	}))
	defer srv.Close()

	svc := ipaddress.NewService(newClient(srv.URL))
	vpns, err := svc.ListRemoteAccessVPNs(context.Background(), "1030011")
	if err != nil {
		t.Fatalf("ListRemoteAccessVPNs() error = %v", err)
	}
	if gotPath != "/ipaddresses/1030011/remote-access-vpns" {
		t.Errorf("path = %q, want %q", gotPath, "/ipaddresses/1030011/remote-access-vpns")
	}
	if len(vpns) != 1 {
		t.Fatalf("ListRemoteAccessVPNs() returned %d VPNs, want 1", len(vpns))
	}
	if vpns[0].ID != "vpn-1" {
		t.Errorf("vpns[0].ID = %q, want %q", vpns[0].ID, "vpn-1")
	}
}

func TestIPEnableRemoteAccessVPN(t *testing.T) {
	vpn := ipaddress.RemoteAccessVPN{
		ID:       "vpn-new",
		PublicIP: "103.0.0.11",
		State:    "Running",
	}

	var gotPath, gotMethod string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(vpnSingleResponse{Status: "Success", Data: vpn})
	}))
	defer srv.Close()

	svc := ipaddress.NewService(newClient(srv.URL))
	result, err := svc.EnableRemoteAccessVPN(context.Background(), "1030011")
	if err != nil {
		t.Fatalf("EnableRemoteAccessVPN() error = %v", err)
	}
	if gotMethod != http.MethodPost {
		t.Errorf("method = %q, want %q", gotMethod, http.MethodPost)
	}
	if gotPath != "/ipaddresses/1030011/remote-access-vpns" {
		t.Errorf("path = %q, want %q", gotPath, "/ipaddresses/1030011/remote-access-vpns")
	}
	if result.ID != "vpn-new" {
		t.Errorf("result.ID = %q, want %q", result.ID, "vpn-new")
	}
}

// TestIPRelease verifies DELETE /ipaddresses/<slug> is called.
func TestIPRelease(t *testing.T) {
	var gotPath, gotMethod string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	svc := ipaddress.NewService(newClient(srv.URL))
	err := svc.Release(context.Background(), "1036521143")
	if err != nil {
		t.Fatalf("Release() error = %v", err)
	}
	if gotMethod != http.MethodDelete {
		t.Errorf("method = %q, want %q", gotMethod, http.MethodDelete)
	}
	if gotPath != "/ipaddresses/1036521143" {
		t.Errorf("path = %q, want %q", gotPath, "/ipaddresses/1036521143")
	}
}

// TestIPRelease_NotFound verifies a 404 returns an error.
func TestIPRelease_NotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprint(w, `{"status":"Error","message":"IP address not found."}`)
	}))
	defer srv.Close()

	svc := ipaddress.NewService(newClient(srv.URL))
	err := svc.Release(context.Background(), "nonexistent")
	if err == nil {
		t.Fatal("Release() expected error, got nil")
	}
}

func TestIPDisableRemoteAccessVPN(t *testing.T) {
	var gotPath, gotMethod string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	svc := ipaddress.NewService(newClient(srv.URL))
	err := svc.DisableRemoteAccessVPN(context.Background(), "1030011", "vpn-1")
	if err != nil {
		t.Fatalf("DisableRemoteAccessVPN() error = %v", err)
	}
	if gotMethod != http.MethodDelete {
		t.Errorf("method = %q, want %q", gotMethod, http.MethodDelete)
	}
	if gotPath != "/ipaddresses/1030011/remote-access-vpns/vpn-1" {
		t.Errorf("path = %q, want %q", gotPath, "/ipaddresses/1030011/remote-access-vpns/vpn-1")
	}
}

// TestIPAddressListPaginates verifies List walks every page so an IP-by-slug lookup
// (used by `loadbalancer delete --release-ip`) doesn't miss an IP on a later page.
func TestIPAddressListPaginates(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Query().Get("page") == "2" {
			fmt.Fprint(w, `{"status":"Success","current_page":2,"last_page":2,"data":[{"slug":"ip-b","ipaddress":"2.2.2.2"}]}`)
		} else {
			fmt.Fprint(w, `{"status":"Success","current_page":1,"last_page":2,"data":[{"slug":"ip-a","ipaddress":"1.1.1.1"}]}`)
		}
	}))
	defer srv.Close()

	ips, err := ipaddress.NewService(newClient(srv.URL)).List(context.Background(), "", "", "")
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(ips) != 2 {
		t.Fatalf("got %d IPs, want 2 (both pages)", len(ips))
	}
	if ips[0].Slug != "ip-a" || ips[1].Slug != "ip-b" {
		t.Errorf("slugs = %q, %q; want ip-a, ip-b", ips[0].Slug, ips[1].Slug)
	}
}
