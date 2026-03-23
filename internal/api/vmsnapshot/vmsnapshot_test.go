package vmsnapshot_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/zsoftly/zcp-cli/internal/api/vmsnapshot"
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

type listVMSnapshotResponse struct {
	Count                  int                     `json:"count"`
	ListVmSnapshotResponse []vmsnapshot.VMSnapshot `json:"listVmSnapshotResponse"`
}

func TestVMSnapshotList(t *testing.T) {
	expected := []vmsnapshot.VMSnapshot{
		{UUID: "vmsnap-1", Name: "snap-a", ZoneUUID: "zone-1", Status: "Ready"},
		{UUID: "vmsnap-2", Name: "snap-b", ZoneUUID: "zone-1", Status: "Ready"},
	}

	var gotZone string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/restapi/vmsnapshot/vmsnapshotList" {
			http.NotFound(w, r)
			return
		}
		gotZone = r.URL.Query().Get("zoneUuid")
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(listVMSnapshotResponse{Count: len(expected), ListVmSnapshotResponse: expected})
	}))
	defer srv.Close()

	svc := vmsnapshot.NewService(newClient(srv.URL))
	snaps, err := svc.List(context.Background(), "zone-1", "")
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(snaps) != 2 {
		t.Fatalf("List() returned %d snapshots, want 2", len(snaps))
	}
	if gotZone != "zone-1" {
		t.Errorf("zoneUuid query param = %q, want %q", gotZone, "zone-1")
	}
	if snaps[0].UUID != "vmsnap-1" {
		t.Errorf("snaps[0].UUID = %q, want %q", snaps[0].UUID, "vmsnap-1")
	}
}

func TestVMSnapshotCreate(t *testing.T) {
	created := vmsnapshot.VMSnapshot{
		UUID:     "vmsnap-new",
		Name:     "my-vmsnap",
		ZoneUUID: "zone-1",
		JobID:    "job-abc",
		Status:   "Creating",
	}

	var gotBody map[string]interface{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "expected POST", http.StatusMethodNotAllowed)
			return
		}
		if r.URL.Path != "/restapi/vmsnapshot/createVmSnapshot" {
			http.NotFound(w, r)
			return
		}
		json.NewDecoder(r.Body).Decode(&gotBody)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(listVMSnapshotResponse{Count: 1, ListVmSnapshotResponse: []vmsnapshot.VMSnapshot{created}})
	}))
	defer srv.Close()

	svc := vmsnapshot.NewService(newClient(srv.URL))
	req := vmsnapshot.CreateRequest{
		Name:               "my-vmsnap",
		ZoneUUID:           "zone-1",
		VirtualMachineUUID: "vm-1",
		Description:        "test snapshot",
		SnapshotMemory:     false,
	}
	snap, err := svc.Create(context.Background(), req)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if snap.UUID != "vmsnap-new" {
		t.Errorf("snap.UUID = %q, want %q", snap.UUID, "vmsnap-new")
	}
	if gotBody["name"] != "my-vmsnap" {
		t.Errorf("body name = %v, want %q", gotBody["name"], "my-vmsnap")
	}
	if gotBody["zoneUuid"] != "zone-1" {
		t.Errorf("body zoneUuid = %v, want %q", gotBody["zoneUuid"], "zone-1")
	}
	if gotBody["virtualmachineUuid"] != "vm-1" {
		t.Errorf("body virtualmachineUuid = %v, want %q", gotBody["virtualmachineUuid"], "vm-1")
	}
}

func TestVMSnapshotDelete(t *testing.T) {
	var gotPath, gotMethod string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	svc := vmsnapshot.NewService(newClient(srv.URL))
	resp, err := svc.Delete(context.Background(), "vmsnap-del-1")
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if resp == nil {
		t.Fatal("Delete() returned nil response")
	}
	if gotMethod != http.MethodDelete {
		t.Errorf("method = %q, want %q", gotMethod, http.MethodDelete)
	}
	if gotPath != "/restapi/vmsnapshot/deleteVmSnapshot/vmsnap-del-1" {
		t.Errorf("path = %q, want %q", gotPath, "/restapi/vmsnapshot/deleteVmSnapshot/vmsnap-del-1")
	}
}

func TestVMSnapshotRevert(t *testing.T) {
	reverted := vmsnapshot.VMSnapshot{
		UUID:     "vmsnap-1",
		Name:     "snap-a",
		ZoneUUID: "zone-1",
		JobID:    "job-revert",
		Status:   "Reverting",
	}

	var gotUUID string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/restapi/vmsnapshot/revertToVmSnapshot" {
			http.NotFound(w, r)
			return
		}
		gotUUID = r.URL.Query().Get("uuid")
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(listVMSnapshotResponse{Count: 1, ListVmSnapshotResponse: []vmsnapshot.VMSnapshot{reverted}})
	}))
	defer srv.Close()

	svc := vmsnapshot.NewService(newClient(srv.URL))
	snap, err := svc.Revert(context.Background(), "vmsnap-1")
	if err != nil {
		t.Fatalf("Revert() error = %v", err)
	}
	if snap.UUID != "vmsnap-1" {
		t.Errorf("snap.UUID = %q, want %q", snap.UUID, "vmsnap-1")
	}
	if gotUUID != "vmsnap-1" {
		t.Errorf("uuid query param = %q, want %q", gotUUID, "vmsnap-1")
	}
}
