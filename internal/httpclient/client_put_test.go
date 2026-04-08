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

func TestPutInjectsBearerToken(t *testing.T) {
	var gotMethod, gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotAuth = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("{}"))
	}))
	defer srv.Close()

	client := httpclient.New(httpclient.Options{
		BaseURL: srv.URL, BearerToken: "tok", Timeout: 5 * time.Second,
	})

	err := client.Put(context.Background(), "/test", nil, map[string]string{"name": "updated"}, nil)
	if err != nil {
		t.Fatalf("Put() error = %v", err)
	}
	if gotMethod != http.MethodPut {
		t.Errorf("method = %q, want PUT", gotMethod)
	}
	if gotAuth != "Bearer tok" {
		t.Errorf("Authorization header = %q, want %q", gotAuth, "Bearer tok")
	}
}

func TestPutWithQueryParams(t *testing.T) {
	var gotQuery url.Values
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.Query()
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("{}"))
	}))
	defer srv.Close()

	client := httpclient.New(httpclient.Options{
		BaseURL: srv.URL, BearerToken: "tok", Timeout: 5 * time.Second,
	})

	q := url.Values{"uuid": {"vm-123"}, "size": {"3"}}
	err := client.Put(context.Background(), "/restapi/kubernetes/scaleKubernetes", q, nil, nil)
	if err != nil {
		t.Fatalf("Put() error = %v", err)
	}
	if gotQuery.Get("uuid") != "vm-123" {
		t.Errorf("uuid param = %q, want %q", gotQuery.Get("uuid"), "vm-123")
	}
	if gotQuery.Get("size") != "3" {
		t.Errorf("size param = %q, want %q", gotQuery.Get("size"), "3")
	}
}

func TestPutDecodesResponse(t *testing.T) {
	type resp struct {
		Name string `json:"name"`
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(resp{Name: "updated"})
	}))
	defer srv.Close()

	client := httpclient.New(httpclient.Options{
		BaseURL: srv.URL, BearerToken: "tok", Timeout: 5 * time.Second,
	})

	var result resp
	if err := client.Put(context.Background(), "/test", nil, map[string]string{"name": "updated"}, &result); err != nil {
		t.Fatalf("Put() error = %v", err)
	}
	if result.Name != "updated" {
		t.Errorf("Name = %q, want %q", result.Name, "updated")
	}
}

func TestPutHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  "Error",
			"message": "Unauthenticated.",
		})
	}))
	defer srv.Close()

	client := httpclient.New(httpclient.Options{
		BaseURL: srv.URL, BearerToken: "bad-token", Timeout: 5 * time.Second,
	})

	err := client.Put(context.Background(), "/test", nil, nil, nil)
	if err == nil {
		t.Fatal("expected error for 401, got nil")
	}
}
