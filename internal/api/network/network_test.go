package network_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/zsoftly/zcp-cli/internal/api/network"
	"github.com/zsoftly/zcp-cli/internal/httpclient"
)

// helpers

func newClient(baseURL string) *httpclient.Client {
	return httpclient.New(httpclient.Options{
		BaseURL:     baseURL,
		BearerToken: "test-token",
		Timeout:     5 * time.Second,
	})
}

func makeNetwork(slug, name string) network.Network {
	return network.Network{
		ID:          "1",
		Slug:        slug,
		Name:        name,
		Status:      "Active",
		NetworkType: "Isolated",
		Gateway:     "10.0.0.1",
		CIDR:        "10.0.0.0/24",
		ZoneSlug:    "yow-1",
	}
}

// TestNetworkList verifies URL path and response parsing.
func TestNetworkList(t *testing.T) {
	networks := []network.Network{
		makeNetwork("web-network", "web-network"),
		makeNetwork("db-network", "db-network"),
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/networks" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "Success",
			"data":   networks,
		})
	}))
	defer srv.Close()

	svc := network.NewService(newClient(srv.URL))

	result, err := svc.List(context.Background())
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(result) != 2 {
		t.Fatalf("List() returned %d networks, want 2", len(result))
	}
	if result[0].Slug != "web-network" {
		t.Errorf("result[0].Slug = %q, want %q", result[0].Slug, "web-network")
	}
	if result[1].Name != "db-network" {
		t.Errorf("result[1].Name = %q, want %q", result[1].Name, "db-network")
	}
}

// TestNetworkCreate verifies POST body and response parsing.
func TestNetworkCreate(t *testing.T) {
	created := makeNetwork("my-network", "my-network")

	var gotBody map[string]interface{}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "expected POST", http.StatusMethodNotAllowed)
			return
		}
		if r.URL.Path != "/networks" {
			http.NotFound(w, r)
			return
		}
		json.NewDecoder(r.Body).Decode(&gotBody)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "Success",
			"data":   created,
		})
	}))
	defer srv.Close()

	svc := network.NewService(newClient(srv.URL))

	req := network.CreateRequest{
		Name:         "my-network",
		CategorySlug: "default-isolated",
	}

	net, err := svc.Create(context.Background(), req)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if net.Slug != "my-network" {
		t.Errorf("net.Slug = %q, want %q", net.Slug, "my-network")
	}
	if gotBody["name"] != "my-network" {
		t.Errorf("body[name] = %v, want %q", gotBody["name"], "my-network")
	}
	if gotBody["category_slug"] != "default-isolated" {
		t.Errorf("body[category_slug] = %v, want %q", gotBody["category_slug"], "default-isolated")
	}
}

// TestNetworkUpdate verifies PUT path and response parsing.
func TestNetworkUpdate(t *testing.T) {
	updated := makeNetwork("my-network", "renamed-network")
	updated.Name = "renamed-network"

	var gotPath, gotMethod string
	var gotBody map[string]interface{}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotMethod = r.Method
		json.NewDecoder(r.Body).Decode(&gotBody)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "Success",
			"data":   updated,
		})
	}))
	defer srv.Close()

	svc := network.NewService(newClient(srv.URL))

	net, err := svc.Update(context.Background(), "my-network", network.UpdateRequest{
		Name: "renamed-network",
	})
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if gotMethod != http.MethodPut {
		t.Errorf("method = %q, want %q", gotMethod, http.MethodPut)
	}
	if gotPath != "/networks/my-network" {
		t.Errorf("path = %q, want %q", gotPath, "/networks/my-network")
	}
	if net.Name != "renamed-network" {
		t.Errorf("net.Name = %q, want %q", net.Name, "renamed-network")
	}
}

// TestListCategories verifies the network categories endpoint.
func TestListCategories(t *testing.T) {
	categories := []network.Category{
		{ID: "1", Slug: "default-isolated", Name: "Default Isolated"},
		{ID: "2", Slug: "vpc-tier", Name: "VPC Tier"},
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/network/categories" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "Success",
			"data":   categories,
		})
	}))
	defer srv.Close()

	svc := network.NewService(newClient(srv.URL))
	result, err := svc.ListCategories(context.Background())
	if err != nil {
		t.Fatalf("ListCategories() error = %v", err)
	}
	if len(result) != 2 {
		t.Fatalf("ListCategories() returned %d, want 2", len(result))
	}
	if result[0].Slug != "default-isolated" {
		t.Errorf("result[0].Slug = %q, want %q", result[0].Slug, "default-isolated")
	}
}

// TestListEgressRules verifies the egress rules list endpoint.
func TestListEgressRules(t *testing.T) {
	rules := []network.EgressRule{
		{ID: "1", Protocol: "tcp", StartPort: "80", EndPort: "80", Status: "Active"},
		{ID: "2", Protocol: "all", Status: "Active"},
	}

	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "Success",
			"data":   rules,
		})
	}))
	defer srv.Close()

	svc := network.NewService(newClient(srv.URL))
	result, err := svc.ListEgressRules(context.Background(), "my-network")
	if err != nil {
		t.Fatalf("ListEgressRules() error = %v", err)
	}
	if gotPath != "/networks/my-network/egress-firewall-rules" {
		t.Errorf("path = %q, want %q", gotPath, "/networks/my-network/egress-firewall-rules")
	}
	if len(result) != 2 {
		t.Fatalf("ListEgressRules() returned %d, want 2", len(result))
	}
	if result[0].Protocol != "tcp" {
		t.Errorf("result[0].Protocol = %q, want %q", result[0].Protocol, "tcp")
	}
}

// TestCreateEgressRule verifies POST body and path for egress rule creation.
func TestCreateEgressRule(t *testing.T) {
	created := network.EgressRule{
		ID: "10", Protocol: "tcp", StartPort: "443", EndPort: "443", Status: "Active",
	}

	var gotPath, gotMethod string
	var gotBody map[string]interface{}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotMethod = r.Method
		json.NewDecoder(r.Body).Decode(&gotBody)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "Success",
			"data":   created,
		})
	}))
	defer srv.Close()

	svc := network.NewService(newClient(srv.URL))
	rule, err := svc.CreateEgressRule(context.Background(), "my-network", network.CreateEgressRuleRequest{
		Protocol:  "tcp",
		StartPort: "443",
		EndPort:   "443",
	})
	if err != nil {
		t.Fatalf("CreateEgressRule() error = %v", err)
	}
	if gotMethod != http.MethodPost {
		t.Errorf("method = %q, want %q", gotMethod, http.MethodPost)
	}
	if gotPath != "/networks/my-network/egress-firewall-rules" {
		t.Errorf("path = %q, want %q", gotPath, "/networks/my-network/egress-firewall-rules")
	}
	if rule.ID != "10" {
		t.Errorf("rule.ID = %q, want %q", rule.ID, "10")
	}
	if gotBody["protocol"] != "tcp" {
		t.Errorf("body[protocol] = %v, want %q", gotBody["protocol"], "tcp")
	}
}

// TestDeleteEgressRule verifies DELETE path includes slug and rule ID.
func TestDeleteEgressRule(t *testing.T) {
	var gotPath, gotMethod string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	svc := network.NewService(newClient(srv.URL))

	err := svc.DeleteEgressRule(context.Background(), "my-network", "42")
	if err != nil {
		t.Fatalf("DeleteEgressRule() error = %v", err)
	}
	if gotMethod != http.MethodDelete {
		t.Errorf("method = %q, want %q", gotMethod, http.MethodDelete)
	}
	want := fmt.Sprintf("/networks/my-network/egress-firewall-rules/%s", "42")
	if gotPath != want {
		t.Errorf("path = %q, want %q", gotPath, want)
	}
}
