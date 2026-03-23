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

func TestPutInjectsAuthHeaders(t *testing.T) {
	var gotMethod, gotAPIKey, gotSecretKey string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotAPIKey = r.Header.Get("apikey")
		gotSecretKey = r.Header.Get("secretkey")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("{}"))
	}))
	defer srv.Close()

	client := httpclient.New(httpclient.Options{
		BaseURL: srv.URL, APIKey: "k", SecretKey: "s", Timeout: 5 * time.Second,
	})

	err := client.Put(context.Background(), "/test", nil, map[string]string{"name": "updated"}, nil)
	if err != nil {
		t.Fatalf("Put() error = %v", err)
	}
	if gotMethod != http.MethodPut {
		t.Errorf("method = %q, want PUT", gotMethod)
	}
	if gotAPIKey != "k" {
		t.Errorf("apikey header = %q, want %q", gotAPIKey, "k")
	}
	if gotSecretKey != "s" {
		t.Errorf("secretkey header = %q, want %q", gotSecretKey, "s")
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
		BaseURL: srv.URL, APIKey: "k", SecretKey: "s", Timeout: 5 * time.Second,
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
		BaseURL: srv.URL, APIKey: "k", SecretKey: "s", Timeout: 5 * time.Second,
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
			"listErrorResponse": map[string]string{"errorCode": "401", "errorMsg": "Unauthorized"},
		})
	}))
	defer srv.Close()

	client := httpclient.New(httpclient.Options{
		BaseURL: srv.URL, APIKey: "bad", SecretKey: "creds", Timeout: 5 * time.Second,
	})

	err := client.Put(context.Background(), "/test", nil, nil, nil)
	if err == nil {
		t.Fatal("expected error for 401, got nil")
	}
}
