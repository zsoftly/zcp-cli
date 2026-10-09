package commands

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
)

type aclRulesCommandResponse struct {
	Rules []struct {
		ID string `json:"id"`
	} `json:"rules"`
	NextToken string `json:"next_token"`
}

func newACLRulesServer(t *testing.T, failPage int, requestedPages *[]string) *httptest.Server {
	t.Helper()
	rules := []map[string]interface{}{
		{"id": "r1", "number": 1}, {"id": "r2", "number": 2},
		{"id": "r3", "number": 3}, {"id": "r4", "number": 4},
	}
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/vpcs/my-vpc/network-acl-list" {
			fmt.Fprint(w, `{"status":"Success","data":[{"id":"acl-1","name":"web-acl"}]}`)
			return
		}
		page := 1
		if raw := r.URL.Query().Get("page"); raw != "" {
			var err error
			page, err = strconv.Atoi(raw)
			if err != nil {
				t.Errorf("invalid page %q: %v", raw, err)
			}
		}
		*requestedPages = append(*requestedPages, strconv.Itoa(page))
		if page == failPage {
			http.Error(w, "later page failed", http.StatusInternalServerError)
			return
		}
		pageSize := 2
		if raw := r.URL.Query().Get("per_page"); raw != "" {
			var err error
			pageSize, err = strconv.Atoi(raw)
			if err != nil {
				t.Errorf("invalid per_page %q: %v", raw, err)
			}
		}
		start := (page - 1) * pageSize
		end := start + pageSize
		if start > len(rules) {
			start = len(rules)
		}
		if end > len(rules) {
			end = len(rules)
		}
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "Success", "current_page": page, "data": rules[start:end], "total": len(rules),
		})
	}))
}

func TestACLRulesMaxItemsResumeAndStructuredOutput(t *testing.T) {
	var pages []string
	srv := newACLRulesServer(t, 0, &pages)
	defer srv.Close()

	stdout, _, err := execCmd(t, NewACLCmd(), "rules", "my-vpc", "acl-1", "--max-items", "2", "--output", "json", "--api-url", srv.URL)
	if err != nil {
		t.Fatalf("first command error = %v", err)
	}
	var first aclRulesCommandResponse
	if err := json.Unmarshal([]byte(stdout), &first); err != nil {
		t.Fatalf("decoding first output: %v\n%s", err, stdout)
	}
	if len(first.Rules) != 2 || first.Rules[0].ID != "r1" || first.Rules[1].ID != "r2" || first.NextToken == "" {
		t.Fatalf("first output = %+v, want r1/r2 and a token", first)
	}

	stdout, _, err = execCmd(t, NewACLCmd(), "rules", "my-vpc", "acl-1", "--max-items", "2", "--starting-token", first.NextToken, "--output", "json", "--api-url", srv.URL)
	if err != nil {
		t.Fatalf("resume command error = %v", err)
	}
	var second aclRulesCommandResponse
	if err := json.Unmarshal([]byte(stdout), &second); err != nil {
		t.Fatalf("decoding resume output: %v\n%s", err, stdout)
	}
	if len(second.Rules) != 2 || second.Rules[0].ID != "r3" || second.Rules[1].ID != "r4" || second.NextToken != "" {
		t.Errorf("resume output = %+v, want r3/r4 and no token", second)
	}
	if got, want := strings.Join(pages, ","), "1,2"; got != want {
		t.Errorf("rule page requests = %q, want %q", got, want)
	}
}

func TestACLRulesPaginationFlags(t *testing.T) {
	var pages []string
	srv := newACLRulesServer(t, 0, &pages)
	defer srv.Close()

	stdout, stderr, err := execCmd(t, NewACLCmd(), "rules", "my-vpc", "acl-1", "--max-items", "1", "--api-url", srv.URL)
	if err != nil {
		t.Fatalf("table command error = %v", err)
	}
	if !strings.Contains(stdout, "r1") || !strings.Contains(stderr, "Next token: ") {
		t.Errorf("table output = %q, stderr = %q, want first rule and next token guidance", stdout, stderr)
	}

	stdout, _, err = execCmd(t, NewACLCmd(), "rules", "my-vpc", "acl-1", "--page-size", "3", "--output", "json", "--api-url", srv.URL)
	if err != nil {
		t.Fatalf("page-size command error = %v", err)
	}
	if !strings.HasPrefix(strings.TrimSpace(stdout), "[") {
		t.Errorf("page-size JSON output = %q, want legacy array", stdout)
	}

	stdout, _, err = execCmd(t, NewACLCmd(), "rules", "my-vpc", "acl-1", "--max-items", "1", "--output", "yaml", "--api-url", srv.URL)
	if err != nil {
		t.Fatalf("YAML command error = %v", err)
	}
	if !strings.Contains(stdout, "rules:") || !strings.Contains(stdout, "next_token:") {
		t.Errorf("bounded YAML output = %q, want rules envelope and next token", stdout)
	}

	stdout, _, err = execCmd(t, NewACLCmd(), "rules", "my-vpc", "acl-1", "--no-paginate", "--output", "json", "--api-url", srv.URL)
	if err != nil {
		t.Fatalf("no-paginate command error = %v", err)
	}
	var onePage aclRulesCommandResponse
	if err := json.Unmarshal([]byte(stdout), &onePage); err != nil {
		t.Fatalf("decoding no-paginate output: %v\n%s", err, stdout)
	}
	if len(onePage.Rules) != 2 || onePage.NextToken == "" {
		t.Errorf("no-paginate output = %+v, want one page and next token", onePage)
	}

	if got, want := strings.Join(pages, ","), "1,1,2,1,1"; got != want {
		t.Errorf("rule page requests = %q, want %q", got, want)
	}

	for _, args := range [][]string{
		{"rules", "my-vpc", "acl-1", "--no-paginate", "--max-items", "1"},
		{"rules", "my-vpc", "acl-1", "--no-paginate", "--starting-token", "token"},
		{"rules", "my-vpc", "acl-1", "--no-paginate", "--page-size", "2"},
		{"rules", "my-vpc", "acl-1", "--max-items", "0"},
		{"rules", "my-vpc", "acl-1", "--page-size", "0"},
		{"rules", "my-vpc", "acl-1", "--starting-token", ""},
	} {
		if _, _, err := execCmd(t, NewACLCmd(), append(args, "--api-url", srv.URL)...); err == nil {
			t.Errorf("args %v returned nil error", args)
		}
	}
}

func TestACLRulesDoesNotPrintPartialResultsOnLaterFailure(t *testing.T) {
	var pages []string
	srv := newACLRulesServer(t, 2, &pages)
	defer srv.Close()

	stdout, _, err := execCmd(t, NewACLCmd(), "rules", "my-vpc", "acl-1", "--max-items", "4", "--output", "json", "--api-url", srv.URL)
	if err == nil {
		t.Fatal("command error = nil, want later page failure")
	}
	if strings.Contains(stdout, `"id"`) || strings.Contains(stdout, `"next_token"`) {
		t.Errorf("stdout = %q, want no partial structured result", stdout)
	}
}
