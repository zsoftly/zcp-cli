package instance_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/zsoftly/zcp-cli/internal/api/instance"
	"github.com/zsoftly/zcp-cli/internal/httpclient"
)

// helpers

func newClient(baseURL string) *httpclient.Client {
	return httpclient.New(httpclient.Options{
		BaseURL:   baseURL,
		APIKey:    "testkey",
		SecretKey: "testsecret",
		Timeout:   5 * time.Second,
	})
}

type listInstanceResponse struct {
	Count                int                 `json:"count"`
	ListInstanceResponse []instance.Instance `json:"listInstanceResponse"`
}

type listInstanceNetworkResponse struct {
	Count                        int                `json:"count"`
	KongInstanceNetworkResponses []instance.Network `json:"kongInstanceNetworkResponses"`
}

func makeInstance(uuid, name, state string) instance.Instance {
	return instance.Instance{
		UUID:      uuid,
		Name:      name,
		State:     state,
		ZoneUUID:  "zone-uuid-1",
		PrivateIP: "10.0.0.1",
		Memory:    "2048",
	}
}

// TestInstanceList verifies the URL path, required zoneUuid param, and response parsing.
func TestInstanceList(t *testing.T) {
	instances := []instance.Instance{
		makeInstance("vm-1", "web-01", "Running"),
		makeInstance("vm-2", "db-01", "Stopped"),
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/restapi/instance/instanceList" {
			http.NotFound(w, r)
			return
		}
		zoneUUID := r.URL.Query().Get("zoneUuid")
		if zoneUUID == "" {
			http.Error(w, "zoneUuid required", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(listInstanceResponse{
			Count:                len(instances),
			ListInstanceResponse: instances,
		})
	}))
	defer srv.Close()

	svc := instance.NewService(newClient(srv.URL))

	result, err := svc.List(context.Background(), "zone-uuid-1", "")
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(result) != 2 {
		t.Fatalf("List() returned %d instances, want 2", len(result))
	}
	if result[0].UUID != "vm-1" {
		t.Errorf("result[0].UUID = %q, want %q", result[0].UUID, "vm-1")
	}
	if result[1].Name != "db-01" {
		t.Errorf("result[1].Name = %q, want %q", result[1].Name, "db-01")
	}
}

// TestInstanceListZoneUUIDSent verifies zoneUuid is sent as a query param.
func TestInstanceListZoneUUIDSent(t *testing.T) {
	var gotZone string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotZone = r.URL.Query().Get("zoneUuid")
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(listInstanceResponse{Count: 0})
	}))
	defer srv.Close()

	svc := instance.NewService(newClient(srv.URL))
	svc.List(context.Background(), "my-zone-123", "")

	if gotZone != "my-zone-123" {
		t.Errorf("zoneUuid query param = %q, want %q", gotZone, "my-zone-123")
	}
}

// TestInstanceGet verifies vmUuid filter is sent and single result is returned.
func TestInstanceGet(t *testing.T) {
	expected := makeInstance("vm-99", "target-vm", "Running")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		vmUUID := r.URL.Query().Get("vmUuid")
		if vmUUID != "vm-99" {
			http.Error(w, "unexpected vmUuid", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(listInstanceResponse{
			Count:                1,
			ListInstanceResponse: []instance.Instance{expected},
		})
	}))
	defer srv.Close()

	svc := instance.NewService(newClient(srv.URL))

	inst, err := svc.Get(context.Background(), "zone-1", "vm-99")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if inst.UUID != "vm-99" {
		t.Errorf("inst.UUID = %q, want %q", inst.UUID, "vm-99")
	}
	if inst.Name != "target-vm" {
		t.Errorf("inst.Name = %q, want %q", inst.Name, "target-vm")
	}
}

// TestInstanceGetNotFound verifies that an empty list returns an error.
func TestInstanceGetNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(listInstanceResponse{Count: 0, ListInstanceResponse: nil})
	}))
	defer srv.Close()

	svc := instance.NewService(newClient(srv.URL))

	_, err := svc.Get(context.Background(), "zone-1", "nonexistent-uuid")
	if err == nil {
		t.Fatal("Get() expected error for not found, got nil")
	}
}

// TestInstanceCreate verifies POST body and response parsing.
func TestInstanceCreate(t *testing.T) {
	created := makeInstance("new-vm-1", "my-vm", "Running")

	var gotBody map[string]interface{}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "expected POST", http.StatusMethodNotAllowed)
			return
		}
		if r.URL.Path != "/restapi/instance/createInstance" {
			http.NotFound(w, r)
			return
		}
		json.NewDecoder(r.Body).Decode(&gotBody)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(listInstanceResponse{
			Count:                1,
			ListInstanceResponse: []instance.Instance{created},
		})
	}))
	defer srv.Close()

	svc := instance.NewService(newClient(srv.URL))

	req := instance.CreateRequest{
		Name:                "my-vm",
		ZoneUUID:            "zone-1",
		TemplateUUID:        "tmpl-1",
		ComputeOfferingUUID: "co-1",
		NetworkUUID:         "net-1",
	}

	inst, err := svc.Create(context.Background(), req)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if inst.UUID != "new-vm-1" {
		t.Errorf("inst.UUID = %q, want %q", inst.UUID, "new-vm-1")
	}

	if gotBody["name"] != "my-vm" {
		t.Errorf("body[name] = %v, want %q", gotBody["name"], "my-vm")
	}
	if gotBody["zoneUuid"] != "zone-1" {
		t.Errorf("body[zoneUuid] = %v, want %q", gotBody["zoneUuid"], "zone-1")
	}
}

// TestInstanceStart verifies GET with uuid and response parsing.
func TestInstanceStart(t *testing.T) {
	started := makeInstance("vm-start-1", "web-01", "Running")

	var gotUUID string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/restapi/instance/startInstance" {
			http.NotFound(w, r)
			return
		}
		gotUUID = r.URL.Query().Get("uuid")
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(listInstanceResponse{
			Count:                1,
			ListInstanceResponse: []instance.Instance{started},
		})
	}))
	defer srv.Close()

	svc := instance.NewService(newClient(srv.URL))

	inst, err := svc.Start(context.Background(), "vm-start-1")
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	if gotUUID != "vm-start-1" {
		t.Errorf("uuid query param = %q, want %q", gotUUID, "vm-start-1")
	}
	if inst.State != "Running" {
		t.Errorf("inst.State = %q, want %q", inst.State, "Running")
	}
}

// TestInstanceStop verifies forceStop param values ("true"/"false").
func TestInstanceStop(t *testing.T) {
	stopped := makeInstance("vm-stop-1", "web-01", "Stopped")

	tests := []struct {
		name      string
		force     bool
		wantForce string
	}{
		{"graceful stop", false, "false"},
		{"force stop", true, "true"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotForce string

			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/restapi/instance/stopInstance" {
					http.NotFound(w, r)
					return
				}
				gotForce = r.URL.Query().Get("forceStop")
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(listInstanceResponse{
					Count:                1,
					ListInstanceResponse: []instance.Instance{stopped},
				})
			}))
			defer srv.Close()

			svc := instance.NewService(newClient(srv.URL))

			_, err := svc.Stop(context.Background(), "vm-stop-1", tt.force)
			if err != nil {
				t.Fatalf("Stop() error = %v", err)
			}
			if gotForce != tt.wantForce {
				t.Errorf("forceStop = %q, want %q", gotForce, tt.wantForce)
			}
		})
	}
}

// TestInstanceDestroy verifies expunge param and no error on success.
func TestInstanceDestroy(t *testing.T) {
	tests := []struct {
		name        string
		expunge     bool
		wantExpunge string
	}{
		{"soft delete", false, "false"},
		{"hard expunge", true, "true"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotExpunge, gotUUID string

			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/restapi/instance/destroyInstance" {
					http.NotFound(w, r)
					return
				}
				gotUUID = r.URL.Query().Get("uuid")
				gotExpunge = r.URL.Query().Get("expunge")
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(listInstanceResponse{Count: 0})
			}))
			defer srv.Close()

			svc := instance.NewService(newClient(srv.URL))

			err := svc.Destroy(context.Background(), "vm-del-1", tt.expunge)
			if err != nil {
				t.Fatalf("Destroy() error = %v", err)
			}
			if gotUUID != "vm-del-1" {
				t.Errorf("uuid = %q, want %q", gotUUID, "vm-del-1")
			}
			if gotExpunge != tt.wantExpunge {
				t.Errorf("expunge = %q, want %q", gotExpunge, tt.wantExpunge)
			}
		})
	}
}

// TestInstanceListNetworks verifies path and response parsing.
func TestInstanceListNetworks(t *testing.T) {
	networks := []instance.Network{
		{UUID: "net-1", Name: "public", Type: "Shared", PrivateIP: "10.0.0.5", DefaultNetwork: true},
		{UUID: "net-2", Name: "private", Type: "Isolated", PrivateIP: "192.168.1.10"},
	}

	var gotUUID string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/restapi/instance/instanceNetworkList" {
			http.NotFound(w, r)
			return
		}
		gotUUID = r.URL.Query().Get("uuid")
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(listInstanceNetworkResponse{
			Count:                        len(networks),
			KongInstanceNetworkResponses: networks,
		})
	}))
	defer srv.Close()

	svc := instance.NewService(newClient(srv.URL))

	result, err := svc.ListNetworks(context.Background(), "vm-1")
	if err != nil {
		t.Fatalf("ListNetworks() error = %v", err)
	}
	if gotUUID != "vm-1" {
		t.Errorf("uuid query param = %q, want %q", gotUUID, "vm-1")
	}
	if len(result) != 2 {
		t.Fatalf("ListNetworks() returned %d networks, want 2", len(result))
	}
	if result[0].Name != "public" {
		t.Errorf("result[0].Name = %q, want %q", result[0].Name, "public")
	}
	if !result[0].DefaultNetwork {
		t.Errorf("result[0].DefaultNetwork = false, want true")
	}
}

// TestInstanceGetStatus verifies path and status field.
func TestInstanceGetStatus(t *testing.T) {
	type statusResponse struct {
		UUID   string `json:"uuid"`
		Status string `json:"status"`
	}

	var gotUUID string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/restapi/instance/vmStatus" {
			http.NotFound(w, r)
			return
		}
		gotUUID = r.URL.Query().Get("uuid")
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(statusResponse{
			UUID:   "vm-status-1",
			Status: "Running",
		})
	}))
	defer srv.Close()

	svc := instance.NewService(newClient(srv.URL))

	status, err := svc.GetStatus(context.Background(), "vm-status-1")
	if err != nil {
		t.Fatalf("GetStatus() error = %v", err)
	}
	if gotUUID != "vm-status-1" {
		t.Errorf("uuid query param = %q, want %q", gotUUID, "vm-status-1")
	}
	if status.UUID != "vm-status-1" {
		t.Errorf("status.UUID = %q, want %q", status.UUID, "vm-status-1")
	}
	if status.Status != "Running" {
		t.Errorf("status.Status = %q, want %q", status.Status, "Running")
	}
}

// TestInstanceRecover verifies the recoverVm path and response.
func TestInstanceRecover(t *testing.T) {
	recovered := makeInstance("vm-rec-1", "web-01", "Running")

	var gotUUID string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/restapi/instance/recoverVm" {
			http.NotFound(w, r)
			return
		}
		gotUUID = r.URL.Query().Get("uuid")
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(listInstanceResponse{
			Count:                1,
			ListInstanceResponse: []instance.Instance{recovered},
		})
	}))
	defer srv.Close()

	svc := instance.NewService(newClient(srv.URL))

	inst, err := svc.Recover(context.Background(), "vm-rec-1")
	if err != nil {
		t.Fatalf("Recover() error = %v", err)
	}
	if gotUUID != "vm-rec-1" {
		t.Errorf("uuid query param = %q, want %q", gotUUID, "vm-rec-1")
	}
	if inst.UUID != "vm-rec-1" {
		t.Errorf("inst.UUID = %q, want %q", inst.UUID, "vm-rec-1")
	}
}

// TestInstanceListAPIError verifies that non-2xx responses return errors.
func TestInstanceListAPIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"listErrorResponse": map[string]string{
				"errorCode": "UNAUTHORIZED",
				"errorMsg":  "Invalid API key",
			},
		})
	}))
	defer srv.Close()

	svc := instance.NewService(newClient(srv.URL))

	_, err := svc.List(context.Background(), "zone-1", "")
	if err == nil {
		t.Fatal("List() expected error for 401, got nil")
	}
}
