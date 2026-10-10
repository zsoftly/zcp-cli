package scheduler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/zsoftly/zcp-cli/pkg/httpclient"
)

func testService(s *httptest.Server) *Service {
	return NewService(httpclient.New(httpclient.Options{BaseURL: s.URL, BearerToken: "test", MaxRetries: 0}))
}

func TestListFetchesAllPages(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("actionable_type") != "Virtual Machine Backup" {
			t.Errorf("actionable_type = %q", r.URL.Query().Get("actionable_type"))
		}
		switch r.URL.Query().Get("page") {
		case "1":
			fmt.Fprint(w, `{"current_page":1,"total":2,"data":[{"id":"1"}]}`)
		case "2":
			fmt.Fprint(w, `{"current_page":2,"total":2,"data":[{"id":"2"}]}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer s.Close()
	got, err := testService(s).List(context.Background(), "yul-1", "test")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[1].ID != "2" {
		t.Fatalf("policies=%+v", got)
	}
}

func TestListReturnsNoPartialResultsWhenLaterPageFails(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("page") == "1" {
			fmt.Fprint(w, `{"current_page":1,"total":2,"data":[{"id":"1"}]}`)
			return
		}
		http.Error(w, "failed", http.StatusInternalServerError)
	}))
	defer s.Close()
	got, err := testService(s).List(context.Background(), "", "")
	if err == nil || got != nil {
		t.Fatalf("got=%+v err=%v, want nil/error", got, err)
	}
}

func TestMutationsUseSchedulerRoutes(t *testing.T) {
	paths := make([]string, 0, 4)
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.Method+" "+r.URL.Path)
		fmt.Fprint(w, `{"data":{"id":"policy"}}`)
	}))
	defer s.Close()
	svc := testService(s)
	ctx := context.Background()
	if _, err := svc.Update(ctx, "policy", UpdateRequest{Interval: "dailyAt", At: "01:00", Timezone: "UTC", RetentionPolicy: 1}); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Pause(ctx, "policy"); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Resume(ctx, "policy"); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.RunNow(ctx, "policy"); err != nil {
		t.Fatal(err)
	}
	want := []string{"PUT /scheduler-actions/policy", "PATCH /scheduler-actions/policy/pause", "PATCH /scheduler-actions/policy/resume", "POST /scheduler/policy/run-now"}
	for i := range want {
		if paths[i] != want[i] {
			t.Errorf("path %d = %q, want %q", i, paths[i], want[i])
		}
	}
}

func TestPolicyDecodesQuotedDay(t *testing.T) {
	var policy Policy
	if err := json.Unmarshal([]byte(`{"day":"0"}`), &policy); err != nil {
		t.Fatal(err)
	}
	if policy.Day == nil || *policy.Day != 0 {
		t.Fatalf("day=%v", policy.Day)
	}
}

func TestPolicyOutputOmitsProviderAndAccountData(t *testing.T) {
	var policy Policy
	if err := json.Unmarshal([]byte(`{"id":"policy","config":{"token":"secret"},"project":{"slug":"test","token":"secret"},"actionable":{"slug":"vm","access_key_token":"secret"}}`), &policy); err != nil {
		t.Fatal(err)
	}
	encoded, err := json.Marshal(policy)
	if err != nil {
		t.Fatal(err)
	}
	if string(encoded) == "" || strings.Contains(string(encoded), "secret") || strings.Contains(string(encoded), "config") {
		t.Fatalf("unsafe output: %s", encoded)
	}
}

func TestPolicyPreservesFailedLastRunState(t *testing.T) {
	var policy Policy
	if err := json.Unmarshal([]byte(`{"last_run_status":"failed","last_run_time":"2026-10-10T01:00:00Z","last_error_message":"provider error","is_healthy":false}`), &policy); err != nil {
		t.Fatal(err)
	}
	if policy.IsHealthy == nil || *policy.IsHealthy || policy.LastRunStatus != "failed" || policy.LastErrorMessage != "provider error" {
		t.Fatalf("policy=%+v", policy)
	}
}
