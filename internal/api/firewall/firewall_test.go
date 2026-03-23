package firewall_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/zsoftly/zcp-cli/internal/api/firewall"
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

type listFirewallRuleResponse struct {
	Count                    int                     `json:"count"`
	ListFirewallRuleResponse []firewall.FirewallRule `json:"listFirewallRuleResponse"`
}

func TestFirewallList(t *testing.T) {
	expected := []firewall.FirewallRule{
		{UUID: "fw-1", Protocol: "tcp", StartPort: "80", EndPort: "80", ZoneUUID: "zone-1"},
		{UUID: "fw-2", Protocol: "udp", StartPort: "53", EndPort: "53", ZoneUUID: "zone-1"},
	}

	var gotZone string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/restapi/firewallrule/firewallRuleList" {
			http.NotFound(w, r)
			return
		}
		gotZone = r.URL.Query().Get("zoneUuid")
		if gotZone == "" {
			http.Error(w, "zoneUuid required", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(listFirewallRuleResponse{Count: len(expected), ListFirewallRuleResponse: expected})
	}))
	defer srv.Close()

	svc := firewall.NewService(newClient(srv.URL))
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
	if rules[0].UUID != "fw-1" {
		t.Errorf("rules[0].UUID = %q, want %q", rules[0].UUID, "fw-1")
	}
}

func TestFirewallCreate(t *testing.T) {
	created := firewall.FirewallRule{
		UUID:          "fw-new",
		Protocol:      "tcp",
		StartPort:     "443",
		EndPort:       "443",
		CIDRList:      "0.0.0.0/0",
		IPAddressUUID: "ip-1",
		ZoneUUID:      "zone-1",
	}

	var gotBody map[string]interface{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "expected POST", http.StatusMethodNotAllowed)
			return
		}
		if r.URL.Path != "/restapi/firewallrule/createFirewallRule" {
			http.NotFound(w, r)
			return
		}
		json.NewDecoder(r.Body).Decode(&gotBody)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(listFirewallRuleResponse{Count: 1, ListFirewallRuleResponse: []firewall.FirewallRule{created}})
	}))
	defer srv.Close()

	svc := firewall.NewService(newClient(srv.URL))
	req := firewall.CreateRequest{
		IPAddressUUID: "ip-1",
		Protocol:      "tcp",
		StartPort:     "443",
		EndPort:       "443",
		CIDRList:      "0.0.0.0/0",
	}
	rule, err := svc.Create(context.Background(), req)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if rule.UUID != "fw-new" {
		t.Errorf("rule.UUID = %q, want %q", rule.UUID, "fw-new")
	}
	if gotBody["ipAddressUuid"] != "ip-1" {
		t.Errorf("body ipAddressUuid = %v, want %q", gotBody["ipAddressUuid"], "ip-1")
	}
	if gotBody["protocol"] != "tcp" {
		t.Errorf("body protocol = %v, want %q", gotBody["protocol"], "tcp")
	}
	if gotBody["startPort"] != "443" {
		t.Errorf("body startPort = %v, want %q", gotBody["startPort"], "443")
	}
	if gotBody["endPort"] != "443" {
		t.Errorf("body endPort = %v, want %q", gotBody["endPort"], "443")
	}
	if gotBody["cidrList"] != "0.0.0.0/0" {
		t.Errorf("body cidrList = %v, want %q", gotBody["cidrList"], "0.0.0.0/0")
	}
}

func TestFirewallDelete(t *testing.T) {
	var gotPath, gotMethod string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	svc := firewall.NewService(newClient(srv.URL))
	err := svc.Delete(context.Background(), "fw-del-1")
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if gotMethod != http.MethodDelete {
		t.Errorf("method = %q, want %q", gotMethod, http.MethodDelete)
	}
	if gotPath != "/restapi/firewallrule/deleteFirewallRule/fw-del-1" {
		t.Errorf("path = %q, want %q", gotPath, "/restapi/firewallrule/deleteFirewallRule/fw-del-1")
	}
}
