package commands

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// execCaptureStdio runs a command while capturing both os.Stdout and
// os.Stderr, in addition to cobra's own out/err buffers. vm-backup delete
// writes its success and not-found messages directly to os.Stdout/os.Stderr
// (not cmd.OutOrStdout()/ErrOrStderr()), so execCapture (which only swaps
// os.Stdout) is not enough to observe the not-found message.
func execCaptureStdio(t *testing.T, cmd *cobra.Command, args ...string) (stdout, stderr string, err error) {
	t.Helper()
	oldOut, oldErr := os.Stdout, os.Stderr
	outR, outW, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe() for stdout: %v", err)
	}
	defer outR.Close()
	errR, errW, err := os.Pipe()
	if err != nil {
		outW.Close()
		t.Fatalf("os.Pipe() for stderr: %v", err)
	}
	defer errR.Close()
	os.Stdout = outW
	os.Stderr = errW
	// Restore the real streams even if execCmd panics or calls t.Fatal.
	defer func() { os.Stdout, os.Stderr = oldOut, oldErr }()
	cobraOut, cobraErr, runErr := execCmd(t, cmd, args...)
	outW.Close()
	errW.Close()
	os.Stdout, os.Stderr = oldOut, oldErr
	pipedOut, err := io.ReadAll(outR)
	if err != nil {
		t.Fatalf("reading captured stdout: %v", err)
	}
	pipedErr, err := io.ReadAll(errR)
	if err != nil {
		t.Fatalf("reading captured stderr: %v", err)
	}
	return cobraOut + string(pipedOut), cobraErr + string(pipedErr), runErr
}

// ─── backup/vm-backup --interval validation ─────────────────────────────────
//
// The API only accepts "dailyAt" and "hourlyAt" for --interval; every other
// value ("daily", "weekly", "monthly", "hourly", "weeklyAt", "monthlyAt") is
// rejected server-side with "The selected interval is invalid." (verified
// live 2026-09-06). These tests confirm the CLI now rejects bad values
// up front instead of round-tripping to the API.

func TestBackupCreateRejectsInvalidInterval(t *testing.T) {
	cmd := NewBackupCmd()
	root := newTestRoot()
	root.AddCommand(cmd)

	root.SetOut(&bytes.Buffer{})
	root.SetErr(&bytes.Buffer{})
	root.SetArgs([]string{"backup", "create",
		"--volume", "root-1234", "--interval", "daily",
		"--region", "yul-1", "--billing-cycle", "hourly",
		"--plan", "backup-yul", "--project", "default-9"})

	err := root.Execute()
	if err == nil {
		t.Fatal("expected validation error for --interval daily")
	}
	want := `--interval must be one of: dailyAt, hourlyAt (got "daily")`
	if !strings.Contains(err.Error(), want) {
		t.Errorf("error = %q, want containing %q", err, want)
	}
}

func TestVMBackupCreateRejectsInvalidInterval(t *testing.T) {
	cmd := NewVMBackupCmd()
	root := newTestRoot()
	root.AddCommand(cmd)

	root.SetOut(&bytes.Buffer{})
	root.SetErr(&bytes.Buffer{})
	root.SetArgs([]string{"vm-backup", "create", "my-vm", "--interval", "weekly"})

	err := root.Execute()
	if err == nil {
		t.Fatal("expected validation error for --interval weekly")
	}
	want := `--interval must be one of: dailyAt, hourlyAt (got "weekly")`
	if !strings.Contains(err.Error(), want) {
		t.Errorf("error = %q, want containing %q", err, want)
	}
}

func TestVMBackupCreateDefaultIntervalPassesValidation(t *testing.T) {
	// No --interval given: the default value must pass client-side validation.
	// This asserts against the flag's default directly instead of executing
	// the command, since executing it would proceed past validation and reach
	// the network.
	cmd := NewVMBackupCmd()
	create, _, err := cmd.Find([]string{"create"})
	if err != nil {
		t.Fatalf("Find(create) error = %v", err)
	}
	flag := create.Flags().Lookup("interval")
	if flag == nil {
		t.Fatal("--interval flag not found on vm-backup create")
	}
	if err := validateBackupInterval(flag.DefValue); err != nil {
		t.Errorf("default --interval value %q should pass validation, got: %v", flag.DefValue, err)
	}
	if err := validateBackupInterval("dailyAt"); err != nil {
		t.Errorf("validateBackupInterval(\"dailyAt\") = %v, want nil", err)
	}
	if err := validateBackupInterval("hourlyAt"); err != nil {
		t.Errorf("validateBackupInterval(\"hourlyAt\") = %v, want nil", err)
	}
}

// ─── vm-backup delete ────────────────────────────────────────────────────────
//
// Verified live 2026-09-06: the service-cancellation endpoint accepts
// service_name "Backups" for VM backup schedules, and returns a 403 with
// message "The selected service not found." for a slug that does not exist.

func TestVMBackupDeleteRequestsCancellation(t *testing.T) {
	var gotPath, gotMethod string
	var gotBody map[string]interface{}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		json.NewDecoder(r.Body).Decode(&gotBody)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"status": "Success", "message": "Ok"})
	}))
	defer srv.Close()

	t.Setenv("ZCP_BEARER_TOKEN", "test-tok")

	stdout, _, err := execCaptureStdio(t, NewVMBackupCmd(), "delete", "vmb-001001-0001", "--yes", "--api-url", srv.URL)
	if err != nil {
		t.Fatalf("delete error = %v", err)
	}
	if gotMethod != http.MethodPost {
		t.Errorf("method = %q, want %q", gotMethod, http.MethodPost)
	}
	if gotPath != "/billing/service-cancel-requests/vmb-001001-0001" {
		t.Errorf("path = %q, want %q", gotPath, "/billing/service-cancel-requests/vmb-001001-0001")
	}
	if gotBody["service_name"] != "Backups" {
		t.Errorf("body[service_name] = %v, want %q", gotBody["service_name"], "Backups")
	}
	if !strings.Contains(stdout, `Deletion requested for VM backup "vmb-001001-0001"`) {
		t.Errorf("stdout = %q, want it to contain the deletion-requested message", stdout)
	}
}

func TestVMBackupDeleteNotFoundIsNoop(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  "Error",
			"message": "The selected service not found.",
		})
	}))
	defer srv.Close()

	t.Setenv("ZCP_BEARER_TOKEN", "test-tok")

	stdout, stderr, err := execCaptureStdio(t, NewVMBackupCmd(), "delete", "does-not-exist", "--yes", "--api-url", srv.URL)
	if err != nil {
		t.Fatalf("expected nil error for not-found no-op, got: %v", err)
	}
	if !strings.Contains(stdout+stderr, "not found") {
		t.Errorf("output = %q, want it to contain a not-found message", stdout+stderr)
	}
}

func TestVMBackupDeleteOtherErrorPropagates(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  "Error",
			"message": "Something else went wrong.",
		})
	}))
	defer srv.Close()

	t.Setenv("ZCP_BEARER_TOKEN", "test-tok")

	_, _, err := execCaptureStdio(t, NewVMBackupCmd(), "delete", "some-slug", "--yes", "--api-url", srv.URL)
	if err == nil {
		t.Fatal("expected error to propagate for an unrelated 403 message")
	}
}
