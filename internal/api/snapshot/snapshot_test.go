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

type listSnapshotResponse struct {
	Count                int                 `json:"count"`
	ListSnapShotResponse []snapshot.Snapshot `json:"listSnapShotResponse"`
}

func newTestClient(t *testing.T, srv *httptest.Server) *httpclient.Client {
	t.Helper()
	return httpclient.New(httpclient.Options{
		BaseURL:   srv.URL,
		APIKey:    "k",
		SecretKey: "s",
		Timeout:   5 * time.Second,
	})
}

func TestSnapshotList(t *testing.T) {
	expected := []snapshot.Snapshot{
		{UUID: "snap-1", Name: "snap-a", Status: "BackedUp", VolumeUUID: "vol-1", ZoneUUID: "zone-1"},
		{UUID: "snap-2", Name: "snap-b", Status: "BackedUp", VolumeUUID: "vol-2", ZoneUUID: "zone-1"},
	}

	var gotZone string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/restapi/snapshot/snapshotList" {
			http.NotFound(w, r)
			return
		}
		gotZone = r.URL.Query().Get("zoneUuid")
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(listSnapshotResponse{Count: len(expected), ListSnapShotResponse: expected})
	}))
	defer srv.Close()

	svc := snapshot.NewService(newTestClient(t, srv))
	snapshots, err := svc.List(context.Background(), "zone-1", "")
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(snapshots) != 2 {
		t.Fatalf("List() returned %d snapshots, want 2", len(snapshots))
	}
	if gotZone != "zone-1" {
		t.Errorf("zoneUuid query param = %q, want %q", gotZone, "zone-1")
	}
	if snapshots[0].UUID != "snap-1" {
		t.Errorf("snapshots[0].UUID = %q, want %q", snapshots[0].UUID, "snap-1")
	}
}

func TestSnapshotListWithUUIDFilter(t *testing.T) {
	var gotUUID string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUUID = r.URL.Query().Get("uuid")
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(listSnapshotResponse{Count: 0, ListSnapShotResponse: nil})
	}))
	defer srv.Close()

	svc := snapshot.NewService(newTestClient(t, srv))
	svc.List(context.Background(), "", "snap-xyz")

	if gotUUID != "snap-xyz" {
		t.Errorf("uuid query param = %q, want %q", gotUUID, "snap-xyz")
	}
}

func TestSnapshotCreate(t *testing.T) {
	expectedSnap := snapshot.Snapshot{
		UUID:       "snap-new",
		Name:       "my-snap",
		Status:     "BackingUp",
		VolumeUUID: "vol-1",
		ZoneUUID:   "zone-1",
	}

	var gotBody map[string]interface{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "want POST", http.StatusMethodNotAllowed)
			return
		}
		if r.URL.Path != "/restapi/snapshot/createSnapshot" {
			http.NotFound(w, r)
			return
		}
		json.NewDecoder(r.Body).Decode(&gotBody)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(listSnapshotResponse{Count: 1, ListSnapShotResponse: []snapshot.Snapshot{expectedSnap}})
	}))
	defer srv.Close()

	svc := snapshot.NewService(newTestClient(t, srv))
	req := snapshot.CreateRequest{
		Name:       "my-snap",
		VolumeUUID: "vol-1",
		ZoneUUID:   "zone-1",
	}
	snap, err := svc.Create(context.Background(), req)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if snap.UUID != "snap-new" {
		t.Errorf("snap.UUID = %q, want %q", snap.UUID, "snap-new")
	}
	if gotBody["name"] != "my-snap" {
		t.Errorf("body name = %v, want %q", gotBody["name"], "my-snap")
	}
	if gotBody["volumeUuid"] != "vol-1" {
		t.Errorf("body volumeUuid = %v, want %q", gotBody["volumeUuid"], "vol-1")
	}
	if gotBody["zoneUuid"] != "zone-1" {
		t.Errorf("body zoneUuid = %v, want %q", gotBody["zoneUuid"], "zone-1")
	}
}

func TestSnapshotDelete(t *testing.T) {
	var gotPath, gotMethod string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	svc := snapshot.NewService(newTestClient(t, srv))
	err := svc.Delete(context.Background(), "snap-abc")
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if gotMethod != http.MethodDelete {
		t.Errorf("method = %q, want %q", gotMethod, http.MethodDelete)
	}
	if gotPath != "/restapi/snapshot/deleteSnapshot/snap-abc" {
		t.Errorf("path = %q, want %q", gotPath, "/restapi/snapshot/deleteSnapshot/snap-abc")
	}
}
