package subuser_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/zsoftly/zcp-cli/pkg/api/subuser"
	"github.com/zsoftly/zcp-cli/pkg/httpclient"
)

func newClient(baseURL string) *httpclient.Client {
	return httpclient.New(httpclient.Options{
		BaseURL:     baseURL,
		BearerToken: "test-token",
		Timeout:     5 * time.Second,
	})
}

const oneUserBody = `{"status":"Success","data":[{"id":"u1","name":"Test","email":"t@z.ca","user_type":"sub_user","is_blocked":false,"user_status":"Active","role":{"id":"r1","name":"Service Viewer","slug":"service-viewer"},"projects":[{"id":"p1","name":"Default","slug":"default-9"}]}]}`

func TestList(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/users" {
			t.Errorf("path = %q", r.URL.Path)
		}
		io.WriteString(w, oneUserBody)
	}))
	defer srv.Close()

	users, err := subuser.NewService(newClient(srv.URL)).List(context.Background())
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(users) != 1 {
		t.Fatalf("got %d users, want 1", len(users))
	}
	u := users[0]
	if u.RoleSlug() != "service-viewer" {
		t.Errorf("RoleSlug() = %q", u.RoleSlug())
	}
	if got := u.ProjectSlugs(); len(got) != 1 || got[0] != "default-9" {
		t.Errorf("ProjectSlugs() = %v", got)
	}
}

func TestCreate(t *testing.T) {
	var body subuser.CreateRequest
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/users" {
			t.Errorf("method/path = %s %s", r.Method, r.URL.Path)
		}
		json.NewDecoder(r.Body).Decode(&body)
		io.WriteString(w, `{"status":"Success","data":{"id":"u9","name":"New","email":"new@z.ca"}}`)
	}))
	defer srv.Close()

	u, err := subuser.NewService(newClient(srv.URL)).Create(context.Background(), subuser.CreateRequest{
		Name: "New", Email: "new@z.ca", Password: "Abc12345!", Role: "service-viewer",
		Projects: []string{"default-9"}, IsUserPassword: true, AuthUser: "customer",
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if u.ID != "u9" {
		t.Errorf("id = %q", u.ID)
	}
	if !body.IsUserPassword || body.AuthUser != "customer" || len(body.Projects) != 1 {
		t.Errorf("sent body = %+v", body)
	}
}

func TestUpdateByID(t *testing.T) {
	var method, path string
	var body subuser.UpdateRequest
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		method, path = r.Method, r.URL.Path
		json.NewDecoder(r.Body).Decode(&body)
		io.WriteString(w, `{"status":"Success","data":{"id":"u1","is_blocked":true}}`)
	}))
	defer srv.Close()

	_, err := subuser.NewService(newClient(srv.URL)).Update(context.Background(), "u1", subuser.UpdateRequest{
		Email: "t@z.ca", Projects: []string{"default-9"}, IsBlocked: true,
	})
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if method != http.MethodPut || path != "/users/u1" {
		t.Errorf("got %s %s, want PUT /users/u1", method, path)
	}
	if !body.IsBlocked || body.Email != "t@z.ca" {
		t.Errorf("sent body = %+v", body)
	}
}

func TestDeleteByID(t *testing.T) {
	var method, path string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		method, path = r.Method, r.URL.Path
		io.WriteString(w, `{"status":"Success","message":"deleted"}`)
	}))
	defer srv.Close()

	if err := subuser.NewService(newClient(srv.URL)).Delete(context.Background(), "u1"); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if method != http.MethodDelete || path != "/users/u1" {
		t.Errorf("got %s %s, want DELETE /users/u1", method, path)
	}
}
