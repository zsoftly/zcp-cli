package snapshotpolicy_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/zsoftly/zcp-cli/internal/api/snapshotpolicy"
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

type listSnapshotPoliciesResponse struct {
	Count                        int                             `json:"count"`
	ListSnapShotPoliciesResponse []snapshotpolicy.SnapshotPolicy `json:"listSnapShotPoliciesResponse"`
}

func TestSnapshotPolicyList(t *testing.T) {
	expected := []snapshotpolicy.SnapshotPolicy{
		{UUID: "sp-1", VolumeUUID: "vol-1", IntervalType: "daily", Status: "Active"},
		{UUID: "sp-2", VolumeUUID: "vol-1", IntervalType: "weekly", Status: "Active"},
	}

	var gotVolumeUUID string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/restapi/snapshotPolicy/snapshotPolicyList" {
			http.NotFound(w, r)
			return
		}
		gotVolumeUUID = r.URL.Query().Get("volumeUuid")
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(listSnapshotPoliciesResponse{Count: len(expected), ListSnapShotPoliciesResponse: expected})
	}))
	defer srv.Close()

	svc := snapshotpolicy.NewService(newClient(srv.URL))
	policies, err := svc.List(context.Background(), "vol-1", "")
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(policies) != 2 {
		t.Fatalf("List() returned %d policies, want 2", len(policies))
	}
	if gotVolumeUUID != "vol-1" {
		t.Errorf("volumeUuid query param = %q, want %q", gotVolumeUUID, "vol-1")
	}
	if policies[0].UUID != "sp-1" {
		t.Errorf("policies[0].UUID = %q, want %q", policies[0].UUID, "sp-1")
	}
}

func TestSnapshotPolicyCreate(t *testing.T) {
	created := snapshotpolicy.SnapshotPolicy{
		UUID:             "sp-new",
		VolumeUUID:       "vol-1",
		IntervalType:     "daily",
		ScheduleTime:     "02:00",
		TimeZone:         "UTC",
		MaximumSnapshots: "7",
		Status:           "Active",
	}

	var gotBody map[string]interface{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "expected POST", http.StatusMethodNotAllowed)
			return
		}
		if r.URL.Path != "/restapi/snapshotPolicy/createSnapshotPolicy" {
			http.NotFound(w, r)
			return
		}
		json.NewDecoder(r.Body).Decode(&gotBody)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(listSnapshotPoliciesResponse{Count: 1, ListSnapShotPoliciesResponse: []snapshotpolicy.SnapshotPolicy{created}})
	}))
	defer srv.Close()

	svc := snapshotpolicy.NewService(newClient(srv.URL))
	req := snapshotpolicy.CreateRequest{
		VolumeUUID:       "vol-1",
		IntervalType:     "daily",
		Timer:            "02:00",
		TimeZone:         "UTC",
		MaximumSnapshots: "7",
	}
	policy, err := svc.Create(context.Background(), req)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if policy.UUID != "sp-new" {
		t.Errorf("policy.UUID = %q, want %q", policy.UUID, "sp-new")
	}
	if gotBody["volumeUuid"] != "vol-1" {
		t.Errorf("body volumeUuid = %v, want %q", gotBody["volumeUuid"], "vol-1")
	}
	if gotBody["intervalType"] != "daily" {
		t.Errorf("body intervalType = %v, want %q", gotBody["intervalType"], "daily")
	}
	if gotBody["timer"] != "02:00" {
		t.Errorf("body timer = %v, want %q", gotBody["timer"], "02:00")
	}
	if gotBody["timeZone"] != "UTC" {
		t.Errorf("body timeZone = %v, want %q", gotBody["timeZone"], "UTC")
	}
	if gotBody["maximumSnapshots"] != "7" {
		t.Errorf("body maximumSnapshots = %v, want %q", gotBody["maximumSnapshots"], "7")
	}
}

func TestSnapshotPolicyDelete(t *testing.T) {
	var gotPath, gotMethod string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	svc := snapshotpolicy.NewService(newClient(srv.URL))
	err := svc.Delete(context.Background(), "sp-del-1")
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if gotMethod != http.MethodDelete {
		t.Errorf("method = %q, want %q", gotMethod, http.MethodDelete)
	}
	if gotPath != "/restapi/snapshotPolicy/deleteSnapshotPolicy/sp-del-1" {
		t.Errorf("path = %q, want %q", gotPath, "/restapi/snapshotPolicy/deleteSnapshotPolicy/sp-del-1")
	}
}
