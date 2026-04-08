package dashboard_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/zsoftly/zcp-cli/internal/api/dashboard"
	"github.com/zsoftly/zcp-cli/internal/httpclient"
)

func newClient(baseURL string) *httpclient.Client {
	return httpclient.New(httpclient.Options{
		BaseURL:     baseURL,
		BearerToken: "test-token",
		Timeout:     5 * time.Second,
	})
}

func TestGetServiceCounts(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/analytics/account/services/counts" {
			http.NotFound(w, r)
			return
		}
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "Success",
			"data": map[string]int{
				"instance":     5,
				"kubernetes":   2,
				"volume":       10,
				"snapshot":     3,
				"network":      4,
				"vpc":          1,
				"publicIp":     6,
				"firewall":     7,
				"loadBalancer": 2,
				"vpn":          1,
				"sshKey":       3,
				"template":     8,
			},
		})
	}))
	defer srv.Close()

	svc := dashboard.NewService(newClient(srv.URL))
	counts, err := svc.GetServiceCounts(context.Background())
	if err != nil {
		t.Fatalf("GetServiceCounts() error = %v", err)
	}

	if counts.Instance != 5 {
		t.Errorf("Instance = %d, want 5", counts.Instance)
	}
	if counts.Kubernetes != 2 {
		t.Errorf("Kubernetes = %d, want 2", counts.Kubernetes)
	}
	if counts.Volume != 10 {
		t.Errorf("Volume = %d, want 10", counts.Volume)
	}
	if counts.PublicIP != 6 {
		t.Errorf("PublicIP = %d, want 6", counts.PublicIP)
	}
	if counts.Template != 8 {
		t.Errorf("Template = %d, want 8", counts.Template)
	}
}

func TestGetServiceCountsAPIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{
			"message": "Invalid credentials",
		})
	}))
	defer srv.Close()

	svc := dashboard.NewService(newClient(srv.URL))
	_, err := svc.GetServiceCounts(context.Background())
	if err == nil {
		t.Fatal("expected error for 401, got nil")
	}
}

func TestGetServiceCountsBadStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "Error",
			"data":   nil,
		})
	}))
	defer srv.Close()

	svc := dashboard.NewService(newClient(srv.URL))
	_, err := svc.GetServiceCounts(context.Background())
	if err == nil {
		t.Fatal("expected error for non-Success status, got nil")
	}
	if !strings.Contains(err.Error(), "unexpected response status") {
		t.Errorf("error = %q, want it to contain 'unexpected response status'", err.Error())
	}
}

func TestCancelService(t *testing.T) {
	var gotPath, gotMethod string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotMethod = r.Method
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "Success",
			"data": map[string]string{
				"message": "Cancellation request submitted",
			},
		})
	}))
	defer srv.Close()

	svc := dashboard.NewService(newClient(srv.URL))
	resp, err := svc.CancelService(context.Background(), "vm-abc-123", "not_needed_anymore")
	if err != nil {
		t.Fatalf("CancelService() error = %v", err)
	}

	if gotMethod != http.MethodPost {
		t.Errorf("method = %s, want POST", gotMethod)
	}
	if gotPath != "/billing/service-cancel-requests/vm-abc-123" {
		t.Errorf("path = %q, want %q", gotPath, "/billing/service-cancel-requests/vm-abc-123")
	}
	if resp.Message != "Cancellation request submitted" {
		t.Errorf("Message = %q, want %q", resp.Message, "Cancellation request submitted")
	}
}

func TestCancelServiceEmptySlug(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("server should not have been called")
	}))
	defer srv.Close()

	svc := dashboard.NewService(newClient(srv.URL))
	_, err := svc.CancelService(context.Background(), "", "test")
	if err == nil {
		t.Fatal("expected error for empty slug, got nil")
	}
	if !strings.Contains(err.Error(), "service slug is required") {
		t.Errorf("error = %q, want it to contain 'service slug is required'", err.Error())
	}
}

func TestCancelServiceAPIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{
			"message": "Service not found",
		})
	}))
	defer srv.Close()

	svc := dashboard.NewService(newClient(srv.URL))
	_, err := svc.CancelService(context.Background(), "nonexistent-slug", "test")
	if err == nil {
		t.Fatal("expected error for 404, got nil")
	}
}
