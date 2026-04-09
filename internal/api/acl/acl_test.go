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
