package portforward_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/zsoftly/zcp-cli/pkg/api/portforward"
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
		if r.URL.Path != "/ipaddresses/1.2.3.4/port-forwarding-rules" {
			t.Errorf("path = %q", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		// The live API returns the start port as public_port/private_port, not
		// public_start_port/private_start_port (verified 2026-07-19).
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "Success",
			"data": []map[string]interface{}{
				{"id": "pf-1", "protocol": "tcp", "public_port": "8080", "public_end_port": "8080", "private_port": "80", "private_end_port": "80"},
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
	// Regression: the ports must decode, not render blank.
	if rules[0].PublicStartPort != "8080" {
		t.Errorf("PublicStartPort = %q, want %q", rules[0].PublicStartPort, "8080")
	}
	if rules[0].PrivateStartPort != "80" {
		t.Errorf("PrivateStartPort = %q, want %q", rules[0].PrivateStartPort, "80")
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

// TestCreateAsyncAck covers the live API's asynchronous create: it returns
// success with data: null and no rule object. Create must not error, and the
// returned rule has an empty ID so the command can report an accepted request
// instead of a blank table.
func TestCreateAsyncAck(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"Success","message":"Creating port forwarding rule.","data":null}`))
	}))
	defer srv.Close()

	svc := portforward.NewService(newClient(srv.URL))
	rule, err := svc.Create(context.Background(), "1.2.3.4", portforward.CreateRequest{Protocol: "tcp"})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if rule.ID != "" {
		t.Errorf("ID = %q, want empty (async ack)", rule.ID)
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

// TestListVMRefString verifies that virtual_machine returned as a plain string
// slug (older API shape) decodes without error and populates Slug.
func TestListVMRefString(t *testing.T) {
	payload := `{"status":"Success","data":[{"id":"pf-3","protocol":"tcp","public_port":"9090","private_port":"90","virtual_machine":"old-vm-slug","state":"Active"}]}`

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(payload))
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
	if rules[0].VirtualMachine.Slug != "old-vm-slug" {
		t.Errorf("VirtualMachine.Slug = %q, want %q", rules[0].VirtualMachine.Slug, "old-vm-slug")
	}
}

// TestListVMRefObject verifies that a list response where virtual_machine is a
// nested object (the real API shape) decodes without error and exposes the slug.
func TestListVMRefObject(t *testing.T) {
	payload := `{"status":"Success","data":[{"id":"pf-2","protocol":"tcp","public_port":"8080","private_port":"80","virtual_machine":{"id":"vm-uuid","slug":"my-vm","name":"My VM"},"state":"Active"}]}`

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(payload))
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
	if rules[0].VirtualMachine.Slug != "my-vm" {
		t.Errorf("VirtualMachine.Slug = %q, want %q", rules[0].VirtualMachine.Slug, "my-vm")
	}
	if rules[0].VirtualMachine.Name != "My VM" {
		t.Errorf("VirtualMachine.Name = %q, want %q", rules[0].VirtualMachine.Name, "My VM")
	}
}
