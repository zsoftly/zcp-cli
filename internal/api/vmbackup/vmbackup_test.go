package vmbackup_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/zsoftly/zcp-cli/internal/api/vmbackup"
	"github.com/zsoftly/zcp-cli/internal/httpclient"
)

func newTestClient(t *testing.T, srv *httptest.Server) *httpclient.Client {
	t.Helper()
	return httpclient.New(httpclient.Options{
		BaseURL:     srv.URL,
		BearerToken: "test-token",
		Timeout:     5 * time.Second,
	})
}

func TestVMBackupList(t *testing.T) {
	backups := []vmbackup.VMBackup{
		{ID: "vmb-1", Name: "daily-backup", Slug: "vmb-001001-0001", State: "Active", VirtualMachineID: "vm-1"},
		{ID: "vmb-2", Name: "weekly-backup", Slug: "vmb-001001-0002", State: "Active", VirtualMachineID: "vm-2"},
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/virtual-machines/backups" {
			http.NotFound(w, r)
			return
		}
		if r.Method != http.MethodGet {
			http.Error(w, "want GET", http.StatusMethodNotAllowed)
			return
		}
		data, _ := json.Marshal(backups)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  "Success",
			"message": "Ok",
			"data":    json.RawMessage(data),
			"total":   len(backups),
		})
	}))
	defer srv.Close()

	svc := vmbackup.NewService(newTestClient(t, srv))
	result, err := svc.List(context.Background())
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(result) != 2 {
		t.Fatalf("List() returned %d backups, want 2", len(result))
	}
	if result[0].ID != "vmb-1" {
		t.Errorf("result[0].ID = %q, want %q", result[0].ID, "vmb-1")
	}
	if result[1].Slug != "vmb-001001-0002" {
		t.Errorf("result[1].Slug = %q, want %q", result[1].Slug, "vmb-001001-0002")
	}
}

func TestVMBackupCreate(t *testing.T) {
	var gotPath string
	var gotBody map[string]interface{}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "want POST", http.StatusMethodNotAllowed)
			return
		}
		gotPath = r.URL.Path
		json.NewDecoder(r.Body).Decode(&gotBody)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  "Success",
			"message": "VM backup created.",
		})
	}))
	defer srv.Close()

	svc := vmbackup.NewService(newTestClient(t, srv))
	req := vmbackup.CreateRequest{
		Interval:      "daily",
		CloudProvider: "nimbo",
		Region:        "yow-1",
		BillingCycle:  "hourly",
		Plan:          "vm-backup-basic",
		PseudoService: "Virtual Machine Backup",
		Project:       "default",
	}
	resp, err := svc.Create(context.Background(), "my-vm", req)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if resp.Status != "Success" {
		t.Errorf("resp.Status = %q, want %q", resp.Status, "Success")
	}
	if gotPath != "/virtual-machines/my-vm/backups" {
		t.Errorf("path = %q, want %q", gotPath, "/virtual-machines/my-vm/backups")
	}
	if gotBody["cloud_provider"] != "nimbo" {
		t.Errorf("body cloud_provider = %v, want %q", gotBody["cloud_provider"], "nimbo")
	}
}

func TestVMBackupDelete(t *testing.T) {
	var gotPath, gotMethod string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	svc := vmbackup.NewService(newTestClient(t, srv))
	err := svc.Delete(context.Background(), "vmb-001001-0001")
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if gotMethod != http.MethodDelete {
		t.Errorf("method = %q, want %q", gotMethod, http.MethodDelete)
	}
	if gotPath != "/virtual-machines/backups/vmb-001001-0001" {
		t.Errorf("path = %q, want %q", gotPath, "/virtual-machines/backups/vmb-001001-0001")
	}
}

func TestVMBackupDelete_Error(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	svc := vmbackup.NewService(newTestClient(t, srv))
	err := svc.Delete(context.Background(), "does-not-exist")
	if err == nil {
		t.Fatal("Delete() expected error on 404, got nil")
	}
}
