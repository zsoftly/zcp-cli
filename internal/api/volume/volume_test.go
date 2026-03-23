package volume_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/zsoftly/zcp-cli/internal/api/volume"
	"github.com/zsoftly/zcp-cli/internal/httpclient"
)

type listVolumeResponse struct {
	Count              int             `json:"count"`
	ListVolumeResponse []volume.Volume `json:"listVolumeResponse"`
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

func TestVolumeList(t *testing.T) {
	expected := []volume.Volume{
		{UUID: "vol-1", Name: "disk-a", Status: "Ready", ZoneUUID: "zone-1", VolumeType: "DATADISK"},
		{UUID: "vol-2", Name: "disk-b", Status: "Ready", ZoneUUID: "zone-1", VolumeType: "DATADISK"},
	}

	var gotZone string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/restapi/volume/volumeList" {
			http.NotFound(w, r)
			return
		}
		gotZone = r.URL.Query().Get("zoneUuid")
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(listVolumeResponse{Count: len(expected), ListVolumeResponse: expected})
	}))
	defer srv.Close()

	svc := volume.NewService(newTestClient(t, srv))
	volumes, err := svc.List(context.Background(), "zone-1", "", "")
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(volumes) != 2 {
		t.Fatalf("List() returned %d volumes, want 2", len(volumes))
	}
	if gotZone != "zone-1" {
		t.Errorf("zoneUuid query param = %q, want %q", gotZone, "zone-1")
	}
	if volumes[0].UUID != "vol-1" {
		t.Errorf("volumes[0].UUID = %q, want %q", volumes[0].UUID, "vol-1")
	}
}

func TestVolumeListOptionalFilters(t *testing.T) {
	var gotVM, gotUUID string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotVM = r.URL.Query().Get("vmUuid")
		gotUUID = r.URL.Query().Get("uuid")
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(listVolumeResponse{Count: 0, ListVolumeResponse: nil})
	}))
	defer srv.Close()

	svc := volume.NewService(newTestClient(t, srv))
	svc.List(context.Background(), "zone-1", "vm-abc", "vol-xyz")

	if gotVM != "vm-abc" {
		t.Errorf("vmUuid query param = %q, want %q", gotVM, "vm-abc")
	}
	if gotUUID != "vol-xyz" {
		t.Errorf("uuid query param = %q, want %q", gotUUID, "vol-xyz")
	}
}

func TestVolumeCreate(t *testing.T) {
	expectedVol := volume.Volume{
		UUID:                "vol-new",
		Name:                "my-disk",
		Status:              "Pending",
		ZoneUUID:            "zone-1",
		StorageOfferingUUID: "offer-1",
		JobID:               "job-123",
	}

	var gotBody map[string]interface{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "want POST", http.StatusMethodNotAllowed)
			return
		}
		if r.URL.Path != "/restapi/volume/createVolume" {
			http.NotFound(w, r)
			return
		}
		json.NewDecoder(r.Body).Decode(&gotBody)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(listVolumeResponse{Count: 1, ListVolumeResponse: []volume.Volume{expectedVol}})
	}))
	defer srv.Close()

	svc := volume.NewService(newTestClient(t, srv))
	req := volume.CreateRequest{
		Name:                "my-disk",
		ZoneUUID:            "zone-1",
		StorageOfferingUUID: "offer-1",
		DiskSize:            50,
	}
	vol, err := svc.Create(context.Background(), req)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if vol.UUID != "vol-new" {
		t.Errorf("vol.UUID = %q, want %q", vol.UUID, "vol-new")
	}
	if vol.JobID != "job-123" {
		t.Errorf("vol.JobID = %q, want %q", vol.JobID, "job-123")
	}
	if gotBody["name"] != "my-disk" {
		t.Errorf("body name = %v, want %q", gotBody["name"], "my-disk")
	}
	if gotBody["zoneUuid"] != "zone-1" {
		t.Errorf("body zoneUuid = %v, want %q", gotBody["zoneUuid"], "zone-1")
	}
	if gotBody["storageOfferingUuid"] != "offer-1" {
		t.Errorf("body storageOfferingUuid = %v, want %q", gotBody["storageOfferingUuid"], "offer-1")
	}
}

func TestVolumeAttach(t *testing.T) {
	var gotVolumeUUID, gotInstanceUUID string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/restapi/volume/attachVolume" {
			http.NotFound(w, r)
			return
		}
		gotVolumeUUID = r.URL.Query().Get("uuid")
		gotInstanceUUID = r.URL.Query().Get("instanceUuid")
		result := volume.Volume{UUID: "vol-1", Status: "Attached", VMInstanceUUID: "vm-1"}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(listVolumeResponse{Count: 1, ListVolumeResponse: []volume.Volume{result}})
	}))
	defer srv.Close()

	svc := volume.NewService(newTestClient(t, srv))
	vol, err := svc.Attach(context.Background(), "vol-1", "vm-1")
	if err != nil {
		t.Fatalf("Attach() error = %v", err)
	}
	if gotVolumeUUID != "vol-1" {
		t.Errorf("uuid query param = %q, want %q", gotVolumeUUID, "vol-1")
	}
	if gotInstanceUUID != "vm-1" {
		t.Errorf("instanceUuid query param = %q, want %q", gotInstanceUUID, "vm-1")
	}
	if vol.VMInstanceUUID != "vm-1" {
		t.Errorf("vol.VMInstanceUUID = %q, want %q", vol.VMInstanceUUID, "vm-1")
	}
}

func TestVolumeDetach(t *testing.T) {
	var gotUUID string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/restapi/volume/detachVolume" {
			http.NotFound(w, r)
			return
		}
		gotUUID = r.URL.Query().Get("uuid")
		result := volume.Volume{UUID: "vol-1", Status: "Detached"}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(listVolumeResponse{Count: 1, ListVolumeResponse: []volume.Volume{result}})
	}))
	defer srv.Close()

	svc := volume.NewService(newTestClient(t, srv))
	vol, err := svc.Detach(context.Background(), "vol-1")
	if err != nil {
		t.Fatalf("Detach() error = %v", err)
	}
	if gotUUID != "vol-1" {
		t.Errorf("uuid query param = %q, want %q", gotUUID, "vol-1")
	}
	if vol.Status != "Detached" {
		t.Errorf("vol.Status = %q, want %q", vol.Status, "Detached")
	}
}

func TestVolumeDelete(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			http.Error(w, "want DELETE", http.StatusMethodNotAllowed)
			return
		}
		gotPath = r.URL.Path
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	svc := volume.NewService(newTestClient(t, srv))
	resp, err := svc.Delete(context.Background(), "vol-abc")
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if gotPath != "/restapi/volume/deleteVolume/vol-abc" {
		t.Errorf("DELETE path = %q, want %q", gotPath, "/restapi/volume/deleteVolume/vol-abc")
	}
	if resp == nil {
		t.Fatal("Delete() returned nil response")
	}
}

func TestVolumeResize(t *testing.T) {
	var gotUUID, gotOffering, gotDiskSize, gotShrink string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/restapi/volume/resizeVolume" {
			http.NotFound(w, r)
			return
		}
		gotUUID = r.URL.Query().Get("uuid")
		gotOffering = r.URL.Query().Get("storageOfferingUuid")
		gotDiskSize = r.URL.Query().Get("diskSize")
		gotShrink = r.URL.Query().Get("isShrink")
		result := volume.Volume{UUID: "vol-1", StorageDiskSize: "100"}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(listVolumeResponse{Count: 1, ListVolumeResponse: []volume.Volume{result}})
	}))
	defer srv.Close()

	svc := volume.NewService(newTestClient(t, srv))
	vol, err := svc.Resize(context.Background(), "vol-1", "offer-2", 100, true)
	if err != nil {
		t.Fatalf("Resize() error = %v", err)
	}
	if gotUUID != "vol-1" {
		t.Errorf("uuid param = %q, want %q", gotUUID, "vol-1")
	}
	if gotOffering != "offer-2" {
		t.Errorf("storageOfferingUuid param = %q, want %q", gotOffering, "offer-2")
	}
	if gotDiskSize != "100" {
		t.Errorf("diskSize param = %q, want %q", gotDiskSize, "100")
	}
	if gotShrink != "true" {
		t.Errorf("isShrink param = %q, want %q", gotShrink, "true")
	}
	if vol.StorageDiskSize != "100" {
		t.Errorf("vol.StorageDiskSize = %q, want %q", vol.StorageDiskSize, "100")
	}
}
