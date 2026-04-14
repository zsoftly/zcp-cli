package commands

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// newTestRoot creates a minimal root command with all persistent flags that
// buildClientAndPrinter expects. Attach subcommands with root.AddCommand().
func newTestRoot() *cobra.Command {
	root := &cobra.Command{Use: "zcp"}
	root.PersistentFlags().String("profile", "", "")
	root.PersistentFlags().StringP("output", "o", "table", "")
	root.PersistentFlags().String("api-url", "", "")
	root.PersistentFlags().Int("timeout", 30, "")
	root.PersistentFlags().Bool("debug", false, "")
	root.PersistentFlags().Bool("no-color", false, "")
	root.PersistentFlags().Bool("pager", false, "")
	root.PersistentFlags().BoolP("auto-approve", "y", false, "")
	return root
}

// execCmd runs a command with args and returns stdout, stderr, and any error.
func execCmd(t *testing.T, cmd *cobra.Command, args ...string) (stdout, stderr string, err error) {
	t.Helper()
	root := newTestRoot()
	root.AddCommand(cmd)

	var outBuf, errBuf bytes.Buffer
	root.SetOut(&outBuf)
	root.SetErr(&errBuf)
	root.SetArgs(append([]string{cmd.Use}, args...))

	err = root.Execute()
	return outBuf.String(), errBuf.String(), err
}

// ─── Environment variable resolution ────────────────────────────────────────

func TestResolveProjectFlagTakesPrecedence(t *testing.T) {
	os.Setenv("ZCP_PROJECT", "env-project")
	defer os.Unsetenv("ZCP_PROJECT")

	result := resolveProject("flag-project")
	if result != "flag-project" {
		t.Errorf("resolveProject = %q, want %q", result, "flag-project")
	}
}

func TestResolveProjectFallsBackToEnv(t *testing.T) {
	os.Setenv("ZCP_PROJECT", "env-project")
	defer os.Unsetenv("ZCP_PROJECT")

	result := resolveProject("")
	if result != "env-project" {
		t.Errorf("resolveProject = %q, want %q", result, "env-project")
	}
}

func TestResolveProjectEmptyWhenNeitherSet(t *testing.T) {
	os.Unsetenv("ZCP_PROJECT")

	result := resolveProject("")
	if result != "" {
		t.Errorf("resolveProject = %q, want empty", result)
	}
}

func TestResolveRegionFlagTakesPrecedence(t *testing.T) {
	os.Setenv("ZCP_REGION", "env-region")
	defer os.Unsetenv("ZCP_REGION")

	result := resolveRegion("flag-region")
	if result != "flag-region" {
		t.Errorf("resolveRegion = %q, want %q", result, "flag-region")
	}
}

func TestResolveRegionFallsBackToEnv(t *testing.T) {
	os.Setenv("ZCP_REGION", "env-region")
	defer os.Unsetenv("ZCP_REGION")

	result := resolveRegion("")
	if result != "env-region" {
		t.Errorf("resolveRegion = %q, want %q", result, "env-region")
	}
}

func TestResolveCloudProviderFlagTakesPrecedence(t *testing.T) {
	os.Setenv("ZCP_CLOUD_PROVIDER", "env-cp")
	defer os.Unsetenv("ZCP_CLOUD_PROVIDER")

	result := resolveCloudProvider("flag-cp")
	if result != "flag-cp" {
		t.Errorf("resolveCloudProvider = %q, want %q", result, "flag-cp")
	}
}

func TestResolveCloudProviderFallsBackToEnv(t *testing.T) {
	os.Setenv("ZCP_CLOUD_PROVIDER", "env-cp")
	defer os.Unsetenv("ZCP_CLOUD_PROVIDER")

	result := resolveCloudProvider("")
	if result != "env-cp" {
		t.Errorf("resolveCloudProvider = %q, want %q", result, "env-cp")
	}
}

// ─── Kubernetes billing-cycle validation ────────────────────────────────────

func TestK8sCreateRequiresBillingCycle(t *testing.T) {
	cmd := NewKubernetesCmd()
	root := newTestRoot()
	root.AddCommand(cmd)

	root.SetOut(&bytes.Buffer{})
	root.SetErr(&bytes.Buffer{})
	root.SetArgs([]string{"kubernetes", "create",
		"--name", "test", "--version", "v1.28.4", "--plan", "k8s-1",
		"--cloud-provider", "nimbo", "--region", "noida", "--project", "default",
		"--workers", "1", "--ssh-key", "mykey"})

	err := root.Execute()
	if err == nil {
		t.Fatal("expected error when --billing-cycle is missing")
	}
	if !strings.Contains(err.Error(), "--billing-cycle is required") {
		t.Errorf("error = %q, want '--billing-cycle is required'", err)
	}
}

// ─── Finding 4: company rejects all-empty, sends only changed fields ────────

func TestCompanyRejectsNoFlags(t *testing.T) {
	_, _, err := execCmd(t, newProfileInfoCompanyCmd())
	if err == nil {
		t.Fatal("expected error when no flags set")
	}
	if !strings.Contains(err.Error(), "at least one flag is required") {
		t.Errorf("error = %q, want 'at least one flag is required'", err)
	}
}

func TestCompanyAcceptsPartialFlags(t *testing.T) {
	// Will fail at config loading (no profile), but should pass validation
	_, _, err := execCmd(t, newProfileInfoCompanyCmd(), "--billing-name", "ZSoftly")
	if err == nil {
		// Would only happen if a real profile exists — that's fine too
		return
	}
	if strings.Contains(err.Error(), "at least one flag is required") {
		t.Errorf("should NOT reject when a flag is set, got: %v", err)
	}
}

// ─── Finding 4: time-settings rejects all-empty ─────────────────────────────

func TestTimeSettingsRejectsNoFlags(t *testing.T) {
	_, _, err := execCmd(t, newProfileInfoTimeSettingsCmd())
	if err == nil {
		t.Fatal("expected error when no flags set")
	}
	if !strings.Contains(err.Error(), "at least one flag is required") {
		t.Errorf("error = %q, want 'at least one flag is required'", err)
	}
}

func TestTimeSettingsAcceptsTimezone(t *testing.T) {
	_, _, err := execCmd(t, newProfileInfoTimeSettingsCmd(), "--timezone", "UTC")
	if err == nil {
		return
	}
	if strings.Contains(err.Error(), "at least one flag is required") {
		t.Errorf("should NOT reject when --timezone is set, got: %v", err)
	}
}

// ─── Finding 5: disable-api respects confirmation ───────────────────────────

func TestDisableAPICancelledOnNo(t *testing.T) {
	cmd := newProfileInfoDisableAPICmd()
	root := newTestRoot()
	root.AddCommand(cmd)

	var errBuf bytes.Buffer
	root.SetOut(&bytes.Buffer{})
	root.SetErr(&errBuf)
	root.SetIn(bytes.NewBufferString("n\n"))
	root.SetArgs([]string{"disable-api"})

	err := root.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(errBuf.String(), "Cancelled") {
		t.Errorf("expected 'Cancelled' in stderr, got: %q", errBuf.String())
	}
}

// ─── Finding 6: vmbackup at/immediate range validation ──────────────────────

func TestVMBackupAtOutOfRange(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{"at negative", []string{"vm", "--at", "-1", "--cloud-provider", "x", "--region", "x", "--billing-cycle", "x", "--plan", "x", "--psudo-service", "x", "--project", "x"}, "--at must be between 0 and 23"},
		{"at 24", []string{"vm", "--at", "24", "--cloud-provider", "x", "--region", "x", "--billing-cycle", "x", "--plan", "x", "--psudo-service", "x", "--project", "x"}, "--at must be between 0 and 23"},
		{"immediate 2", []string{"vm", "--immediate", "2", "--cloud-provider", "x", "--region", "x", "--billing-cycle", "x", "--plan", "x", "--psudo-service", "x", "--project", "x"}, "--immediate must be 0 or 1"},
		{"immediate -1", []string{"vm", "--immediate", "-1", "--cloud-provider", "x", "--region", "x", "--billing-cycle", "x", "--plan", "x", "--psudo-service", "x", "--project", "x"}, "--immediate must be 0 or 1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := NewVMBackupCmd()
			root := newTestRoot()
			root.AddCommand(cmd)

			root.SetOut(&bytes.Buffer{})
			root.SetErr(&bytes.Buffer{})
			root.SetArgs(append([]string{"vm-backup", "create"}, tt.args...))

			err := root.Execute()
			if err == nil {
				t.Fatal("expected validation error")
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Errorf("error = %q, want containing %q", err, tt.want)
			}
		})
	}
}

func TestVMBackupAtValidValues(t *testing.T) {
	// at=12, immediate=1 — should pass validation, fail later at config/API
	cmd := NewVMBackupCmd()
	root := newTestRoot()
	root.AddCommand(cmd)

	root.SetOut(&bytes.Buffer{})
	root.SetErr(&bytes.Buffer{})
	root.SetArgs([]string{"vm-backup", "create", "my-vm",
		"--at", "12", "--immediate", "1",
		"--cloud-provider", "x", "--region", "x", "--billing-cycle", "x",
		"--plan", "x", "--psudo-service", "x", "--project", "x"})

	err := root.Execute()
	if err != nil {
		// Config/API errors are expected, but NOT validation errors
		msg := err.Error()
		if strings.Contains(msg, "--at must be") || strings.Contains(msg, "--immediate must be") {
			t.Errorf("valid values should pass validation, got: %v", err)
		}
	}
}
