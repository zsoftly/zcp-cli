package instance_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/zsoftly/zcp-cli/internal/api/instance"
	"github.com/zsoftly/zcp-cli/internal/httpclient"
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
	vms, err := svc.List(context.Background())
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
