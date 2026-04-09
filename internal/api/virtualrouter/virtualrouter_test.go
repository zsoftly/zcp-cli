package virtualrouter_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/zsoftly/zcp-cli/internal/api/virtualrouter"
	"github.com/zsoftly/zcp-cli/internal/httpclient"
)

func newClient(baseURL string) *httpclient.Client {
	return httpclient.New(httpclient.Options{
		BaseURL:     baseURL,
		BearerToken: "test-token",
		Timeout:     5 * time.Second,
	})
}

func makeRouter(slug, name string) virtualrouter.VirtualRouter {
	return virtualrouter.VirtualRouter{
		ID:       "1",
		Slug:     slug,
		Name:     name,
		Status:   "Running",
		State:    "Running",
		Gateway:  "10.0.0.1",
		PublicIP: "203.0.113.10",
		GuestIP:  "10.0.0.1",
		ZoneSlug: "yow-1",
		Role:     "VIRTUAL_ROUTER",
	}
}

func TestVirtualRouterList(t *testing.T) {
	routers := []virtualrouter.VirtualRouter{
		makeRouter("router-1", "vr-web"),
		makeRouter("router-2", "vr-db"),
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/virtual-routers" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "Success",
			"data":   routers,
		})
	}))
	defer srv.Close()

	svc := virtualrouter.NewService(newClient(srv.URL))
	result, err := svc.List(context.Background())
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(result) != 2 {
		t.Fatalf("List() returned %d routers, want 2", len(result))
	}
	if result[0].Slug != "router-1" {
		t.Errorf("result[0].Slug = %q, want %q", result[0].Slug, "router-1")
	}
	if result[1].Name != "vr-db" {
		t.Errorf("result[1].Name = %q, want %q", result[1].Name, "vr-db")
	}
}

func TestVirtualRouterCreate(t *testing.T) {
	created := makeRouter("new-router", "my-router")

	var gotBody map[string]interface{}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "expected POST", http.StatusMethodNotAllowed)
			return
		}
		if r.URL.Path != "/virtual-routers" {
			http.NotFound(w, r)
			return
		}
		json.NewDecoder(r.Body).Decode(&gotBody)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "Success",
			"data":   created,
		})
	}))
	defer srv.Close()

	svc := virtualrouter.NewService(newClient(srv.URL))
	req := virtualrouter.CreateRequest{
		Name:        "my-router",
		NetworkSlug: "web-network",
	}
	vr, err := svc.Create(context.Background(), req)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if vr.Slug != "new-router" {
		t.Errorf("vr.Slug = %q, want %q", vr.Slug, "new-router")
	}
	if gotBody["vr_name"] != "my-router" {
		t.Errorf("body[name] = %v, want %q", gotBody["vr_name"], "my-router")
	}
	if gotBody["network_slug"] != "web-network" {
		t.Errorf("body[network_slug] = %v, want %q", gotBody["network_slug"], "web-network")
	}
}

func TestVirtualRouterReboot(t *testing.T) {
	rebooted := makeRouter("router-1", "vr-web")
	rebooted.Status = "Running"

	var gotPath string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "Success",
			"data":   rebooted,
		})
	}))
	defer srv.Close()

	svc := virtualrouter.NewService(newClient(srv.URL))
	vr, err := svc.Reboot(context.Background(), "router-1")
	if err != nil {
		t.Fatalf("Reboot() error = %v", err)
	}
	if gotPath != "/virtual-routers/router-1/reboot" {
		t.Errorf("path = %q, want %q", gotPath, "/virtual-routers/router-1/reboot")
	}
	if vr.Status != "Running" {
		t.Errorf("vr.Status = %q, want %q", vr.Status, "Running")
	}
}
