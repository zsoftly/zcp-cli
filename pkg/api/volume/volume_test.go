package volume_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/zsoftly/zcp-cli/pkg/api/volume"
	"github.com/zsoftly/zcp-cli/pkg/httpclient"
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
	volumes, err := svc.List(context.Background(), "", "")
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

func TestVolumeListPagination(t *testing.T) {
	pages := map[string][]volume.Volume{
		"1": {{ID: "vol-1", Slug: "vol-1"}},
		"2": {{ID: "vol-2", Slug: "vol-2"}},
	}
	var requestedPages []string
	var requestedPagesMu sync.Mutex
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		page := r.URL.Query().Get("page")
		if page == "" {
			page = "1"
		}
		requestedPagesMu.Lock()
		requestedPages = append(requestedPages, page)
		requestedPagesMu.Unlock()
		if got := r.URL.Query().Get("filter[region]"); got != "ca-central" {
			t.Errorf("region filter = %q, want ca-central", got)
		}
		if got := r.URL.Query().Get("filter[project]"); got != "project-a" {
			t.Errorf("project filter = %q, want project-a", got)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(listResponse{
			Status:      "Success",
			CurrentPage: map[string]int{"1": 1, "2": 2}[page],
			Data:        pages[page],
			Total:       2,
		})
	}))
	defer srv.Close()

	volumes, err := volume.NewService(newTestClient(t, srv)).List(context.Background(), "ca-central", "project-a")
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if got, want := len(volumes), 2; got != want {
		t.Fatalf("List() returned %d volumes, want %d", got, want)
	}
	if got, want := volumes[1].Slug, "vol-2"; got != want {
		t.Errorf("volumes[1].Slug = %q, want %q", got, want)
	}
	requestedPagesMu.Lock()
	got := append([]string(nil), requestedPages...)
	requestedPagesMu.Unlock()
	if want := []string{"1", "2"}; len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Errorf("requested pages = %v, want %v", got, want)
	}
}

func TestVolumeListPaginationRejectsIgnoredPage(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(listResponse{
			Status:      "Success",
			CurrentPage: 1,
			Data:        []volume.Volume{{ID: "vol-1", Slug: "vol-1"}},
			Total:       2,
		})
	}))
	defer srv.Close()

	_, err := volume.NewService(newTestClient(t, srv)).List(context.Background(), "", "")
	if err == nil {
		t.Fatal("List() error = nil, want an error when the API ignores page 2")
	}
}

func TestVolumeListPaginationRejectsMissingCurrentPage(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		page := r.URL.Query().Get("page")
		w.Header().Set("Content-Type", "application/json")
		if page == "" {
			json.NewEncoder(w).Encode(listResponse{
				Status:      "Success",
				CurrentPage: 1,
				Data:        []volume.Volume{{ID: "vol-1", Slug: "vol-1"}},
				Total:       2,
			})
			return
		}
		json.NewEncoder(w).Encode(listResponse{
			Status: "Success",
			Data:   []volume.Volume{{ID: "vol-1", Slug: "vol-1"}},
			Total:  2,
		})
	}))
	defer srv.Close()

	_, err := volume.NewService(newTestClient(t, srv)).List(context.Background(), "", "")
	if err == nil {
		t.Fatal("List() error = nil, want an error when the API omits current_page for page 2")
	}
}

func TestVolumeListPaginationRejectsEmptyPageBeforeTotal(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		page := r.URL.Query().Get("page")
		if page == "" {
			page = "1"
		}
		w.Header().Set("Content-Type", "application/json")
		if page == "1" {
			json.NewEncoder(w).Encode(listResponse{
				Status:      "Success",
				CurrentPage: 1,
				Data:        []volume.Volume{{ID: "vol-1", Slug: "vol-1"}},
				Total:       2,
			})
			return
		}
		json.NewEncoder(w).Encode(listResponse{
			Status:      "Success",
			CurrentPage: 2,
			Data:        []volume.Volume{},
			Total:       2,
		})
	}))
	defer srv.Close()

	_, err := volume.NewService(newTestClient(t, srv)).List(context.Background(), "", "")
	if err == nil {
		t.Fatal("List() error = nil, want an error for an empty page before the reported total")
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
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(volume.ActionResponse{Status: "Success", Message: "Attaching block storage."})
	}))
	defer srv.Close()

	svc := volume.NewService(newTestClient(t, srv))
	resp, err := svc.Attach(context.Background(), "root-4153", "test-vm-1")
	if err != nil {
		t.Fatalf("Attach() error = %v", err)
	}
	if gotPath != "/blockstorages/root-4153/attach" {
		t.Errorf("path = %q, want %q", gotPath, "/blockstorages/root-4153/attach")
	}
	if gotBody["virtual_machine"] != "test-vm-1" {
		t.Errorf("body virtual_machine = %v, want %q", gotBody["virtual_machine"], "test-vm-1")
	}
	if resp.Message != "Attaching block storage." {
		t.Errorf("resp.Message = %q, want %q", resp.Message, "Attaching block storage.")
	}
	if resp.Status != "Success" {
		t.Errorf("resp.Status = %q, want %q", resp.Status, "Success")
	}
}

func TestVolumeDetach(t *testing.T) {
	var gotPath, gotMethod string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(volume.ActionResponse{Status: "Success", Message: "Detaching block storage."})
	}))
	defer srv.Close()

	svc := volume.NewService(newTestClient(t, srv))
	resp, err := svc.Detach(context.Background(), "root-4153")
	if err != nil {
		t.Fatalf("Detach() error = %v", err)
	}
	if gotMethod != http.MethodPost {
		t.Errorf("method = %q, want %q", gotMethod, http.MethodPost)
	}
	if gotPath != "/blockstorages/root-4153/detach" {
		t.Errorf("path = %q, want %q", gotPath, "/blockstorages/root-4153/detach")
	}
	if resp.Status != "Success" {
		t.Errorf("resp.Status = %q, want %q", resp.Status, "Success")
	}
	if resp.Message != "Detaching block storage." {
		t.Errorf("resp.Message = %q, want %q", resp.Message, "Detaching block storage.")
	}
}

func TestVolumeDelete(t *testing.T) {
	var gotPath, gotMethod string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	svc := volume.NewService(newTestClient(t, srv))
	err := svc.Delete(context.Background(), "bs-001001-0042")
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if gotMethod != http.MethodDelete {
		t.Errorf("method = %q, want %q", gotMethod, http.MethodDelete)
	}
	if gotPath != "/blockstorages/bs-001001-0042" {
		t.Errorf("path = %q, want %q", gotPath, "/blockstorages/bs-001001-0042")
	}
}

func TestVolumeDelete_Error(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnprocessableEntity)
	}))
	defer srv.Close()

	svc := volume.NewService(newTestClient(t, srv))
	err := svc.Delete(context.Background(), "bs-attached")
	if err == nil {
		t.Fatal("Delete() expected error on non-2xx, got nil")
	}
}
