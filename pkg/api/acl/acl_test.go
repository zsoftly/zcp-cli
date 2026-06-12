package acl_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/zsoftly/zcp-cli/pkg/api/acl"
	"github.com/zsoftly/zcp-cli/pkg/httpclient"
)

func newClient(baseURL string) *httpclient.Client {
	return httpclient.New(httpclient.Options{
		BaseURL:     baseURL,
		BearerToken: "test-token",
		Timeout:     5 * time.Second,
	})
}

func TestACLList(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/vpcs/my-vpc/network-acl-list" {
			t.Errorf("path = %q", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "Success",
			"data": []map[string]string{
				{"slug": "acl-1", "name": "default-acl", "vpcSlug": "my-vpc"},
			},
		})
	}))
	defer srv.Close()

	svc := acl.NewService(newClient(srv.URL))
	acls, err := svc.List(context.Background(), "my-vpc")
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(acls) != 1 {
		t.Fatalf("got %d ACLs, want 1", len(acls))
	}
	if acls[0].Slug != "acl-1" {
		t.Errorf("slug = %q, want %q", acls[0].Slug, "acl-1")
	}
}

func TestACLCreate(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %q, want POST", r.Method)
		}
		if r.URL.Path != "/vpcs/my-vpc/network-acl-list" {
			t.Errorf("path = %q", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  "Success",
			"message": "Adding network ACL list.",
		})
	}))
	defer srv.Close()

	svc := acl.NewService(newClient(srv.URL))
	err := svc.Create(context.Background(), "my-vpc", acl.ACLCreateRequest{Name: "web-acl", VPC: "my-vpc"})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
}

func TestACLReplaceNetworkACL(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/networks/my-net/replace-acl-list" {
			t.Errorf("path = %q", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"status": "Success", "data": nil})
	}))
	defer srv.Close()

	svc := acl.NewService(newClient(srv.URL))
	err := svc.ReplaceNetworkACL(context.Background(), "my-net", "acl-1")
	if err != nil {
		t.Fatalf("ReplaceNetworkACL() error = %v", err)
	}
}

// TestReplaceNetworkACLSendsACLID verifies the replace body uses the acl_id
// field (the live API rejects any other key).
func TestReplaceNetworkACLSendsACLID(t *testing.T) {
	var gotBody map[string]interface{}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/networks/web-tier/replace-acl-list" {
			http.NotFound(w, r)
			return
		}
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Errorf("decoding request body: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"status":"Success","data":{"job_id":"j1"}}`)
	}))
	defer srv.Close()

	svc := acl.NewService(newClient(srv.URL))

	if err := svc.ReplaceNetworkACL(context.Background(), "web-tier", "acl-uuid-1"); err != nil {
		t.Fatalf("ReplaceNetworkACL() error = %v", err)
	}
	if gotBody["acl_id"] != "acl-uuid-1" {
		t.Errorf("body[acl_id] = %v, want %q", gotBody["acl_id"], "acl-uuid-1")
	}
}

// TestResolveACLByName verifies name → ID resolution against the VPC ACL list.
func TestResolveACLByName(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"status":"Success","data":[
			{"id":"uuid-web","name":"web-acl","description":"web"},
			{"id":"uuid-db","name":"db-acl","description":"db"}
		]}`)
	}))
	defer srv.Close()

	svc := acl.NewService(newClient(srv.URL))

	id, err := svc.Resolve(context.Background(), "my-vpc", "db-acl")
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if id != "uuid-db" {
		t.Errorf("Resolve(db-acl) = %q, want %q", id, "uuid-db")
	}
	if id, _ := svc.Resolve(context.Background(), "my-vpc", "uuid-web"); id != "uuid-web" {
		t.Errorf("Resolve(uuid-web) = %q, want passthrough by ID", id)
	}
	if _, err := svc.Resolve(context.Background(), "my-vpc", "missing"); err == nil {
		t.Error("Resolve(missing) expected error, got nil")
	}
}

// TestCreateRuleSendsSnakeCase verifies the rule create body matches the live
// API validation (snake_case fields, ports only for tcp/udp).
func TestCreateRuleSendsSnakeCase(t *testing.T) {
	var gotBody map[string]interface{}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/vpcs/my-vpc/network-acl-list/acl-1/network-acl" || r.Method != http.MethodPost {
			http.NotFound(w, r)
			return
		}
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Errorf("decoding request body: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"status":"Success","data":null}`)
	}))
	defer srv.Close()

	svc := acl.NewService(newClient(srv.URL))

	start, end := 80, 80
	err := svc.CreateRule(context.Background(), "my-vpc", "acl-1", acl.RuleCreateRequest{
		Number:      1,
		Protocol:    "tcp",
		CIDRList:    "0.0.0.0/0",
		Action:      "allow",
		TrafficType: "ingress",
		StartPort:   &start,
		EndPort:     &end,
	})
	if err != nil {
		t.Fatalf("CreateRule() error = %v", err)
	}
	for k, want := range map[string]interface{}{
		"protocol": "tcp", "cidr_list": "0.0.0.0/0", "action": "allow",
		"traffic_type": "ingress", "start_port": float64(80), "end_port": float64(80),
	} {
		if gotBody[k] != want {
			t.Errorf("body[%s] = %v, want %v", k, gotBody[k], want)
		}
	}
	for _, absent := range []string{"icmp_type", "icmp_code", "protocol_number"} {
		if _, ok := gotBody[absent]; ok {
			t.Errorf("body[%s] present, want omitted for tcp rules", absent)
		}
	}
}

// TestListRules verifies rule list parsing from the live response shape.
func TestListRules(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"status":"Success","data":[
			{"id":"r1","protocol":"tcp","start_port":"80","end_port":"80","traffictype":"Ingress",
			 "state":"Active","cidrlist":"0.0.0.0/0","aclid":"acl-1","aclname":"web-acl","number":1,"action":"Allow"}
		]}`)
	}))
	defer srv.Close()

	svc := acl.NewService(newClient(srv.URL))

	rules, err := svc.ListRules(context.Background(), "my-vpc", "acl-1")
	if err != nil {
		t.Fatalf("ListRules() error = %v", err)
	}
	if len(rules) != 1 || rules[0].ID != "r1" || rules[0].CIDRList != "0.0.0.0/0" || rules[0].Number != 1 {
		t.Errorf("ListRules() = %+v, want one rule r1 with cidr 0.0.0.0/0 number 1", rules)
	}
}

// TestDeleteRulePath verifies the DELETE route shape.
func TestDeleteRulePath(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.Method + " " + r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"status":"Success","data":null}`)
	}))
	defer srv.Close()

	svc := acl.NewService(newClient(srv.URL))

	if err := svc.DeleteRule(context.Background(), "my-vpc", "acl-1", "rule-9"); err != nil {
		t.Fatalf("DeleteRule() error = %v", err)
	}
	want := "DELETE /vpcs/my-vpc/network-acl-list/acl-1/network-acl/rule-9"
	if gotPath != want {
		t.Errorf("request = %q, want %q", gotPath, want)
	}
}

// TestUpdateRulePathAndBody verifies the PUT route shape and that multi-CIDR
// lists pass through unchanged.
func TestUpdateRulePathAndBody(t *testing.T) {
	var gotPath string
	var gotBody map[string]interface{}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.Method + " " + r.URL.Path
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Errorf("decoding request body: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"status":"Success","data":null}`)
	}))
	defer srv.Close()

	svc := acl.NewService(newClient(srv.URL))

	icmp := -1
	err := svc.UpdateRule(context.Background(), "my-vpc", "acl-1", "rule-9", acl.RuleCreateRequest{
		Number:      3,
		Protocol:    "icmp",
		CIDRList:    "10.30.1.0/24,10.30.2.0/24,10.30.3.0/24",
		Action:      "allow",
		TrafficType: "ingress",
		ICMPType:    &icmp,
		ICMPCode:    &icmp,
	})
	if err != nil {
		t.Fatalf("UpdateRule() error = %v", err)
	}
	want := "PUT /vpcs/my-vpc/network-acl-list/acl-1/network-acl/rule-9"
	if gotPath != want {
		t.Errorf("request = %q, want %q", gotPath, want)
	}
	if gotBody["cidr_list"] != "10.30.1.0/24,10.30.2.0/24,10.30.3.0/24" {
		t.Errorf("body[cidr_list] = %v, want the full multi-CIDR list", gotBody["cidr_list"])
	}
}

// TestResolveSkipsEmptyID verifies a name match without a usable ID is not
// returned, and that the legacy slug field works as an alias.
func TestResolveSkipsEmptyID(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"status":"Success","data":[
			{"id":"","name":"ghost-acl","description":"no id"},
			{"id":"uuid-1","slug":"legacy-slug","name":"web-acl","description":"ok"}
		]}`)
	}))
	defer srv.Close()

	svc := acl.NewService(newClient(srv.URL))

	if _, err := svc.Resolve(context.Background(), "my-vpc", "ghost-acl"); err == nil {
		t.Error("Resolve(ghost-acl) = nil error, want not-found for an ACL with no ID")
	}
	id, err := svc.Resolve(context.Background(), "my-vpc", "legacy-slug")
	if err != nil {
		t.Fatalf("Resolve(legacy-slug) error = %v", err)
	}
	if id != "uuid-1" {
		t.Errorf("Resolve(legacy-slug) = %q, want %q", id, "uuid-1")
	}
}
