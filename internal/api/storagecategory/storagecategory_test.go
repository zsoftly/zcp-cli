package storagecategory_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/zsoftly/zcp-cli/internal/api/storagecategory"
	"github.com/zsoftly/zcp-cli/internal/httpclient"
)

func TestStorageCategoryList(t *testing.T) {
	expected := []storagecategory.StorageCategory{
		{ID: "sc-1", Name: "SSD Storage", Slug: "ssd-storage", Status: true},
		{ID: "sc-2", Name: "NVMe", Slug: "nvme", Status: true},
		{ID: "sc-3", Name: "HDD Storage", Slug: "hdd-storage", Status: true},
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/storage-categories" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		data, _ := json.Marshal(expected)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  "Success",
			"message": "OK",
			"data":    json.RawMessage(data),
		})
	}))
	defer srv.Close()

	client := httpclient.New(httpclient.Options{
		BaseURL:     srv.URL,
		BearerToken: "test-token",
		Timeout:     5 * time.Second,
	})

	svc := storagecategory.NewService(client)
	categories, err := svc.List(context.Background())
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	if len(categories) != 3 {
		t.Fatalf("List() returned %d categories, want 3", len(categories))
	}
	if categories[0].Slug != "ssd-storage" {
		t.Errorf("categories[0].Slug = %q, want %q", categories[0].Slug, "ssd-storage")
	}
	if categories[1].Name != "NVMe" {
		t.Errorf("categories[1].Name = %q, want %q", categories[1].Name, "NVMe")
	}
}

func TestStorageCategoryListAPIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(401)
		json.NewEncoder(w).Encode(map[string]string{
			"message": "Unauthenticated.",
		})
	}))
	defer srv.Close()

	client := httpclient.New(httpclient.Options{
		BaseURL:     srv.URL,
		BearerToken: "test-token",
		Timeout:     5 * time.Second,
	})

	svc := storagecategory.NewService(client)
	_, err := svc.List(context.Background())
	if err == nil {
		t.Fatal("expected error for 401, got nil")
	}
}
