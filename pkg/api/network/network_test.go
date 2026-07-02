package network_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/zsoftly/zcp-cli/pkg/api/network"
	"github.com/zsoftly/zcp-cli/pkg/httpclient"
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
		Status:      true,
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

	result, err := svc.List(context.Background(), "", "")
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

func TestNetworkGetWithFilters(t *testing.T) {
	networks := []network.Network{
		makeNetwork("web-network", "web-network"),
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/networks" {
			http.NotFound(w, r)
			return
		}
		if got := r.URL.Query().Get("filter[region]"); got != "yow-1" {
			t.Errorf("filter[region] = %q, want %q", got, "yow-1")
		}
		if got := r.URL.Query().Get("filter[project]"); got != "default-9" {
			t.Errorf("filter[project] = %q, want %q", got, "default-9")
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "Success",
			"data":   networks,
		})
	}))
	defer srv.Close()

	svc := network.NewService(newClient(srv.URL))
	net, err := svc.Get(context.Background(), "web-network", "yow-1", "default-9")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if net.Slug != "web-network" {
		t.Errorf("net.Slug = %q, want %q", net.Slug, "web-network")
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
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Errorf("decoding request body: %v", err)
		}
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
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Errorf("decoding request body: %v", err)
		}
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
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Errorf("decoding request body: %v", err)
		}
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

// TestNetworkListBoolStatus verifies that a network list response with boolean
// status (as returned by the live API) decodes without error.
func TestNetworkListBoolStatus(t *testing.T) {
	payload := `{"status":"Success","data":[{"id":"abc","slug":"test-net","name":"Test Net","status":true,"type":"Isolated","gateway":"10.0.0.1","cidr":"10.0.0.0/24","zone_slug":"yow-1","is_default":false}]}`

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/networks" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, payload)
	}))
	defer srv.Close()

	svc := network.NewService(newClient(srv.URL))
	result, err := svc.List(context.Background(), "", "")
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(result) != 1 {
		t.Fatalf("List() returned %d networks, want 1", len(result))
	}
	if !result[0].Status {
		t.Errorf("result[0].Status = false, want true")
	}
	if result[0].NetworkType != "Isolated" {
		t.Errorf("result[0].NetworkType = %q, want %q", result[0].NetworkType, "Isolated")
	}
}

// TestNetworkListStringStatus verifies that "status":"Active" (older API shape)
// decodes as Status=true.
func TestNetworkListStringStatus(t *testing.T) {
	payload := `{"status":"Success","data":[{"id":"old","slug":"old-net","name":"Old Net","status":"Active","type":"Isolated"}]}`

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/networks" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, payload)
	}))
	defer srv.Close()

	svc := network.NewService(newClient(srv.URL))
	result, err := svc.List(context.Background(), "", "")
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(result) != 1 {
		t.Fatalf("List() returned %d networks, want 1", len(result))
	}
	if !result[0].Status {
		t.Errorf("result[0].Status = false, want true (decoded from string %q)", "Active")
	}
}

// TestNetworkListNetworkTypeKey verifies that "network_type":"Isolated" (older
// API key name) populates NetworkType correctly.
func TestNetworkListNetworkTypeKey(t *testing.T) {
	payload := `{"status":"Success","data":[{"id":"old2","slug":"old-net2","name":"Old Net 2","status":true,"network_type":"Shared"}]}`

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/networks" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, payload)
	}))
	defer srv.Close()

	svc := network.NewService(newClient(srv.URL))
	result, err := svc.List(context.Background(), "", "")
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(result) != 1 {
		t.Fatalf("List() returned %d networks, want 1", len(result))
	}
	if result[0].NetworkType != "Shared" {
		t.Errorf("NetworkType = %q, want %q", result[0].NetworkType, "Shared")
	}
}

// TestDeleteNetwork verifies DELETE /networks/<slug> is called correctly.
func TestDeleteNetwork(t *testing.T) {
	var gotPath, gotMethod string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	svc := network.NewService(newClient(srv.URL))
	err := svc.Delete(context.Background(), "en-001001-0018")
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if gotMethod != http.MethodDelete {
		t.Errorf("method = %q, want %q", gotMethod, http.MethodDelete)
	}
	if gotPath != "/networks/en-001001-0018" {
		t.Errorf("path = %q, want %q", gotPath, "/networks/en-001001-0018")
	}
}

// TestDeleteNetwork_HasVMs verifies that an error response surfaces correctly.
func TestDeleteNetwork_HasVMs(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnprocessableEntity)
		fmt.Fprint(w, `{"status":"Error","message":"You cannot delete the network while virtual machines are created using it."}`)
	}))
	defer srv.Close()

	svc := network.NewService(newClient(srv.URL))
	err := svc.Delete(context.Background(), "en-001001-0018")
	if err == nil {
		t.Fatal("Delete() expected error, got nil")
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

// TestNetworkCreateVPCSubnet verifies the POST body for a VPC subnet (tier):
// the API requires type "Vpc" (exact case) plus the vpc slug, billing cycle,
// gateway, and netmask — and no network_plan.
func TestNetworkCreateVPCSubnet(t *testing.T) {
	created := makeNetwork("web-tier", "web-tier")

	var gotBody map[string]interface{}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Errorf("decoding request body: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "Success",
			"data":   created,
		})
	}))
	defer srv.Close()

	svc := network.NewService(newClient(srv.URL))

	_, err := svc.Create(context.Background(), network.CreateRequest{
		Name:          "web-tier",
		CloudProvider: "nimbo",
		Region:        "yul-1",
		Project:       "default-9",
		VPC:           "my-vpc",
		BillingCycle:  "hourly",
		Type:          "Vpc",
		Gateway:       "10.30.1.1",
		Netmask:       "255.255.255.0",
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if gotBody["vpc"] != "my-vpc" {
		t.Errorf("body[vpc] = %v, want %q", gotBody["vpc"], "my-vpc")
	}
	if gotBody["type"] != "Vpc" {
		t.Errorf("body[type] = %v, want %q (exact case is required by the API)", gotBody["type"], "Vpc")
	}
	if gotBody["billing_cycle"] != "hourly" {
		t.Errorf("body[billing_cycle] = %v, want %q", gotBody["billing_cycle"], "hourly")
	}
	if _, present := gotBody["network_plan"]; present {
		t.Errorf("body[network_plan] = %v, want omitted for VPC subnets", gotBody["network_plan"])
	}
	if _, present := gotBody["category_slug"]; present {
		t.Errorf("body[category_slug] = %v, want omitted when empty", gotBody["category_slug"])
	}
}

// TestNetworkCreateIsolatedSendsPlan verifies network_plan and type are sent
// for isolated networks.
func TestNetworkCreateIsolatedSendsPlan(t *testing.T) {
	created := makeNetwork("my-net", "my-net")

	var gotBody map[string]interface{}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Errorf("decoding request body: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "Success",
			"data":   created,
		})
	}))
	defer srv.Close()

	svc := network.NewService(newClient(srv.URL))

	_, err := svc.Create(context.Background(), network.CreateRequest{
		Name:          "my-net",
		CloudProvider: "nimbo",
		Region:        "yow-1",
		Project:       "default-9",
		Type:          "Isolated",
		NetworkPlan:   "inet-yow",
		BillingCycle:  "hourly",
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if gotBody["network_plan"] != "inet-yow" {
		t.Errorf("body[network_plan] = %v, want %q", gotBody["network_plan"], "inet-yow")
	}
	if gotBody["type"] != "Isolated" {
		t.Errorf("body[type] = %v, want %q", gotBody["type"], "Isolated")
	}
}

// TestNetworkGetDetail verifies GET /networks/{slug} parsing, including the
// CloudStack meta block that holds CIDR, state, and VPC membership.
func TestNetworkGetDetail(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/networks/web-tier" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"status":"Success","data":{
			"id":"a1","slug":"web-tier","name":"web-tier","network_id":"cs-uuid",
			"meta":{"cidr":"10.30.1.0/24","netmask":"255.255.255.0","type":"Isolated",
				"state":"Allocated","vpc_id":"vpc-uuid","acl_name":"default_allow","zone_name":"yul-1"}
		}}`)
	}))
	defer srv.Close()

	svc := network.NewService(newClient(srv.URL))

	d, err := svc.GetDetail(context.Background(), "web-tier")
	if err != nil {
		t.Fatalf("GetDetail() error = %v", err)
	}
	if d.Meta.CIDR != "10.30.1.0/24" {
		t.Errorf("Meta.CIDR = %q, want %q", d.Meta.CIDR, "10.30.1.0/24")
	}
	if d.Meta.VPCID != "vpc-uuid" {
		t.Errorf("Meta.VPCID = %q, want %q", d.Meta.VPCID, "vpc-uuid")
	}
	if d.Meta.State != "Allocated" {
		t.Errorf("Meta.State = %q, want %q", d.Meta.State, "Allocated")
	}
	if d.Meta.ACLName != "default_allow" {
		t.Errorf("Meta.ACLName = %q, want %q", d.Meta.ACLName, "default_allow")
	}
}
