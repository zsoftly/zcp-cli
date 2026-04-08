package acl_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/zsoftly/zcp-cli/internal/api/acl"
	"github.com/zsoftly/zcp-cli/internal/httpclient"
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

func TestACLCreateRule(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %q, want POST", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "Success",
			"data":   map[string]interface{}{"slug": "rule-1", "protocol": "tcp", "action": "allow"},
		})
	}))
	defer srv.Close()

	svc := acl.NewService(newClient(srv.URL))
	rule, err := svc.CreateRule(context.Background(), "my-vpc", acl.ACLRuleCreateRequest{Protocol: "tcp", Action: "allow"})
	if err != nil {
		t.Fatalf("CreateRule() error = %v", err)
	}
	if rule.Slug != "rule-1" {
		t.Errorf("slug = %q, want %q", rule.Slug, "rule-1")
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
