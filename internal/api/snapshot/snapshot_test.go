package snapshot_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/zsoftly/zcp-cli/internal/api/snapshot"
	"github.com/zsoftly/zcp-cli/internal/httpclient"
)

type listResponse struct {
	Status      string              `json:"status"`
	Message     string              `json:"message"`
	CurrentPage int                 `json:"current_page"`
	Data        []snapshot.Snapshot `json:"data"`
	Total       int                 `json:"total"`
}

type singleResponse struct {
	Status  string            `json:"status"`
	Message string            `json:"message"`
	Data    snapshot.Snapshot `json:"data"`
}

func newTestClient(t *testing.T, srv *httptest.Server) *httpclient.Client {
	t.Helper()
	return httpclient.New(httpclient.Options{
		BaseURL:     srv.URL,
		BearerToken: "test-token",
		Timeout:     5 * time.Second,
	})
}

func TestSnapshotList(t *testing.T) {
	expected := []snapshot.Snapshot{
		{ID: "snap-1", Name: "snap-a", Slug: "snap-a", BlockstorageID: "vol-1"},
		{ID: "snap-2", Name: "snap-b", Slug: "snap-b", BlockstorageID: "vol-2"},
	}

	var gotInclude string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/blockstorages/snapshots" {
			http.NotFound(w, r)
			return
		}
		gotInclude = r.URL.Query().Get("include")
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(listResponse{
			Status:  "Success",
			Message: "Ok",
			Data:    expected,
			Total:   len(expected),
		})
	}))
	defer srv.Close()

	svc := snapshot.NewService(newTestClient(t, srv))
	snapshots, err := svc.List(context.Background())
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(snapshots) != 2 {
		t.Fatalf("List() returned %d snapshots, want 2", len(snapshots))
	}
	if gotInclude == "" {
		t.Error("include query param was empty, expected relations")
	}
	if snapshots[0].ID != "snap-1" {
		t.Errorf("snapshots[0].ID = %q, want %q", snapshots[0].ID, "snap-1")
	}
}

func TestSnapshotCreate(t *testing.T) {
	expectedSnap := snapshot.Snapshot{
		ID:             "snap-new",
		Name:           "my-snap",
		Slug:           "my-snap",
		BlockstorageID: "vol-1",
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
			Data:    expectedSnap,
		})
	}))
	defer srv.Close()

	svc := snapshot.NewService(newTestClient(t, srv))
	req := snapshot.CreateRequest{
		Name:          "my-snap",
		Plan:          "snapshot-per-gb",
		Service:       "Block Storage Snapshot",
		CloudProvider: "nimbo",
		Region:        "noida",
		BillingCycle:  "hourly",
		Project:       "default-73",
	}
	snap, err := svc.Create(context.Background(), "root-4153", req)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if snap.ID != "snap-new" {
		t.Errorf("snap.ID = %q, want %q", snap.ID, "snap-new")
	}
	if gotPath != "/blockstorages/root-4153/snapshots" {
		t.Errorf("path = %q, want %q", gotPath, "/blockstorages/root-4153/snapshots")
	}
	if gotBody["name"] != "my-snap" {
		t.Errorf("body name = %v, want %q", gotBody["name"], "my-snap")
	}
}

func TestSnapshotRevert(t *testing.T) {
	expectedSnap := snapshot.Snapshot{
		ID:             "snap-1",
		Name:           "snap-a",
		Slug:           "snap-a",
		BlockstorageID: "vol-1",
	}

	var gotPath, gotMethod string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(singleResponse{
			Status:  "Success",
			Message: "Ok",
			Data:    expectedSnap,
		})
	}))
	defer srv.Close()

	svc := snapshot.NewService(newTestClient(t, srv))
	snap, err := svc.Revert(context.Background(), "root-4153", "snap-a")
	if err != nil {
		t.Fatalf("Revert() error = %v", err)
	}
	if gotMethod != http.MethodPost {
		t.Errorf("method = %q, want %q", gotMethod, http.MethodPost)
	}
	if gotPath != "/blockstorages/root-4153/snapshots/snap-a/revert" {
		t.Errorf("path = %q, want %q", gotPath, "/blockstorages/root-4153/snapshots/snap-a/revert")
	}
	if snap.ID != "snap-1" {
		t.Errorf("snap.ID = %q, want %q", snap.ID, "snap-1")
	}
}
