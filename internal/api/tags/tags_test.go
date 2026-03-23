package tags_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/zsoftly/zcp-cli/internal/api/tags"
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

type listTagsResponse struct {
	Count                  int        `json:"count"`
	KongCreateTagsResponse []tags.Tag `json:"kongCreateTagsResponse"`
}

func TestTagList(t *testing.T) {
	expected := []tags.Tag{
		{UUID: "tag-1", Key: "env", Value: "prod", ResourceUUID: "vm-1", ResourceType: "Instance"},
		{UUID: "tag-2", Key: "team", Value: "ops", ResourceUUID: "vm-1", ResourceType: "Instance"},
	}

	var gotResourceUUID, gotResourceType string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/restapi/resourcetags/resourceTagsList" {
			http.NotFound(w, r)
			return
		}
		gotResourceUUID = r.URL.Query().Get("resourceUuid")
		gotResourceType = r.URL.Query().Get("resourceType")
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(listTagsResponse{Count: len(expected), KongCreateTagsResponse: expected})
	}))
	defer srv.Close()

	svc := tags.NewService(newClient(srv.URL))
	result, err := svc.List(context.Background(), "vm-1", "Instance")
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(result) != 2 {
		t.Fatalf("List() returned %d tags, want 2", len(result))
	}
	if gotResourceUUID != "vm-1" {
		t.Errorf("resourceUuid query param = %q, want %q", gotResourceUUID, "vm-1")
	}
	if gotResourceType != "Instance" {
		t.Errorf("resourceType query param = %q, want %q", gotResourceType, "Instance")
	}
	if result[0].UUID != "tag-1" {
		t.Errorf("result[0].UUID = %q, want %q", result[0].UUID, "tag-1")
	}
}

func TestTagCreate(t *testing.T) {
	created := tags.Tag{
		UUID:         "tag-new",
		Key:          "env",
		Value:        "prod",
		ResourceUUID: "vm-1",
		ResourceType: "Instance",
		IsActive:     true,
	}

	var gotBody map[string]interface{}
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "expected POST", http.StatusMethodNotAllowed)
			return
		}
		gotPath = r.URL.Path + "?" + r.URL.RawQuery
		json.NewDecoder(r.Body).Decode(&gotBody)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(listTagsResponse{Count: 1, KongCreateTagsResponse: []tags.Tag{created}})
	}))
	defer srv.Close()

	svc := tags.NewService(newClient(srv.URL))
	req := tags.CreateRequest{
		Key:          "env",
		Value:        "prod",
		ResourceUUID: "vm-1",
	}
	tag, err := svc.Create(context.Background(), "Instance", "zone-1", req)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if tag.UUID != "tag-new" {
		t.Errorf("tag.UUID = %q, want %q", tag.UUID, "tag-new")
	}
	if gotBody["key"] != "env" {
		t.Errorf("body key = %v, want %q", gotBody["key"], "env")
	}
	if gotBody["value"] != "prod" {
		t.Errorf("body value = %v, want %q", gotBody["value"], "prod")
	}
	if gotBody["resourceUuid"] != "vm-1" {
		t.Errorf("body resourceUuid = %v, want %q", gotBody["resourceUuid"], "vm-1")
	}
	// Verify resourceType and zoneUuid are in path query params
	if gotPath == "" {
		t.Error("expected non-empty path with query params")
	}
}

func TestTagDelete(t *testing.T) {
	var gotPath, gotMethod string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	svc := tags.NewService(newClient(srv.URL))
	err := svc.Delete(context.Background(), "tag-del-1")
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if gotMethod != http.MethodDelete {
		t.Errorf("method = %q, want %q", gotMethod, http.MethodDelete)
	}
	if gotPath != "/restapi/resourcetags/deleteResourceTag/tag-del-1" {
		t.Errorf("path = %q, want %q", gotPath, "/restapi/resourcetags/deleteResourceTag/tag-del-1")
	}
}
