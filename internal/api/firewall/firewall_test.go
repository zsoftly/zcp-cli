package firewall_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/zsoftly/zcp-cli/internal/api/firewall"
	"github.com/zsoftly/zcp-cli/internal/httpclient"
)

func newClient(baseURL string) *httpclient.Client {
	return httpclient.New(httpclient.Options{
		BaseURL:     baseURL,
		BearerToken: "test-token",
		Timeout:     5 * time.Second,
	})
}

type listResponse struct {
	Status  string                  `json:"status"`
	Message string                  `json:"message"`
	Data    []firewall.FirewallRule `json:"data"`
}

type singleResponse struct {
	Status  string                `json:"status"`
	Message string                `json:"message"`
	Data    firewall.FirewallRule `json:"data"`
}

func TestFirewallList(t *testing.T) {
	expected := []firewall.FirewallRule{
		{ID: "fw-1", Protocol: "tcp", StartPort: "80", EndPort: "80", CIDRList: "0.0.0.0/0"},
		{ID: "fw-2", Protocol: "udp", StartPort: "53", EndPort: "53", CIDRList: "10.0.0.0/8"},
	}

	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(listResponse{Status: "Success", Data: expected})
	}))
	defer srv.Close()

	svc := firewall.NewService(newClient(srv.URL))
	rules, err := svc.List(context.Background(), "1030011")
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if gotPath != "/ipaddresses/1030011/firewall-rules" {
		t.Errorf("path = %q, want %q", gotPath, "/ipaddresses/1030011/firewall-rules")
	}
	if len(rules) != 2 {
		t.Fatalf("List() returned %d rules, want 2", len(rules))
	}
	if rules[0].ID != "fw-1" {
		t.Errorf("rules[0].ID = %q, want %q", rules[0].ID, "fw-1")
	}
	if rules[0].StartPort != "80" {
		t.Errorf("rules[0].StartPort = %v, want %q", rules[0].StartPort, "80")
	}
}

func TestFirewallCreate(t *testing.T) {
	created := firewall.FirewallRule{
		ID:        "fw-new",
		Protocol:  "tcp",
		StartPort: "443",
		EndPort:   "443",
		CIDRList:  "0.0.0.0/0",
		State:     "Active",
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
		json.NewEncoder(w).Encode(singleResponse{Status: "Success", Data: created})
	}))
	defer srv.Close()

	svc := firewall.NewService(newClient(srv.URL))
	req := firewall.CreateRequest{
		Protocol:  "tcp",
		StartPort: "443",
		EndPort:   "443",
		CIDRList:  "0.0.0.0/0",
	}
	rule, err := svc.Create(context.Background(), "1030011", req)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if gotPath != "/ipaddresses/1030011/firewall-rules" {
		t.Errorf("path = %q, want %q", gotPath, "/ipaddresses/1030011/firewall-rules")
	}
	if rule.ID != "fw-new" {
		t.Errorf("rule.ID = %q, want %q", rule.ID, "fw-new")
	}
	if gotBody["protocol"] != "tcp" {
		t.Errorf("body protocol = %v, want %q", gotBody["protocol"], "tcp")
	}
	// JSON numbers are float64
	if gotBody["start_port"] != "443" {
		t.Errorf("body start_port = %v, want %q", gotBody["start_port"], "443")
	}
	if gotBody["end_port"] != "443" {
		t.Errorf("body end_port = %v, want %q", gotBody["end_port"], "443")
	}
	if gotBody["cidr_list"] != "0.0.0.0/0" {
		t.Errorf("body cidr_list = %v, want %q", gotBody["cidr_list"], "0.0.0.0/0")
	}
}

func TestFirewallDelete(t *testing.T) {
	var gotPath, gotMethod string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	svc := firewall.NewService(newClient(srv.URL))
	err := svc.Delete(context.Background(), "1030011", "fw-del-1")
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if gotMethod != http.MethodDelete {
		t.Errorf("method = %q, want %q", gotMethod, http.MethodDelete)
	}
	if gotPath != "/ipaddresses/1030011/firewall-rules/fw-del-1" {
		t.Errorf("path = %q, want %q", gotPath, "/ipaddresses/1030011/firewall-rules/fw-del-1")
	}
}
