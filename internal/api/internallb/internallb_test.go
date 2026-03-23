package internallb_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/zsoftly/zcp-cli/internal/api/internallb"
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

type listInternalLbResponse struct {
	Count                  int                     `json:"count"`
	ListInternalLbResponse []internallb.InternalLB `json:"listInternalLbResponse"`
}

func TestInternalLBList(t *testing.T) {
	expected := []internallb.InternalLB{
		{UUID: "ilb-1", Name: "lb-one", NetworkUUID: "net-1", ZoneUUID: "zone-1"},
		{UUID: "ilb-2", Name: "lb-two", NetworkUUID: "net-1", ZoneUUID: "zone-1"},
	}

	var gotZone string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/restapi/internallb/internalLbList" {
			http.NotFound(w, r)
			return
		}
		gotZone = r.URL.Query().Get("zoneUuid")
		if gotZone == "" {
			http.Error(w, "zoneUuid required", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(listInternalLbResponse{Count: len(expected), ListInternalLbResponse: expected})
	}))
	defer srv.Close()

	svc := internallb.NewService(newClient(srv.URL))
	lbs, err := svc.List(context.Background(), "zone-1", "", "")
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(lbs) != 2 {
		t.Fatalf("List() returned %d LBs, want 2", len(lbs))
	}
	if gotZone != "zone-1" {
		t.Errorf("zoneUuid query param = %q, want %q", gotZone, "zone-1")
	}
	if lbs[0].UUID != "ilb-1" {
		t.Errorf("lbs[0].UUID = %q, want %q", lbs[0].UUID, "ilb-1")
	}
}

func TestInternalLBListWithFilters(t *testing.T) {
	var gotUUID, gotNetworkUUID string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUUID = r.URL.Query().Get("uuid")
		gotNetworkUUID = r.URL.Query().Get("networkUuid")
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(listInternalLbResponse{Count: 0, ListInternalLbResponse: nil})
	}))
	defer srv.Close()

	svc := internallb.NewService(newClient(srv.URL))
	_, err := svc.List(context.Background(), "zone-1", "ilb-1", "net-1")
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if gotUUID != "ilb-1" {
		t.Errorf("uuid query param = %q, want %q", gotUUID, "ilb-1")
	}
	if gotNetworkUUID != "net-1" {
		t.Errorf("networkUuid query param = %q, want %q", gotNetworkUUID, "net-1")
	}
}

func TestInternalLBCreate(t *testing.T) {
	created := internallb.InternalLB{
		UUID:         "ilb-new",
		Name:         "new-lb",
		NetworkUUID:  "net-1",
		SourcePort:   "80",
		InstancePort: "8080",
		Algorithm:    "roundrobin",
	}

	var gotBody map[string]interface{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "expected POST", http.StatusMethodNotAllowed)
			return
		}
		if r.URL.Path != "/restapi/internallb/createInternalLb" {
			http.NotFound(w, r)
			return
		}
		json.NewDecoder(r.Body).Decode(&gotBody)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(listInternalLbResponse{Count: 1, ListInternalLbResponse: []internallb.InternalLB{created}})
	}))
	defer srv.Close()

	svc := internallb.NewService(newClient(srv.URL))
	req := internallb.CreateRequest{
		Name:         "new-lb",
		NetworkUUID:  "net-1",
		SourcePort:   "80",
		InstancePort: "8080",
		Algorithm:    "roundrobin",
	}
	result, err := svc.Create(context.Background(), req)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if result.UUID != "ilb-new" {
		t.Errorf("result.UUID = %q, want %q", result.UUID, "ilb-new")
	}
	if gotBody["name"] != "new-lb" {
		t.Errorf("body name = %v, want %q", gotBody["name"], "new-lb")
	}
	if gotBody["networkUuid"] != "net-1" {
		t.Errorf("body networkUuid = %v, want %q", gotBody["networkUuid"], "net-1")
	}
	if gotBody["algorithm"] != "roundrobin" {
		t.Errorf("body algorithm = %v, want %q", gotBody["algorithm"], "roundrobin")
	}
}

func TestInternalLBCreateEmptyResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(listInternalLbResponse{Count: 0, ListInternalLbResponse: nil})
	}))
	defer srv.Close()

	svc := internallb.NewService(newClient(srv.URL))
	_, err := svc.Create(context.Background(), internallb.CreateRequest{Name: "x", NetworkUUID: "net-1"})
	if err == nil {
		t.Fatal("Create() expected error on empty response, got nil")
	}
}

func TestInternalLBDelete(t *testing.T) {
	var gotPath, gotMethod string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	svc := internallb.NewService(newClient(srv.URL))
	err := svc.Delete(context.Background(), "ilb-del-1")
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if gotMethod != http.MethodDelete {
		t.Errorf("method = %q, want %q", gotMethod, http.MethodDelete)
	}
	if gotPath != "/restapi/internallb/deleteInternalLb/ilb-del-1" {
		t.Errorf("path = %q, want %q", gotPath, "/restapi/internallb/deleteInternalLb/ilb-del-1")
	}
}

func TestInternalLBDeleteError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "not found", http.StatusNotFound)
	}))
	defer srv.Close()

	svc := internallb.NewService(newClient(srv.URL))
	err := svc.Delete(context.Background(), "missing")
	if err == nil {
		t.Fatal("Delete() expected error on 404, got nil")
	}
}

func TestInternalLBListError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "server error", http.StatusInternalServerError)
	}))
	defer srv.Close()

	svc := internallb.NewService(newClient(srv.URL))
	_, err := svc.List(context.Background(), "zone-1", "", "")
	if err == nil {
		t.Fatal("List() expected error on 500, got nil")
	}
}
