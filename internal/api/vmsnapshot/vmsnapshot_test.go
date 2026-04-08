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
		BaseURL:     baseURL,
		BearerToken: "test-token",
		Timeout:     5 * time.Second,
	})
}

func TestVMSnapshotList(t *testing.T) {
	expected := []vmsnapshot.VMSnapshot{
		{ID: "id-1", Name: "snap-a", Slug: "snap-a", State: "Ready", RegionID: "rgn-1", VirtualMachineID: "vm-1"},
		{ID: "id-2", Name: "snap-b", Slug: "snap-b", State: "Ready", RegionID: "rgn-1", VirtualMachineID: "vm-2"},
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/virtual-machines/snapshots" {
			http.NotFound(w, r)
			return
		}
		data, _ := json.Marshal(expected)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(vmsnapshot.Envelope{
			Status:  "Success",
			Message: "ok",
			Data:    data,
			Total:   len(expected),
		})
	}))
	defer srv.Close()

	svc := vmsnapshot.NewService(newClient(srv.URL))
	snaps, err := svc.List(context.Background())
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(snaps) != 2 {
		t.Fatalf("List() returned %d snapshots, want 2", len(snaps))
	}
	if snaps[0].Slug != "snap-a" {
		t.Errorf("snaps[0].Slug = %q, want %q", snaps[0].Slug, "snap-a")
	}
}

func TestVMSnapshotCreate(t *testing.T) {
	var gotPath, gotMethod string
	var gotBody map[string]interface{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		json.NewDecoder(r.Body).Decode(&gotBody)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(vmsnapshot.ActionResponse{
			Status:  "Success",
			Message: "Snapshot creation initiated",
		})
	}))
	defer srv.Close()

	svc := vmsnapshot.NewService(newClient(srv.URL))
	req := vmsnapshot.CreateRequest{
		Name:         "my-vmsnap",
		BillingCycle: "monthly",
		Plan:         "basic",
		IsMemory:     false,
		IsVMSnapshot: true,
		Project:      "proj-1",
	}
	resp, err := svc.Create(context.Background(), "my-vm", req)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if resp.Status != "Success" {
		t.Errorf("resp.Status = %q, want %q", resp.Status, "Success")
	}
	if gotMethod != http.MethodPost {
		t.Errorf("method = %q, want %q", gotMethod, http.MethodPost)
	}
	if gotPath != "/virtual-machines/my-vm/snapshots" {
		t.Errorf("path = %q, want %q", gotPath, "/virtual-machines/my-vm/snapshots")
	}
	if gotBody["name"] != "my-vmsnap" {
		t.Errorf("body name = %v, want %q", gotBody["name"], "my-vmsnap")
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
	err := svc.Delete(context.Background(), "snap-del-1")
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if gotMethod != http.MethodDelete {
		t.Errorf("method = %q, want %q", gotMethod, http.MethodDelete)
	}
	if gotPath != "/virtual-machines/snapshots/snap-del-1" {
		t.Errorf("path = %q, want %q", gotPath, "/virtual-machines/snapshots/snap-del-1")
	}
}

func TestVMSnapshotRevert(t *testing.T) {
	var gotPath, gotMethod string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(vmsnapshot.ActionResponse{
			Status:  "Success",
			Message: "Revert initiated",
		})
	}))
	defer srv.Close()

	svc := vmsnapshot.NewService(newClient(srv.URL))
	resp, err := svc.Revert(context.Background(), "snap-1")
	if err != nil {
		t.Fatalf("Revert() error = %v", err)
	}
	if resp.Status != "Success" {
		t.Errorf("resp.Status = %q, want %q", resp.Status, "Success")
	}
	if gotMethod != http.MethodPost {
		t.Errorf("method = %q, want %q", gotMethod, http.MethodPost)
	}
	if gotPath != "/virtual-machines/snapshots/snap-1/revert" {
		t.Errorf("path = %q, want %q", gotPath, "/virtual-machines/snapshots/snap-1/revert")
	}
}
