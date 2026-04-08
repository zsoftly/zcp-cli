package portforward_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/zsoftly/zcp-cli/internal/api/portforward"
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
		if r.URL.Path != "/ipaddresses/1.2.3.4/port-forwarding-rules" {
			t.Errorf("path = %q", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "Success",
			"data": []map[string]interface{}{
				{"id": "pf-1", "protocol": "tcp", "public_start_port": "8080", "private_start_port": "80"},
			},
		})
	}))
	defer srv.Close()

	svc := portforward.NewService(newClient(srv.URL))
	rules, err := svc.List(context.Background(), "1.2.3.4")
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(rules) != 1 {
		t.Fatalf("got %d rules, want 1", len(rules))
	}
	if rules[0].ID != "pf-1" {
		t.Errorf("ID = %q, want %q", rules[0].ID, "pf-1")
	}
}

func TestCreate(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %q", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "Success",
			"data":   map[string]interface{}{"id": "pf-new", "protocol": "tcp"},
		})
	}))
	defer srv.Close()

	svc := portforward.NewService(newClient(srv.URL))
	rule, err := svc.Create(context.Background(), "1.2.3.4", portforward.CreateRequest{Protocol: "tcp"})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if rule.ID != "pf-new" {
		t.Errorf("ID = %q, want %q", rule.ID, "pf-new")
	}
}

func TestDelete(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("method = %q", r.Method)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	svc := portforward.NewService(newClient(srv.URL))
	err := svc.Delete(context.Background(), "1.2.3.4", "pf-1")
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
}
