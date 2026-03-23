package host_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/zsoftly/zcp-cli/internal/api/host"
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

type listHostResponse struct {
	Count            int         `json:"count"`
	ListHostResponse []host.Host `json:"listHostResponse"`
}

func TestHostList(t *testing.T) {
	expected := []host.Host{
		{UUID: "host-1", Name: "hypervisor-01", Hypervisor: "KVM", CPUCores: 32, VMCount: 10},
		{UUID: "host-2", Name: "hypervisor-02", Hypervisor: "KVM", CPUCores: 64, VMCount: 5},
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/restapi/host/hostList" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(listHostResponse{Count: len(expected), ListHostResponse: expected})
	}))
	defer srv.Close()

	svc := host.NewService(newClient(srv.URL))
	hosts, err := svc.List(context.Background(), "")
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(hosts) != 2 {
		t.Fatalf("List() returned %d hosts, want 2", len(hosts))
	}
	if hosts[0].UUID != "host-1" {
		t.Errorf("hosts[0].UUID = %q, want %q", hosts[0].UUID, "host-1")
	}
	if hosts[0].Hypervisor != "KVM" {
		t.Errorf("hosts[0].Hypervisor = %q, want %q", hosts[0].Hypervisor, "KVM")
	}
}

func TestHostListWithUUID(t *testing.T) {
	expected := []host.Host{
		{UUID: "host-1", Name: "hypervisor-01", Hypervisor: "KVM", CPUCores: 32},
	}

	var gotUUID string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUUID = r.URL.Query().Get("uuid")
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(listHostResponse{Count: 1, ListHostResponse: expected})
	}))
	defer srv.Close()

	svc := host.NewService(newClient(srv.URL))
	hosts, err := svc.List(context.Background(), "host-1")
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(hosts) != 1 {
		t.Fatalf("List() returned %d hosts, want 1", len(hosts))
	}
	if gotUUID != "host-1" {
		t.Errorf("uuid query param = %q, want %q", gotUUID, "host-1")
	}
}

func TestHostListNoUUIDParam(t *testing.T) {
	var gotUUID string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUUID = r.URL.Query().Get("uuid")
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(listHostResponse{Count: 0, ListHostResponse: nil})
	}))
	defer srv.Close()

	svc := host.NewService(newClient(srv.URL))
	_, err := svc.List(context.Background(), "")
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if gotUUID != "" {
		t.Errorf("uuid query param should be empty, got %q", gotUUID)
	}
}

func TestHostListError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "forbidden", http.StatusForbidden)
	}))
	defer srv.Close()

	svc := host.NewService(newClient(srv.URL))
	_, err := svc.List(context.Background(), "")
	if err == nil {
		t.Fatal("List() expected error on 403, got nil")
	}
}
