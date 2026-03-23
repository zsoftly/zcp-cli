package egress_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/zsoftly/zcp-cli/internal/api/egress"
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

type listEgressRuleResponse struct {
	Count                  int                 `json:"count"`
	ListEgressRuleResponse []egress.EgressRule `json:"listEgressRuleResponse"`
}

func TestEgressList(t *testing.T) {
	expected := []egress.EgressRule{
		{UUID: "egr-1", Protocol: "tcp", StartPort: "80", EndPort: "80", ZoneUUID: "zone-1"},
		{UUID: "egr-2", Protocol: "all", ZoneUUID: "zone-1"},
	}

	var gotZone string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/restapi/egressrule/egressRuleList" {
			http.NotFound(w, r)
			return
		}
		gotZone = r.URL.Query().Get("zoneUuid")
		if gotZone == "" {
			http.Error(w, "zoneUuid required", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(listEgressRuleResponse{Count: len(expected), ListEgressRuleResponse: expected})
	}))
	defer srv.Close()

	svc := egress.NewService(newClient(srv.URL))
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
	if rules[0].UUID != "egr-1" {
		t.Errorf("rules[0].UUID = %q, want %q", rules[0].UUID, "egr-1")
	}
}

func TestEgressCreate(t *testing.T) {
	created := egress.EgressRule{
		UUID:        "egr-new",
		Protocol:    "tcp",
		StartPort:   "8080",
		EndPort:     "8080",
		NetworkUUID: "net-1",
		ZoneUUID:    "zone-1",
	}

	var gotBody map[string]interface{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "expected POST", http.StatusMethodNotAllowed)
			return
		}
		if r.URL.Path != "/restapi/egressrule/createEgressRule" {
			http.NotFound(w, r)
			return
		}
		json.NewDecoder(r.Body).Decode(&gotBody)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(listEgressRuleResponse{Count: 1, ListEgressRuleResponse: []egress.EgressRule{created}})
	}))
	defer srv.Close()

	svc := egress.NewService(newClient(srv.URL))
	req := egress.CreateRequest{
		NetworkUUID: "net-1",
		Protocol:    "tcp",
		StartPort:   "8080",
		EndPort:     "8080",
	}
	rule, err := svc.Create(context.Background(), req)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if rule.UUID != "egr-new" {
		t.Errorf("rule.UUID = %q, want %q", rule.UUID, "egr-new")
	}
	if gotBody["networkUuid"] != "net-1" {
		t.Errorf("body networkUuid = %v, want %q", gotBody["networkUuid"], "net-1")
	}
	if gotBody["protocol"] != "tcp" {
		t.Errorf("body protocol = %v, want %q", gotBody["protocol"], "tcp")
	}
	if gotBody["startPort"] != "8080" {
		t.Errorf("body startPort = %v, want %q", gotBody["startPort"], "8080")
	}
	if gotBody["endPort"] != "8080" {
		t.Errorf("body endPort = %v, want %q", gotBody["endPort"], "8080")
	}
}

func TestEgressDelete(t *testing.T) {
	var gotPath, gotMethod string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	svc := egress.NewService(newClient(srv.URL))
	err := svc.Delete(context.Background(), "egr-del-1")
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if gotMethod != http.MethodDelete {
		t.Errorf("method = %q, want %q", gotMethod, http.MethodDelete)
	}
	if gotPath != "/restapi/egressrule/deleteEgressRule/egr-del-1" {
		t.Errorf("path = %q, want %q", gotPath, "/restapi/egressrule/deleteEgressRule/egr-del-1")
	}
}
