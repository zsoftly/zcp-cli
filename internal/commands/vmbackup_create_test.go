package commands

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/zsoftly/zcp-cli/pkg/api/vmbackup"
	"gopkg.in/yaml.v3"
)

func vmBackupCreateArgs(apiURL string, extra ...string) []string {
	args := []string{
		"create", "vm-1",
		"--cloud-provider", "zsoftly",
		"--region", "yul-1",
		"--project", "project-1",
		"--billing-cycle", "hourly",
		"--plan", "backup-yul",
		"--pseudo-service", "vm-backup",
		"--interval", "dailyAt",
		"--at", "3",
		"--api-url", apiURL,
	}
	return append(args, extra...)
}

func writeVMBackupList(w http.ResponseWriter, data string) {
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{"status":"Success","current_page":1,"last_page":1,"data":%s}`, data)
}

func TestVMBackupCreatePrintsResolvedSlugAsJSON(t *testing.T) {
	t.Setenv("ZCP_BEARER_TOKEN", "test-token")
	listCalls, postCalls := 0, 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/virtual-machines/backups":
			listCalls++
			if listCalls == 1 {
				writeVMBackupList(w, `[]`)
				return
			}
			writeVMBackupList(w, `[{"slug":"vmb-new","virtual_machine":{"slug":"vm-1"},"interval":"dailyAt","at":"3"}]`)
		case r.Method == http.MethodPost && r.URL.Path == "/virtual-machines/vm-1/backups":
			postCalls++
			fmt.Fprint(w, `{"status":"Success","message":"accepted","data":{}}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	stdout, stderr, err := execCmd(t, NewVMBackupCmd(), vmBackupCreateArgs(srv.URL, "--output", "json")...)
	if err != nil {
		t.Fatalf("vm-backup create: %v", err)
	}
	if postCalls != 1 || listCalls != 4 {
		t.Fatalf("post calls = %d, list calls = %d, want 1 and 4", postCalls, listCalls)
	}
	if stderr != "" || strings.Contains(stdout, "accepted") {
		t.Fatalf("stdout=%q stderr=%q, status text must not contaminate structured output", stdout, stderr)
	}
	var got vmbackup.VMBackup
	if err := json.Unmarshal([]byte(stdout), &got); err != nil {
		t.Fatalf("JSON output = %q: %v", stdout, err)
	}
	if got.Slug != "vmb-new" || got.VMSlug() != "vm-1" || got.Interval != "dailyAt" || got.At != 3 {
		t.Fatalf("created backup = %#v, want slug=vmb-new vm=vm-1 interval=dailyAt at=3", got)
	}
}

func TestVMBackupCreatePrintsResolvedSlugAsTable(t *testing.T) {
	t.Setenv("ZCP_BEARER_TOKEN", "test-token")
	listCalls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/virtual-machines/backups":
			listCalls++
			if listCalls == 1 {
				writeVMBackupList(w, `[]`)
				return
			}
			writeVMBackupList(w, `[{"slug":"vmb-new","name":"daily","virtual_machine":{"slug":"vm-1"},"interval":"dailyAt","at":3}]`)
		case r.Method == http.MethodPost && r.URL.Path == "/virtual-machines/vm-1/backups":
			fmt.Fprint(w, `{"status":"Success","data":null}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	stdout, stderr, err := execCmd(t, NewVMBackupCmd(), vmBackupCreateArgs(srv.URL)...)
	if err != nil || !strings.Contains(stdout, "vmb-new") || stderr != "" {
		t.Fatalf("stdout=%q stderr=%q err=%v", stdout, stderr, err)
	}
}

func TestVMBackupCreateExcludesPreexistingMatchingSchedule(t *testing.T) {
	t.Setenv("ZCP_BEARER_TOKEN", "test-token")
	listCalls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/virtual-machines/backups":
			listCalls++
			if listCalls == 1 {
				writeVMBackupList(w, `[{"slug":"vmb-old","virtual_machine":{"slug":"vm-1"},"interval":"dailyAt","at":3}]`)
				return
			}
			writeVMBackupList(w, `[{"slug":"vmb-old","virtual_machine":{"slug":"vm-1"},"interval":"dailyAt","at":3},{"slug":"vmb-new","virtual_machine":{"slug":"vm-1"},"interval":"dailyAt","at":3}]`)
		case r.Method == http.MethodPost && r.URL.Path == "/virtual-machines/vm-1/backups":
			fmt.Fprint(w, `{"status":"Success","data":{}}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	stdout, _, err := execCmd(t, NewVMBackupCmd(), vmBackupCreateArgs(srv.URL, "--output", "json")...)
	if err != nil || !strings.Contains(stdout, `"slug": "vmb-new"`) || strings.Contains(stdout, "vmb-old") {
		t.Fatalf("stdout=%q err=%v", stdout, err)
	}
}

func TestVMBackupCreateUsesSlugFromActionResponse(t *testing.T) {
	t.Setenv("ZCP_BEARER_TOKEN", "test-token")
	listCalls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/virtual-machines/backups":
			listCalls++
			writeVMBackupList(w, `[]`)
		case r.Method == http.MethodPost && r.URL.Path == "/virtual-machines/vm-1/backups":
			fmt.Fprint(w, `{"status":"Success","data":{"slug":"vmb-from-action"}}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	stdout, _, err := execCmd(t, NewVMBackupCmd(), vmBackupCreateArgs(srv.URL, "--output", "json")...)
	if err != nil || listCalls != 1 || !strings.Contains(stdout, `"slug": "vmb-from-action"`) || !strings.Contains(stdout, `"interval": "dailyAt"`) || !strings.Contains(stdout, `"at": 3`) || !strings.Contains(stdout, `"virtual_machine"`) {
		t.Fatalf("stdout=%q list calls=%d err=%v", stdout, listCalls, err)
	}
}

func TestVMBackupCreatePrintsValidYAML(t *testing.T) {
	t.Setenv("ZCP_BEARER_TOKEN", "test-token")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/virtual-machines/backups":
			writeVMBackupList(w, `[]`)
		case r.Method == http.MethodPost && r.URL.Path == "/virtual-machines/vm-1/backups":
			fmt.Fprint(w, `{"status":"Success","data":{"slug":"vmb-yaml"}}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	stdout, stderr, err := execCmd(t, NewVMBackupCmd(), vmBackupCreateArgs(srv.URL, "--output", "yaml")...)
	if err != nil || stderr != "" {
		t.Fatalf("stdout=%q stderr=%q err=%v", stdout, stderr, err)
	}
	var got map[string]interface{}
	if err := yaml.Unmarshal([]byte(stdout), &got); err != nil {
		t.Fatalf("YAML output = %q: %v", stdout, err)
	}
	if got["slug"] != "vmb-yaml" {
		t.Fatalf("slug = %#v, want vmb-yaml", got["slug"])
	}
}

func TestVMBackupCreateReportsAmbiguousLookupAfterCreate(t *testing.T) {
	t.Setenv("ZCP_BEARER_TOKEN", "test-token")
	postCalls, listCalls := 0, 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/virtual-machines/backups":
			listCalls++
			if listCalls == 1 {
				writeVMBackupList(w, `[]`)
				return
			}
			writeVMBackupList(w, `[{"slug":"vmb-one","virtual_machine":{"slug":"vm-1"},"interval":"dailyAt","at":3},{"slug":"vmb-two","virtual_machine":{"slug":"vm-1"},"interval":"dailyAt","at":3}]`)
		case r.Method == http.MethodPost && r.URL.Path == "/virtual-machines/vm-1/backups":
			postCalls++
			fmt.Fprint(w, `{"status":"Success","data":{}}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	_, _, err := execCmd(t, NewVMBackupCmd(), vmBackupCreateArgs(srv.URL)...)
	if err == nil || !strings.Contains(err.Error(), "was created") || !strings.Contains(err.Error(), "cannot determine") || postCalls != 1 {
		t.Fatalf("err=%v post calls=%d", err, postCalls)
	}
}

func TestVMBackupCreateRejectsCandidateWhenLaterPollIsAmbiguous(t *testing.T) {
	t.Setenv("ZCP_BEARER_TOKEN", "test-token")
	postCalls, listCalls := 0, 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/virtual-machines/backups":
			listCalls++
			switch listCalls {
			case 1:
				writeVMBackupList(w, `[]`)
			case 2:
				writeVMBackupList(w, `[{"slug":"vmb-other","virtual_machine":{"slug":"vm-1"},"interval":"dailyAt","at":3}]`)
			default:
				writeVMBackupList(w, `[{"slug":"vmb-other","virtual_machine":{"slug":"vm-1"},"interval":"dailyAt","at":3},{"slug":"vmb-created","virtual_machine":{"slug":"vm-1"},"interval":"dailyAt","at":3}]`)
			}
		case r.Method == http.MethodPost && r.URL.Path == "/virtual-machines/vm-1/backups":
			postCalls++
			fmt.Fprint(w, `{"status":"Success","data":{}}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	_, _, err := execCmd(t, NewVMBackupCmd(), vmBackupCreateArgs(srv.URL)...)
	if err == nil || !strings.Contains(err.Error(), "cannot determine") || postCalls != 1 || listCalls != 4 {
		t.Fatalf("err=%v post calls=%d list calls=%d", err, postCalls, listCalls)
	}
}

func TestVMBackupCreateReportsFailedLookupAfterCreate(t *testing.T) {
	t.Setenv("ZCP_BEARER_TOKEN", "test-token")
	listCalls, postCalls := 0, 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/virtual-machines/backups":
			listCalls++
			if listCalls == 1 {
				writeVMBackupList(w, `[]`)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"status":"Success","current_page":1,"last_page":1,"data":{}}`)
		case r.Method == http.MethodPost && r.URL.Path == "/virtual-machines/vm-1/backups":
			postCalls++
			fmt.Fprint(w, `{"status":"Success","data":{}}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	_, _, err := execCmd(t, NewVMBackupCmd(), vmBackupCreateArgs(srv.URL)...)
	if err == nil || postCalls != 1 || !strings.Contains(err.Error(), "was created") || !strings.Contains(err.Error(), "listing backups after create") {
		t.Fatalf("err=%v post calls=%d", err, postCalls)
	}
}

func TestVMBackupCreateDoesNotPostWhenBaselineListFails(t *testing.T) {
	t.Setenv("ZCP_BEARER_TOKEN", "test-token")
	postCalled := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			postCalled = true
		}
		http.Error(w, `{"message":"list unavailable"}`, http.StatusServiceUnavailable)
	}))
	defer srv.Close()

	_, _, err := execCmd(t, NewVMBackupCmd(), vmBackupCreateArgs(srv.URL)...)
	if err == nil || !strings.Contains(err.Error(), "listing existing backups before create") || postCalled {
		t.Fatalf("err=%v post called=%t", err, postCalled)
	}
}
