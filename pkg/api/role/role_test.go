package role_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/zsoftly/zcp-cli/pkg/api/role"
	"github.com/zsoftly/zcp-cli/pkg/httpclient"
)

func newClient(baseURL string) *httpclient.Client {
	return httpclient.New(httpclient.Options{
		BaseURL:     baseURL,
		BearerToken: "test-token",
		Timeout:     5 * time.Second,
	})
}

func TestList(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/roles" {
			t.Errorf("path = %q", r.URL.Path)
		}
		roles := []role.Role{
			{ID: "r1", Name: "Owner", Slug: "owner", Description: "default role"},
			{ID: "r2", Name: "Service Viewer", Slug: "service-viewer"},
		}
		data, _ := json.Marshal(roles)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"status": "Success", "data": json.RawMessage(data)})
	}))
	defer srv.Close()

	roles, err := role.NewService(newClient(srv.URL)).List(context.Background())
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(roles) != 2 || roles[0].Slug != "owner" {
		t.Fatalf("unexpected roles: %+v", roles)
	}
}

func TestGetBySlugIncludesPermissions(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/roles/owner" {
			t.Errorf("path = %q, want /roles/owner", r.URL.Path)
		}
		body := `{"status":"Success","data":{"id":"r1","name":"Owner","slug":"owner","permissions":[{"slug":"dns-read","name":"DNS Read","category":"DNS"}],"users":[{"id":"u1","name":"Z","email":"z@z.ca"}]}}`
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, body)
	}))
	defer srv.Close()

	r, err := role.NewService(newClient(srv.URL)).Get(context.Background(), "owner")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if len(r.Permissions) != 1 || r.Permissions[0].Slug != "dns-read" {
		t.Errorf("permissions = %+v", r.Permissions)
	}
	if len(r.Users) != 1 || r.Users[0].Email != "z@z.ca" {
		t.Errorf("users = %+v", r.Users)
	}
}

func TestCreateSendsPermissionSlugs(t *testing.T) {
	var gotBody map[string]interface{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/roles" {
			t.Errorf("method/path = %s %s", r.Method, r.URL.Path)
		}
		json.NewDecoder(r.Body).Decode(&gotBody)
		io.WriteString(w, `{"status":"Success","data":{"id":"r9","name":"custom","slug":"custom"}}`)
	}))
	defer srv.Close()

	r, err := role.NewService(newClient(srv.URL)).Create(context.Background(), role.CreateRequest{
		Name: "custom", Description: "d", Permissions: []string{"dns-read", "project-read"},
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if r.Slug != "custom" {
		t.Errorf("slug = %q", r.Slug)
	}
	perms, _ := gotBody["permissions"].([]interface{})
	if len(perms) != 2 {
		t.Errorf("sent permissions = %v, want 2 slugs", gotBody["permissions"])
	}
}

func TestUpdateUsesPutBySlug(t *testing.T) {
	var method, path string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		method, path = r.Method, r.URL.Path
		io.WriteString(w, `{"status":"Success","data":{"slug":"custom","description":"v2"}}`)
	}))
	defer srv.Close()

	_, err := role.NewService(newClient(srv.URL)).Update(context.Background(), "custom", role.UpdateRequest{
		Name: "custom", Description: "v2", Permissions: []string{"dns-read"},
	})
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if method != http.MethodPut || path != "/roles/custom" {
		t.Errorf("got %s %s, want PUT /roles/custom", method, path)
	}
}

func TestDeleteBySlug(t *testing.T) {
	var method, path string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		method, path = r.Method, r.URL.Path
		io.WriteString(w, `{"status":"Success","message":"Role deleted successfully."}`)
	}))
	defer srv.Close()

	if err := role.NewService(newClient(srv.URL)).Delete(context.Background(), "custom"); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if method != http.MethodDelete || path != "/roles/custom" {
		t.Errorf("got %s %s, want DELETE /roles/custom", method, path)
	}
}
