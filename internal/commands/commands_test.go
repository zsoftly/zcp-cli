package commands

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"runtime"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/spf13/cobra"
	"github.com/zsoftly/zcp-cli/internal/config"
	"github.com/zsoftly/zcp-cli/pkg/httpclient"
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

func TestResolveRegionEmptyWhenNeitherSet(t *testing.T) {
	os.Unsetenv("ZCP_REGION")

	result := resolveRegion("")
	if result != "" {
		t.Errorf("resolveRegion = %q, want empty", result)
	}
}

func TestResolveCloudProviderFlagTakesPrecedence(t *testing.T) {
	os.Setenv("ZCP_CLOUD_PROVIDER", "env-cp")
	defer os.Unsetenv("ZCP_CLOUD_PROVIDER")

	result := cloudProviderFlagOrEnv("flag-cp")
	if result != "flag-cp" {
		t.Errorf("resolveCloudProvider = %q, want %q", result, "flag-cp")
	}
}

func TestResolveCloudProviderFallsBackToEnv(t *testing.T) {
	os.Setenv("ZCP_CLOUD_PROVIDER", "env-cp")
	defer os.Unsetenv("ZCP_CLOUD_PROVIDER")

	result := cloudProviderFlagOrEnv("")
	if result != "env-cp" {
		t.Errorf("resolveCloudProvider = %q, want %q", result, "env-cp")
	}
}

func TestResolveCloudProviderEmptyWhenNeitherSet(t *testing.T) {
	os.Unsetenv("ZCP_CLOUD_PROVIDER")

	result := cloudProviderFlagOrEnv("")
	if result != "" {
		t.Errorf("resolveCloudProvider = %q, want empty", result)
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
		{"at negative", []string{"vm", "--at", "-1", "--cloud-provider", "x", "--region", "x", "--billing-cycle", "x", "--plan", "x", "--pseudo-service", "x", "--project", "x"}, "--at must be between 0 and 23"},
		{"at 24", []string{"vm", "--at", "24", "--cloud-provider", "x", "--region", "x", "--billing-cycle", "x", "--plan", "x", "--pseudo-service", "x", "--project", "x"}, "--at must be between 0 and 23"},
		{"immediate 2", []string{"vm", "--immediate", "2", "--cloud-provider", "x", "--region", "x", "--billing-cycle", "x", "--plan", "x", "--pseudo-service", "x", "--project", "x"}, "--immediate must be 0 or 1"},
		{"immediate -1", []string{"vm", "--immediate", "-1", "--cloud-provider", "x", "--region", "x", "--billing-cycle", "x", "--plan", "x", "--pseudo-service", "x", "--project", "x"}, "--immediate must be 0 or 1"},
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
		"--plan", "x", "--pseudo-service", "x", "--project", "x"})

	err := root.Execute()
	if err != nil {
		// Config/API errors are expected, but NOT validation errors
		msg := err.Error()
		if strings.Contains(msg, "--at must be") || strings.Contains(msg, "--immediate must be") {
			t.Errorf("valid values should pass validation, got: %v", err)
		}
	}
}

// ─── BUG 12: instance get transient-routing-error retry loop ────────────────

// routingErrBody is the exact 403 payload the CMP returns when the routing layer
// has not yet indexed a newly-created VM slug.
const routingErrBody = `{"status":"Error","message":"The route virtual-machines/test-vm could not be found."}`

// minVMBody is the minimum valid 200 envelope for a VirtualMachine GET.
const minVMBody = `{"status":"Success","message":"OK","data":{"id":"a1b2c3","vm_id":"vm-1","name":"test-vm","slug":"test-vm","hostname":"test-vm","username":"ubuntu","state":"Running","request_status":true,"is_vnf":false,"has_contract":false,"is_metrics_hidden":false,"is_restricted":false,"has_autoscale":false,"all_time_consumption":0,"created_at":"2026-01-01T00:00:00.000000Z","updated_at":"2026-01-01T00:00:00.000000Z","networks":[]}}`

// withFastRetry overrides instanceGetRetryWait for the duration of a test.
func withFastRetry(t *testing.T) {
	t.Helper()
	orig := instanceGetRetryWait
	instanceGetRetryWait = func(int) time.Duration { return time.Millisecond }
	t.Cleanup(func() { instanceGetRetryWait = orig })
}

// TestInstanceGetRetrySucceeds verifies that instance get retries on transient
// 403 routing errors and succeeds once the server starts returning 200.
func TestInstanceGetRetrySucceeds(t *testing.T) {
	withFastRetry(t)

	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := calls.Add(1)
		if n < 3 {
			w.WriteHeader(http.StatusForbidden)
			fmt.Fprint(w, routingErrBody)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, minVMBody)
	}))
	defer srv.Close()

	os.Setenv("ZCP_BEARER_TOKEN", "test-tok")
	defer os.Unsetenv("ZCP_BEARER_TOKEN")

	stdout, stderr, err := execCmd(t, NewInstanceCmd(), "get", "test-vm", "--api-url", srv.URL)
	if err != nil {
		t.Fatalf("expected success after retries, got: %v\nstdout: %s\nstderr: %s", err, stdout, stderr)
	}
	if calls.Load() != 3 {
		t.Errorf("server called %d times, want 3 (2 routing errors + 1 success)", calls.Load())
	}
	if !strings.Contains(stderr, "routing not ready") {
		t.Errorf("expected retry message in stderr, got: %q", stderr)
	}
}

// TestInstanceGetRetryExhausted verifies that instance get surfaces the error
// after exhausting all 5 retry attempts.
func TestInstanceGetRetryExhausted(t *testing.T) {
	withFastRetry(t)

	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusForbidden)
		fmt.Fprint(w, routingErrBody)
	}))
	defer srv.Close()

	os.Setenv("ZCP_BEARER_TOKEN", "test-tok")
	defer os.Unsetenv("ZCP_BEARER_TOKEN")

	_, _, err := execCmd(t, NewInstanceCmd(), "get", "test-vm", "--api-url", srv.URL)
	if err == nil {
		t.Fatal("expected error after exhausting retries, got nil")
	}
	if !strings.Contains(err.Error(), "instance get") {
		t.Errorf("error = %q, want it to contain 'instance get'", err)
	}
	if calls.Load() != 5 {
		t.Errorf("server called %d times, want 5 (all attempts exhausted)", calls.Load())
	}
}

// TestInstanceGetNonRoutingErrorNoRetry verifies that a non-routing 403 (e.g.
// a plain "forbidden") is returned immediately without any retry.
func TestInstanceGetNonRoutingErrorNoRetry(t *testing.T) {
	withFastRetry(t)

	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusForbidden)
		fmt.Fprint(w, `{"status":"Error","message":"Access denied."}`)
	}))
	defer srv.Close()

	os.Setenv("ZCP_BEARER_TOKEN", "test-tok")
	defer os.Unsetenv("ZCP_BEARER_TOKEN")

	_, _, err := execCmd(t, NewInstanceCmd(), "get", "test-vm", "--api-url", srv.URL)
	if err == nil {
		t.Fatal("expected error for 403 forbidden, got nil")
	}
	if calls.Load() != 1 {
		t.Errorf("server called %d times, want 1 (no retry on non-routing error)", calls.Load())
	}
}

// TestNoShorthandCollisions walks the full command tree and triggers persistent-flag
// merging on every subcommand. A duplicate shorthand (e.g. -y registered locally
// when the root already owns -y for --auto-approve) causes cobra to panic here
// rather than at runtime.
func TestNoShorthandCollisions(t *testing.T) {
	root := newTestRoot()
	root.AddCommand(
		NewACLCmd(),
		NewAffinityGroupCmd(),
		NewAuthCmd(),
		NewAutoscaleCmd(),
		NewBackupCmd(),
		NewBillingCmd(),
		NewBillingCycleCmd(),
		NewCloudProviderCmd(),
		NewCurrencyCmd(),
		NewDashboardCmd(),
		NewDNSCmd(),
		NewEgressCmd(),
		NewFirewallCmd(),
		NewIPCmd(),
		NewISOCmd(),
		NewInstanceCmd(),
		NewKubernetesCmd(),
		NewLoadBalancerCmd(),
		NewMarketplaceCmd(),
		NewMonitoringCmd(),
		NewNetworkCmd(),
		NewObjectStorageCmd(),
		NewPlanCmd(),
		NewPortForwardCmd(),
		NewProductCmd(),
		NewProfileCmd(),
		NewProjectCmd(),
		NewRegionCmd(),
		NewServerCmd(),
		NewSnapshotCmd(),
		NewSSHKeyCmd(),
		NewStorageCategoryCmd(),
		NewStoreCmd(),
		NewSupportCmd(),
		NewTemplateCmd(),
		NewUserProfileCmd(),
		NewVirtualRouterCmd(),
		NewVMBackupCmd(),
		NewVMSnapshotCmd(),
		NewVolumeCmd(),
		NewVPCCmd(),
		NewVPNCmd(),
	)

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("flag shorthand collision in command tree: %v", r)
		}
	}()

	var walk func(*cobra.Command)
	walk = func(c *cobra.Command) {
		c.InheritedFlags() // triggers mergePersistentFlags; panics on shorthand collision
		for _, sub := range c.Commands() {
			walk(sub)
		}
	}
	walk(root)
}

// ─── network create: VPC subnet vs isolated validation ──────────────────────

func networkCreateExec(t *testing.T, args ...string) error {
	t.Helper()
	root := newTestRoot()
	root.AddCommand(NewNetworkCmd())
	root.SetOut(&bytes.Buffer{})
	root.SetErr(&bytes.Buffer{})
	root.SetArgs(append([]string{"network", "create"}, args...))
	return root.Execute()
}

func TestNetworkCreateVPCRequiresBillingCycle(t *testing.T) {
	err := networkCreateExec(t,
		"--name", "web-tier", "--vpc", "my-vpc",
		"--gateway", "10.30.1.1", "--netmask", "255.255.255.0",
		"--cloud-provider", "nimbo", "--region", "yul-1", "--project", "default")
	if err == nil {
		t.Fatal("expected error when --billing-cycle is missing for a VPC subnet")
	}
	if !strings.Contains(err.Error(), "--billing-cycle is required") {
		t.Errorf("error = %q, want '--billing-cycle is required'", err)
	}
}

func TestNetworkCreateVPCRequiresGatewayNetmask(t *testing.T) {
	err := networkCreateExec(t,
		"--name", "web-tier", "--vpc", "my-vpc", "--billing-cycle", "hourly",
		"--cloud-provider", "nimbo", "--region", "yul-1", "--project", "default")
	if err == nil {
		t.Fatal("expected error when --gateway/--netmask are missing for a VPC subnet")
	}
	if !strings.Contains(err.Error(), "--gateway and --netmask are required") {
		t.Errorf("error = %q, want '--gateway and --netmask are required'", err)
	}
}

func TestNetworkCreateVPCRejectsConflictingType(t *testing.T) {
	err := networkCreateExec(t,
		"--name", "web-tier", "--vpc", "my-vpc", "--type", "Isolated",
		"--gateway", "10.30.1.1", "--netmask", "255.255.255.0", "--billing-cycle", "hourly",
		"--cloud-provider", "nimbo", "--region", "yul-1", "--project", "default")
	if err == nil {
		t.Fatal("expected error when --type conflicts with --vpc")
	}
	if !strings.Contains(err.Error(), "--type cannot be combined with --vpc") {
		t.Errorf("error = %q, want '--type cannot be combined with --vpc'", err)
	}
}

func TestNetworkCreateVPCRejectsNetworkPlan(t *testing.T) {
	err := networkCreateExec(t,
		"--name", "web-tier", "--vpc", "my-vpc", "--network-plan", "pnet-yul",
		"--gateway", "10.30.1.1", "--netmask", "255.255.255.0", "--billing-cycle", "hourly",
		"--cloud-provider", "nimbo", "--region", "yul-1", "--project", "default")
	if err == nil {
		t.Fatal("expected error when --network-plan is combined with --vpc")
	}
	if !strings.Contains(err.Error(), "--network-plan cannot be combined with --vpc") {
		t.Errorf("error = %q, want '--network-plan cannot be combined with --vpc'", err)
	}
}

func TestNetworkCreateIsolatedRequiresNetworkPlan(t *testing.T) {
	err := networkCreateExec(t,
		"--name", "my-net",
		"--cloud-provider", "nimbo", "--region", "yow-1", "--project", "default")
	if err == nil {
		t.Fatal("expected error when --network-plan is missing for an isolated network")
	}
	if !strings.Contains(err.Error(), "--network-plan is required") {
		t.Errorf("error = %q, want '--network-plan is required'", err)
	}
}

func TestNetworkCreateRejectsUnknownType(t *testing.T) {
	err := networkCreateExec(t,
		"--name", "my-net", "--type", "Shared", "--network-plan", "inet-yow",
		"--cloud-provider", "nimbo", "--region", "yow-1", "--project", "default")
	if err == nil {
		t.Fatal("expected error for unknown --type")
	}
	if !strings.Contains(err.Error(), "--type must be Isolated or L2") {
		t.Errorf("error = %q, want '--type must be Isolated or L2'", err)
	}
}

// ─── acl create-rule validation ──────────────────────────────────────────────

func aclCreateRuleExec(t *testing.T, args ...string) error {
	t.Helper()
	root := newTestRoot()
	root.AddCommand(NewACLCmd())
	root.SetOut(&bytes.Buffer{})
	root.SetErr(&bytes.Buffer{})
	root.SetArgs(append([]string{"acl", "create-rule", "my-vpc", "web-acl"}, args...))
	return root.Execute()
}

func TestACLCreateRuleTCPRequiresPorts(t *testing.T) {
	err := aclCreateRuleExec(t, "--protocol", "tcp", "--cidr", "0.0.0.0/0")
	if err == nil {
		t.Fatal("expected error when tcp rule has no ports")
	}
	if !strings.Contains(err.Error(), "--start-port and --end-port are required") {
		t.Errorf("error = %q, want port requirement message", err)
	}
}

func TestACLCreateRuleRejectsBadProtocol(t *testing.T) {
	err := aclCreateRuleExec(t, "--protocol", "gre", "--cidr", "0.0.0.0/0")
	if err == nil {
		t.Fatal("expected error for unknown protocol")
	}
	if !strings.Contains(err.Error(), "--protocol must be") {
		t.Errorf("error = %q, want protocol validation message", err)
	}
}

func TestACLCreateRuleRequiresCIDR(t *testing.T) {
	err := aclCreateRuleExec(t, "--protocol", "all")
	if err == nil {
		t.Fatal("expected error when --cidr is missing")
	}
	if !strings.Contains(err.Error(), "--cidr is required") {
		t.Errorf("error = %q, want '--cidr is required'", err)
	}
}

func TestACLCreateRuleRejectsInvalidPortValues(t *testing.T) {
	err := aclCreateRuleExec(t, "--protocol", "tcp", "--cidr", "0.0.0.0/0",
		"--start-port", "0", "--end-port", "70000")
	if err == nil {
		t.Fatal("expected error for out-of-range ports")
	}
	if !strings.Contains(err.Error(), "ports must be between 1 and 65535") {
		t.Errorf("error = %q, want port range message", err)
	}
}

func TestACLCreateRuleRejectsInvertedPortRange(t *testing.T) {
	err := aclCreateRuleExec(t, "--protocol", "tcp", "--cidr", "0.0.0.0/0",
		"--start-port", "443", "--end-port", "80")
	if err == nil {
		t.Fatal("expected error when end-port < start-port")
	}
	if !strings.Contains(err.Error(), "must not be lower than") {
		t.Errorf("error = %q, want inverted-range message", err)
	}
}

// ─── Cloud provider auto-detection (Option A) ───────────────────────────────

// providersServer returns an httptest server that serves the given JSON array
// (the `data` payload) at /cloud-providers.
func providersServer(t *testing.T, dataJSON string) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/cloud-providers" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"status":"Success","message":"OK","data":%s}`, dataJSON)
	}))
	t.Cleanup(srv.Close)
	return srv
}

// isolateConfigDir points config Save/Load at a temp dir on every platform.
// Windows resolves the config path from APPDATA (XDG_CONFIG_HOME is ignored), so
// setting only XDG would leak writes into the real user config on Windows.
func isolateConfigDir(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	if runtime.GOOS == "windows" {
		t.Setenv("APPDATA", dir)
	} else {
		t.Setenv("XDG_CONFIG_HOME", dir)
	}
}

func TestDetectCloudProviderSingleActivePersists(t *testing.T) {
	isolateConfigDir(t)
	cfg := &config.Config{
		ActiveProfile: "default",
		Profiles:      map[string]config.Profile{"default": {Name: "default", BearerToken: "t"}},
	}
	if err := config.Save(cfg); err != nil {
		t.Fatalf("save: %v", err)
	}
	// One active provider, one inactive — only the active one counts.
	srv := providersServer(t, `[{"slug":"nimbo","status":true},{"slug":"ceph","status":false}]`)
	client := httpclient.New(httpclient.Options{BaseURL: srv.URL, BearerToken: "t", Timeout: 5 * time.Second})

	slug, err := detectCloudProvider(context.Background(), client, cfg, "default")
	if err != nil {
		t.Fatalf("detectCloudProvider error: %v", err)
	}
	if slug != "nimbo" {
		t.Fatalf("slug = %q, want nimbo", slug)
	}
	got, err := config.Load()
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	if got.Profiles["default"].CloudProvider != "nimbo" {
		t.Fatalf("persisted CloudProvider = %q, want nimbo", got.Profiles["default"].CloudProvider)
	}
}

// Mirrors the real production catalog: ceph (Object Storage), dns (Dns Domain),
// nimbo (Virtual Machine + infra). The compute provider must be picked by the
// "Virtual Machine" service regardless of catalog order.
func TestDetectCloudProviderPicksComputeProvider(t *testing.T) {
	isolateConfigDir(t)
	cfg := &config.Config{
		ActiveProfile: "default",
		Profiles:      map[string]config.Profile{"default": {Name: "default", BearerToken: "t"}},
	}
	if err := config.Save(cfg); err != nil {
		t.Fatalf("save: %v", err)
	}
	srv := providersServer(t, `[
		{"slug":"ceph","status":true,"services":["Object Storage"]},
		{"slug":"dns","status":true,"services":["Dns Domain"]},
		{"slug":"nimbo","status":true,"services":["Block Storage","Virtual Machine","VPC"]}
	]`)
	client := httpclient.New(httpclient.Options{BaseURL: srv.URL, BearerToken: "t", Timeout: 5 * time.Second})

	slug, err := detectCloudProvider(context.Background(), client, cfg, "default")
	if err != nil {
		t.Fatalf("detectCloudProvider error: %v", err)
	}
	if slug != "nimbo" {
		t.Fatalf("slug = %q, want nimbo (the Virtual Machine provider)", slug)
	}
	got, _ := config.Load()
	if got.Profiles["default"].CloudProvider != "nimbo" {
		t.Fatalf("persisted CloudProvider = %q, want nimbo", got.Profiles["default"].CloudProvider)
	}
}

// When several providers are active but none advertises compute, do not guess.
func TestDetectCloudProviderAmbiguousNoComputeDoesNotPersist(t *testing.T) {
	isolateConfigDir(t)
	cfg := &config.Config{
		ActiveProfile: "default",
		Profiles:      map[string]config.Profile{"default": {Name: "default", BearerToken: "t"}},
	}
	if err := config.Save(cfg); err != nil {
		t.Fatalf("save: %v", err)
	}
	srv := providersServer(t, `[
		{"slug":"ceph","status":true,"services":["Object Storage"]},
		{"slug":"dns","status":true,"services":["Dns Domain"]}
	]`)
	client := httpclient.New(httpclient.Options{BaseURL: srv.URL, BearerToken: "t", Timeout: 5 * time.Second})

	slug, err := detectCloudProvider(context.Background(), client, cfg, "default")
	if err != nil {
		t.Fatalf("detectCloudProvider error: %v", err)
	}
	if slug != "" {
		t.Fatalf("slug = %q, want empty (no compute provider, must not guess)", slug)
	}
	got, _ := config.Load()
	if got.Profiles["default"].CloudProvider != "" {
		t.Fatalf("CloudProvider should not be persisted, got %q", got.Profiles["default"].CloudProvider)
	}
}

func TestResolveCloudProviderFallsBackToProfile(t *testing.T) {
	t.Setenv("ZCP_CLOUD_PROVIDER", "")
	t.Setenv("ZCP_BEARER_TOKEN", "")
	t.Setenv("ZCP_PROFILE", "")
	isolateConfigDir(t)
	cfg := &config.Config{
		ActiveProfile: "default",
		Profiles:      map[string]config.Profile{"default": {Name: "default", BearerToken: "t", CloudProvider: "nimbo"}},
	}
	if err := config.Save(cfg); err != nil {
		t.Fatalf("save: %v", err)
	}
	root := newTestRoot()

	if got := resolveCloudProvider(root, ""); got != "nimbo" {
		t.Errorf("resolveCloudProvider with stored profile = %q, want nimbo", got)
	}
	if got := resolveCloudProvider(root, "override"); got != "override" {
		t.Errorf("explicit flag must win, got %q", got)
	}
}

// ─── object-storage command-layer validation (fails before any API call) ────

func TestParseKVTags(t *testing.T) {
	m, err := parseKVTags([]string{"env=prod", "team=data", "blank="})
	if err != nil {
		t.Fatalf("parseKVTags error = %v", err)
	}
	if m["env"] != "prod" || m["team"] != "data" {
		t.Errorf("parsed = %v, want env=prod team=data", m)
	}
	if v, ok := m["blank"]; !ok || v != "" {
		t.Errorf("empty value should be allowed, got %v ok=%v", v, ok)
	}
	if _, err := parseKVTags([]string{"noequals"}); err == nil {
		t.Error("expected error for a tag without '='")
	}
	if _, err := parseKVTags([]string{"=v"}); err == nil {
		t.Error("expected error for an empty key")
	}
}

// runOS executes the full object-storage command tree with the given args and
// returns the error. Used to exercise flag validation that runs before any
// network call.
func runOS(args ...string) error {
	root := newTestRoot()
	root.AddCommand(NewObjectStorageCmd())
	root.SetOut(&bytes.Buffer{})
	root.SetErr(&bytes.Buffer{})
	root.SetArgs(append([]string{"object-storage"}, args...))
	return root.Execute()
}

func TestOSFlagValidation(t *testing.T) {
	cases := []struct {
		name string
		args []string
		want string
	}{
		{"lifecycle expire requires --days", []string{"bucket", "lifecycle", "expire", "s", "b"}, "--days"},
		{"object url rejects --expires 0", []string{"object", "url", "s", "b", "k", "--expires", "0"}, "--expires"},
		{"object url rejects > 7 days", []string{"object", "url", "s", "b", "k", "--expires", "200h"}, "--expires"},
		{"object put-url rejects --expires 0", []string{"object", "put-url", "s", "b", "k", "--expires", "0"}, "--expires"},
		{"lifecycle expire requires a duration flag", []string{"bucket", "lifecycle", "expire", "s", "b"}, "at least one"},
		{"set-acl rejects bad value", []string{"bucket", "set-acl", "s", "b", "--acl", "bogus"}, "unsupported"},
		{"cors set requires origin+method", []string{"bucket", "cors", "set", "s", "b", "--origin", "*"}, "--method"},
		{"bucket tag set requires --tag", []string{"bucket", "tag", "set", "s", "b"}, "--tag"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := runOS(c.args...)
			if err == nil {
				t.Fatalf("expected error, got nil")
			}
			if !strings.Contains(err.Error(), c.want) {
				t.Errorf("error = %q, want it to mention %q", err.Error(), c.want)
			}
		})
	}
}
