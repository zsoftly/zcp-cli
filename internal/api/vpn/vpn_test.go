package vpn_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/zsoftly/zcp-cli/internal/api/vpn"
	"github.com/zsoftly/zcp-cli/internal/httpclient"
)

func newClient(baseURL string) *httpclient.Client {
	return httpclient.New(httpclient.Options{
		BaseURL:     baseURL,
		BearerToken: "test-token",
		Timeout:     5 * time.Second,
	})
}

// apiEnvelope mirrors the ZCP response envelope used by the services.
type apiEnvelope struct {
	Status string      `json:"status"`
	Data   interface{} `json:"data"`
}

// ---------------------------------------------------------------------------
// CustomerGateway tests
// ---------------------------------------------------------------------------

func TestCustomerGatewayList(t *testing.T) {
	expected := []vpn.CustomerGateway{
		{Slug: "cgw-1", Name: "gw-one", CIDRList: "10.0.0.0/8", IKEPolicy: "aes-128-sha1-modp1536"},
		{Slug: "cgw-2", Name: "gw-two", CIDRList: "192.168.0.0/16", IKEPolicy: "aes-256-sha256-modp2048"},
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/vpn-customer-gateways" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(apiEnvelope{Status: "ok", Data: expected})
	}))
	defer srv.Close()

	svc := vpn.NewCustomerGatewayService(newClient(srv.URL))
	cgws, err := svc.List(context.Background())
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(cgws) != 2 {
		t.Fatalf("List() returned %d customer gateways, want 2", len(cgws))
	}
	if cgws[0].Slug != "cgw-1" {
		t.Errorf("cgws[0].Slug = %q, want %q", cgws[0].Slug, "cgw-1")
	}
	if cgws[1].CIDRList != "192.168.0.0/16" {
		t.Errorf("cgws[1].CIDRList = %q, want %q", cgws[1].CIDRList, "192.168.0.0/16")
	}
}

func TestCustomerGatewayCreate(t *testing.T) {
	created := vpn.CustomerGateway{Slug: "cgw-new", Name: "my-cgw", CIDRList: "10.0.0.0/8"}

	var gotBody map[string]interface{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "expected POST", http.StatusMethodNotAllowed)
			return
		}
		if r.URL.Path != "/vpn-customer-gateways" {
			http.NotFound(w, r)
			return
		}
		json.NewDecoder(r.Body).Decode(&gotBody)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(apiEnvelope{Status: "ok", Data: created})
	}))
	defer srv.Close()

	svc := vpn.NewCustomerGatewayService(newClient(srv.URL))
	req := vpn.CustomerGatewayRequest{
		Name:     "my-cgw",
		Gateway:  "203.0.113.1",
		CIDRList: "10.0.0.0/8",
		IPSecPSK: "s3cr3t",
	}
	result, err := svc.Create(context.Background(), req)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if result.Slug != "cgw-new" {
		t.Errorf("result.Slug = %q, want %q", result.Slug, "cgw-new")
	}
	if gotBody["name"] != "my-cgw" {
		t.Errorf("body name = %v, want %q", gotBody["name"], "my-cgw")
	}
	if gotBody["gateway"] != "203.0.113.1" {
		t.Errorf("body gateway = %v, want %q", gotBody["gateway"], "203.0.113.1")
	}
}

func TestCustomerGatewayUpdate(t *testing.T) {
	updated := vpn.CustomerGateway{Slug: "cgw-1", Name: "updated-cgw", CIDRList: "172.16.0.0/12"}

	var gotBody map[string]interface{}
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			http.Error(w, "expected PUT", http.StatusMethodNotAllowed)
			return
		}
		gotPath = r.URL.Path
		json.NewDecoder(r.Body).Decode(&gotBody)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(apiEnvelope{Status: "ok", Data: updated})
	}))
	defer srv.Close()

	svc := vpn.NewCustomerGatewayService(newClient(srv.URL))
	req := vpn.CustomerGatewayRequest{
		Name:     "updated-cgw",
		CIDRList: "172.16.0.0/12",
	}
	result, err := svc.Update(context.Background(), "cgw-1", req)
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if gotPath != "/vpn-customer-gateways/cgw-1" {
		t.Errorf("path = %q, want %q", gotPath, "/vpn-customer-gateways/cgw-1")
	}
	if result.Slug != "cgw-1" {
		t.Errorf("result.Slug = %q, want %q", result.Slug, "cgw-1")
	}
	if gotBody["name"] != "updated-cgw" {
		t.Errorf("body name = %v, want %q", gotBody["name"], "updated-cgw")
	}
}

func TestCustomerGatewayDelete(t *testing.T) {
	var gotPath, gotMethod string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	svc := vpn.NewCustomerGatewayService(newClient(srv.URL))
	err := svc.Delete(context.Background(), "cgw-del-1")
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if gotMethod != http.MethodDelete {
		t.Errorf("method = %q, want %q", gotMethod, http.MethodDelete)
	}
	if gotPath != "/vpn-customer-gateways/cgw-del-1" {
		t.Errorf("path = %q, want %q", gotPath, "/vpn-customer-gateways/cgw-del-1")
	}
}

func TestCustomerGatewayListError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "server error", http.StatusInternalServerError)
	}))
	defer srv.Close()

	svc := vpn.NewCustomerGatewayService(newClient(srv.URL))
	_, err := svc.List(context.Background())
	if err == nil {
		t.Fatal("List() expected error on 500, got nil")
	}
}

// ---------------------------------------------------------------------------
// User tests
// ---------------------------------------------------------------------------

func TestVPNUserList(t *testing.T) {
	expected := []vpn.User{
		{Slug: "usr-1", UserName: "alice", Status: "Enabled"},
		{Slug: "usr-2", UserName: "bob", Status: "Disabled"},
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/vpn-users" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(apiEnvelope{Status: "ok", Data: expected})
	}))
	defer srv.Close()

	svc := vpn.NewUserService(newClient(srv.URL))
	users, err := svc.List(context.Background())
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(users) != 2 {
		t.Fatalf("List() returned %d users, want 2", len(users))
	}
	if users[0].Slug != "usr-1" {
		t.Errorf("users[0].Slug = %q, want %q", users[0].Slug, "usr-1")
	}
	if users[0].UserName != "alice" {
		t.Errorf("users[0].UserName = %q, want %q", users[0].UserName, "alice")
	}
}

func TestVPNUserCreate(t *testing.T) {
	created := vpn.User{Slug: "usr-new", UserName: "carol", Status: "Enabled"}

	var gotBody map[string]interface{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "expected POST", http.StatusMethodNotAllowed)
			return
		}
		if r.URL.Path != "/vpn-users" {
			http.NotFound(w, r)
			return
		}
		json.NewDecoder(r.Body).Decode(&gotBody)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(apiEnvelope{Status: "ok", Data: created})
	}))
	defer srv.Close()

	svc := vpn.NewUserService(newClient(srv.URL))
	result, err := svc.Create(context.Background(), vpn.UserCreateRequest{
		Username: "carol",
		Password: "p@ssw0rd",
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if result.Slug != "usr-new" {
		t.Errorf("result.Slug = %q, want %q", result.Slug, "usr-new")
	}
	if gotBody["username"] != "carol" {
		t.Errorf("body username = %v, want %q", gotBody["username"], "carol")
	}
	if gotBody["password"] != "p@ssw0rd" {
		t.Errorf("body password = %v, want %q", gotBody["password"], "p@ssw0rd")
	}
}

func TestVPNUserDelete(t *testing.T) {
	var gotPath, gotMethod string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	svc := vpn.NewUserService(newClient(srv.URL))
	err := svc.Delete(context.Background(), "usr-alice")
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if gotMethod != http.MethodDelete {
		t.Errorf("method = %q, want %q", gotMethod, http.MethodDelete)
	}
	if gotPath != "/vpn-users/usr-alice" {
		t.Errorf("path = %q, want %q", gotPath, "/vpn-users/usr-alice")
	}
}

func TestVPNUserDeleteError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "not found", http.StatusNotFound)
	}))
	defer srv.Close()

	svc := vpn.NewUserService(newClient(srv.URL))
	err := svc.Delete(context.Background(), "nonexistent")
	if err == nil {
		t.Fatal("Delete() expected error on 404, got nil")
	}
}

func TestVPNUserListError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "server error", http.StatusInternalServerError)
	}))
	defer srv.Close()

	svc := vpn.NewUserService(newClient(srv.URL))
	_, err := svc.List(context.Background())
	if err == nil {
		t.Fatal("List() expected error on 500, got nil")
	}
}
