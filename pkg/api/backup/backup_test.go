package backup_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/zsoftly/zcp-cli/pkg/api/backup"
	"github.com/zsoftly/zcp-cli/pkg/httpclient"
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
	backups, err := svc.List(context.Background(), "", "")
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

func TestBackupListPagination(t *testing.T) {
	page1 := `{"status":"Success","message":"Ok","current_page":1,"last_page":2,"total":2,
		"data":[{"id":"bak-1","name":"a","slug":"backup-a","blockstorage_id":"vol-1"}]}`
	page2 := `{"status":"Success","message":"Ok","current_page":2,"last_page":2,"total":2,
		"data":[{"id":"bak-2","name":"b","slug":"backup-b","blockstorage_id":"vol-2"}]}`

	var calls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Query().Get("page") == "2" {
			fmt.Fprint(w, page2)
			return
		}
		fmt.Fprint(w, page1)
	}))
	defer srv.Close()

	svc := backup.NewService(newTestClient(t, srv))
	backups, err := svc.List(context.Background(), "", "")
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if calls != 2 {
		t.Fatalf("server received %d requests, want 2", calls)
	}
	if len(backups) != 2 {
		t.Fatalf("List() returned %d backups, want 2", len(backups))
	}
	if backups[0].Slug != "backup-a" || backups[1].Slug != "backup-b" {
		t.Errorf("backups slugs = [%q, %q], want [backup-a, backup-b]", backups[0].Slug, backups[1].Slug)
	}
}

func TestBackupListPaginationIgnoresEchoedCurrentPage(t *testing.T) {
	// A server that ignores the ?page query parameter entirely and always
	// echoes current_page:1, last_page:2 must not send List into an
	// unbounded loop, and must not hand back the repeated page as duplicate
	// rows. List requests page 2, sees current_page:1 echoed back, and
	// returns an error after exactly 2 requests.
	const body = `{"status":"Success","message":"Ok","current_page":1,"last_page":2,"total":2,
		"data":[{"id":"bak-1","name":"a","slug":"backup-a","blockstorage_id":"vol-1"}]}`

	var calls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, body)
	}))
	defer srv.Close()

	svc := backup.NewService(newTestClient(t, srv))
	backups, err := svc.List(context.Background(), "", "")
	if err == nil {
		t.Fatalf("List() error = nil, want an error for a repeated page (got %d backups)", len(backups))
	}
	if !strings.Contains(err.Error(), "requested page 2 but the API returned page 1") {
		t.Errorf("List() error = %q, want it to name the page mismatch", err)
	}
	if calls != 2 {
		t.Fatalf("server received %d requests, want exactly 2", calls)
	}
	if backups != nil {
		t.Fatalf("List() returned %d backups alongside the error, want none", len(backups))
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

func TestBackupDelete(t *testing.T) {
	var gotPath, gotMethod string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	svc := backup.NewService(newTestClient(t, srv))
	err := svc.Delete(context.Background(), "bk-001001-0001")
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if gotMethod != http.MethodDelete {
		t.Errorf("method = %q, want %q", gotMethod, http.MethodDelete)
	}
	if gotPath != "/blockstorages/backups/bk-001001-0001" {
		t.Errorf("path = %q, want %q", gotPath, "/blockstorages/backups/bk-001001-0001")
	}
}

func TestBackupAtDecoding(t *testing.T) {
	tests := []struct {
		name    string
		json    string
		wantAt  int
		wantErr bool
	}{
		{name: "quoted numeric string", json: `{"id":"bak-1","at":"3"}`, wantAt: 3},
		{name: "json number", json: `{"id":"bak-1","at":3}`, wantAt: 3},
		{name: "null", json: `{"id":"bak-1","at":null}`, wantAt: 0},
		{name: "missing", json: `{"id":"bak-1"}`, wantAt: 0},
		{name: "non-numeric string", json: `{"id":"bak-1","at":"soon"}`, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var b backup.Backup
			err := json.Unmarshal([]byte(tt.json), &b)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("Unmarshal() expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("Unmarshal() error = %v", err)
			}
			if b.At != tt.wantAt {
				t.Errorf("At = %d, want %d", b.At, tt.wantAt)
			}
		})
	}
}

func TestBackupListDecodesNestedBlockstorage(t *testing.T) {
	body := `{
		"status": "Success",
		"message": "Ok",
		"current_page": 1,
		"total": 1,
		"data": [{
			"id": "bak-1",
			"name": "backup-jtest-dailyAt",
			"slug": "backup-jtest-dailyat",
			"interval": "dailyAt",
			"day": null,
			"at": "3",
			"scheduled_at": "07-09-2026 03:00:00 am",
			"created_at": "2026-07-09T00:00:00.000000Z",
			"updated_at": "2026-07-09T00:00:00.000000Z",
			"deleted_at": null,
			"all_time_consumption": 7e-05,
			"blockstorage": {"id": "vol-uuid", "slug": "jtest", "name": "jtest", "size": "8"}
		}]
	}`

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, body)
	}))
	defer srv.Close()

	svc := backup.NewService(newTestClient(t, srv))
	backups, err := svc.List(context.Background(), "", "")
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(backups) != 1 {
		t.Fatalf("List() returned %d backups, want 1", len(backups))
	}
	b := backups[0]
	if b.At != 3 {
		t.Errorf("At = %d, want 3", b.At)
	}
	if b.BlockstorageID != "" {
		t.Errorf("BlockstorageID = %q, want empty (list responses have no top-level field)", b.BlockstorageID)
	}
	if got := b.VolumeSlug(); got != "jtest" {
		t.Errorf("VolumeSlug() = %q, want %q", got, "jtest")
	}
}

func TestBackupDelete_Error(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	svc := backup.NewService(newTestClient(t, srv))
	err := svc.Delete(context.Background(), "bk-does-not-exist")
	if err == nil {
		t.Fatal("Delete() expected error, got nil")
	}
}
