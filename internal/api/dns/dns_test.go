package dns_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/zsoftly/zcp-cli/internal/api/dns"
	"github.com/zsoftly/zcp-cli/internal/httpclient"
)

func newClient(baseURL string) *httpclient.Client {
	return httpclient.New(httpclient.Options{
		BaseURL:     baseURL,
		BearerToken: "test-token",
		Timeout:     5 * time.Second,
	})
}

func TestDNSDomainList(t *testing.T) {
	expected := []dns.Domain{
		{ID: "1", Name: "example.com", Slug: "example-com-1", Status: "active"},
		{ID: "2", Name: "test.org", Slug: "test-org-2", Status: "active"},
	}

	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		if r.URL.Path != "/dns/domains" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  "Success",
			"message": "OK",
			"data":    expected,
			"total":   len(expected),
		})
	}))
	defer srv.Close()

	svc := dns.NewService(newClient(srv.URL))
	domains, err := svc.List(context.Background())
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if gotPath != "/dns/domains" {
		t.Errorf("path = %q, want %q", gotPath, "/dns/domains")
	}
	if len(domains) != 2 {
		t.Fatalf("List() returned %d domains, want 2", len(domains))
	}
	if domains[0].Name != "example.com" {
		t.Errorf("domains[0].Name = %q, want %q", domains[0].Name, "example.com")
	}
	if domains[1].Slug != "test-org-2" {
		t.Errorf("domains[1].Slug = %q, want %q", domains[1].Slug, "test-org-2")
	}
}

func TestDNSDomainShow(t *testing.T) {
	expected := dns.Domain{
		ID:   "1",
		Name: "example.com",
		Slug: "example-com-1",
		Records: []dns.Record{
			{ID: "10", Name: "www", Type: "A", Content: "192.0.2.1", TTL: 3600},
		},
	}

	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  "Success",
			"message": "OK",
			"data":    expected,
		})
	}))
	defer srv.Close()

	svc := dns.NewService(newClient(srv.URL))
	domain, err := svc.Show(context.Background(), "example-com-1")
	if err != nil {
		t.Fatalf("Show() error = %v", err)
	}
	if gotPath != "/dns/domains/example-com-1" {
		t.Errorf("path = %q, want %q", gotPath, "/dns/domains/example-com-1")
	}
	if domain.Name != "example.com" {
		t.Errorf("domain.Name = %q, want %q", domain.Name, "example.com")
	}
	if len(domain.Records) != 1 {
		t.Fatalf("domain.Records has %d entries, want 1", len(domain.Records))
	}
	if domain.Records[0].Type != "A" {
		t.Errorf("records[0].Type = %q, want %q", domain.Records[0].Type, "A")
	}
}

func TestDNSDomainCreate(t *testing.T) {
	created := dns.Domain{
		ID:          "3",
		Name:        "new.com",
		Slug:        "new-com-3",
		DNSProvider: "powerdns",
		Status:      "active",
	}

	var gotBody map[string]interface{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "expected POST", http.StatusMethodNotAllowed)
			return
		}
		if r.URL.Path != "/dns/domains" {
			http.NotFound(w, r)
			return
		}
		json.NewDecoder(r.Body).Decode(&gotBody)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  "Success",
			"message": "Created",
			"data":    created,
		})
	}))
	defer srv.Close()

	svc := dns.NewService(newClient(srv.URL))
	req := dns.CreateDomainRequest{
		Name:        "new.com",
		Project:     "default-60",
		DNSProvider: "powerdns",
	}
	domain, err := svc.Create(context.Background(), req)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if domain.Name != "new.com" {
		t.Errorf("domain.Name = %q, want %q", domain.Name, "new.com")
	}
	if gotBody["name"] != "new.com" {
		t.Errorf("body name = %v, want %q", gotBody["name"], "new.com")
	}
	if gotBody["dns_provider"] != "powerdns" {
		t.Errorf("body dns_provider = %v, want %q", gotBody["dns_provider"], "powerdns")
	}
}

func TestDNSDomainDelete(t *testing.T) {
	var gotPath, gotMethod string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  "Success",
			"message": "Deleted",
		})
	}))
	defer srv.Close()

	svc := dns.NewService(newClient(srv.URL))
	err := svc.Delete(context.Background(), "example-com-1")
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if gotMethod != http.MethodDelete {
		t.Errorf("method = %q, want %q", gotMethod, http.MethodDelete)
	}
	if gotPath != "/dns/domains/example-com-1" {
		t.Errorf("path = %q, want %q", gotPath, "/dns/domains/example-com-1")
	}
}

func TestDNSRecordCreate(t *testing.T) {
	domainWithRecord := dns.Domain{
		ID:   "1",
		Name: "example.com",
		Slug: "example-com-1",
		Records: []dns.Record{
			{ID: "20", Name: "mail", Type: "MX", Content: "mail.example.com", TTL: 3600},
		},
	}

	var gotPath string
	var gotBody map[string]interface{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		if r.Method != http.MethodPost {
			http.Error(w, "expected POST", http.StatusMethodNotAllowed)
			return
		}
		json.NewDecoder(r.Body).Decode(&gotBody)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  "Success",
			"message": "Created",
			"data":    domainWithRecord,
		})
	}))
	defer srv.Close()

	svc := dns.NewService(newClient(srv.URL))
	req := dns.CreateRecordRequest{
		Name:    "mail",
		Type:    "MX",
		Content: "mail.example.com",
		TTL:     3600,
	}
	domain, err := svc.CreateRecord(context.Background(), "example-com-1", req)
	if err != nil {
		t.Fatalf("CreateRecord() error = %v", err)
	}
	if gotPath != "/dns/domains/example-com-1/records" {
		t.Errorf("path = %q, want %q", gotPath, "/dns/domains/example-com-1/records")
	}
	if domain.Name != "example.com" {
		t.Errorf("domain.Name = %q, want %q", domain.Name, "example.com")
	}
	if gotBody["type"] != "MX" {
		t.Errorf("body type = %v, want %q", gotBody["type"], "MX")
	}
}

func TestDNSRecordDelete(t *testing.T) {
	var gotPath, gotMethod string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  "Success",
			"message": "Deleted",
		})
	}))
	defer srv.Close()

	svc := dns.NewService(newClient(srv.URL))
	err := svc.DeleteRecord(context.Background(), "example-com-1", 42)
	if err != nil {
		t.Fatalf("DeleteRecord() error = %v", err)
	}
	if gotMethod != http.MethodDelete {
		t.Errorf("method = %q, want %q", gotMethod, http.MethodDelete)
	}
	if gotPath != "/dns/domains/example-com-1/records" {
		t.Errorf("path = %q, want %q", gotPath, "/dns/domains/example-com-1/records")
	}
}
