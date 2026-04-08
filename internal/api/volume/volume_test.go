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

type listResponse struct {
	Status      string          `json:"status"`
	Message     string          `json:"message"`
	CurrentPage int             `json:"current_page"`
	Data        []volume.Volume `json:"data"`
	Total       int             `json:"total"`
}

type singleResponse struct {
	Status  string        `json:"status"`
	Message string        `json:"message"`
	Data    volume.Volume `json:"data"`
}

func newTestClient(t *testing.T, srv *httptest.Server) *httpclient.Client {
	t.Helper()
	return httpclient.New(httpclient.Options{
		BaseURL:     srv.URL,
		BearerToken: "test-token",
		Timeout:     5 * time.Second,
	})
}

func TestVolumeList(t *testing.T) {
	expected := []volume.Volume{
		{ID: "vol-1", Name: "ROOT-4153", Slug: "root-4153", Size: "50", VolumeType: "ROOT"},
		{ID: "vol-2", Name: "data-disk", Slug: "data-disk", Size: "100", VolumeType: "DATA"},
	}

	var gotInclude string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/blockstorages" {
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

	svc := volume.NewService(newTestClient(t, srv))
	volumes, err := svc.List(context.Background())
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(volumes) != 2 {
		t.Fatalf("List() returned %d volumes, want 2", len(volumes))
	}
	if gotInclude == "" {
		t.Error("include query param was empty, expected relations")
	}
	if volumes[0].ID != "vol-1" {
		t.Errorf("volumes[0].ID = %q, want %q", volumes[0].ID, "vol-1")
	}
	if volumes[0].VolumeType != "ROOT" {
		t.Errorf("volumes[0].VolumeType = %q, want %q", volumes[0].VolumeType, "ROOT")
	}
}

func TestVolumeCreate(t *testing.T) {
	expectedVol := volume.Volume{
		ID:   "vol-new",
		Name: "my-volume",
		Slug: "my-volume",
		Size: "50",
	}

	var gotBody map[string]interface{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "want POST", http.StatusMethodNotAllowed)
			return
		}
		if r.URL.Path != "/blockstorages" {
			http.NotFound(w, r)
			return
		}
		json.NewDecoder(r.Body).Decode(&gotBody)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(singleResponse{
			Status:  "Success",
			Message: "Ok",
			Data:    expectedVol,
		})
	}))
	defer srv.Close()

	svc := volume.NewService(newTestClient(t, srv))
	req := volume.CreateRequest{
		Name:            "my-volume",
		Project:         "default-73",
		CloudProvider:   "nimbo",
		Region:          "noida",
		BillingCycle:    "hourly",
		StorageCategory: "nvme",
		Plan:            "50-gb-2",
	}
	vol, err := svc.Create(context.Background(), req)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if vol.ID != "vol-new" {
		t.Errorf("vol.ID = %q, want %q", vol.ID, "vol-new")
	}
	if gotBody["name"] != "my-volume" {
		t.Errorf("body name = %v, want %q", gotBody["name"], "my-volume")
	}
	if gotBody["cloud_provider"] != "nimbo" {
		t.Errorf("body cloud_provider = %v, want %q", gotBody["cloud_provider"], "nimbo")
	}
}

func TestVolumeAttach(t *testing.T) {
	var gotPath string
	var gotBody map[string]interface{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "want POST", http.StatusMethodNotAllowed)
			return
		}
		gotPath = r.URL.Path
		json.NewDecoder(r.Body).Decode(&gotBody)
		result := volume.Volume{ID: "vol-1", Slug: "root-4153", VirtualMachineID: "vm-1"}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(singleResponse{Status: "Success", Message: "Ok", Data: result})
	}))
	defer srv.Close()

	svc := volume.NewService(newTestClient(t, srv))
	vol, err := svc.Attach(context.Background(), "root-4153", "test-vm-1")
	if err != nil {
		t.Fatalf("Attach() error = %v", err)
	}
	if gotPath != "/blockstorages/root-4153/attach" {
		t.Errorf("path = %q, want %q", gotPath, "/blockstorages/root-4153/attach")
	}
	if gotBody["virtual_machine"] != "test-vm-1" {
		t.Errorf("body virtual_machine = %v, want %q", gotBody["virtual_machine"], "test-vm-1")
	}
	if vol.VirtualMachineID != "vm-1" {
		t.Errorf("vol.VirtualMachineID = %q, want %q", vol.VirtualMachineID, "vm-1")
	}
}

func TestVolumeDetach(t *testing.T) {
	var gotPath, gotMethod string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		result := volume.Volume{ID: "vol-1", Slug: "root-4153"}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(singleResponse{Status: "Success", Message: "Ok", Data: result})
	}))
	defer srv.Close()

	svc := volume.NewService(newTestClient(t, srv))
	vol, err := svc.Detach(context.Background(), "root-4153")
	if err != nil {
		t.Fatalf("Detach() error = %v", err)
	}
	if gotMethod != http.MethodPost {
		t.Errorf("method = %q, want %q", gotMethod, http.MethodPost)
	}
	if gotPath != "/blockstorages/root-4153/detach" {
		t.Errorf("path = %q, want %q", gotPath, "/blockstorages/root-4153/detach")
	}
	if vol.Slug != "root-4153" {
		t.Errorf("vol.Slug = %q, want %q", vol.Slug, "root-4153")
	}
}
