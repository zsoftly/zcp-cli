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
		BaseURL:     baseURL,
		BearerToken: "test-token",
		Timeout:     5 * time.Second,
	})
}

type envelope struct {
	Status  string      `json:"status"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
	Total   int         `json:"total,omitempty"`
}

func TestLoadBalancerList(t *testing.T) {
	expected := []loadbalancer.LoadBalancer{
		{
			ID:    "lb-1",
			Name:  "web-lb",
			Slug:  "web-lb",
			State: "Running",
			IPAddress: &loadbalancer.IPAddress{
				ID:        "ip-1",
				IPAddress: "1.2.3.4",
				Slug:      "ip-1",
			},
			Region: &loadbalancer.Region{
				ID:   "region-1",
				Name: "US East",
				Slug: "us-east",
			},
		},
		{
			ID:    "lb-2",
			Name:  "ssl-lb",
			Slug:  "ssl-lb",
			State: "Running",
		},
	}

	var gotInclude string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/load-balancers" {
			http.NotFound(w, r)
			return
		}
		if r.Method != http.MethodGet {
			http.Error(w, "expected GET", http.StatusMethodNotAllowed)
			return
		}
		gotInclude = r.URL.Query().Get("include")
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(envelope{Status: "Success", Message: "OK", Data: expected, Total: len(expected)})
	}))
	defer srv.Close()

	svc := loadbalancer.NewService(newClient(srv.URL))
	lbs, err := svc.List(context.Background())
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(lbs) != 2 {
		t.Fatalf("List() returned %d load balancers, want 2", len(lbs))
	}
	if gotInclude == "" {
		t.Error("expected include query parameter to be set")
	}
	if lbs[0].Slug != "web-lb" {
		t.Errorf("lbs[0].Slug = %q, want %q", lbs[0].Slug, "web-lb")
	}
	if lbs[0].IPAddress == nil || lbs[0].IPAddress.IPAddress != "1.2.3.4" {
		t.Errorf("lbs[0].IPAddress.IPAddress = %v, want %q", lbs[0].IPAddress, "1.2.3.4")
	}
}

func TestLoadBalancerListEmpty(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(envelope{Status: "Success", Message: "OK", Data: []loadbalancer.LoadBalancer{}, Total: 0})
	}))
	defer srv.Close()

	svc := loadbalancer.NewService(newClient(srv.URL))
	lbs, err := svc.List(context.Background())
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(lbs) != 0 {
		t.Fatalf("List() returned %d load balancers, want 0", len(lbs))
	}
}

func TestLoadBalancerCreate(t *testing.T) {
	created := loadbalancer.LoadBalancer{
		ID:    "lb-new",
		Name:  "my-lb",
		Slug:  "my-lb",
		State: "Creating",
	}

	var gotBody map[string]interface{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "expected POST", http.StatusMethodNotAllowed)
			return
		}
		if r.URL.Path != "/load-balancers" {
			http.NotFound(w, r)
			return
		}
		json.NewDecoder(r.Body).Decode(&gotBody)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(envelope{Status: "Success", Message: "OK", Data: created})
	}))
	defer srv.Close()

	svc := loadbalancer.NewService(newClient(srv.URL))
	req := loadbalancer.CreateRequest{
		Name:          "my-lb",
		CloudProvider: "nimbo",
		Project:       "default-33",
		Region:        "ixg-belagavi",
		Network:       "d-net-test",
		Plan:          "load-balancer",
		BillingCycle:  "hourly",
		AcquireNewIP:  true,
		Rules:         []loadbalancer.CreateRuleSpec{},
	}
	result, err := svc.Create(context.Background(), req)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if result.Slug != "my-lb" {
		t.Errorf("result.Slug = %q, want %q", result.Slug, "my-lb")
	}
	if gotBody["name"] != "my-lb" {
		t.Errorf("body name = %v, want %q", gotBody["name"], "my-lb")
	}
	if gotBody["cloud_provider"] != "nimbo" {
		t.Errorf("body cloud_provider = %v, want %q", gotBody["cloud_provider"], "nimbo")
	}
	if gotBody["billing_cycle"] != "hourly" {
		t.Errorf("body billing_cycle = %v, want %q", gotBody["billing_cycle"], "hourly")
	}
	if gotBody["aquire_new_ip"] != true {
		t.Errorf("body aquire_new_ip = %v, want true", gotBody["aquire_new_ip"])
	}
}

func TestLoadBalancerCreateRule(t *testing.T) {
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
		json.NewEncoder(w).Encode(envelope{Status: "Success", Message: "OK", Data: nil})
	}))
	defer srv.Close()

	svc := loadbalancer.NewService(newClient(srv.URL))
	req := loadbalancer.CreateRuleRequest{
		Rules: []loadbalancer.CreateRuleSpec{
			{
				Name:            "web-rule",
				PublicPort:      "80",
				PrivatePort:     "8080",
				Protocol:        "tcp",
				Algorithm:       "roundrobin",
				VirtualMachines: []loadbalancer.VMAttachment{},
			},
		},
	}
	err := svc.CreateRule(context.Background(), "my-lb", req)
	if err != nil {
		t.Fatalf("CreateRule() error = %v", err)
	}
	if gotPath != "/load-balancers/my-lb/load-balancer-rules" {
		t.Errorf("path = %q, want %q", gotPath, "/load-balancers/my-lb/load-balancer-rules")
	}
	rules, ok := gotBody["rules"].([]interface{})
	if !ok || len(rules) != 1 {
		t.Fatalf("body rules length = %v, want 1", gotBody["rules"])
	}
	rule := rules[0].(map[string]interface{})
	if rule["name"] != "web-rule" {
		t.Errorf("rule name = %v, want %q", rule["name"], "web-rule")
	}
	if rule["algorithm"] != "roundrobin" {
		t.Errorf("rule algorithm = %v, want %q", rule["algorithm"], "roundrobin")
	}
}

func TestLoadBalancerAttachVM(t *testing.T) {
	var gotBody map[string]interface{}
	var gotPath, gotMethod string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		json.NewDecoder(r.Body).Decode(&gotBody)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(envelope{Status: "Success", Message: "OK", Data: nil})
	}))
	defer srv.Close()

	svc := loadbalancer.NewService(newClient(srv.URL))
	req := loadbalancer.AttachVMRequest{
		VirtualMachines: []string{"vm-slug-1", "vm-slug-2"},
		CloudProvider:   "nimbo",
		Region:          "ixg-belagavi",
		Project:         "default-33",
	}
	err := svc.AttachVM(context.Background(), "my-lb", "rule-123", req)
	if err != nil {
		t.Fatalf("AttachVM() error = %v", err)
	}
	if gotMethod != http.MethodPost {
		t.Errorf("method = %q, want %q", gotMethod, http.MethodPost)
	}
	if gotPath != "/load-balancers/my-lb/load-balancer-rules/rule-123/attach" {
		t.Errorf("path = %q, want %q", gotPath, "/load-balancers/my-lb/load-balancer-rules/rule-123/attach")
	}
	vms, ok := gotBody["virtual_machines"].([]interface{})
	if !ok || len(vms) != 2 {
		t.Fatalf("body virtual_machines length = %v, want 2", gotBody["virtual_machines"])
	}
	if vms[0] != "vm-slug-1" {
		t.Errorf("virtual_machines[0] = %v, want %q", vms[0], "vm-slug-1")
	}
	if gotBody["cloud_provider"] != "nimbo" {
		t.Errorf("body cloud_provider = %v, want %q", gotBody["cloud_provider"], "nimbo")
	}
}

func TestLoadBalancerListError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "server error", http.StatusInternalServerError)
	}))
	defer srv.Close()

	svc := loadbalancer.NewService(newClient(srv.URL))
	_, err := svc.List(context.Background())
	if err == nil {
		t.Fatal("List() expected error on 500, got nil")
	}
}

func TestLoadBalancerCreateError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "bad request", http.StatusBadRequest)
	}))
	defer srv.Close()

	svc := loadbalancer.NewService(newClient(srv.URL))
	_, err := svc.Create(context.Background(), loadbalancer.CreateRequest{Name: "x"})
	if err == nil {
		t.Fatal("Create() expected error on 400, got nil")
	}
}

func TestLoadBalancerCreateRuleError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "not found", http.StatusNotFound)
	}))
	defer srv.Close()

	svc := loadbalancer.NewService(newClient(srv.URL))
	err := svc.CreateRule(context.Background(), "missing-lb", loadbalancer.CreateRuleRequest{})
	if err == nil {
		t.Fatal("CreateRule() expected error on 404, got nil")
	}
}

func TestLoadBalancerAttachVMError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "not found", http.StatusNotFound)
	}))
	defer srv.Close()

	svc := loadbalancer.NewService(newClient(srv.URL))
	err := svc.AttachVM(context.Background(), "missing-lb", "rule-1", loadbalancer.AttachVMRequest{})
	if err == nil {
		t.Fatal("AttachVM() expected error on 404, got nil")
	}
}
