package securitygroup_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/zsoftly/zcp-cli/internal/api/securitygroup"
	"github.com/zsoftly/zcp-cli/internal/httpclient"
)

func newClient(baseURL string) *httpclient.Client {
	return httpclient.New(httpclient.Options{
		BaseURL:     baseURL,
		BearerToken: "test-token",
		Timeout:     5 * time.Second,
	})
}

type listSecurityGroupResponse struct {
	Count                     int                           `json:"count"`
	ListSecurityGroupResponse []securitygroup.SecurityGroup `json:"listSecurityGroupResponse"`
}

func makeTestSG(uuid, name string) securitygroup.SecurityGroup {
	return securitygroup.SecurityGroup{
		UUID:        uuid,
		Name:        name,
		Description: "test sg",
		IsActive:    true,
		Status:      "active",
		FirewallRules: []securitygroup.FirewallRule{
			{UUID: "fw-rule-1", Protocol: "TCP", StartPort: "80", EndPort: "80", CIDRList: "0.0.0.0/0"},
		},
		EgressRules: []securitygroup.EgressRule{
			{UUID: "eg-rule-1", Protocol: "ALL"},
		},
	}
}

func TestSecurityGroupList(t *testing.T) {
	expected := []securitygroup.SecurityGroup{
		makeTestSG("sg-1", "web-sg"),
		makeTestSG("sg-2", "db-sg"),
	}

	var gotPath, gotUUID string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotUUID = r.URL.Query().Get("uuid")
		if r.URL.Path != "/restapi/securitygroup/securityList" {
			http.NotFound(w, r)
			return
		}
		// If uuid filter is applied, return only matching
		result := expected
		if gotUUID == "sg-1" {
			result = expected[:1]
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(listSecurityGroupResponse{Count: len(result), ListSecurityGroupResponse: result})
	}))
	defer srv.Close()

	svc := securitygroup.NewService(newClient(srv.URL))

	// Test unfiltered list
	groups, err := svc.List(context.Background(), "")
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if gotPath != "/restapi/securitygroup/securityList" {
		t.Errorf("path = %q, want %q", gotPath, "/restapi/securitygroup/securityList")
	}
	if len(groups) != 2 {
		t.Fatalf("List() returned %d groups, want 2", len(groups))
	}
	if groups[0].UUID != "sg-1" {
		t.Errorf("groups[0].UUID = %q, want %q", groups[0].UUID, "sg-1")
	}
	if len(groups[0].FirewallRules) != 1 {
		t.Errorf("groups[0].FirewallRules count = %d, want 1", len(groups[0].FirewallRules))
	}
	if groups[0].FirewallRules[0].UUID != "fw-rule-1" {
		t.Errorf("FirewallRules[0].UUID = %q, want %q", groups[0].FirewallRules[0].UUID, "fw-rule-1")
	}
	if len(groups[0].EgressRules) != 1 {
		t.Errorf("groups[0].EgressRules count = %d, want 1", len(groups[0].EgressRules))
	}

	// Test uuid filter
	filtered, err := svc.List(context.Background(), "sg-1")
	if err != nil {
		t.Fatalf("List(uuid) error = %v", err)
	}
	if gotUUID != "sg-1" {
		t.Errorf("uuid query param = %q, want %q", gotUUID, "sg-1")
	}
	if len(filtered) != 1 {
		t.Fatalf("List(uuid) returned %d groups, want 1", len(filtered))
	}
}

func TestSecurityGroupCreate(t *testing.T) {
	created := makeTestSG("sg-new", "my-sg")

	var gotBody map[string]interface{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "expected POST", http.StatusMethodNotAllowed)
			return
		}
		if r.URL.Path != "/restapi/securitygroup/createSecurityGroup" {
			http.NotFound(w, r)
			return
		}
		json.NewDecoder(r.Body).Decode(&gotBody)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(listSecurityGroupResponse{Count: 1, ListSecurityGroupResponse: []securitygroup.SecurityGroup{created}})
	}))
	defer srv.Close()

	svc := securitygroup.NewService(newClient(srv.URL))
	req := securitygroup.CreateGroupRequest{Name: "my-sg", Description: "my security group"}
	sg, err := svc.Create(context.Background(), req)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if sg.UUID != "sg-new" {
		t.Errorf("sg.UUID = %q, want %q", sg.UUID, "sg-new")
	}
	if gotBody["name"] != "my-sg" {
		t.Errorf("body name = %v, want %q", gotBody["name"], "my-sg")
	}
	if gotBody["description"] != "my security group" {
		t.Errorf("body description = %v, want %q", gotBody["description"], "my security group")
	}
}

func TestSecurityGroupDelete(t *testing.T) {
	var gotPath, gotMethod string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	svc := securitygroup.NewService(newClient(srv.URL))
	err := svc.Delete(context.Background(), "sg-del-1")
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if gotMethod != http.MethodDelete {
		t.Errorf("method = %q, want %q", gotMethod, http.MethodDelete)
	}
	if gotPath != "/restapi/securitygroup/deleteSecurityGroup/sg-del-1" {
		t.Errorf("path = %q, want %q", gotPath, "/restapi/securitygroup/deleteSecurityGroup/sg-del-1")
	}
}

func TestSecurityGroupCreateFirewallRule(t *testing.T) {
	sg := makeTestSG("sg-1", "web-sg")

	var gotBody map[string]interface{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "expected POST", http.StatusMethodNotAllowed)
			return
		}
		if r.URL.Path != "/restapi/securitygroup/createSecurityGroupFirewallRule" {
			http.NotFound(w, r)
			return
		}
		json.NewDecoder(r.Body).Decode(&gotBody)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(listSecurityGroupResponse{Count: 1, ListSecurityGroupResponse: []securitygroup.SecurityGroup{sg}})
	}))
	defer srv.Close()

	svc := securitygroup.NewService(newClient(srv.URL))
	req := securitygroup.CreateFirewallRuleRequest{
		SecurityGroupUUID: "sg-1",
		Protocol:          "TCP",
		StartPort:         "443",
		EndPort:           "443",
		CIDRList:          "10.0.0.0/8",
	}
	result, err := svc.CreateFirewallRule(context.Background(), req)
	if err != nil {
		t.Fatalf("CreateFirewallRule() error = %v", err)
	}
	if result.UUID != "sg-1" {
		t.Errorf("result.UUID = %q, want %q", result.UUID, "sg-1")
	}
	if gotBody["securityGroupUuid"] != "sg-1" {
		t.Errorf("body securityGroupUuid = %v, want %q", gotBody["securityGroupUuid"], "sg-1")
	}
	if gotBody["protocol"] != "TCP" {
		t.Errorf("body protocol = %v, want %q", gotBody["protocol"], "TCP")
	}
	if gotBody["startPort"] != "443" {
		t.Errorf("body startPort = %v, want %q", gotBody["startPort"], "443")
	}
	if gotBody["cidrList"] != "10.0.0.0/8" {
		t.Errorf("body cidrList = %v, want %q", gotBody["cidrList"], "10.0.0.0/8")
	}
}

func TestSecurityGroupDeleteRule(t *testing.T) {
	var gotMethod, gotPath string
	var gotQuery map[string]string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		gotQuery = map[string]string{
			"securityGroupUuid": r.URL.Query().Get("securityGroupUuid"),
			"ruleType":          r.URL.Query().Get("ruleType"),
			"ruleUuid":          r.URL.Query().Get("ruleUuid"),
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	svc := securitygroup.NewService(newClient(srv.URL))
	err := svc.DeleteRule(context.Background(), "sg-1", "firewall", "fw-rule-1")
	if err != nil {
		t.Fatalf("DeleteRule() error = %v", err)
	}
	if gotMethod != http.MethodDelete {
		t.Errorf("method = %q, want %q", gotMethod, http.MethodDelete)
	}
	if gotPath != "/restapi/securitygroup/deleteSecurityGroupRule" {
		t.Errorf("path = %q, want %q", gotPath, "/restapi/securitygroup/deleteSecurityGroupRule")
	}
	if gotQuery["securityGroupUuid"] != "sg-1" {
		t.Errorf("query securityGroupUuid = %q, want %q", gotQuery["securityGroupUuid"], "sg-1")
	}
	if gotQuery["ruleType"] != "firewall" {
		t.Errorf("query ruleType = %q, want %q", gotQuery["ruleType"], "firewall")
	}
	if gotQuery["ruleUuid"] != "fw-rule-1" {
		t.Errorf("query ruleUuid = %q, want %q", gotQuery["ruleUuid"], "fw-rule-1")
	}
}
