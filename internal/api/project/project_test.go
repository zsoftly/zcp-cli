package project_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/zsoftly/zcp-cli/internal/api/project"
	"github.com/zsoftly/zcp-cli/internal/httpclient"
)

func newClient(baseURL string) *httpclient.Client {
	return httpclient.New(httpclient.Options{
		BaseURL:     baseURL,
		BearerToken: "test-token",
		Timeout:     5 * time.Second,
	})
}

// wrap produces a STKCNSL envelope response.
func wrap(data interface{}) map[string]interface{} {
	return map[string]interface{}{
		"status": "Success",
		"data":   data,
	}
}

func TestProjectList(t *testing.T) {
	expected := []project.Project{
		{ID: "1", Name: "Alpha", Slug: "alpha", Description: "First project"},
		{ID: "2", Name: "Beta", Slug: "beta", Description: "Second project"},
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/projects" {
			http.NotFound(w, r)
			return
		}
		if r.Method != http.MethodGet {
			http.Error(w, "expected GET", http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(wrap(expected))
	}))
	defer srv.Close()

	svc := project.NewService(newClient(srv.URL))
	projects, err := svc.List(context.Background())
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(projects) != 2 {
		t.Fatalf("List() returned %d projects, want 2", len(projects))
	}
	if projects[0].Slug != "alpha" {
		t.Errorf("projects[0].Slug = %q, want %q", projects[0].Slug, "alpha")
	}
}

func TestProjectCreate(t *testing.T) {
	created := project.Project{
		ID:          "3",
		Name:        "Gamma",
		Slug:        "gamma",
		Description: "New project",
	}

	var gotBody map[string]interface{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "expected POST", http.StatusMethodNotAllowed)
			return
		}
		if r.URL.Path != "/projects" {
			http.NotFound(w, r)
			return
		}
		json.NewDecoder(r.Body).Decode(&gotBody)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(wrap(created))
	}))
	defer srv.Close()

	svc := project.NewService(newClient(srv.URL))
	result, err := svc.Create(context.Background(), project.CreateRequest{
		Name:        "Gamma",
		Description: "New project",
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if result.Slug != "gamma" {
		t.Errorf("result.Slug = %q, want %q", result.Slug, "gamma")
	}
	if gotBody["name"] != "Gamma" {
		t.Errorf("body name = %v, want %q", gotBody["name"], "Gamma")
	}
}

func TestProjectUpdate(t *testing.T) {
	updated := project.Project{
		ID:          "1",
		Name:        "Alpha Renamed",
		Slug:        "alpha",
		Description: "Updated description",
	}

	var gotBody map[string]interface{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			http.Error(w, "expected PUT", http.StatusMethodNotAllowed)
			return
		}
		if r.URL.Path != "/projects/alpha" {
			http.NotFound(w, r)
			return
		}
		json.NewDecoder(r.Body).Decode(&gotBody)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(wrap(updated))
	}))
	defer srv.Close()

	svc := project.NewService(newClient(srv.URL))
	result, err := svc.Update(context.Background(), "alpha", project.UpdateRequest{
		Name:        "Alpha Renamed",
		Description: "Updated description",
	})
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if result.Name != "Alpha Renamed" {
		t.Errorf("result.Name = %q, want %q", result.Name, "Alpha Renamed")
	}
}

func TestProjectDashboard(t *testing.T) {
	expected := []project.DashboardService{
		{Name: "web-app", Type: "compute", Status: "Running", Count: 3},
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/projects/dashboard/alpha/services" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(wrap(expected))
	}))
	defer srv.Close()

	svc := project.NewService(newClient(srv.URL))
	services, err := svc.Dashboard(context.Background(), "alpha")
	if err != nil {
		t.Fatalf("Dashboard() error = %v", err)
	}
	if len(services) != 1 {
		t.Fatalf("Dashboard() returned %d services, want 1", len(services))
	}
	if services[0].Name != "web-app" {
		t.Errorf("services[0].Name = %q, want %q", services[0].Name, "web-app")
	}
}

func TestProjectListIcons(t *testing.T) {
	expected := []project.Icon{
		{ID: "1", Name: "server", URL: "https://icons.example.com/server.svg"},
		{ID: "2", Name: "database", URL: "https://icons.example.com/database.svg"},
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/project-icons" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(wrap(expected))
	}))
	defer srv.Close()

	svc := project.NewService(newClient(srv.URL))
	icons, err := svc.ListIcons(context.Background())
	if err != nil {
		t.Fatalf("ListIcons() error = %v", err)
	}
	if len(icons) != 2 {
		t.Fatalf("ListIcons() returned %d icons, want 2", len(icons))
	}
	if icons[0].Name != "server" {
		t.Errorf("icons[0].Name = %q, want %q", icons[0].Name, "server")
	}
}

func TestProjectListUsers(t *testing.T) {
	expected := []project.User{
		{ID: "10", Name: "Alice", Email: "alice@example.com", Role: "admin"},
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/projects/alpha/users" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(wrap(expected))
	}))
	defer srv.Close()

	svc := project.NewService(newClient(srv.URL))
	users, err := svc.ListUsers(context.Background(), "alpha")
	if err != nil {
		t.Fatalf("ListUsers() error = %v", err)
	}
	if len(users) != 1 {
		t.Fatalf("ListUsers() returned %d users, want 1", len(users))
	}
	if users[0].Email != "alice@example.com" {
		t.Errorf("users[0].Email = %q, want %q", users[0].Email, "alice@example.com")
	}
}

func TestProjectAddUser(t *testing.T) {
	added := project.User{
		ID:    "11",
		Name:  "Bob",
		Email: "bob@example.com",
		Role:  "member",
	}

	var gotBody map[string]interface{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "expected POST", http.StatusMethodNotAllowed)
			return
		}
		if r.URL.Path != "/projects/alpha/users" {
			http.NotFound(w, r)
			return
		}
		json.NewDecoder(r.Body).Decode(&gotBody)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(wrap(added))
	}))
	defer srv.Close()

	svc := project.NewService(newClient(srv.URL))
	result, err := svc.AddUser(context.Background(), "alpha", project.AddUserRequest{
		Email: "bob@example.com",
		Role:  "member",
	})
	if err != nil {
		t.Fatalf("AddUser() error = %v", err)
	}
	if result.Email != "bob@example.com" {
		t.Errorf("result.Email = %q, want %q", result.Email, "bob@example.com")
	}
	if gotBody["email"] != "bob@example.com" {
		t.Errorf("body email = %v, want %q", gotBody["email"], "bob@example.com")
	}
}

func TestProjectListError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "internal error", http.StatusInternalServerError)
	}))
	defer srv.Close()

	svc := project.NewService(newClient(srv.URL))
	_, err := svc.List(context.Background())
	if err == nil {
		t.Fatal("List() expected error on 500, got nil")
	}
}
