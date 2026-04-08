package userprofile_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/zsoftly/zcp-cli/internal/api/userprofile"
	"github.com/zsoftly/zcp-cli/internal/httpclient"
)

func newClient(baseURL string) *httpclient.Client {
	return httpclient.New(httpclient.Options{
		BaseURL:     baseURL,
		BearerToken: "test-token",
		Timeout:     5 * time.Second,
	})
}

// envelope wraps data in the STKCNSL response envelope.
func envelope(data interface{}) map[string]interface{} {
	return map[string]interface{}{
		"status":  "Success",
		"message": "OK",
		"data":    data,
	}
}

func TestGetProfile(t *testing.T) {
	profileData := map[string]interface{}{
		"user": map[string]interface{}{
			"id":    "user-123",
			"name":  "Test User",
			"email": "test@example.com",
			"account": map[string]interface{}{
				"id":     "acct-123",
				"crn":    "001001",
				"status": "ACTIVE",
			},
		},
	}

	var gotPath, gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotAuth = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(envelope(profileData))
	}))
	defer srv.Close()

	svc := userprofile.NewService(newClient(srv.URL))
	p, err := svc.Get(context.Background())
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if gotPath != "/profile" {
		t.Errorf("path = %q, want %q", gotPath, "/profile")
	}
	if gotAuth != "Bearer test-token" {
		t.Errorf("auth = %q, want %q", gotAuth, "Bearer test-token")
	}
	if p.User.ID != "user-123" {
		t.Errorf("user.ID = %q, want %q", p.User.ID, "user-123")
	}
	if p.User.Name != "Test User" {
		t.Errorf("user.Name = %q, want %q", p.User.Name, "Test User")
	}
	if p.User.Account.CRN != "001001" {
		t.Errorf("account.CRN = %q, want %q", p.User.Account.CRN, "001001")
	}
}

func TestUpdateProfile(t *testing.T) {
	var gotBody map[string]interface{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			http.Error(w, "expected PUT", http.StatusMethodNotAllowed)
			return
		}
		json.NewDecoder(r.Body).Decode(&gotBody)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(envelope(map[string]interface{}{
			"user": map[string]interface{}{
				"id":   "user-123",
				"name": "Updated Name",
			},
		}))
	}))
	defer srv.Close()

	svc := userprofile.NewService(newClient(srv.URL))
	p, err := svc.Update(context.Background(), userprofile.UpdateProfileRequest{Name: "Updated Name"})
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if gotBody["name"] != "Updated Name" {
		t.Errorf("body name = %v, want %q", gotBody["name"], "Updated Name")
	}
	if p.User.Name != "Updated Name" {
		t.Errorf("user.Name = %q, want %q", p.User.Name, "Updated Name")
	}
}

func TestChangePassword(t *testing.T) {
	var gotPath, gotMethod string
	var gotBody map[string]interface{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotMethod = r.Method
		json.NewDecoder(r.Body).Decode(&gotBody)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(envelope(nil))
	}))
	defer srv.Close()

	svc := userprofile.NewService(newClient(srv.URL))
	err := svc.ChangePassword(context.Background(), userprofile.ChangePasswordRequest{
		CurrentPassword:    "old123",
		NewPassword:        "new456",
		NewPasswordConfirm: "new456",
	})
	if err != nil {
		t.Fatalf("ChangePassword() error = %v", err)
	}
	if gotMethod != http.MethodPost {
		t.Errorf("method = %q, want %q", gotMethod, http.MethodPost)
	}
	if gotPath != "/users/change-password" {
		t.Errorf("path = %q, want %q", gotPath, "/users/change-password")
	}
	if gotBody["current_password"] != "old123" {
		t.Errorf("body current_password = %v, want %q", gotBody["current_password"], "old123")
	}
}

func TestEnableAPI(t *testing.T) {
	var gotPath, gotMethod string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotMethod = r.Method
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(envelope(nil))
	}))
	defer srv.Close()

	svc := userprofile.NewService(newClient(srv.URL))
	err := svc.EnableAPI(context.Background())
	if err != nil {
		t.Fatalf("EnableAPI() error = %v", err)
	}
	if gotMethod != http.MethodPost {
		t.Errorf("method = %q, want %q", gotMethod, http.MethodPost)
	}
	if gotPath != "/profile/api/enable" {
		t.Errorf("path = %q, want %q", gotPath, "/profile/api/enable")
	}
}

func TestDisableAPI(t *testing.T) {
	var gotPath, gotMethod string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotMethod = r.Method
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	svc := userprofile.NewService(newClient(srv.URL))
	err := svc.DisableAPI(context.Background())
	if err != nil {
		t.Fatalf("DisableAPI() error = %v", err)
	}
	if gotMethod != http.MethodDelete {
		t.Errorf("method = %q, want %q", gotMethod, http.MethodDelete)
	}
	if gotPath != "/profile/api/disable" {
		t.Errorf("path = %q, want %q", gotPath, "/profile/api/disable")
	}
}

func TestCreateUser(t *testing.T) {
	created := map[string]interface{}{
		"id":    "user-new",
		"name":  "New User",
		"email": "new@example.com",
	}

	var gotBody map[string]interface{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "expected POST", http.StatusMethodNotAllowed)
			return
		}
		json.NewDecoder(r.Body).Decode(&gotBody)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(envelope(created))
	}))
	defer srv.Close()

	svc := userprofile.NewService(newClient(srv.URL))
	u, err := svc.CreateUser(context.Background(), userprofile.CreateUserRequest{
		Name:  "New User",
		Email: "new@example.com",
	})
	if err != nil {
		t.Fatalf("CreateUser() error = %v", err)
	}
	if u.ID != "user-new" {
		t.Errorf("user.ID = %q, want %q", u.ID, "user-new")
	}
	if gotBody["email"] != "new@example.com" {
		t.Errorf("body email = %v, want %q", gotBody["email"], "new@example.com")
	}
}

func TestDeleteUser(t *testing.T) {
	var gotPath, gotMethod string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotMethod = r.Method
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	svc := userprofile.NewService(newClient(srv.URL))
	err := svc.DeleteUser(context.Background(), "user-del-1")
	if err != nil {
		t.Fatalf("DeleteUser() error = %v", err)
	}
	if gotMethod != http.MethodDelete {
		t.Errorf("method = %q, want %q", gotMethod, http.MethodDelete)
	}
	if gotPath != "/users/user-del-1" {
		t.Errorf("path = %q, want %q", gotPath, "/users/user-del-1")
	}
}
