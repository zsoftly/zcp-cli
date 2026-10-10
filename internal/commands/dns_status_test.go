package commands

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDNSShowResolvesOmittedStatusFromList(t *testing.T) {
	var listCalls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/dns/domains/example-com-1":
			fmt.Fprint(w, `{"status":"Success","data":{"slug":"example-com-1","name":"example.com"}}`)
		case "/dns/domains":
			listCalls++
			fmt.Fprint(w, `{"status":"Success","current_page":1,"total":1,"data":[{"slug":"example-com-1","status":false}]}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	t.Setenv("ZCP_BEARER_TOKEN", "test-token")

	out, _, err := execCmd(t, NewDNSCmd(), "show", "example-com-1", "--api-url", srv.URL)
	if err != nil {
		t.Fatalf("dns show error = %v", err)
	}
	if !strings.Contains(out, "false") {
		t.Errorf("dns show output = %q, want resolved explicit false", out)
	}
	if listCalls != 1 {
		t.Errorf("list calls = %d, want 1", listCalls)
	}
}

func TestDNSShowPreservesExplicitFalseWithoutListLookup(t *testing.T) {
	var listCalls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/dns/domains/example-com-1" {
			fmt.Fprint(w, `{"status":"Success","data":{"slug":"example-com-1","name":"example.com","status":false}}`)
			return
		}
		if r.URL.Path == "/dns/domains" {
			listCalls++
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()
	t.Setenv("ZCP_BEARER_TOKEN", "test-token")

	out, _, err := execCmd(t, NewDNSCmd(), "show", "example-com-1", "--api-url", srv.URL)
	if err != nil {
		t.Fatalf("dns show error = %v", err)
	}
	if !strings.Contains(out, "false") {
		t.Errorf("dns show output = %q, want explicit false", out)
	}
	if listCalls != 0 {
		t.Errorf("list calls = %d, want no lookup when show reports status", listCalls)
	}
}

func TestDNSShowLeavesStatusUnknownWhenListHasNoMatchingSlug(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/dns/domains/example-com-1":
			fmt.Fprint(w, `{"status":"Success","data":{"slug":"example-com-1","name":"example.com"}}`)
		case "/dns/domains":
			fmt.Fprint(w, `{"status":"Success","current_page":1,"total":1,"data":[{"slug":"other-domain","status":true}]}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	t.Setenv("ZCP_BEARER_TOKEN", "test-token")

	out, _, err := execCmd(t, NewDNSCmd(), "show", "example-com-1", "--api-url", srv.URL)
	if err != nil {
		t.Fatalf("dns show error = %v", err)
	}
	if !strings.Contains(out, "-") || strings.Contains(out, "true") {
		t.Errorf("dns show output = %q, want unknown status marker without a fabricated value", out)
	}
}

func TestDNSShowReturnsListLookupErrorRatherThanFabricatingStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/dns/domains/example-com-1":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"status":"Success","data":{"slug":"example-com-1","name":"example.com"}}`)
		case "/dns/domains":
			http.Error(w, "unavailable", http.StatusServiceUnavailable)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	t.Setenv("ZCP_BEARER_TOKEN", "test-token")

	out, _, err := execCmd(t, NewDNSCmd(), "show", "example-com-1", "--api-url", srv.URL)
	if err == nil {
		t.Fatal("dns show error = nil, want list lookup error")
	}
	if !strings.Contains(err.Error(), "resolving status") {
		t.Errorf("dns show error = %q, want status resolution context", err)
	}
	if strings.Contains(out, "true") || strings.Contains(out, "false") {
		t.Errorf("dns show output = %q, must not fabricate a status", out)
	}
}
