package commands

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestObjectStorageCredentialsAndCreateUseVisibleKey(t *testing.T) {
	t.Setenv("ZCP_BEARER_TOKEN", "test-token")
	visible := time.Now().Add(time.Minute).UTC().Format(time.RFC3339)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/object-storages/store":
			fmt.Fprint(w, `{"status":"Success","data":{"slug":"store","region":{"slug":"os-yul","cloud_provider_setup":{"config":{"s3_endpoint":"https://s3.example.test"}}}}}`)
		case r.Method == http.MethodPost && r.URL.Path == "/object-storages":
			fmt.Fprint(w, `{"status":"Success","data":{"slug":"store","name":"store"}}`)
		case r.Method == http.MethodGet && r.URL.Path == "/object-storages/store/keys":
			fmt.Fprintf(w, `{"status":"Success","data":[{"id":"key-1","api_key":"new-key","api_secret":"new-secret","status":"active","is_primary":true,"secret_visible_until":%q}]}`, visible)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	for _, args := range [][]string{{"credentials", "store"}, {"create", "--name", "store", "--project", "project", "--region", "os-yul", "--billing-cycle", "hourly", "--storage-gb", "60"}} {
		args = append(args, "--output", "json", "--api-url", srv.URL)
		stdout, stderr, err := execCmd(t, NewObjectStorageCmd(), args...)
		if err != nil || !strings.Contains(stdout, `"api_key": "new-key"`) || !strings.Contains(stdout, `"api_secret": "new-secret"`) || strings.Contains(stderr, "new-secret") {
			t.Fatalf("%v output=%q stderr=%q err=%v", args, stdout, stderr, err)
		}
	}
}

func TestObjectStorageCreateReportsCreatedStoreWhenKeyIsNotVisible(t *testing.T) {
	t.Setenv("ZCP_BEARER_TOKEN", "test-token")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/object-storages":
			fmt.Fprint(w, `{"status":"Success","data":{"slug":"new-store"}}`)
		case r.Method == http.MethodGet && r.URL.Path == "/object-storages/new-store":
			fmt.Fprint(w, `{"status":"Success","data":{"slug":"new-store","region":{"slug":"os-yul","cloud_provider_setup":{"config":{"s3_endpoint":"https://s3.example.test"}}}}}`)
		case r.Method == http.MethodGet && r.URL.Path == "/object-storages/new-store/keys":
			fmt.Fprint(w, `{"status":"Success","data":[{"id":"key-1","api_key":"old","api_secret":"should-not-print","status":"active","secret_visible_until":"2000-01-01T00:00:00Z"}]}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	stdout, stderr, err := execCmd(t, NewObjectStorageCmd(), "create", "--name", "new-store", "--project", "project", "--region", "os-yul", "--billing-cycle", "hourly", "--storage-gb", "60", "--output", "json", "--api-url", srv.URL)
	if err == nil || !strings.Contains(err.Error(), `storage "new-store" was created`) || strings.Contains(stdout, "should-not-print") || strings.Contains(stderr, "should-not-print") {
		t.Fatalf("output=%q stderr=%q err=%v", stdout, stderr, err)
	}
}

func TestObjectStorageCreateReportsCreatedStoreWhenReadBackFails(t *testing.T) {
	t.Setenv("ZCP_BEARER_TOKEN", "test-token")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && r.URL.Path == "/object-storages" {
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"status":"Success","data":{"slug":"new-store"}}`)
			return
		}
		http.Error(w, `{"message":"read-back unavailable"}`, http.StatusServiceUnavailable)
	}))
	defer srv.Close()
	_, _, err := execCmd(t, NewObjectStorageCmd(), "create", "--name", "new-store", "--project", "project", "--region", "os-yul", "--billing-cycle", "hourly", "--storage-gb", "60", "--output", "json", "--api-url", srv.URL)
	if err == nil || !strings.Contains(err.Error(), `storage "new-store" was created`) || !strings.Contains(err.Error(), "read-back unavailable") {
		t.Fatalf("create error=%v", err)
	}
}

func TestObjectStorageKeysRespectVendorLimits(t *testing.T) {
	t.Setenv("ZCP_BEARER_TOKEN", "test-token")
	postCalled, deleteCalled := false, false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/object-storages/store/keys":
			fmt.Fprint(w, `{"status":"Success","data":[{"id":"key-1","api_key":"old","status":"active"},{"id":"key-2","api_key":"new","status":"active"}]}`)
		case r.Method == http.MethodPost && r.URL.Path == "/object-storages/store/keys":
			postCalled = true
			fmt.Fprint(w, `{"status":"Success","data":{"id":"key-3"}}`)
		case r.Method == http.MethodDelete && r.URL.Path == "/object-storages/store/keys/key-1":
			deleteCalled = true
			w.WriteHeader(http.StatusNoContent)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	if _, _, err := execCmd(t, NewObjectStorageCmd(), "keys", "create", "store", "--api-url", srv.URL); err == nil || postCalled {
		t.Fatal("keys create sent POST despite two active keys")
	}
	if _, _, err := execCmd(t, NewObjectStorageCmd(), "keys", "delete", "store", "key-1", "-y", "--api-url", srv.URL); err != nil || !deleteCalled {
		t.Fatalf("keys delete with two active keys = %v, called=%t", err, deleteCalled)
	}
}

func TestObjectStorageKeysCreatePrintsOnlyFreshVisibleSecret(t *testing.T) {
	t.Setenv("ZCP_BEARER_TOKEN", "test-token")
	visible := time.Now().Add(time.Minute).UTC().Format(time.RFC3339)
	postCalls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/object-storages/store/keys":
			fmt.Fprint(w, `{"status":"Success","data":[{"id":"key-1","api_key":"old","status":"active"}]}`)
		case r.Method == http.MethodPost && r.URL.Path == "/object-storages/store/keys":
			postCalls++
			fmt.Fprintf(w, `{"status":"Success","data":{"id":"key-2","api_key":"new-key","api_secret":"new-secret","status":"active","secret_visible_until":%q}}`, visible)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	stdout, _, err := execCmd(t, NewObjectStorageCmd(), "keys", "create", "store", "--output", "json", "--api-url", srv.URL)
	if err != nil || postCalls != 1 || !strings.Contains(stdout, `"api_secret": "new-secret"`) {
		t.Fatalf("keys create output=%q postCalls=%d err=%v", stdout, postCalls, err)
	}
}

func TestObjectStorageKeysCreateRejectsInvalidSecretVisibility(t *testing.T) {
	for _, visibleUntil := range []string{"", "invalid", "2000-01-01T00:00:00Z"} {
		t.Run(visibleUntil, func(t *testing.T) {
			t.Setenv("ZCP_BEARER_TOKEN", "test-token")
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				switch {
				case r.Method == http.MethodGet && r.URL.Path == "/object-storages/store/keys":
					fmt.Fprint(w, `{"status":"Success","data":[{"id":"key-1","api_key":"old","status":"active"}]}`)
				case r.Method == http.MethodPost && r.URL.Path == "/object-storages/store/keys":
					fmt.Fprintf(w, `{"status":"Success","data":{"id":"created-key","api_key":"new-key","api_secret":"must-not-print","status":"active","secret_visible_until":%q}}`, visibleUntil)
				default:
					http.NotFound(w, r)
				}
			}))
			defer srv.Close()
			stdout, stderr, err := execCmd(t, NewObjectStorageCmd(), "keys", "create", "store", "--output", "json", "--api-url", srv.URL)
			if err == nil || !strings.Contains(err.Error(), `key "created-key" was created`) || strings.Contains(stdout, "must-not-print") || strings.Contains(stderr, "must-not-print") {
				t.Fatalf("keys create output=%q stderr=%q err=%v", stdout, stderr, err)
			}
		})
	}
}

func TestObjectStorageKeysListDoesNotPrintSecretAndBlocksLastRevoke(t *testing.T) {
	t.Setenv("ZCP_BEARER_TOKEN", "test-token")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/object-storages/store/keys" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"status":"Success","data":[{"id":"key-1","api_key":"access-key","api_secret":"must-not-print","status":"active","is_primary":true}]}`)
	}))
	defer srv.Close()
	stdout, _, err := execCmd(t, NewObjectStorageCmd(), "keys", "list", "store", "--output", "json", "--api-url", srv.URL)
	if err != nil || strings.Contains(stdout, "must-not-print") || strings.Contains(stdout, "api_secret") {
		t.Fatalf("keys list output=%q err=%v", stdout, err)
	}
	if _, _, err := execCmd(t, NewObjectStorageCmd(), "keys", "delete", "store", "key-1", "-y", "--api-url", srv.URL); err == nil || !strings.Contains(err.Error(), "at least one active key") {
		t.Fatalf("keys delete last active error=%v", err)
	}
}

func TestObjectStorageKeyVendorErrorsPropagate(t *testing.T) {
	t.Setenv("ZCP_BEARER_TOKEN", "test-token")
	listCalls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/object-storages/store/keys":
			if r.Method == http.MethodGet {
				listCalls++
				if listCalls == 1 {
					fmt.Fprint(w, `{"status":"Success","data":[{"id":"key-1","api_key":"key","status":"active"}]}`)
				} else {
					fmt.Fprint(w, `{"status":"Success","data":[{"id":"key-1","api_key":"key","status":"active"},{"id":"key-2","api_key":"key2","status":"active"}]}`)
				}
				return
			}
			http.Error(w, `{"message":"vendor key limit"}`, http.StatusConflict)
		case "/object-storages/store/keys/key-1":
			http.Error(w, `{"message":"vendor revoke failed"}`, http.StatusConflict)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	if _, _, err := execCmd(t, NewObjectStorageCmd(), "keys", "create", "store", "--api-url", srv.URL); err == nil || !strings.Contains(err.Error(), "vendor key limit") {
		t.Fatalf("keys create error=%v", err)
	}
	if _, _, err := execCmd(t, NewObjectStorageCmd(), "keys", "delete", "store", "key-1", "-y", "--api-url", srv.URL); err == nil || !strings.Contains(err.Error(), "vendor revoke failed") {
		t.Fatalf("keys delete error=%v", err)
	}
}
