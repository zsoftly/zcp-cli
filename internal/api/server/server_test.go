package server_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/zsoftly/zcp-cli/internal/api/server"
	"github.com/zsoftly/zcp-cli/internal/httpclient"
)

func TestServerList(t *testing.T) {
	expected := []server.Server{
		{ID: "srv-1", Name: "Cloud Compute", Slug: "cloud-compute", Status: true},
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/servers" {
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

	svc := server.NewService(client)
	servers, err := svc.List(context.Background())
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	if len(servers) != 1 {
		t.Fatalf("List() returned %d servers, want 1", len(servers))
	}
	if servers[0].Name != "Cloud Compute" {
		t.Errorf("servers[0].Name = %q, want %q", servers[0].Name, "Cloud Compute")
	}
	if servers[0].Slug != "cloud-compute" {
		t.Errorf("servers[0].Slug = %q, want %q", servers[0].Slug, "cloud-compute")
	}
}

func TestServerListAPIError(t *testing.T) {
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

	svc := server.NewService(client)
	_, err := svc.List(context.Background())
	if err == nil {
		t.Fatal("expected error for 401, got nil")
	}
}
