package loadbalancer_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/zsoftly/zcp-cli/internal/api/loadbalancer"
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

type listLoadBalancerRuleResponse struct {
	Count                        int                 `json:"count"`
	ListLoadBalancerRuleResponse []loadbalancer.Rule `json:"listLoadBalancerRuleResponse"`
}

func TestLoadBalancerList(t *testing.T) {
	expected := []loadbalancer.Rule{
		{UUID: "lb-1", Name: "web-lb", PublicPort: "80", PrivatePort: "8080", ZoneUUID: "zone-1"},
		{UUID: "lb-2", Name: "ssl-lb", PublicPort: "443", PrivatePort: "8443", ZoneUUID: "zone-1"},
	}

	var gotZone string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/restapi/loadbalancerrule/loadBalancerRuleList" {
			http.NotFound(w, r)
			return
		}
		gotZone = r.URL.Query().Get("zoneUuid")
		if gotZone == "" {
			http.Error(w, "zoneUuid required", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(listLoadBalancerRuleResponse{Count: len(expected), ListLoadBalancerRuleResponse: expected})
	}))
	defer srv.Close()

	svc := loadbalancer.NewService(newClient(srv.URL))
	rules, err := svc.List(context.Background(), "zone-1", "", "")
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(rules) != 2 {
		t.Fatalf("List() returned %d rules, want 2", len(rules))
	}
	if gotZone != "zone-1" {
		t.Errorf("zoneUuid query param = %q, want %q", gotZone, "zone-1")
	}
	if rules[0].UUID != "lb-1" {
		t.Errorf("rules[0].UUID = %q, want %q", rules[0].UUID, "lb-1")
	}
}

func TestLoadBalancerListWithFilters(t *testing.T) {
	var gotUUID, gotIPAddressUUID string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUUID = r.URL.Query().Get("uuid")
		gotIPAddressUUID = r.URL.Query().Get("ipAddressUuid")
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(listLoadBalancerRuleResponse{Count: 0, ListLoadBalancerRuleResponse: nil})
	}))
	defer srv.Close()

	svc := loadbalancer.NewService(newClient(srv.URL))
	_, err := svc.List(context.Background(), "zone-1", "lb-1", "ip-1")
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if gotUUID != "lb-1" {
		t.Errorf("uuid query param = %q, want %q", gotUUID, "lb-1")
	}
	if gotIPAddressUUID != "ip-1" {
		t.Errorf("ipAddressUuid query param = %q, want %q", gotIPAddressUUID, "ip-1")
	}
}

func TestLoadBalancerCreate(t *testing.T) {
	created := loadbalancer.Rule{
		UUID:        "lb-new",
		Name:        "my-lb",
		PublicPort:  "80",
		PrivatePort: "8080",
		Algorithm:   "roundrobin",
	}

	var gotBody map[string]interface{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "expected POST", http.StatusMethodNotAllowed)
			return
		}
		if r.URL.Path != "/restapi/loadbalancerrule/createLoadBalancerRule" {
			http.NotFound(w, r)
			return
		}
		json.NewDecoder(r.Body).Decode(&gotBody)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(listLoadBalancerRuleResponse{Count: 1, ListLoadBalancerRuleResponse: []loadbalancer.Rule{created}})
	}))
	defer srv.Close()

	svc := loadbalancer.NewService(newClient(srv.URL))
	req := loadbalancer.CreateRequest{
		Name:         "my-lb",
		PublicIPUUID: "ip-1",
		PublicPort:   "80",
		PrivatePort:  "8080",
		Algorithm:    "roundrobin",
	}
	result, err := svc.Create(context.Background(), req)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if result.UUID != "lb-new" {
		t.Errorf("result.UUID = %q, want %q", result.UUID, "lb-new")
	}
	if gotBody["name"] != "my-lb" {
		t.Errorf("body name = %v, want %q", gotBody["name"], "my-lb")
	}
	if gotBody["publicIpUuid"] != "ip-1" {
		t.Errorf("body publicIpUuid = %v, want %q", gotBody["publicIpUuid"], "ip-1")
	}
	if gotBody["algorithm"] != "roundrobin" {
		t.Errorf("body algorithm = %v, want %q", gotBody["algorithm"], "roundrobin")
	}
}

func TestLoadBalancerCreateEmptyResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(listLoadBalancerRuleResponse{Count: 0, ListLoadBalancerRuleResponse: nil})
	}))
	defer srv.Close()

	svc := loadbalancer.NewService(newClient(srv.URL))
	_, err := svc.Create(context.Background(), loadbalancer.CreateRequest{Name: "x", PublicIPUUID: "ip-1"})
	if err == nil {
		t.Fatal("Create() expected error on empty response, got nil")
	}
}

func TestLoadBalancerUpdate(t *testing.T) {
	updated := loadbalancer.Rule{
		UUID:      "lb-1",
		Name:      "updated-lb",
		Algorithm: "leastconn",
	}

	var gotBody map[string]interface{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			http.Error(w, "expected PUT", http.StatusMethodNotAllowed)
			return
		}
		if r.URL.Path != "/restapi/loadbalancerrule/updateLoadBalancerRule" {
			http.NotFound(w, r)
			return
		}
		json.NewDecoder(r.Body).Decode(&gotBody)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(listLoadBalancerRuleResponse{Count: 1, ListLoadBalancerRuleResponse: []loadbalancer.Rule{updated}})
	}))
	defer srv.Close()

	svc := loadbalancer.NewService(newClient(srv.URL))
	req := loadbalancer.UpdateRequest{
		UUID:      "lb-1",
		Name:      "updated-lb",
		Algorithm: "leastconn",
	}
	result, err := svc.Update(context.Background(), req)
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if result.UUID != "lb-1" {
		t.Errorf("result.UUID = %q, want %q", result.UUID, "lb-1")
	}
	if gotBody["name"] != "updated-lb" {
		t.Errorf("body name = %v, want %q", gotBody["name"], "updated-lb")
	}
	if gotBody["algorithm"] != "leastconn" {
		t.Errorf("body algorithm = %v, want %q", gotBody["algorithm"], "leastconn")
	}
}

func TestLoadBalancerUpdateEmptyResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(listLoadBalancerRuleResponse{Count: 0, ListLoadBalancerRuleResponse: nil})
	}))
	defer srv.Close()

	svc := loadbalancer.NewService(newClient(srv.URL))
	_, err := svc.Update(context.Background(), loadbalancer.UpdateRequest{UUID: "lb-1", Name: "x"})
	if err == nil {
		t.Fatal("Update() expected error on empty response, got nil")
	}
}

func TestLoadBalancerDelete(t *testing.T) {
	var gotPath, gotMethod string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	svc := loadbalancer.NewService(newClient(srv.URL))
	err := svc.Delete(context.Background(), "lb-del-1")
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if gotMethod != http.MethodDelete {
		t.Errorf("method = %q, want %q", gotMethod, http.MethodDelete)
	}
	if gotPath != "/restapi/loadbalancerrule/deleteLoadBalancerRule/lb-del-1" {
		t.Errorf("path = %q, want %q", gotPath, "/restapi/loadbalancerrule/deleteLoadBalancerRule/lb-del-1")
	}
}

func TestLoadBalancerDeleteError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "not found", http.StatusNotFound)
	}))
	defer srv.Close()

	svc := loadbalancer.NewService(newClient(srv.URL))
	err := svc.Delete(context.Background(), "missing")
	if err == nil {
		t.Fatal("Delete() expected error on 404, got nil")
	}
}

func TestLoadBalancerListError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "server error", http.StatusInternalServerError)
	}))
	defer srv.Close()

	svc := loadbalancer.NewService(newClient(srv.URL))
	_, err := svc.List(context.Background(), "zone-1", "", "")
	if err == nil {
		t.Fatal("List() expected error on 500, got nil")
	}
}
