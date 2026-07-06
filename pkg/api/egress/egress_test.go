package egress_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/zsoftly/zcp-cli/pkg/api/egress"
	"github.com/zsoftly/zcp-cli/pkg/httpclient"
)

func newClient(baseURL string) *httpclient.Client {
	return httpclient.New(httpclient.Options{
		BaseURL:     baseURL,
		BearerToken: "test-token",
		Timeout:     5 * time.Second,
	})
}

func TestEgressList(t *testing.T) {
	expected := []egress.EgressRule{
		{ID: "1", Protocol: "tcp", StartPort: "80", EndPort: "80", Status: "Active"},
		{ID: "2", Protocol: "all", Status: "Active"},
	}

	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "Success",
			"data":   expected,
		})
	}))
	defer srv.Close()

	svc := egress.NewService(newClient(srv.URL))
	rules, err := svc.List(context.Background(), "my-network")
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if gotPath != "/networks/my-network/egress-firewall-rules" {
		t.Errorf("path = %q, want %q", gotPath, "/networks/my-network/egress-firewall-rules")
	}
	if len(rules) != 2 {
		t.Fatalf("List() returned %d rules, want 2", len(rules))
	}
	if rules[0].ID != "1" {
		t.Errorf("rules[0].ID = %q, want %q", rules[0].ID, "1")
	}
}

func TestEgressCreate(t *testing.T) {
	created := egress.EgressRule{
		ID: "10", Protocol: "tcp", StartPort: "8080", EndPort: "8080", Status: "Active",
	}

	var gotBody map[string]interface{}
	var gotPath, gotMethod string
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

	svc := egress.NewService(newClient(srv.URL))
	req := egress.CreateRequest{
		NetworkSlug: "my-network",
		Protocol:    "tcp",
		StartPort:   "8080",
		EndPort:     "8080",
	}
	rule, err := svc.Create(context.Background(), req)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
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
		t.Errorf("body protocol = %v, want %q", gotBody["protocol"], "tcp")
	}
	if gotBody["start_port"] != "8080" {
		t.Errorf("body start_port = %v, want %q", gotBody["start_port"], "8080")
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
	err := svc.Delete(context.Background(), "my-network", "42")
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if gotMethod != http.MethodDelete {
		t.Errorf("method = %q, want %q", gotMethod, http.MethodDelete)
	}
	want := fmt.Sprintf("/networks/my-network/egress-firewall-rules/%s", "42")
	if gotPath != want {
		t.Errorf("path = %q, want %q", gotPath, want)
	}
}

// TestCreateFallbackMatchesDestCIDR verifies the data:null create fallback
// matches on destcidr_list (the echoed request CIDR), not the top-level cidr
// (the network's source CIDR), and maps state from _original.
func TestCreateFallbackMatchesDestCIDR(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodPost {
			fmt.Fprint(w, `{"status":"Success","message":"Creating egress firewall rule.","data":null}`)
			return
		}
		fmt.Fprint(w, `{"status":"Success","data":[
			{"id":"other","protocol":"tcp","start_port":80,"end_port":80,"cidr":"10.0.0.0/24","destcidr_list":"192.168.0.0/16","_original":{"state":"Active"}},
			{"id":"mine","protocol":"tcp","start_port":80,"end_port":80,"cidr":"10.0.0.0/24","destcidr_list":"0.0.0.0/0","_original":{"state":"Active"}}
		]}`)
	}))
	defer srv.Close()

	svc := egress.NewService(newClient(srv.URL))

	rule, err := svc.Create(context.Background(), egress.CreateRequest{
		NetworkSlug: "my-net", Protocol: "tcp", StartPort: "80", EndPort: "80", CIDR: "0.0.0.0/0",
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if rule.ID != "mine" {
		t.Errorf("Create() matched rule %q, want %q (must match destcidr_list, not cidr)", rule.ID, "mine")
	}
	if rule.Status != "Active" {
		t.Errorf("rule.Status = %q, want %q (from _original.state)", rule.Status, "Active")
	}
}

// TestCreateFallbackNoMatchErrors verifies an explicit error when the created
// rule is not yet visible, instead of an empty-ID success.
func TestCreateFallbackNoMatchErrors(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodPost {
			fmt.Fprint(w, `{"status":"Success","data":null}`)
			return
		}
		fmt.Fprint(w, `{"status":"Success","data":[]}`)
	}))
	defer srv.Close()

	svc := egress.NewService(newClient(srv.URL))

	_, err := svc.Create(context.Background(), egress.CreateRequest{
		NetworkSlug: "my-net", Protocol: "tcp", StartPort: "80", EndPort: "80", CIDR: "0.0.0.0/0",
	})
	if err == nil {
		t.Fatal("Create() = nil error, want explicit not-yet-visible error")
	}
}

// TestCreateFallbackMatchesICMPTypeAndCode verifies the fallback lookup picks
// the right ICMP rule when two rules to the same CIDR differ only in ICMP
// type/code (ICMP rules carry no ports to discriminate on).
func TestCreateFallbackMatchesICMPTypeAndCode(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodPost {
			fmt.Fprint(w, `{"status":"Success","data":null}`)
			return
		}
		fmt.Fprint(w, `{"status":"Success","data":[
			{"id":"echo-reply","protocol":"icmp","icmp_type":0,"icmp_code":0,"destcidr_list":"0.0.0.0/0","_original":{"state":"Active"}},
			{"id":"echo-request","protocol":"icmp","icmp_type":8,"icmp_code":0,"destcidr_list":"0.0.0.0/0","_original":{"state":"Active"}}
		]}`)
	}))
	defer srv.Close()

	svc := egress.NewService(newClient(srv.URL))

	rule, err := svc.Create(context.Background(), egress.CreateRequest{
		NetworkSlug: "my-net", Protocol: "icmp", ICMPType: "8", ICMPCode: "0", CIDR: "0.0.0.0/0",
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if rule.ID != "echo-request" {
		t.Errorf("Create() matched rule %q, want %q (must discriminate on icmp_type/icmp_code)", rule.ID, "echo-request")
	}
}
