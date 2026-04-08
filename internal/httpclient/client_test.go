package httpclient_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/zsoftly/zcp-cli/internal/httpclient"
)

func TestGetInjectsBearerToken(t *testing.T) {
	var gotAuth string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"ok": "true"})
	}))
	defer srv.Close()

	client := httpclient.New(httpclient.Options{
		BaseURL:     srv.URL,
		BearerToken: "my-test-token",
		Timeout:     5 * time.Second,
	})

	var result map[string]string
	err := client.Get(context.Background(), "/test", url.Values{}, &result)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if gotAuth != "Bearer my-test-token" {
		t.Errorf("Authorization header = %q, want %q", gotAuth, "Bearer my-test-token")
	}
}

func TestGetDecodesJSON(t *testing.T) {
	type testResp struct {
		Name  string `json:"name"`
		Count int    `json:"count"`
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(testResp{Name: "zone-1", Count: 3})
	}))
	defer srv.Close()

	client := httpclient.New(httpclient.Options{
		BaseURL:     srv.URL,
		BearerToken: "tok",
		Timeout:     5 * time.Second,
	})

	var result testResp
	if err := client.Get(context.Background(), "/data", url.Values{}, &result); err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if result.Name != "zone-1" {
		t.Errorf("Name = %q, want %q", result.Name, "zone-1")
	}
	if result.Count != 3 {
		t.Errorf("Count = %d, want 3", result.Count)
	}
}

func TestGetQueryParams(t *testing.T) {
	var gotQuery url.Values

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.Query()
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("{}"))
	}))
	defer srv.Close()

	client := httpclient.New(httpclient.Options{
		BaseURL:     srv.URL,
		BearerToken: "tok",
		Timeout:     5 * time.Second,
	})

	q := url.Values{"zoneUuid": {"abc-123"}}
	client.Get(context.Background(), "/test", q, nil)

	if gotQuery.Get("zoneUuid") != "abc-123" {
		t.Errorf("zoneUuid query param = %q, want %q", gotQuery.Get("zoneUuid"), "abc-123")
	}
}

func TestGetHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  "Error",
			"message": "Unauthenticated.",
		})
	}))
	defer srv.Close()

	client := httpclient.New(httpclient.Options{
		BaseURL:     srv.URL,
		BearerToken: "bad-token",
		Timeout:     5 * time.Second,
	})

	err := client.Get(context.Background(), "/protected", url.Values{}, nil)
	if err == nil {
		t.Fatal("expected error for 401, got nil")
	}
}

func TestGetContextCancellation(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// never responds
		select {}
	}))
	defer srv.Close()

	client := httpclient.New(httpclient.Options{
		BaseURL:     srv.URL,
		BearerToken: "tok",
		Timeout:     10 * time.Second,
	})

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	err := client.Get(ctx, "/slow", url.Values{}, nil)
	if err == nil {
		t.Fatal("expected error for cancelled context, got nil")
	}
}
