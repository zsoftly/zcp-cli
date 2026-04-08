package region_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/zsoftly/zcp-cli/internal/api/region"
	"github.com/zsoftly/zcp-cli/internal/httpclient"
)

func TestRegionList(t *testing.T) {
	expected := []region.Region{
		{
			ID:      "region-1",
			Name:    "NOIDA",
			Slug:    "noida",
			Country: "India",
			Status:  true,
			CloudProvider: &region.CloudProvider{
				ID:          "cp-1",
				Name:        "nimbo",
				DisplayName: "Webberstop Cloud",
				Slug:        "nimbo",
			},
		},
		{
			ID:      "region-2",
			Name:    "Mumbai",
			Slug:    "mumbai",
			Country: "India",
			Status:  true,
		},
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/regions" {
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

	svc := region.NewService(client)
	regions, err := svc.List(context.Background())
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	if len(regions) != 2 {
		t.Fatalf("List() returned %d regions, want 2", len(regions))
	}
	if regions[0].ID != "region-1" {
		t.Errorf("regions[0].ID = %q, want %q", regions[0].ID, "region-1")
	}
	if regions[0].Name != "NOIDA" {
		t.Errorf("regions[0].Name = %q, want %q", regions[0].Name, "NOIDA")
	}
	if regions[0].CloudProvider == nil {
		t.Fatal("regions[0].CloudProvider is nil, want non-nil")
	}
	if regions[0].CloudProvider.DisplayName != "Webberstop Cloud" {
		t.Errorf("regions[0].CloudProvider.DisplayName = %q, want %q",
			regions[0].CloudProvider.DisplayName, "Webberstop Cloud")
	}
	if regions[1].Slug != "mumbai" {
		t.Errorf("regions[1].Slug = %q, want %q", regions[1].Slug, "mumbai")
	}
}

func TestRegionListAPIError(t *testing.T) {
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

	svc := region.NewService(client)
	_, err := svc.List(context.Background())
	if err == nil {
		t.Fatal("expected error for 401, got nil")
	}
}
