package sshkey_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/zsoftly/zcp-cli/internal/api/sshkey"
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

func TestSSHKeyList(t *testing.T) {
	expected := []map[string]interface{}{
		{"id": "key-1", "name": "my-key", "slug": "my-key", "public_key": "ssh-rsa AAA..."},
		{"id": "key-2", "name": "other-key", "slug": "other-key", "public_key": "ssh-ed25519 BBB..."},
	}

	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		if r.URL.Path != "/users/ssh-keys" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(envelope(expected))
	}))
	defer srv.Close()

	svc := sshkey.NewService(newClient(srv.URL))
	keys, err := svc.List(context.Background())
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if gotPath != "/users/ssh-keys" {
		t.Errorf("path = %q, want %q", gotPath, "/users/ssh-keys")
	}
	if len(keys) != 2 {
		t.Fatalf("List() returned %d keys, want 2", len(keys))
	}
	if keys[0].ID != "key-1" {
		t.Errorf("keys[0].ID = %q, want %q", keys[0].ID, "key-1")
	}
	if keys[1].Name != "other-key" {
		t.Errorf("keys[1].Name = %q, want %q", keys[1].Name, "other-key")
	}
}

func TestSSHKeyCreate(t *testing.T) {
	created := map[string]interface{}{
		"id":         "key-new",
		"name":       "imported-key",
		"slug":       "imported-key",
		"public_key": "ssh-rsa AAAAB3NzaC1yc2EAAAA test@host",
	}

	var gotBody map[string]interface{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "expected POST", http.StatusMethodNotAllowed)
			return
		}
		if r.URL.Path != "/users/ssh-keys" {
			http.NotFound(w, r)
			return
		}
		json.NewDecoder(r.Body).Decode(&gotBody)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(envelope(created))
	}))
	defer srv.Close()

	svc := sshkey.NewService(newClient(srv.URL))
	req := sshkey.CreateRequest{
		Name:      "imported-key",
		PublicKey: "ssh-rsa AAAAB3NzaC1yc2EAAAA test@host",
	}
	key, err := svc.Create(context.Background(), req)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if key.ID != "key-new" {
		t.Errorf("key.ID = %q, want %q", key.ID, "key-new")
	}
	if gotBody["name"] != "imported-key" {
		t.Errorf("body name = %v, want %q", gotBody["name"], "imported-key")
	}
	if gotBody["public_key"] != "ssh-rsa AAAAB3NzaC1yc2EAAAA test@host" {
		t.Errorf("body public_key = %v, want public_key value", gotBody["public_key"])
	}
}

func TestSSHKeyDelete(t *testing.T) {
	var gotPath, gotMethod string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	svc := sshkey.NewService(newClient(srv.URL))
	err := svc.Delete(context.Background(), "key-del-1")
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if gotMethod != http.MethodDelete {
		t.Errorf("method = %q, want %q", gotMethod, http.MethodDelete)
	}
	if gotPath != "/users/ssh-keys/key-del-1" {
		t.Errorf("path = %q, want %q", gotPath, "/users/ssh-keys/key-del-1")
	}
}
