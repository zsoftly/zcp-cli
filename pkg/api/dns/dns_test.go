package dns_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/zsoftly/zcp-cli/pkg/api/dns"
	"github.com/zsoftly/zcp-cli/pkg/httpclient"
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
		{ID: "1", Name: "example.com", Slug: "example-com-1", Status: true},
		{ID: "2", Name: "test.org", Slug: "test-org-2", Status: true},
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
		Status:      true,
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

// TestDNSDomainListRealShape decodes the real API JSON shape:
// project_id as a UUID string, status as a bool.
func TestDNSDomainListRealShape(t *testing.T) {
	payload := `{"status":"Success","message":"OK","current_page":1,"total":1,"data":[{"id":"a1dd5370","name":"examdomain.com","slug":"examdomaincom","project_id":"a1c29c89-11ea-4ab1-a3df-c4ebd895cffe","account_id":"a1c29c88","status":true,"created_at":"2026-05-25T14:33:17.000000Z","updated_at":"2026-05-25T14:33:17.000000Z"}]}`

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(payload))
	}))
	defer srv.Close()

	svc := dns.NewService(newClient(srv.URL))
	domains, err := svc.List(context.Background())
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(domains) != 1 {
		t.Fatalf("got %d domains, want 1", len(domains))
	}
	if domains[0].ProjectID != "a1c29c89-11ea-4ab1-a3df-c4ebd895cffe" {
		t.Errorf("ProjectID = %q, want UUID string", domains[0].ProjectID)
	}
	if !domains[0].Status {
		t.Errorf("Status = false, want true")
	}
}

func TestDNSRecordDecodeRRset(t *testing.T) {
	// The live PowerDNS-backed API returns record sets: no id, contents array.
	raw := []byte(`{"name":"www.example.com.","type":"A","ttl":3600,"contents":["192.0.2.10","192.0.2.11"]}`)
	var rec dns.Record
	if err := json.Unmarshal(raw, &rec); err != nil {
		t.Fatalf("Unmarshal RRset: %v", err)
	}
	if rec.Content != "192.0.2.10, 192.0.2.11" {
		t.Errorf("Content = %q, want joined contents", rec.Content)
	}
	if len(rec.Contents) != 2 {
		t.Errorf("Contents = %v, want 2 values", rec.Contents)
	}

	// Legacy shape with id/content still decodes.
	raw = []byte(`{"id":"41","name":"www","type":"A","content":"192.0.2.10","ttl":600}`)
	rec = dns.Record{}
	if err := json.Unmarshal(raw, &rec); err != nil {
		t.Fatalf("Unmarshal legacy: %v", err)
	}
	if rec.ID != "41" || rec.Content != "192.0.2.10" {
		t.Errorf("legacy decode = %+v, want id 41 content 192.0.2.10", rec)
	}
}

func TestDNSDeleteRecordByName(t *testing.T) {
	var gotPath, gotName, gotType string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotName = r.URL.Query().Get("name")
		gotType = r.URL.Query().Get("type")
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"status": "Success", "message": "deleted"})
	}))
	defer srv.Close()

	svc := dns.NewService(newClient(srv.URL))
	if err := svc.DeleteRecordByName(context.Background(), "example-com-1", "www.example.com.", "A"); err != nil {
		t.Fatalf("DeleteRecordByName() error = %v", err)
	}
	if gotPath != "/dns/domains/example-com-1/records" {
		t.Errorf("path = %q", gotPath)
	}
	if gotName != "www.example.com." || gotType != "A" {
		t.Errorf("query = name %q type %q, want www.example.com. / A", gotName, gotType)
	}
}

func TestCanonicalRecordFQDN(t *testing.T) {
	cases := []struct{ name, zone, want string }{
		{"www", "example.com", "www.example.com."},
		{"www", "example.com.", "www.example.com."},
		{"WWW.Example.com.", "example.com", "www.example.com."},
		{"www.example.com", "example.com", "www.example.com."},
		{"@", "example.com", "example.com."},
		{"", "example.com", "example.com."},
		{"example.com", "example.com", "example.com."},
	}
	for _, c := range cases {
		if got := dns.CanonicalRecordFQDN(c.name, c.zone); got != c.want {
			t.Errorf("CanonicalRecordFQDN(%q, %q) = %q, want %q", c.name, c.zone, got, c.want)
		}
	}
}
