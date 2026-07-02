package root

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// TestEnforceScopeInjectsProfileDefaults locks in the scope-gate fix: the gate
// must inject the active profile's default region/project onto the command's
// flags so commands using the bare flag+env resolvers (e.g. `instance create`)
// see them. Without the injection this command errored "--project is required"
// even though the gate accepted the profile default; with it, resolution gets
// past scope and fails on the next required flag (--template), proving the
// defaults reached the command layer. No API call is made.
func TestEnforceScopeInjectsProfileDefaults(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("config path override uses XDG_CONFIG_HOME, not supported on windows")
	}

	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "zcp"), 0o700); err != nil {
		t.Fatal(err)
	}
	cfg := `active_profile: test
profiles:
  test:
    name: test
    bearer_token: fake-token
    region: yul-1
    project: default-9
    cloud_provider: nimbo
`
	if err := os.WriteFile(filepath.Join(dir, "zcp", "config.yaml"), []byte(cfg), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("XDG_CONFIG_HOME", dir)
	t.Setenv("ZCP_REGION", "")
	t.Setenv("ZCP_PROJECT", "")
	t.Setenv("ZCP_PROFILE", "")

	rootCmd.SetOut(&bytes.Buffer{})
	rootCmd.SetErr(&bytes.Buffer{})
	rootCmd.SetArgs([]string{"instance", "create", "--name", "test-vm"})

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected an error (no --template given), got nil")
	}
	if strings.Contains(err.Error(), "--region is required") || strings.Contains(err.Error(), "--project is required") {
		t.Fatalf("profile defaults were not injected into the command's flags: %v", err)
	}
	if !strings.Contains(err.Error(), "--template is required") {
		t.Fatalf("expected to fail on --template (past scope resolution), got: %v", err)
	}
}
