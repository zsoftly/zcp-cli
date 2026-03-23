package portforward_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/zsoftly/zcp-cli/internal/api/portforward"
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

type listPortForwardingResponse struct {
	Count                      int                           `json:"count"`
	ListPortForwardingResponse []portforward.PortForwardRule `json:"listPortForwardingResponse"`
}

func TestPortForwardList(t *testing.T) {
	expected := []portforward.PortForwardRule{
		{UUID: "pf-1", Protocol: "tcp", PublicPort: "2222", PrivatePort: "22", ZoneUUID: "zone-1"},
		{UUID: "pf-2", Protocol: "tcp", PublicPort: "8080", PrivatePort: "80", ZoneUUID: "zone-1"},
	}

	var gotZone string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/restapi/portforwardingrule/portForwardingRuleList" {
			http.NotFound(w, r)
			return
		}
		gotZone = r.URL.Query().Get("zoneUuid")
		if gotZone == "" {
			http.Error(w, "zoneUuid required", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(listPortForwardingResponse{Count: len(expected), ListPortForwardingResponse: expected})
	}))
	defer srv.Close()

	svc := portforward.NewService(newClient(srv.URL))
	rules, err := svc.List(context.Background(), "zone-1", "", "", "")
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(rules) != 2 {
		t.Fatalf("List() returned %d rules, want 2", len(rules))
	}
	if gotZone != "zone-1" {
		t.Errorf("zoneUuid query param = %q, want %q", gotZone, "zone-1")
	}
	if rules[0].UUID != "pf-1" {
		t.Errorf("rules[0].UUID = %q, want %q", rules[0].UUID, "pf-1")
	}
}

func TestPortForwardCreate(t *testing.T) {
	created := portforward.PortForwardRule{
		UUID:          "pf-new",
		Protocol:      "tcp",
		PublicPort:    "2222",
		PrivatePort:   "22",
		IPAddressUUID: "ip-1",
		ZoneUUID:      "zone-1",
	}

	var gotBody map[string]interface{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "expected POST", http.StatusMethodNotAllowed)
			return
		}
		if r.URL.Path != "/restapi/portforwardingrule/createPortForwardingRule" {
			http.NotFound(w, r)
			return
		}
		json.NewDecoder(r.Body).Decode(&gotBody)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(listPortForwardingResponse{Count: 1, ListPortForwardingResponse: []portforward.PortForwardRule{created}})
	}))
	defer srv.Close()

	svc := portforward.NewService(newClient(srv.URL))
	req := portforward.CreateRequest{
		IPAddressUUID:      "ip-1",
		Protocol:           "tcp",
		PublicPort:         "2222",
		PrivatePort:        "22",
		VirtualMachineUUID: "vm-1",
	}
	rule, err := svc.Create(context.Background(), req)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if rule.UUID != "pf-new" {
		t.Errorf("rule.UUID = %q, want %q", rule.UUID, "pf-new")
	}
	if gotBody["ipAddressUuid"] != "ip-1" {
		t.Errorf("body ipAddressUuid = %v, want %q", gotBody["ipAddressUuid"], "ip-1")
	}
	if gotBody["protocol"] != "tcp" {
		t.Errorf("body protocol = %v, want %q", gotBody["protocol"], "tcp")
	}
	if gotBody["publicPort"] != "2222" {
		t.Errorf("body publicPort = %v, want %q", gotBody["publicPort"], "2222")
	}
	if gotBody["privatePort"] != "22" {
		t.Errorf("body privatePort = %v, want %q", gotBody["privatePort"], "22")
	}
	if gotBody["virtualmachineUuid"] != "vm-1" {
		t.Errorf("body virtualmachineUuid = %v, want %q", gotBody["virtualmachineUuid"], "vm-1")
	}
}

func TestPortForwardDelete(t *testing.T) {
	var gotPath, gotMethod string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	svc := portforward.NewService(newClient(srv.URL))
	err := svc.Delete(context.Background(), "pf-del-1")
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if gotMethod != http.MethodDelete {
		t.Errorf("method = %q, want %q", gotMethod, http.MethodDelete)
	}
	if gotPath != "/restapi/portforwardingrule/deletePortForwardingRule/pf-del-1" {
		t.Errorf("path = %q, want %q", gotPath, "/restapi/portforwardingrule/deletePortForwardingRule/pf-del-1")
	}
}
