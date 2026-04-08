package backup_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/zsoftly/zcp-cli/internal/api/backup"
	"github.com/zsoftly/zcp-cli/internal/httpclient"
)

type listResponse struct {
	Status      string          `json:"status"`
	Message     string          `json:"message"`
	CurrentPage int             `json:"current_page"`
	Data        []backup.Backup `json:"data"`
	Total       int             `json:"total"`
}

type singleResponse struct {
	Status  string        `json:"status"`
	Message string        `json:"message"`
	Data    backup.Backup `json:"data"`
}

func newTestClient(t *testing.T, srv *httptest.Server) *httpclient.Client {
	t.Helper()
	return httpclient.New(httpclient.Options{
		BaseURL:     srv.URL,
		BearerToken: "test-token",
		Timeout:     5 * time.Second,
	})
}

func TestBackupList(t *testing.T) {
	expected := []backup.Backup{
		{ID: "bak-1", Name: "backup-a", Slug: "backup-a", BlockstorageID: "vol-1"},
		{ID: "bak-2", Name: "backup-b", Slug: "backup-b", BlockstorageID: "vol-2"},
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/blockstorages/backups" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(listResponse{
			Status:  "Success",
			Message: "Ok",
			Data:    expected,
			Total:   len(expected),
		})
	}))
	defer srv.Close()

	svc := backup.NewService(newTestClient(t, srv))
	backups, err := svc.List(context.Background())
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(backups) != 2 {
		t.Fatalf("List() returned %d backups, want 2", len(backups))
	}
	if backups[0].ID != "bak-1" {
		t.Errorf("backups[0].ID = %q, want %q", backups[0].ID, "bak-1")
	}
}

func TestBackupCreate(t *testing.T) {
	expectedBackup := backup.Backup{
		ID:             "bak-new",
		Name:           "my-backup",
		Slug:           "my-backup",
		BlockstorageID: "vol-1",
		Interval:       "dailyAt",
		At:             1,
		Immediate:      true,
	}

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
		json.NewEncoder(w).Encode(singleResponse{
			Status:  "Success",
			Message: "Ok",
			Data:    expectedBackup,
		})
	}))
	defer srv.Close()

	svc := backup.NewService(newTestClient(t, srv))
	req := backup.CreateRequest{
		Interval:      "dailyAt",
		At:            1,
		Immediate:     1,
		CloudProvider: "nimbo",
		Region:        "noida",
		BillingCycle:  "hourly",
		Plan:          "backup-1",
		PseudoService: "Virtual Machine Backup",
		Project:       "default-73",
	}
	bak, err := svc.Create(context.Background(), "root-4153", req)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if bak.ID != "bak-new" {
		t.Errorf("bak.ID = %q, want %q", bak.ID, "bak-new")
	}
	if gotPath != "/blockstorages/root-4153/backups" {
		t.Errorf("path = %q, want %q", gotPath, "/blockstorages/root-4153/backups")
	}
	if gotBody["interval"] != "dailyAt" {
		t.Errorf("body interval = %v, want %q", gotBody["interval"], "dailyAt")
	}
}
