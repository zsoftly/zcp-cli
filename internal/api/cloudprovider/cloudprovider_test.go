package cloudprovider_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/zsoftly/zcp-cli/internal/api/cloudprovider"
	"github.com/zsoftly/zcp-cli/internal/httpclient"
)

func TestCloudProviderList(t *testing.T) {
	expected := []cloudprovider.CloudProvider{
		{
			ID:          "cp-1",
			Name:        "nimbo",
			DisplayName: "Webberstop Cloud",
			Slug:        "nimbo",
			Status:      true,
			Services:    []string{"Virtual Machine", "Block Storage"},
		},
		{
			ID:          "cp-2",
			Name:        "stratus",
			DisplayName: "CS",
			Slug:        "stratus",
			Status:      true,
			Services:    []string{"IP Address", "VM Snapshot"},
		},
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/cloud-providers" {
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

	svc := cloudprovider.NewService(client)
	providers, err := svc.List(context.Background())
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	if len(providers) != 2 {
		t.Fatalf("List() returned %d providers, want 2", len(providers))
	}
	if providers[0].DisplayName != "Webberstop Cloud" {
		t.Errorf("providers[0].DisplayName = %q, want %q",
			providers[0].DisplayName, "Webberstop Cloud")
	}
	if len(providers[0].Services) != 2 {
		t.Errorf("providers[0].Services has %d items, want 2", len(providers[0].Services))
	}
}

func TestCloudProviderListAPIError(t *testing.T) {
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

	svc := cloudprovider.NewService(client)
	_, err := svc.List(context.Background())
	if err == nil {
		t.Fatal("expected error for 401, got nil")
	}
}
