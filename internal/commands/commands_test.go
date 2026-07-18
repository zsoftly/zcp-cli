package commands

import (
	"bytes"
	"context"
	"encoding/json"
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
		"--cloud-provider", "nimbo", "--region", "noida", "--project", "default-9",
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

// minVMListBody is the minimum valid 200 envelope for resolving a VM reference.
const minVMListBody = `{"status":"Success","message":"OK","data":[{"id":"a1b2c3","vm_id":"vm-1","name":"test-vm","slug":"test-vm","hostname":"test-vm","username":"ubuntu","state":"Running","request_status":true,"is_vnf":false,"has_contract":false,"is_metrics_hidden":false,"is_restricted":false,"has_autoscale":false,"all_time_consumption":0,"created_at":"2026-01-01T00:00:00.000000Z","updated_at":"2026-01-01T00:00:00.000000Z","networks":[]}]}`

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
		if r.Method == http.MethodGet && r.URL.Path == "/virtual-machines" {
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, minVMListBody)
			return
		}
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
		if r.Method == http.MethodGet && r.URL.Path == "/virtual-machines" {
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, minVMListBody)
			return
		}
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

// vmListWithBillingCycle is a single-VM list whose billing cycle is hourly, used to
// verify the delete command derives billing_cycle for the service-cancel request.
const vmListWithBillingCycle = `{"status":"Success","message":"OK","data":[{"id":"a1b2c3","vm_id":"vm-1","name":"test-vm","slug":"test-vm","hostname":"test-vm","username":"ubuntu","state":"Running","request_status":true,"billing_cycle":{"id":"bc1","name":"Hourly","slug":"hourly","duration":1,"unit":"hour"},"is_vnf":false,"has_contract":false,"is_metrics_hidden":false,"is_restricted":false,"has_autoscale":false,"all_time_consumption":0,"created_at":"2026-01-01T00:00:00.000000Z","updated_at":"2026-01-01T00:00:00.000000Z","networks":[]}]}`

// TestInstanceDeleteUsesServiceCancel verifies that `zcp instance delete` routes through
// the unified service-cancellation endpoint (which releases the auto-assigned public IP),
// NOT the direct DELETE endpoint (which ignores delete_public_ip). Regression test for the
// public-IP-leak bug.
func TestInstanceDeleteUsesServiceCancel(t *testing.T) {
	var cancelBody map[string]interface{}
	var cancelPath string
	var directDelete atomic.Bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/virtual-machines":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, vmListWithBillingCycle)
		case r.Method == http.MethodPost && strings.HasPrefix(r.URL.Path, "/billing/service-cancel-requests/"):
			cancelPath = r.URL.Path
			json.NewDecoder(r.Body).Decode(&cancelBody)
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"status":"Success","message":"Cancellation scheduled","data":null}`)
		case r.Method == http.MethodDelete && r.URL.Path == "/virtual-machines/test-vm":
			directDelete.Store(true)
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"status":"Success","message":"deleted","data":null}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	os.Setenv("ZCP_BEARER_TOKEN", "test-tok")
	defer os.Unsetenv("ZCP_BEARER_TOKEN")

	stdout, stderr, err := execCmd(t, NewInstanceCmd(), "delete", "test-vm", "--yes", "--api-url", srv.URL)
	if err != nil {
		t.Fatalf("delete error = %v\nstdout: %s\nstderr: %s", err, stdout, stderr)
	}
	if directDelete.Load() {
		t.Error("delete called the direct DELETE /virtual-machines endpoint (leaks public IP); want service-cancel")
	}
	if cancelPath != "/billing/service-cancel-requests/test-vm" {
		t.Errorf("cancel path = %q, want %q", cancelPath, "/billing/service-cancel-requests/test-vm")
	}
	if cancelBody["service_name"] != "Virtual Machine" {
		t.Errorf("service_name = %v, want %q", cancelBody["service_name"], "Virtual Machine")
	}
	if cancelBody["type"] != "Immediate" {
		t.Errorf("type = %v, want %q", cancelBody["type"], "Immediate")
	}
	if cancelBody["delete_public_ip"] != true {
		t.Errorf("delete_public_ip = %v, want true (public IP must be released)", cancelBody["delete_public_ip"])
	}
	if cancelBody["billing_cycle"] != "hour" {
		t.Errorf("billing_cycle = %v, want %q (derived from VM's hourly cycle)", cancelBody["billing_cycle"], "hour")
	}
}

// TestInstanceDeleteRetainPublicIP verifies --delete-public-ip=false sends
// delete_public_ip:false so the auto-assigned IP is kept.
func TestInstanceDeleteRetainPublicIP(t *testing.T) {
	var cancelBody map[string]interface{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/virtual-machines":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, vmListWithBillingCycle)
		case r.Method == http.MethodPost && strings.HasPrefix(r.URL.Path, "/billing/service-cancel-requests/"):
			json.NewDecoder(r.Body).Decode(&cancelBody)
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"status":"Success","message":"Cancellation scheduled","data":null}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	os.Setenv("ZCP_BEARER_TOKEN", "test-tok")
	defer os.Unsetenv("ZCP_BEARER_TOKEN")

	_, _, err := execCmd(t, NewInstanceCmd(), "delete", "test-vm", "--yes", "--delete-public-ip=false", "--api-url", srv.URL)
	if err != nil {
		t.Fatalf("delete error = %v", err)
	}
	if cancelBody["delete_public_ip"] != false {
		t.Errorf("delete_public_ip = %v, want false", cancelBody["delete_public_ip"])
	}
}

// TestInstanceDeleteForceIsNoOp verifies the deprecated --force flag is accepted but
// does not change the endpoint used (still service-cancel, not a direct expunge).
func TestInstanceDeleteForceIsNoOp(t *testing.T) {
	var sawCancel atomic.Bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/virtual-machines":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, vmListWithBillingCycle)
		case r.Method == http.MethodPost && strings.HasPrefix(r.URL.Path, "/billing/service-cancel-requests/"):
			sawCancel.Store(true)
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"status":"Success","message":"Cancellation scheduled","data":null}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	os.Setenv("ZCP_BEARER_TOKEN", "test-tok")
	defer os.Unsetenv("ZCP_BEARER_TOKEN")

	if _, _, err := execCmd(t, NewInstanceCmd(), "delete", "test-vm", "--yes", "--force", "--api-url", srv.URL); err != nil {
		t.Fatalf("delete --force error = %v", err)
	}
	if !sawCancel.Load() {
		t.Error("--force should still route through service-cancel")
	}
}

// withFastLBRelease overrides lbIPReleaseWait so --release-ip retries don't sleep.
func withFastLBRelease(t *testing.T) {
	t.Helper()
	orig := lbIPReleaseWait
	lbIPReleaseWait = func(int) time.Duration { return time.Millisecond }
	t.Cleanup(func() { lbIPReleaseWait = orig })
}

// TestLoadBalancerDeleteUsesServiceCancel verifies `zcp loadbalancer delete` routes through
// the service-cancel endpoint (matching the CMP Web UI) instead of the direct DELETE, does
// not send delete_public_ip (which the LB workflow does not honor), and by default does NOT
// touch the LB's public IP (a reusable resource).
func TestLoadBalancerDeleteUsesServiceCancel(t *testing.T) {
	var cancelBody map[string]interface{}
	var cancelPath string
	var directDelete, ipReleased atomic.Bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && strings.HasPrefix(r.URL.Path, "/billing/service-cancel-requests/"):
			cancelPath = r.URL.Path
			json.NewDecoder(r.Body).Decode(&cancelBody)
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"status":"Success","message":"Cancellation scheduled","data":null}`)
		case r.Method == http.MethodDelete && r.URL.Path == "/load-balancers/my-lb":
			directDelete.Store(true)
			fmt.Fprint(w, `{"status":"Success","data":null}`)
		case r.Method == http.MethodDelete && strings.HasPrefix(r.URL.Path, "/ipaddresses/"):
			ipReleased.Store(true)
			fmt.Fprint(w, `{"status":"Success","data":null}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	os.Setenv("ZCP_BEARER_TOKEN", "test-tok")
	defer os.Unsetenv("ZCP_BEARER_TOKEN")

	stdout, stderr, err := execCmd(t, NewLoadBalancerCmd(), "delete", "my-lb", "--yes", "--api-url", srv.URL)
	if err != nil {
		t.Fatalf("lb delete error = %v\nstdout: %s\nstderr: %s", err, stdout, stderr)
	}
	if directDelete.Load() {
		t.Error("lb delete called the direct DELETE /load-balancers endpoint; want service-cancel")
	}
	if ipReleased.Load() {
		t.Error("lb delete released the public IP without --release-ip")
	}
	if cancelPath != "/billing/service-cancel-requests/my-lb" {
		t.Errorf("cancel path = %q, want %q", cancelPath, "/billing/service-cancel-requests/my-lb")
	}
	if cancelBody["service_name"] != "Load Balancer" {
		t.Errorf("service_name = %v, want %q", cancelBody["service_name"], "Load Balancer")
	}
	if _, ok := cancelBody["delete_public_ip"]; ok {
		t.Errorf("delete_public_ip should be absent for LB cancel, got %v", cancelBody["delete_public_ip"])
	}
	if cancelBody["billing_cycle"] != "hour" {
		t.Errorf("billing_cycle = %v, want %q (default hourly)", cancelBody["billing_cycle"], "hour")
	}
}

// TestLoadBalancerDeleteReleaseIP verifies --release-ip releases the LB's dedicated public
// IP after the cancel, and --billing-cycle monthly maps to "month". The IP's strategy is
// left empty on purpose: an attached (non-source-NAT) LB IP reports an empty strategy in
// practice, and the release gate is "not SOURCE-NAT", not literally "STATIC".
func TestLoadBalancerDeleteReleaseIP(t *testing.T) {
	withFastLBRelease(t)
	var cancelBody map[string]interface{}
	var releasedPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/load-balancers":
			fmt.Fprint(w, `{"status":"Success","data":[{"slug":"my-lb","name":"my-lb","id":"lb-1","ipaddress":{"id":"ipid","ip_address":"1.2.3.4","slug":"ip-1"}}]}`)
		case r.Method == http.MethodGet && r.URL.Path == "/ipaddresses":
			fmt.Fprint(w, `{"status":"Success","data":[{"slug":"ip-1","ipaddress":"1.2.3.4","strategy":""}]}`)
		case r.Method == http.MethodPost && strings.HasPrefix(r.URL.Path, "/billing/service-cancel-requests/"):
			json.NewDecoder(r.Body).Decode(&cancelBody)
			fmt.Fprint(w, `{"status":"Success","data":null}`)
		case r.Method == http.MethodDelete && strings.HasPrefix(r.URL.Path, "/ipaddresses/"):
			releasedPath = r.URL.Path
			fmt.Fprint(w, `{"status":"Success","data":null}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	os.Setenv("ZCP_BEARER_TOKEN", "test-tok")
	defer os.Unsetenv("ZCP_BEARER_TOKEN")

	stdout, stderr, err := execCmd(t, NewLoadBalancerCmd(), "delete", "my-lb", "--yes", "--release-ip", "--billing-cycle", "monthly", "--api-url", srv.URL)
	if err != nil {
		t.Fatalf("lb delete --release-ip error = %v\nstdout: %s\nstderr: %s", err, stdout, stderr)
	}
	if releasedPath != "/ipaddresses/ip-1" {
		t.Errorf("released IP path = %q, want %q", releasedPath, "/ipaddresses/ip-1")
	}
	if cancelBody["billing_cycle"] != "month" {
		t.Errorf("billing_cycle = %v, want %q", cancelBody["billing_cycle"], "month")
	}
}

// TestLoadBalancerDeleteReleaseIPSkipsSourceNAT verifies --release-ip never releases a
// network source-NAT IP (which would break the network).
func TestLoadBalancerDeleteReleaseIPSkipsSourceNAT(t *testing.T) {
	withFastLBRelease(t)
	var ipReleased atomic.Bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/load-balancers":
			fmt.Fprint(w, `{"status":"Success","data":[{"slug":"my-lb","name":"my-lb","id":"lb-1","ipaddress":{"id":"ipid","ip_address":"1.2.3.4","slug":"ip-1"}}]}`)
		case r.Method == http.MethodGet && r.URL.Path == "/ipaddresses":
			fmt.Fprint(w, `{"status":"Success","data":[{"slug":"ip-1","ipaddress":"1.2.3.4","strategy":"SOURCE-NAT"}]}`)
		case r.Method == http.MethodPost && strings.HasPrefix(r.URL.Path, "/billing/service-cancel-requests/"):
			fmt.Fprint(w, `{"status":"Success","data":null}`)
		case r.Method == http.MethodDelete && strings.HasPrefix(r.URL.Path, "/ipaddresses/"):
			ipReleased.Store(true)
			fmt.Fprint(w, `{"status":"Success","data":null}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	os.Setenv("ZCP_BEARER_TOKEN", "test-tok")
	defer os.Unsetenv("ZCP_BEARER_TOKEN")

	if _, _, err := execCmd(t, NewLoadBalancerCmd(), "delete", "my-lb", "--yes", "--release-ip", "--api-url", srv.URL); err != nil {
		t.Fatalf("lb delete error = %v", err)
	}
	// The critical safety property: a SOURCE-NAT IP is never released (it belongs to the
	// network). The skip reason is printed to os.Stderr, which execCmd does not capture.
	if ipReleased.Load() {
		t.Error("--release-ip released a SOURCE-NAT IP; it must never touch the network source-NAT")
	}
}

// TestLoadBalancerDeleteReleaseIPResolvesName verifies that when a NAME (not a slug) is
// passed with --release-ip, the cancel targets the LB's canonical SLUG (not the raw name),
// so the deletion doesn't 404 and the IP is still released.
func TestLoadBalancerDeleteReleaseIPResolvesName(t *testing.T) {
	withFastLBRelease(t)
	var cancelPath, releasedPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/load-balancers":
			fmt.Fprint(w, `{"status":"Success","data":[{"slug":"lb-canonical","name":"my-lb-name","id":"lb-1","ipaddress":{"id":"ipid","ip_address":"1.2.3.4","slug":"ip-1"}}]}`)
		case r.Method == http.MethodGet && r.URL.Path == "/ipaddresses":
			fmt.Fprint(w, `{"status":"Success","data":[{"slug":"ip-1","ipaddress":"1.2.3.4","strategy":"STATIC"}]}`)
		case r.Method == http.MethodPost && strings.HasPrefix(r.URL.Path, "/billing/service-cancel-requests/"):
			cancelPath = r.URL.Path
			fmt.Fprint(w, `{"status":"Success","data":null}`)
		case r.Method == http.MethodDelete && strings.HasPrefix(r.URL.Path, "/ipaddresses/"):
			releasedPath = r.URL.Path
			fmt.Fprint(w, `{"status":"Success","data":null}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	os.Setenv("ZCP_BEARER_TOKEN", "test-tok")
	defer os.Unsetenv("ZCP_BEARER_TOKEN")

	// Pass the NAME, not the slug.
	if _, _, err := execCmd(t, NewLoadBalancerCmd(), "delete", "my-lb-name", "--yes", "--release-ip", "--api-url", srv.URL); err != nil {
		t.Fatalf("lb delete error = %v", err)
	}
	if cancelPath != "/billing/service-cancel-requests/lb-canonical" {
		t.Errorf("cancel path = %q, want the canonical slug %q (not the name)", cancelPath, "/billing/service-cancel-requests/lb-canonical")
	}
	if releasedPath != "/ipaddresses/ip-1" {
		t.Errorf("released IP path = %q, want %q", releasedPath, "/ipaddresses/ip-1")
	}
}

// TestLoadBalancerDeleteReleaseIPNotFound verifies that when --release-ip can't find the LB
// in the listing (wrong ref, pagination, or already gone), it still deletes via the raw ref
// and does not attempt to release any IP (no silent wrong release, no crash).
func TestLoadBalancerDeleteReleaseIPNotFound(t *testing.T) {
	withFastLBRelease(t)
	var cancelPath string
	var ipReleased atomic.Bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/load-balancers":
			fmt.Fprint(w, `{"status":"Success","data":[]}`)
		case r.Method == http.MethodPost && strings.HasPrefix(r.URL.Path, "/billing/service-cancel-requests/"):
			cancelPath = r.URL.Path
			fmt.Fprint(w, `{"status":"Success","data":null}`)
		case r.Method == http.MethodDelete && strings.HasPrefix(r.URL.Path, "/ipaddresses/"):
			ipReleased.Store(true)
			fmt.Fprint(w, `{"status":"Success","data":null}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	os.Setenv("ZCP_BEARER_TOKEN", "test-tok")
	defer os.Unsetenv("ZCP_BEARER_TOKEN")

	if _, _, err := execCmd(t, NewLoadBalancerCmd(), "delete", "some-lb", "--yes", "--release-ip", "--api-url", srv.URL); err != nil {
		t.Fatalf("lb delete error = %v", err)
	}
	if cancelPath != "/billing/service-cancel-requests/some-lb" {
		t.Errorf("cancel path = %q, want the raw ref %q (deletion must still proceed)", cancelPath, "/billing/service-cancel-requests/some-lb")
	}
	if ipReleased.Load() {
		t.Error("released an IP despite not resolving the LB; must never release an unconfirmed IP")
	}
}

// TestLoadBalancerDeleteRejectsBadBillingCycle verifies an unknown --billing-cycle is
// rejected up front instead of being silently coerced to "month".
func TestLoadBalancerDeleteRejectsBadBillingCycle(t *testing.T) {
	var hit atomic.Bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hit.Store(true)
		http.NotFound(w, r)
	}))
	defer srv.Close()

	os.Setenv("ZCP_BEARER_TOKEN", "test-tok")
	defer os.Unsetenv("ZCP_BEARER_TOKEN")

	_, _, err := execCmd(t, NewLoadBalancerCmd(), "delete", "my-lb", "--yes", "--billing-cycle", "weekly", "--api-url", srv.URL)
	if err == nil {
		t.Fatal("expected an error for an invalid --billing-cycle, got nil")
	}
	if !strings.Contains(err.Error(), "invalid --billing-cycle") {
		t.Errorf("error = %q, want it to mention 'invalid --billing-cycle'", err)
	}
	if hit.Load() {
		t.Error("a bad --billing-cycle should be rejected before any API call")
	}
}

// TestBillingCycleUnit locks the exact-match normalization (no loose prefix matching).
func TestBillingCycleUnit(t *testing.T) {
	cases := []struct {
		in   string
		want string
		ok   bool
	}{
		{"hour", "hour", true}, {"hourly", "hour", true}, {"Hourly", "hour", true},
		{"month", "month", true}, {"monthly", "month", true}, {"MONTHLY", "month", true},
		{"hourlyx", "", false}, {"hourfoo", "", false}, {"weekly", "", false}, {"", "", false},
	}
	for _, c := range cases {
		if got, ok := billingCycleUnit(c.in); got != c.want || ok != c.ok {
			t.Errorf("billingCycleUnit(%q) = (%q,%v), want (%q,%v)", c.in, got, ok, c.want, c.ok)
		}
	}
}

// vmListNoBillingCycle is a single-VM list with no billing cycle metadata.
const vmListNoBillingCycle = `{"status":"Success","message":"OK","data":[{"id":"a1","vm_id":"vm-1","name":"nc-vm","slug":"nc-vm","state":"Running","request_status":true,"networks":[]}]}`

// TestInstanceDeleteRequiresBillingCycle verifies delete fails (and submits no cancel) when
// the VM's billing cycle can't be determined and no --billing-cycle override is given.
func TestInstanceDeleteRequiresBillingCycle(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == "/virtual-machines" {
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, vmListNoBillingCycle)
			return
		}
		if r.Method == http.MethodPost {
			t.Error("cancel must not be submitted when the billing cycle can't be determined")
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()
	os.Setenv("ZCP_BEARER_TOKEN", "test-tok")
	defer os.Unsetenv("ZCP_BEARER_TOKEN")

	_, _, err := execCmd(t, NewInstanceCmd(), "delete", "nc-vm", "--yes", "--api-url", srv.URL)
	if err == nil || !strings.Contains(err.Error(), "could not determine the billing cycle") {
		t.Fatalf("want 'could not determine the billing cycle' error, got %v", err)
	}
}

// TestInstanceDeleteBillingCycleOverride verifies --billing-cycle is used when the VM has none.
func TestInstanceDeleteBillingCycleOverride(t *testing.T) {
	var cancelBody map[string]interface{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/virtual-machines":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, vmListNoBillingCycle)
		case r.Method == http.MethodPost && strings.HasPrefix(r.URL.Path, "/billing/service-cancel-requests/"):
			json.NewDecoder(r.Body).Decode(&cancelBody)
			fmt.Fprint(w, `{"status":"Success","data":null}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	os.Setenv("ZCP_BEARER_TOKEN", "test-tok")
	defer os.Unsetenv("ZCP_BEARER_TOKEN")

	if _, _, err := execCmd(t, NewInstanceCmd(), "delete", "nc-vm", "--yes", "--billing-cycle", "monthly", "--api-url", srv.URL); err != nil {
		t.Fatalf("delete with --billing-cycle override error = %v", err)
	}
	if cancelBody["billing_cycle"] != "month" {
		t.Errorf("billing_cycle = %v, want %q", cancelBody["billing_cycle"], "month")
	}
}

// vmListOfferingBillingCycle is a single-VM list whose billing cycle exists only under
// offering.billing_cycle (top-level billing_cycle absent), as get-shaped responses do.
const vmListOfferingBillingCycle = `{"status":"Success","message":"OK","data":[{"id":"a1","vm_id":"vm-1","name":"off-vm","slug":"off-vm","state":"Running","request_status":true,"offering":{"id":"o1","billing_cycle":{"id":"bc1","name":"Monthly","slug":"monthly","duration":1,"unit":"month"}},"networks":[]}]}`

// TestInstanceDeleteDerivesBillingCycleFromOffering verifies the cancel cycle is taken from
// offering.billing_cycle when the top-level billing_cycle is absent, so the delete succeeds.
func TestInstanceDeleteDerivesBillingCycleFromOffering(t *testing.T) {
	var cancelBody map[string]interface{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/virtual-machines":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, vmListOfferingBillingCycle)
		case r.Method == http.MethodPost && strings.HasPrefix(r.URL.Path, "/billing/service-cancel-requests/"):
			json.NewDecoder(r.Body).Decode(&cancelBody)
			fmt.Fprint(w, `{"status":"Success","data":null}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	os.Setenv("ZCP_BEARER_TOKEN", "test-tok")
	defer os.Unsetenv("ZCP_BEARER_TOKEN")

	if _, _, err := execCmd(t, NewInstanceCmd(), "delete", "off-vm", "--yes", "--api-url", srv.URL); err != nil {
		t.Fatalf("delete should succeed using offering.billing_cycle, got %v", err)
	}
	if cancelBody["billing_cycle"] != "month" {
		t.Errorf("billing_cycle = %v, want %q (from offering)", cancelBody["billing_cycle"], "month")
	}
}

// TestLoadBalancerDeleteReleaseIPAmbiguousName verifies a name matching >1 LB is an error
// (never silently cancel the wrong one).
func TestLoadBalancerDeleteReleaseIPAmbiguousName(t *testing.T) {
	withFastLBRelease(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == "/load-balancers" {
			fmt.Fprint(w, `{"status":"Success","data":[{"slug":"lb-1","name":"dup","id":"1"},{"slug":"lb-2","name":"dup","id":"2"}]}`)
			return
		}
		if r.Method == http.MethodPost {
			t.Error("must not cancel when the name is ambiguous")
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()
	os.Setenv("ZCP_BEARER_TOKEN", "test-tok")
	defer os.Unsetenv("ZCP_BEARER_TOKEN")

	_, _, err := execCmd(t, NewLoadBalancerCmd(), "delete", "dup", "--yes", "--release-ip", "--api-url", srv.URL)
	if err == nil || !strings.Contains(err.Error(), "named") {
		t.Fatalf("want ambiguous-name error, got %v", err)
	}
}

// TestLoadBalancerDeleteReleaseIPFailsLoud verifies that when --release-ip's IP release
// exhausts its retries, the command exits with an error (not silently 0).
func TestLoadBalancerDeleteReleaseIPFailsLoud(t *testing.T) {
	withFastLBRelease(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/load-balancers":
			fmt.Fprint(w, `{"status":"Success","data":[{"slug":"my-lb","name":"my-lb","id":"lb-1","ipaddress":{"id":"ipid","ip_address":"1.2.3.4","slug":"ip-1"}}]}`)
		case r.Method == http.MethodGet && r.URL.Path == "/ipaddresses":
			fmt.Fprint(w, `{"status":"Success","data":[{"slug":"ip-1","ipaddress":"1.2.3.4","strategy":""}]}`)
		case r.Method == http.MethodPost && strings.HasPrefix(r.URL.Path, "/billing/service-cancel-requests/"):
			fmt.Fprint(w, `{"status":"Success","data":null}`)
		case r.Method == http.MethodDelete && strings.HasPrefix(r.URL.Path, "/ipaddresses/"):
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprint(w, `{"status":"Error","message":"boom"}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	os.Setenv("ZCP_BEARER_TOKEN", "test-tok")
	defer os.Unsetenv("ZCP_BEARER_TOKEN")

	_, _, err := execCmd(t, NewLoadBalancerCmd(), "delete", "my-lb", "--yes", "--release-ip", "--api-url", srv.URL)
	if err == nil || !strings.Contains(err.Error(), "could not be released") {
		t.Fatalf("want a non-nil IP-release-failed error, got %v", err)
	}
}

// TestInstanceGetNonRoutingErrorNoRetry verifies that a non-routing 403 (e.g.
// a plain "forbidden") is returned immediately without any retry.
func TestInstanceGetNonRoutingErrorNoRetry(t *testing.T) {
	withFastRetry(t)

	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == "/virtual-machines" {
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, minVMListBody)
			return
		}
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

func TestInstanceRebootRejectsNonRunningVM(t *testing.T) {
	var rebootCalled atomic.Bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/virtual-machines":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"status":"Success","message":"OK","data":[{"id":"vm-1","name":"test-vm","slug":"test-vm","state":"Stopped","request_status":true,"is_vnf":false,"has_contract":false,"is_metrics_hidden":false,"is_restricted":false,"has_autoscale":false,"all_time_consumption":0,"created_at":"2026-01-01T00:00:00.000000Z","updated_at":"2026-01-01T00:00:00.000000Z"}]}`)
		case r.Method == http.MethodGet && r.URL.Path == "/virtual-machines/test-vm":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"status":"Success","message":"OK","data":{"id":"vm-1","name":"test-vm","slug":"test-vm","state":"Stopped","request_status":true,"is_vnf":false,"has_contract":false,"is_metrics_hidden":false,"is_restricted":false,"has_autoscale":false,"all_time_consumption":0,"created_at":"2026-01-01T00:00:00.000000Z","updated_at":"2026-01-01T00:00:00.000000Z"}}`)
		case r.Method == http.MethodPut && r.URL.Path == "/virtual-machines/test-vm/reboot":
			rebootCalled.Store(true)
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"status":"Success","message":"Rebooting virtual machine..."}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	os.Setenv("ZCP_BEARER_TOKEN", "test-tok")
	defer os.Unsetenv("ZCP_BEARER_TOKEN")

	_, _, err := execCmd(t, NewInstanceCmd(), "reboot", "test-vm", "--api-url", srv.URL)
	if err == nil {
		t.Fatal("expected reboot to fail for stopped VM")
	}
	if !strings.Contains(err.Error(), `instance "test-vm" is Stopped`) {
		t.Errorf("error = %q, want stopped-state message", err)
	}
	if rebootCalled.Load() {
		t.Fatal("reboot endpoint was called for a stopped VM")
	}
}

func TestInstanceRebootAllowsRunningVM(t *testing.T) {
	var rebootCalled atomic.Bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/virtual-machines":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, minVMListBody)
		case r.Method == http.MethodGet && r.URL.Path == "/virtual-machines/test-vm":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, minVMBody)
		case r.Method == http.MethodPut && r.URL.Path == "/virtual-machines/test-vm/reboot":
			rebootCalled.Store(true)
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"status":"Success","message":"Rebooting virtual machine..."}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	os.Setenv("ZCP_BEARER_TOKEN", "test-tok")
	defer os.Unsetenv("ZCP_BEARER_TOKEN")

	_, _, err := execCmd(t, NewInstanceCmd(), "reboot", "test-vm", "--api-url", srv.URL)
	if err != nil {
		t.Fatalf("expected reboot success for running VM, got: %v", err)
	}
	if !rebootCalled.Load() {
		t.Fatal("reboot endpoint was not called for a running VM")
	}
}

func TestInstanceRebootRejectsDuplicateName(t *testing.T) {
	var rebootCalled atomic.Bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/virtual-machines":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"status":"Success","message":"OK","data":[{"id":"vm-id-1","name":"dup-vm","slug":"dup-vm","state":"Running","request_status":true,"is_vnf":false,"has_contract":false,"is_metrics_hidden":false,"is_restricted":false,"has_autoscale":false,"all_time_consumption":0,"created_at":"2026-01-01T00:00:00.000000Z","updated_at":"2026-01-01T00:00:00.000000Z"},{"id":"vm-id-2","name":"dup-vm","slug":"dup-vm-2","state":"Running","request_status":true,"is_vnf":false,"has_contract":false,"is_metrics_hidden":false,"is_restricted":false,"has_autoscale":false,"all_time_consumption":0,"created_at":"2026-01-01T00:00:00.000000Z","updated_at":"2026-01-01T00:00:00.000000Z"}]}`)
		case r.Method == http.MethodPut:
			rebootCalled.Store(true)
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"status":"Success","message":"Rebooting virtual machine..."}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	os.Setenv("ZCP_BEARER_TOKEN", "test-tok")
	defer os.Unsetenv("ZCP_BEARER_TOKEN")

	_, _, err := execCmd(t, NewInstanceCmd(), "reboot", "dup-vm", "--api-url", srv.URL)
	if err == nil {
		t.Fatal("expected duplicate name error")
	}
	if !strings.Contains(err.Error(), `instance name "dup-vm" matches 2 instances`) {
		t.Errorf("error = %q, want duplicate-name message", err)
	}
	if !strings.Contains(err.Error(), "vm-id-1") || !strings.Contains(err.Error(), "vm-id-2") {
		t.Errorf("error = %q, want matching instance IDs", err)
	}
	if rebootCalled.Load() {
		t.Fatal("reboot endpoint was called for an ambiguous VM name")
	}
}

func TestInstanceRebootResolvesVMIDToSlug(t *testing.T) {
	var gotRebootPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/virtual-machines":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"status":"Success","message":"OK","data":[{"id":"record-id-1","vm_id":"vm-id-1","name":"test-vm","slug":"test-vm-slug","state":"Running","request_status":true,"is_vnf":false,"has_contract":false,"is_metrics_hidden":false,"is_restricted":false,"has_autoscale":false,"all_time_consumption":0,"created_at":"2026-01-01T00:00:00.000000Z","updated_at":"2026-01-01T00:00:00.000000Z"}]}`)
		case r.Method == http.MethodGet && r.URL.Path == "/virtual-machines/test-vm-slug":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"status":"Success","message":"OK","data":{"id":"record-id-1","vm_id":"vm-id-1","name":"test-vm","slug":"test-vm-slug","state":"Running","request_status":true,"is_vnf":false,"has_contract":false,"is_metrics_hidden":false,"is_restricted":false,"has_autoscale":false,"all_time_consumption":0,"created_at":"2026-01-01T00:00:00.000000Z","updated_at":"2026-01-01T00:00:00.000000Z"}}`)
		case r.Method == http.MethodPut && r.URL.Path == "/virtual-machines/test-vm-slug/reboot":
			gotRebootPath = r.URL.Path
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"status":"Success","message":"Rebooting virtual machine..."}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	os.Setenv("ZCP_BEARER_TOKEN", "test-tok")
	defer os.Unsetenv("ZCP_BEARER_TOKEN")

	_, _, err := execCmd(t, NewInstanceCmd(), "reboot", "vm-id-1", "--api-url", srv.URL)
	if err != nil {
		t.Fatalf("expected reboot by VM ID to succeed, got: %v", err)
	}
	if gotRebootPath != "/virtual-machines/test-vm-slug/reboot" {
		t.Fatalf("reboot path = %q, want slug route", gotRebootPath)
	}
}

func TestInstanceRebootFallsBackToUnscopedLookup(t *testing.T) {
	var rebootCalled atomic.Bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/virtual-machines":
			w.Header().Set("Content-Type", "application/json")
			// The scoped list (region filter applied) does not contain the VM;
			// the unscoped retry (no region filter) resolves it.
			if r.URL.Query().Get("filter[region]") != "" {
				fmt.Fprint(w, `{"status":"Success","message":"OK","total":0,"data":[]}`)
				return
			}
			fmt.Fprint(w, minVMListBody)
		case r.Method == http.MethodPut && r.URL.Path == "/virtual-machines/test-vm/reboot":
			rebootCalled.Store(true)
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"status":"Success","message":"Rebooting virtual machine..."}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	os.Setenv("ZCP_BEARER_TOKEN", "test-tok")
	defer os.Unsetenv("ZCP_BEARER_TOKEN")
	os.Setenv("ZCP_REGION", "yul-1")
	defer os.Unsetenv("ZCP_REGION")

	_, _, err := execCmd(t, NewInstanceCmd(), "reboot", "test-vm", "--api-url", srv.URL)
	if err != nil {
		t.Fatalf("expected unscoped fallback to resolve the VM, got: %v", err)
	}
	if !rebootCalled.Load() {
		t.Fatal("reboot endpoint was not called after unscoped fallback")
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
		"--cloud-provider", "nimbo", "--region", "yul-1", "--project", "default-9")
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
		"--cloud-provider", "nimbo", "--region", "yul-1", "--project", "default-9")
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
		"--cloud-provider", "nimbo", "--region", "yul-1", "--project", "default-9")
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
		"--cloud-provider", "nimbo", "--region", "yul-1", "--project", "default-9")
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
		"--cloud-provider", "nimbo", "--region", "yul-1", "--project", "default-9")
	if err == nil {
		t.Fatal("expected error when --network-plan is missing for an isolated network")
	}
	if !strings.Contains(err.Error(), "--network-plan is required") {
		t.Errorf("error = %q, want '--network-plan is required'", err)
	}
}

func TestNetworkCreateRejectsUnknownType(t *testing.T) {
	err := networkCreateExec(t,
		"--name", "my-net", "--type", "Shared", "--network-plan", "inet-yul",
		"--cloud-provider", "nimbo", "--region", "yul-1", "--project", "default-9")
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

// ─── DNS record-create priority ─────────────────────────────────────────────

func TestDNSRecordCreateMXRequiresPriority(t *testing.T) {
	cases := []struct {
		name string
		args []string
		want string
	}{
		{
			"MX without priority",
			[]string{"record-create", "--domain", "d", "--name", "@", "--type", "MX", "--content", "mail.example.com."},
			"--priority is required for MX",
		},
		{
			"priority on non-MX record",
			[]string{"record-create", "--domain", "d", "--name", "www", "--type", "A", "--content", "192.0.2.1", "--priority", "10"},
			"--priority is only valid for MX",
		},
		{
			"priority above range",
			[]string{"record-create", "--domain", "d", "--name", "@", "--type", "MX", "--content", "mail.example.com.", "--priority", "70000"},
			"--priority must be between 0 and 65535",
		},
		{
			"priority below range",
			[]string{"record-create", "--domain", "d", "--name", "@", "--type", "MX", "--content", "mail.example.com.", "--priority", "-1"},
			"--priority must be between 0 and 65535",
		},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := execCmd(t, NewDNSCmd(), tt.args...)
			if err == nil {
				t.Fatal("expected validation error, got nil")
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Errorf("error = %q, want containing %q", err, tt.want)
			}
		})
	}
}

// A full MX create through the command path must put priority in the request
// body. This is the regression that caused every CLI MX attempt to 403.
func TestDNSRecordCreateMXSendsPriorityEndToEnd(t *testing.T) {
	var gotBody map[string]interface{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/records") {
			json.NewDecoder(r.Body).Decode(&gotBody)
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"status":"Success","message":"Created","data":{"id":"1","name":"example.com","slug":"example-com-1"}}`)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	os.Setenv("ZCP_BEARER_TOKEN", "test-tok")
	defer os.Unsetenv("ZCP_BEARER_TOKEN")

	_, _, err := execCmd(t, NewDNSCmd(),
		"record-create", "--domain", "example-com-1", "--name", "@", "--type", "MX",
		"--content", "mail.example.com.", "--priority", "10", "--api-url", srv.URL)
	if err != nil {
		t.Fatalf("record-create MX error = %v", err)
	}
	if gotBody["priority"] != float64(10) {
		t.Errorf("request priority = %v, want 10", gotBody["priority"])
	}
	if gotBody["type"] != "MX" {
		t.Errorf("request type = %v, want MX", gotBody["type"])
	}
}
