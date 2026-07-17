package instance_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/zsoftly/zcp-cli/pkg/api/instance"
	"github.com/zsoftly/zcp-cli/pkg/httpclient"
)

func newClient(baseURL string) *httpclient.Client {
	return httpclient.New(httpclient.Options{
		BaseURL:     baseURL,
		BearerToken: "test-token",
		Timeout:     5 * time.Second,
	})
}

func TestList(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/virtual-machines" {
			t.Errorf("path = %q", r.URL.Path)
		}
		vms := []instance.VirtualMachine{
			{ID: "vm-1", Name: "test-vm", Slug: "test-vm", State: "Running"},
		}
		data, _ := json.Marshal(vms)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "Success", "data": json.RawMessage(data), "total": 1,
		})
	}))
	defer srv.Close()

	svc := instance.NewService(newClient(srv.URL))
	vms, err := svc.List(context.Background(), "", "")
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(vms) != 1 {
		t.Fatalf("got %d VMs, want 1", len(vms))
	}
	if vms[0].Slug != "test-vm" {
		t.Errorf("slug = %q, want %q", vms[0].Slug, "test-vm")
	}
}

func TestListEmptyResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "Success",
			"data":   []interface{}{},
			"total":  0,
		})
	}))
	defer srv.Close()

	svc := instance.NewService(newClient(srv.URL))
	vms, err := svc.List(context.Background(), "", "")
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(vms) != 0 {
		t.Errorf("got %d VMs, want 0", len(vms))
	}
}

func TestListPaginatesAllPages(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/virtual-machines" {
			t.Errorf("path = %q", r.URL.Path)
		}
		page := r.URL.Query().Get("page")
		var vms []instance.VirtualMachine
		switch page {
		case "1", "":
			vms = []instance.VirtualMachine{{ID: "vm-1", Slug: "vm-1"}, {ID: "vm-2", Slug: "vm-2"}}
		case "2":
			vms = []instance.VirtualMachine{{ID: "vm-3", Slug: "vm-3"}}
		default:
			t.Errorf("unexpected page = %q", page)
		}
		data, _ := json.Marshal(vms)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "Success", "data": json.RawMessage(data),
			"current_page": atoiOr(page, 1), "last_page": 2, "total": 3,
		})
	}))
	defer srv.Close()

	svc := instance.NewService(newClient(srv.URL))
	vms, err := svc.List(context.Background(), "", "")
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(vms) != 3 {
		t.Fatalf("got %d VMs across pages, want 3", len(vms))
	}
	if vms[2].Slug != "vm-3" {
		t.Errorf("last slug = %q, want vm-3 (second page not fetched)", vms[2].Slug)
	}
}

func atoiOr(s string, fallback int) int {
	switch s {
	case "1":
		return 1
	case "2":
		return 2
	default:
		return fallback
	}
}

func TestGet(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/virtual-machines/test-vm" {
			t.Errorf("path = %q", r.URL.Path)
		}
		vm := instance.VirtualMachine{ID: "vm-1", Name: "test-vm", Slug: "test-vm", State: "Running"}
		data, _ := json.Marshal(vm)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "Success", "data": json.RawMessage(data),
		})
	}))
	defer srv.Close()

	svc := instance.NewService(newClient(srv.URL))
	vm, err := svc.Get(context.Background(), "test-vm")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if vm.Slug != "test-vm" {
		t.Errorf("slug = %q, want %q", vm.Slug, "test-vm")
	}
}

func TestGetNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  "Error",
			"message": "not found",
		})
	}))
	defer srv.Close()

	svc := instance.NewService(newClient(srv.URL))
	_, err := svc.Get(context.Background(), "nonexistent")
	if err == nil {
		t.Error("expected error for not found")
	}
}
func TestStart(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut || r.URL.Path != "/virtual-machines/test-vm/start" {
			t.Errorf("method=%s path=%s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"status": "Success", "message": "OK"})
	}))
	defer srv.Close()

	svc := instance.NewService(newClient(srv.URL))
	_, err := svc.Start(context.Background(), "test-vm")
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}
}

func TestStop(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut || r.URL.Path != "/virtual-machines/test-vm/stop" {
			t.Errorf("method=%s path=%s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"status": "Success", "message": "OK"})
	}))
	defer srv.Close()

	svc := instance.NewService(newClient(srv.URL))
	_, err := svc.Stop(context.Background(), "test-vm")
	if err != nil {
		t.Fatalf("Stop() error = %v", err)
	}
}

func TestDelete(t *testing.T) {
	var called bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete || r.URL.Path != "/virtual-machines/test-vm" {
			t.Errorf("method=%s path=%s", r.Method, r.URL.Path)
		}
		if got := r.URL.Query().Get("expunge"); got != "" {
			t.Errorf("expunge query param should be absent, got %q", got)
		}
		if got := r.URL.Query().Get("delete_public_ip"); got != "" {
			t.Errorf("delete_public_ip query param should be absent, got %q", got)
		}
		called = true
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  "Success",
			"message": "Virtual machine deleted successfully.",
			"data":    nil,
		})
	}))
	defer srv.Close()

	svc := instance.NewService(newClient(srv.URL))
	if err := svc.Delete(context.Background(), "test-vm", false, false); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if !called {
		t.Error("DELETE request was never made")
	}
}

func TestDelete_Force(t *testing.T) {
	var called bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete || r.URL.Path != "/virtual-machines/test-vm" {
			t.Errorf("method=%s path=%s", r.Method, r.URL.Path)
		}
		if got := r.URL.Query().Get("expunge"); got != "true" {
			t.Errorf("expunge = %q, want %q", got, "true")
		}
		if got := r.URL.Query().Get("delete_public_ip"); got != "true" {
			t.Errorf("delete_public_ip = %q, want %q", got, "true")
		}
		called = true
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  "Success",
			"message": "Virtual machine deleted successfully.",
			"data":    nil,
		})
	}))
	defer srv.Close()

	svc := instance.NewService(newClient(srv.URL))
	if err := svc.Delete(context.Background(), "test-vm", true, true); err != nil {
		t.Fatalf("Delete(force) error = %v", err)
	}
	if !called {
		t.Error("DELETE request was never made")
	}
}

func TestDelete_DeletePublicIP(t *testing.T) {
	var called bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete || r.URL.Path != "/virtual-machines/test-vm" {
			t.Errorf("method=%s path=%s", r.Method, r.URL.Path)
		}
		if got := r.URL.Query().Get("delete_public_ip"); got != "true" {
			t.Errorf("delete_public_ip = %q, want %q", got, "true")
		}
		if got := r.URL.Query().Get("expunge"); got != "" {
			t.Errorf("expunge query param should be absent, got %q", got)
		}
		called = true
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  "Success",
			"message": "Virtual machine deleted successfully.",
			"data":    nil,
		})
	}))
	defer srv.Close()

	svc := instance.NewService(newClient(srv.URL))
	if err := svc.Delete(context.Background(), "test-vm", false, true); err != nil {
		t.Fatalf("Delete(deletePublicIP) error = %v", err)
	}
	if !called {
		t.Error("DELETE request was never made")
	}
}

func TestDelete_NotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  "Error",
			"message": "Virtual machine not found.",
		})
	}))
	defer srv.Close()

	svc := instance.NewService(newClient(srv.URL))
	if err := svc.Delete(context.Background(), "nonexistent", false, false); err == nil {
		t.Fatal("Delete() expected error for 404, got nil")
	}
}

// TestGetPublicIPAddress verifies the public IP is selected by ip_type ("Public IP"),
// not by the "type" field (which is the IP version, e.g. "IPv4", for every entry), and
// that "" is returned when the VM has no public IP.
func TestGetPublicIPAddress(t *testing.T) {
	cases := []struct {
		name string
		ips  []instance.IPAddresses
		want string
	}{
		{"none", nil, ""},
		{"non-public entry ignored", []instance.IPAddresses{{IPAddress: "10.0.0.5", Type: "IPv4", IPType: "Private IP"}}, ""},
		{"picks public over an earlier non-public", []instance.IPAddresses{
			{IPAddress: "10.0.0.5", Type: "IPv4", IPType: "Private IP"},
			{IPAddress: "203.0.113.7", Type: "IPv4", IPType: "Public IP"},
		}, "203.0.113.7"},
		{"public source-nat", []instance.IPAddresses{{IPAddress: "203.0.113.9", Type: "IPv4", IPType: "Public IP"}}, "203.0.113.9"},
	}
	for _, c := range cases {
		vm := instance.VirtualMachine{IPAddresses: c.ips}
		if got := vm.GetPublicIPAddress(); got != c.want {
			t.Errorf("%s: GetPublicIPAddress() = %q, want %q", c.name, got, c.want)
		}
	}
}

// TestNetworkPrivateIPEmpty verifies NetworkPrivateIP returns "" (not "-") when the VM
// has no network IP, which `instance ssh` relies on to fall through to other addresses.
func TestNetworkPrivateIPEmpty(t *testing.T) {
	vm := instance.VirtualMachine{}
	if got := vm.NetworkPrivateIP(); got != "" {
		t.Errorf("NetworkPrivateIP() = %q, want \"\" (ssh relies on empty for 'no IP')", got)
	}
}
