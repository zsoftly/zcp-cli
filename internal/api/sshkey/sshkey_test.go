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
		BaseURL:   baseURL,
		APIKey:    "testkey",
		SecretKey: "testsecret",
		Timeout:   5 * time.Second,
	})
}

type listSSHKeyResponse struct {
	Count              int             `json:"count"`
	ListSSHKeyResponse []sshkey.SSHKey `json:"listSSHKeyResponse"`
}

func TestSSHKeyList(t *testing.T) {
	expected := []sshkey.SSHKey{
		{UUID: "key-1", Name: "my-key", Status: "active", IsActive: true, DomainName: "default"},
		{UUID: "key-2", Name: "other-key", Status: "active", IsActive: true, DomainName: "default"},
	}

	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		if r.URL.Path != "/restapi/sshkey/sshkeyList" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(listSSHKeyResponse{Count: len(expected), ListSSHKeyResponse: expected})
	}))
	defer srv.Close()

	svc := sshkey.NewService(newClient(srv.URL))
	keys, err := svc.List(context.Background())
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if gotPath != "/restapi/sshkey/sshkeyList" {
		t.Errorf("path = %q, want %q", gotPath, "/restapi/sshkey/sshkeyList")
	}
	if len(keys) != 2 {
		t.Fatalf("List() returned %d keys, want 2", len(keys))
	}
	if keys[0].UUID != "key-1" {
		t.Errorf("keys[0].UUID = %q, want %q", keys[0].UUID, "key-1")
	}
	if keys[1].Name != "other-key" {
		t.Errorf("keys[1].Name = %q, want %q", keys[1].Name, "other-key")
	}
}

func TestSSHKeyCreate(t *testing.T) {
	created := sshkey.SSHKey{
		UUID:       "key-new",
		Name:       "imported-key",
		Status:     "active",
		IsActive:   true,
		DomainName: "default",
	}

	var gotBody map[string]interface{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "expected POST", http.StatusMethodNotAllowed)
			return
		}
		if r.URL.Path != "/restapi/sshkey/createSSHkey" {
			http.NotFound(w, r)
			return
		}
		json.NewDecoder(r.Body).Decode(&gotBody)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(listSSHKeyResponse{Count: 1, ListSSHKeyResponse: []sshkey.SSHKey{created}})
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
	if key.UUID != "key-new" {
		t.Errorf("key.UUID = %q, want %q", key.UUID, "key-new")
	}
	if gotBody["name"] != "imported-key" {
		t.Errorf("body name = %v, want %q", gotBody["name"], "imported-key")
	}
	if gotBody["publicKey"] != "ssh-rsa AAAAB3NzaC1yc2EAAAA test@host" {
		t.Errorf("body publicKey = %v, want publicKey value", gotBody["publicKey"])
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
	if gotPath != "/restapi/sshkey/deleteSSHkey/key-del-1" {
		t.Errorf("path = %q, want %q", gotPath, "/restapi/sshkey/deleteSSHkey/key-del-1")
	}
}
