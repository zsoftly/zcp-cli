package vmbackup_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/zsoftly/zcp-cli/pkg/api/vmbackup"
	"github.com/zsoftly/zcp-cli/pkg/httpclient"
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
	result, err := svc.List(context.Background(), "", "")
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
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
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
		Project:       "default-9",
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
	var gotBody map[string]interface{}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  "Success",
			"message": "We are in the process of deleting this service.",
		})
	}))
	defer srv.Close()

	svc := vmbackup.NewService(newTestClient(t, srv))
	err := svc.Delete(context.Background(), "vmb-001001-0001")
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if gotMethod != http.MethodPost {
		t.Errorf("method = %q, want %q", gotMethod, http.MethodPost)
	}
	if gotPath != "/billing/service-cancel-requests/vmb-001001-0001" {
		t.Errorf("path = %q, want %q", gotPath, "/billing/service-cancel-requests/vmb-001001-0001")
	}
	wantBody := map[string]interface{}{
		"service_name": "Backups",
		"reason":       "not_needed_anymore",
		"type":         "Immediate",
		"status":       "Pending",
	}
	for k, want := range wantBody {
		if got := gotBody[k]; got != want {
			t.Errorf("body[%q] = %v, want %v", k, got, want)
		}
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

func TestVMBackupListPagination(t *testing.T) {
	page1 := `{"status":"Success","message":"Ok","current_page":1,"last_page":2,"total":2,
		"data":[{"id":"vmb-1","name":"a","slug":"vmb-a"}]}`
	page2 := `{"status":"Success","message":"Ok","current_page":2,"last_page":2,"total":2,
		"data":[{"id":"vmb-2","name":"b","slug":"vmb-b"}]}`

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

	svc := vmbackup.NewService(newTestClient(t, srv))
	result, err := svc.List(context.Background(), "", "")
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if calls != 2 {
		t.Fatalf("server received %d requests, want 2", calls)
	}
	if len(result) != 2 {
		t.Fatalf("List() returned %d backups, want 2", len(result))
	}
	if result[0].Slug != "vmb-a" || result[1].Slug != "vmb-b" {
		t.Errorf("result slugs = [%q, %q], want [vmb-a, vmb-b]", result[0].Slug, result[1].Slug)
	}
}

func TestVMBackupListPaginationIgnoresEchoedCurrentPage(t *testing.T) {
	// A server that ignores the ?page query parameter entirely and always
	// echoes current_page:1, last_page:2 must not send List into an
	// unbounded loop: the loop counter, not the server-echoed
	// env.CurrentPage, decides when to stop. Before the fix this stub
	// drove List to maxListPages (1000) requests and 1000 duplicate rows;
	// after the fix it must stop after exactly 2 requests.
	const body = `{"status":"Success","message":"Ok","current_page":1,"last_page":2,"total":2,
		"data":[{"id":"vmb-1","name":"a","slug":"vmb-a"}]}`

	var calls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, body)
	}))
	defer srv.Close()

	svc := vmbackup.NewService(newTestClient(t, srv))
	result, err := svc.List(context.Background(), "", "")
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if calls != 2 {
		t.Fatalf("server received %d requests, want exactly 2", calls)
	}
	if len(result) != 2 {
		t.Fatalf("List() returned %d backups, want 2 (no duplicates beyond 2 pages)", len(result))
	}
}

func TestVMBackupVMSlug(t *testing.T) {
	nested := vmbackup.VMBackup{VirtualMachineID: "vm-fallback", VirtualMachine: &vmbackup.VMRef{Slug: "adj-headscale"}}
	if got := nested.VMSlug(); got != "adj-headscale" {
		t.Errorf("VMSlug() = %q, want %q", got, "adj-headscale")
	}
	fallback := vmbackup.VMBackup{VirtualMachineID: "vm-fallback"}
	if got := fallback.VMSlug(); got != "vm-fallback" {
		t.Errorf("VMSlug() = %q, want %q", got, "vm-fallback")
	}
}

func TestVMBackupAtDecoding(t *testing.T) {
	tests := []struct {
		name    string
		json    string
		wantAt  int
		wantErr bool
	}{
		{name: "quoted numeric string", json: `{"id":"vmb-1","at":"3"}`, wantAt: 3},
		{name: "json number", json: `{"id":"vmb-1","at":3}`, wantAt: 3},
		{name: "null", json: `{"id":"vmb-1","at":null}`, wantAt: 0},
		{name: "non-numeric string", json: `{"id":"vmb-1","at":"soon"}`, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var v vmbackup.VMBackup
			err := json.Unmarshal([]byte(tt.json), &v)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("Unmarshal() expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("Unmarshal() error = %v", err)
			}
			if v.At != tt.wantAt {
				t.Errorf("At = %d, want %d", v.At, tt.wantAt)
			}
		})
	}
}
