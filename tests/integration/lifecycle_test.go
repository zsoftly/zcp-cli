//go:build integration

// Package integration provides full lifecycle tests against the live ZCP API.
//
// Run with:
//
//	go test -tags integration -v -timeout 30m ./tests/integration/
//
// Prerequisites:
//   - A configured zcp profile (default) with valid credentials
//
// Environment variables (all optional):
//
//	ZCP_TEST_REGION          Region slug         (default: auto-detected first active region)
//	ZCP_TEST_CLOUD_PROVIDER  Cloud provider slug  (default: auto-detected from region)
//	ZCP_TEST_PROJECT         Project slug         (default: auto-detected first project)
//	ZCP_TEST_TEMPLATE        Template slug        (default: auto-detected Ubuntu 24.04)
//	ZCP_TEST_PLAN            VM plan slug         (default: auto-detected Small Instance)
//	ZCP_TEST_STORAGE_PLAN    Block storage plan   (default: auto-detected Small-Disk)
//	ZCP_TEST_BILLING_CYCLE   Billing cycle slug   (default: "hourly")
//	ZCP_TEST_STORAGE_CAT     Storage category slug(required with ZCP_TEST_STORAGE_PLAN)
//	ZCP_TEST_NETWORK_PLAN    Network plan slug    (default: auto-detected active plan)
//	ZCP_TEST_SNAPSHOT_PLAN   Snapshot plan slug   (default: auto-detected active plan)
//	ZCP_TEST_API_URL         API base URL (default: configured profile)
//	ZCP_TEST_BEARER_TOKEN    Bearer token (default: configured profile)
//	ZCP_TEST_PREFIX          Prefix for resources created by this run (default: "zcp-test")
package integration

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/hex"
	"encoding/pem"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"golang.org/x/crypto/ssh"

	"github.com/zsoftly/zcp-cli/internal/config"
	"github.com/zsoftly/zcp-cli/pkg/api/cloudprovider"
	"github.com/zsoftly/zcp-cli/pkg/api/instance"
	"github.com/zsoftly/zcp-cli/pkg/api/network"
	"github.com/zsoftly/zcp-cli/pkg/api/plan"
	"github.com/zsoftly/zcp-cli/pkg/api/project"
	"github.com/zsoftly/zcp-cli/pkg/api/region"
	"github.com/zsoftly/zcp-cli/pkg/api/snapshot"
	"github.com/zsoftly/zcp-cli/pkg/api/sshkey"
	"github.com/zsoftly/zcp-cli/pkg/api/storagecategory"
	"github.com/zsoftly/zcp-cli/pkg/api/template"
	"github.com/zsoftly/zcp-cli/pkg/api/volume"
	"github.com/zsoftly/zcp-cli/pkg/httpclient"
)

// ─── helpers ────────────────────────────────────────────────────────────────

var integrationRunID = func() string {
	b := make([]byte, 4)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("%08x", time.Now().UnixNano())
	}
	return hex.EncodeToString(b)
}()

// testID returns a unique prefix for test resources to avoid name collisions.
func testID() string {
	prefix := env("ZCP_TEST_PREFIX", "zcp-test")
	const runSuffixLength = 7
	const maxPrefixLength = 7
	if len(prefix) > maxPrefixLength {
		prefix = prefix[:maxPrefixLength]
	}
	return prefix + "-" + integrationRunID[:runSuffixLength]
}

// env returns the environment variable value or fallback.
func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// testContext returns a context with a generous timeout for API calls.
func testContext(t *testing.T, timeout time.Duration) context.Context {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	t.Cleanup(cancel)
	return ctx
}

// setupClient loads the default profile and returns a configured HTTP client.
func setupClient(t *testing.T) *httpclient.Client {
	t.Helper()
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("loading config: %v", err)
	}
	profile, err := config.ResolveProfile(cfg, "")
	if err != nil {
		t.Fatalf("resolving profile: %v", err)
	}
	apiURL := env("ZCP_TEST_API_URL", config.ActiveAPIURL(profile, ""))
	token := env("ZCP_TEST_BEARER_TOKEN", profile.BearerToken)
	client := httpclient.New(httpclient.Options{
		BaseURL:     apiURL,
		BearerToken: token,
		Timeout:     2 * time.Minute,
		Debug:       os.Getenv("ZCP_TEST_DEBUG") != "",
		DebugOut:    os.Stderr,
	})
	return client
}

// generateSSHKey creates an ephemeral Ed25519 key pair and returns the
// OpenSSH-formatted public key string.
func generateSSHKey(t *testing.T) string {
	t.Helper()
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generating ed25519 key: %v", err)
	}
	signer, err := ssh.NewSignerFromKey(priv)
	if err != nil {
		t.Fatalf("creating signer: %v", err)
	}
	pubBytes := ssh.MarshalAuthorizedKey(signer.PublicKey())
	return strings.TrimSpace(string(pubBytes))
}

// generateSSHKeyPEM is unused but kept for reference; the API only needs the
// public key in authorized_keys format.
func generateSSHKeyPEM(t *testing.T) (pubStr string, privPEM []byte) {
	t.Helper()
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generating ed25519 key: %v", err)
	}
	signer, err := ssh.NewSignerFromKey(priv)
	if err != nil {
		t.Fatalf("creating signer: %v", err)
	}
	pubStr = strings.TrimSpace(string(ssh.MarshalAuthorizedKey(signer.PublicKey())))
	_ = pub
	privPEM = pem.EncodeToMemory(&pem.Block{Type: "OPENSSH PRIVATE KEY", Bytes: priv.Seed()})
	return
}

// waitForInstance polls instance state via Get until it reaches one of targetStates.
func waitForInstance(t *testing.T, svc *instance.Service, ctx context.Context, slug string, targetStates ...string) {
	t.Helper()
	t.Logf("  waiting for instance %s to reach %v ...", slug, targetStates)
	poll := 10 * time.Second
	ticker := time.NewTicker(poll)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			t.Fatalf("timed out waiting for instance to reach %v", targetStates)
		case <-ticker.C:
			vm, err := svc.Meta(ctx, slug)
			if err != nil {
				t.Fatalf("polling instance state: %v", err)
			}
			t.Logf("    current state: %s", vm.State)
			if strings.EqualFold(vm.State, "error") {
				t.Fatalf("instance entered error state (wanted %v)", targetStates)
			}
			for _, target := range targetStates {
				if strings.EqualFold(vm.State, target) {
					t.Logf("  instance reached state: %s", vm.State)
					return
				}
			}
		}
	}
}

// waitForInstanceHostname waits for the asynchronous hostname change to finish.
// The API can update a VM slug along with its hostname, so the immutable record
// ID anchors the lookup. currentSlug is updated as soon as the record is found
// so cleanup can still cancel the VM if the operation later fails or times out.
func waitForInstanceHostname(t *testing.T, svc *instance.Service, ctx context.Context, region, project, id, hostname string, currentSlug *string) *instance.VirtualMachine {
	t.Helper()
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()
	for {
		vms, err := svc.List(ctx, region, project)
		if err != nil {
			t.Fatalf("listing instances while waiting for hostname change: %v", err)
		}
		for i := range vms {
			vm := &vms[i]
			if vm.ID != id {
				continue
			}
			if vm.Slug != "" {
				*currentSlug = vm.Slug
			}
			if vm.Name == hostname && vm.Hostname == hostname && strings.EqualFold(vm.State, "Running") {
				t.Logf("  instance name and hostname changed to %s", hostname)
				return vm
			}
			if strings.EqualFold(vm.State, "Error") {
				t.Fatal("instance entered error state while changing hostname")
			}
			t.Logf("    name=%s hostname=%s state=%s", vm.Name, vm.Hostname, vm.State)
		}
		select {
		case <-ctx.Done():
			t.Fatalf("timed out waiting for instance hostname to change to %s", hostname)
		case <-ticker.C:
		}
	}
}

func waitForInstanceAbsent(t *testing.T, svc *instance.Service, ctx context.Context, region, project, slug string) {
	t.Helper()
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()
	for {
		vms, err := svc.List(ctx, region, project)
		if err != nil {
			t.Fatalf("listing instances while waiting for %s deletion: %v", slug, err)
		}
		if !containsInstance(vms, slug) {
			t.Logf("  instance %s is absent", slug)
			return
		}
		select {
		case <-ctx.Done():
			t.Fatalf("timed out waiting for instance %s deletion", slug)
		case <-ticker.C:
		}
	}
}

func containsInstance(vms []instance.VirtualMachine, slug string) bool {
	for _, vm := range vms {
		if vm.Slug == slug {
			return true
		}
	}
	return false
}

func waitForInstanceAbsentResult(ctx context.Context, svc *instance.Service, region, project, slug string) error {
	return waitUntilAbsent(ctx, func() (bool, error) {
		vms, err := svc.List(ctx, region, project)
		return !containsInstance(vms, slug), err
	})
}

func waitForVolumeAbsentResult(ctx context.Context, svc *volume.Service, region, project, slug string) error {
	return waitUntilAbsent(ctx, func() (bool, error) {
		volumes, err := svc.List(ctx, region, project)
		if err != nil {
			return false, err
		}
		for _, volume := range volumes {
			if volume.Slug == slug {
				return false, nil
			}
		}
		return true, nil
	})
}

func waitUntilAbsent(ctx context.Context, check func() (bool, error)) error {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()
	for {
		absent, err := check()
		if err != nil {
			return err
		}
		if absent {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

func waitForVolumeAbsent(t *testing.T, svc *volume.Service, ctx context.Context, region, project, slug string) {
	t.Helper()
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()
	for {
		volumes, err := svc.List(ctx, region, project)
		if err != nil {
			t.Fatalf("listing volumes while waiting for %s deletion: %v", slug, err)
		}
		found := false
		for _, volume := range volumes {
			found = found || volume.Slug == slug
		}
		if !found {
			t.Logf("  volume %s is absent", slug)
			return
		}
		select {
		case <-ctx.Done():
			t.Fatalf("timed out waiting for volume %s deletion", slug)
		case <-ticker.C:
		}
	}
}

func waitForVolumeDetached(t *testing.T, svc *volume.Service, ctx context.Context, region, project, slug string) {
	t.Helper()
	if err := waitForVolumeDetachedResult(ctx, svc, region, project, slug); err != nil {
		t.Fatalf("waiting for volume %s detachment: %v", slug, err)
	}
	t.Logf("  volume %s is detached", slug)
}

func waitForVolumeDetachedResult(ctx context.Context, svc *volume.Service, region, project, slug string) error {
	return waitUntilAbsent(ctx, func() (bool, error) {
		volumes, err := svc.List(ctx, region, project)
		if err != nil {
			return false, err
		}
		for _, volume := range volumes {
			if volume.Slug == slug && volume.VirtualMachineID == "" {
				return true, nil
			}
		}
		return false, nil
	})
}

func waitForVolumeAttached(t *testing.T, svc *volume.Service, ctx context.Context, region, project, slug, vmID string) {
	t.Helper()
	if err := waitForVolumeAttachedResult(ctx, svc, region, project, slug, vmID); err != nil {
		t.Fatalf("waiting for volume %s attachment to instance %s: %v", slug, vmID, err)
	}
	t.Logf("  volume %s is attached to instance %s", slug, vmID)
}

func waitForVolumeAttachedResult(ctx context.Context, svc *volume.Service, region, project, slug, vmID string) error {
	return waitUntilAbsent(ctx, func() (bool, error) {
		volumes, err := svc.List(ctx, region, project)
		if err != nil {
			return false, err
		}
		for _, volume := range volumes {
			if volume.Slug == slug && volume.VirtualMachineID == vmID {
				return true, nil
			}
		}
		return false, nil
	})
}

func deleteVolumeAfterDetach(ctx context.Context, svc *volume.Service, region, project, slug string) error {
	if _, err := svc.Detach(ctx, slug); err != nil {
		detached, checkErr := isVolumeDetached(ctx, svc, region, project, slug)
		if checkErr != nil {
			return fmt.Errorf("detaching volume: %w (checking state: %v)", err, checkErr)
		}
		if !detached {
			return fmt.Errorf("detaching volume: %w", err)
		}
	} else if err := waitForVolumeDetachedResult(ctx, svc, region, project, slug); err != nil {
		return fmt.Errorf("waiting for volume detachment: %w", err)
	}
	if err := svc.Delete(ctx, slug); err != nil {
		return fmt.Errorf("deleting volume: %w", err)
	}
	if err := waitForVolumeAbsentResult(ctx, svc, region, project, slug); err != nil {
		return fmt.Errorf("waiting for volume removal: %w", err)
	}
	return nil
}

func isVolumeDetached(ctx context.Context, svc *volume.Service, region, project, slug string) (bool, error) {
	volumes, err := svc.List(ctx, region, project)
	if err != nil {
		return false, err
	}
	for _, volume := range volumes {
		if volume.Slug == slug {
			return volume.VirtualMachineID == "", nil
		}
	}
	return true, nil
}

func waitForSnapshotAbsent(t *testing.T, svc *snapshot.Service, ctx context.Context, region, project, slug string) {
	t.Helper()
	if err := waitForSnapshotAbsentResult(ctx, svc, region, project, slug); err != nil {
		t.Fatalf("waiting for snapshot %s deletion: %v", slug, err)
	}
	t.Logf("  snapshot %s is absent", slug)
}

func waitForSnapshotAbsentResult(ctx context.Context, svc *snapshot.Service, region, project, slug string) error {
	return waitUntilAbsent(ctx, func() (bool, error) {
		snapshots, err := svc.List(ctx, region, project)
		if err != nil {
			return false, err
		}
		for _, item := range snapshots {
			if item.Slug == slug {
				return false, nil
			}
		}
		return true, nil
	})
}

func waitForNetworkAbsent(t *testing.T, svc *network.Service, ctx context.Context, region, project, slug string) {
	t.Helper()
	if err := waitForNetworkAbsentResult(ctx, svc, region, project, slug); err != nil {
		t.Fatalf("waiting for network %s deletion: %v", slug, err)
	}
	t.Logf("  network %s is absent", slug)
}

func waitForNetworkAbsentResult(ctx context.Context, svc *network.Service, region, project, slug string) error {
	return waitUntilAbsent(ctx, func() (bool, error) {
		networks, err := svc.List(ctx, region, project)
		if err != nil {
			return false, err
		}
		for _, item := range networks {
			if item.Slug == slug {
				return false, nil
			}
		}
		return true, nil
	})
}

func networkSlugByName(ctx context.Context, svc *network.Service, region, project, name string) (string, error) {
	networks, err := svc.List(ctx, region, project)
	if err != nil {
		return "", err
	}
	for _, item := range networks {
		if item.Name == name {
			return item.Slug, nil
		}
	}
	return "", nil
}

func instanceSlugByName(ctx context.Context, svc *instance.Service, region, project, name string) (string, error) {
	vms, err := svc.List(ctx, region, project)
	if err != nil {
		return "", err
	}
	for _, vm := range vms {
		if vm.Name == name {
			return vm.Slug, nil
		}
	}
	return "", nil
}

func instanceSlugByID(ctx context.Context, svc *instance.Service, region, project, id string) (string, error) {
	vms, err := svc.List(ctx, region, project)
	if err != nil {
		return "", err
	}
	for _, vm := range vms {
		if vm.ID == id {
			return vm.Slug, nil
		}
	}
	return "", nil
}

func volumeSlugByName(ctx context.Context, svc *volume.Service, region, project, name string) (string, error) {
	volumes, err := svc.List(ctx, region, project)
	if err != nil {
		return "", err
	}
	for _, item := range volumes {
		if item.Name == name {
			return item.Slug, nil
		}
	}
	return "", nil
}

func snapshotSlugByName(ctx context.Context, svc *snapshot.Service, region, project, name string) (string, error) {
	snapshots, err := svc.List(ctx, region, project)
	if err != nil {
		return "", err
	}
	for _, item := range snapshots {
		if item.Name == name {
			return item.Slug, nil
		}
	}
	return "", nil
}

// ─── resource auto-detection ────────────────────────────────────────────────

func detectRegion(t *testing.T, client *httpclient.Client) (regionSlug, cloudProviderSlug string) {
	if rs := os.Getenv("ZCP_TEST_REGION"); rs != "" {
		cp := env("ZCP_TEST_CLOUD_PROVIDER", "")
		if cp == "" {
			t.Fatal("ZCP_TEST_CLOUD_PROVIDER must be set when ZCP_TEST_REGION is set")
		}
		return rs, cp
	}
	ctx := testContext(t, 30*time.Second)
	svc := region.NewService(client)
	regions, err := svc.List(ctx)
	if err != nil {
		t.Fatalf("listing regions: %v", err)
	}
	// This suite creates compute resources (VMs, volumes), so only consider
	// providers whose catalog advertises "Virtual Machine" (the object-storage
	// and DNS providers are single-service). Pick the first active region
	// backed by such a provider.
	providers, err := cloudprovider.NewService(client).List(ctx)
	if err != nil {
		t.Fatalf("listing cloud providers: %v", err)
	}
	compute := map[string]bool{}
	for _, p := range providers {
		for _, svcName := range p.Services {
			if p.Status && svcName == "Virtual Machine" {
				compute[p.Slug] = true
			}
		}
	}
	for _, r := range regions {
		if r.Status && r.CloudProvider != nil && compute[r.CloudProvider.Slug] {
			t.Logf("  auto-detected region: %s (%s), cloud_provider: %s", r.Name, r.Slug, r.CloudProvider.Slug)
			return r.Slug, r.CloudProvider.Slug
		}
	}
	t.Fatal("no compute region available (all regions are object-storage/dns)")
	return "", ""
}

func detectProject(t *testing.T, client *httpclient.Client) string {
	if v := os.Getenv("ZCP_TEST_PROJECT"); v != "" {
		return v
	}
	ctx := testContext(t, 30*time.Second)
	svc := project.NewService(client)
	projects, err := svc.List(ctx)
	if err != nil {
		t.Fatalf("listing projects: %v", err)
	}
	if len(projects) > 0 {
		t.Logf("  auto-detected project: %s (%s)", projects[0].Name, projects[0].Slug)
		return projects[0].Slug
	}
	t.Fatal("no projects available")
	return ""
}

func detectTemplate(t *testing.T, client *httpclient.Client, regionSlug string) string {
	if v := os.Getenv("ZCP_TEST_TEMPLATE"); v != "" {
		return v
	}
	ctx := testContext(t, 30*time.Second)
	svc := template.NewService(client)
	templates, err := svc.List(ctx, regionSlug)
	if err != nil {
		t.Fatalf("listing templates: %v", err)
	}
	// Prefer Ubuntu 24.04, then 22.04, then first available
	for _, pref := range []string{"Ubuntu-24.04", "Ubuntu 24.04", "Ubuntu-22.04", "Ubuntu 22.04"} {
		for _, tmpl := range templates {
			if strings.Contains(tmpl.Name, pref) {
				t.Logf("  auto-detected template: %s (%s)", tmpl.Name, tmpl.Slug)
				return tmpl.Slug
			}
		}
	}
	if len(templates) > 0 {
		t.Logf("  using first available template: %s (%s)", templates[0].Name, templates[0].Slug)
		return templates[0].Slug
	}
	t.Fatal("no templates available")
	return ""
}

func detectVMPlan(t *testing.T, client *httpclient.Client, regionSlug string) string {
	if v := os.Getenv("ZCP_TEST_PLAN"); v != "" {
		return v
	}
	ctx := testContext(t, 30*time.Second)
	svc := plan.NewService(client)
	plans, err := svc.List(ctx, plan.ServiceVM, regionSlug)
	if err != nil {
		t.Fatalf("listing VM plans: %v", err)
	}
	// Prefer exact "Small Instance" (no tier suffix)
	for _, p := range plans {
		if p.Name == "Small Instance" && p.Status {
			t.Logf("  auto-detected VM plan: %s (%s)", p.Name, p.Slug)
			return p.Slug
		}
	}
	// Fallback: cheapest option
	for _, pref := range []string{"Small Instance", "Starter"} {
		for _, p := range plans {
			if strings.Contains(p.Name, pref) && p.Status {
				t.Logf("  auto-detected VM plan: %s (%s)", p.Name, p.Slug)
				return p.Slug
			}
		}
	}
	if len(plans) > 0 {
		t.Logf("  using first VM plan: %s (%s)", plans[0].Name, plans[0].Slug)
		return plans[0].Slug
	}
	t.Fatal("no VM plans available")
	return ""
}

func detectStoragePlan(t *testing.T, client *httpclient.Client, regionSlug string) (string, string) {
	if v := os.Getenv("ZCP_TEST_STORAGE_PLAN"); v != "" {
		category := os.Getenv("ZCP_TEST_STORAGE_CAT")
		if category == "" {
			t.Fatal("ZCP_TEST_STORAGE_CAT must be set when ZCP_TEST_STORAGE_PLAN is set")
		}
		return v, category
	}
	ctx := testContext(t, 30*time.Second)
	svc := plan.NewService(client)
	plans, err := svc.List(ctx, plan.ServiceBlockStorage, regionSlug)
	if err != nil {
		t.Fatalf("listing block storage plans: %v", err)
	}
	categories, err := storagecategory.NewService(client).List(ctx, regionSlug)
	if err != nil {
		t.Fatalf("listing storage categories: %v", err)
	}
	categoryByID := make(map[string]string, len(categories))
	for _, category := range categories {
		if category.Status {
			categoryByID[category.ID] = category.Slug
		}
	}
	for _, p := range plans {
		if p.Status {
			if category := categoryByID[p.StorageCategoryID]; category != "" {
				t.Logf("  auto-detected storage plan: %s (%s), category: %s", p.Name, p.Slug, category)
				return p.Slug, category
			}
		}
	}
	t.Fatal("no active block storage plan with an active storage category available")
	return "", ""
}

func detectNetworkPlan(t *testing.T, client *httpclient.Client, regionSlug string) string {
	if v := os.Getenv("ZCP_TEST_NETWORK_PLAN"); v != "" {
		return v
	}
	ctx := testContext(t, 30*time.Second)
	plans, err := plan.NewService(client).List(ctx, plan.ServiceNetwork, regionSlug)
	if err != nil {
		t.Fatalf("listing network plans: %v", err)
	}
	for _, p := range plans {
		if p.Status {
			t.Logf("  auto-detected network plan: %s (%s)", p.Name, p.Slug)
			return p.Slug
		}
	}
	t.Fatal("no active network plan available")
	return ""
}

func detectSnapshotPlan(t *testing.T, client *httpclient.Client, regionSlug string) string {
	if v := os.Getenv("ZCP_TEST_SNAPSHOT_PLAN"); v != "" {
		return v
	}
	ctx := testContext(t, 30*time.Second)
	plans, err := plan.NewService(client).List(ctx, plan.ServiceVMSnapshot, regionSlug)
	if err != nil {
		t.Fatalf("listing VM snapshot plans: %v", err)
	}
	for _, p := range plans {
		if p.Status {
			t.Logf("  auto-detected VM snapshot plan: %s (%s)", p.Name, p.Slug)
			return p.Slug
		}
	}
	t.Fatal("no active VM snapshot plan available")
	return ""
}

// ─── Phase 1: SSH Key Lifecycle ─────────────────────────────────────────────

func TestPhase1_SSHKeyLifecycle(t *testing.T) {
	client := setupClient(t)
	ctx := testContext(t, 60*time.Second)
	svc := sshkey.NewService(client)
	var keySlug string
	t.Cleanup(func() {
		if keySlug == "" {
			return
		}
		cleanCtx, cancel := context.WithTimeout(context.Background(), time.Minute)
		defer cancel()
		keys, err := svc.List(cleanCtx)
		if err != nil {
			t.Errorf("cleanup listing SSH keys: %v", err)
			return
		}
		found := false
		for _, item := range keys {
			if item.Slug == keySlug {
				found = true
				break
			}
		}
		if !found {
			return
		}
		if err := svc.Delete(cleanCtx, keySlug); err != nil {
			t.Errorf("cleanup deleting SSH key %s: %v", keySlug, err)
			return
		}
		keys, err = svc.List(cleanCtx)
		if err != nil {
			t.Errorf("cleanup listing SSH keys after delete: %v", err)
			return
		}
		for _, item := range keys {
			if item.Slug == keySlug {
				t.Errorf("cleanup SSH key %s remains listed", keySlug)
				return
			}
		}
	})
	keyName := testID() + "-key"

	// project and region are mandatory — the API derives the cloud provider
	// from them and 500s ("read property \"id\" on null") if either is missing.
	regionSlug, _ := detectRegion(t, client)
	projectSlug := detectProject(t, client)

	// Generate an ephemeral SSH key pair
	pubKey := generateSSHKey(t)
	t.Logf("generated ephemeral SSH public key: %s...%s", pubKey[:30], pubKey[len(pubKey)-10:])

	// 1. Create (import) SSH key
	t.Log("Step 1: Import SSH key")
	key, err := svc.Create(ctx, sshkey.CreateRequest{
		Name:      keyName,
		PublicKey: pubKey,
		Project:   projectSlug,
		Region:    regionSlug,
	})
	if err != nil {
		t.Fatalf("creating SSH key: %v", err)
	}
	keySlug = key.Slug
	t.Logf("  created SSH key: slug=%s name=%s", key.Slug, key.Name)

	// 2. List and verify it exists
	t.Log("Step 2: List SSH keys")
	keys, err := svc.List(ctx)
	if err != nil {
		t.Fatalf("listing SSH keys: %v", err)
	}
	found := false
	for _, k := range keys {
		if k.Slug == key.Slug {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("newly created SSH key %s not found in list", key.Slug)
	}
	t.Logf("  found %d SSH key(s), our key present: %v", len(keys), found)

	// 3. Delete
	t.Log("Step 3: Delete SSH key")
	if err := svc.Delete(ctx, key.Slug); err != nil {
		t.Fatalf("deleting SSH key: %v", err)
	}
	t.Logf("  deleted SSH key %s", key.Slug)

	// 4. Verify deletion
	t.Log("Step 4: Verify deletion")
	keys, err = svc.List(ctx)
	if err != nil {
		t.Fatalf("listing SSH keys after delete: %v", err)
	}
	for _, k := range keys {
		if k.Slug == key.Slug {
			t.Errorf("deleted SSH key %s still appears in list", key.Slug)
			return
		}
	}
	keySlug = ""
	t.Log("  SSH key no longer in list — lifecycle complete")
}

// Phase 2 (Security Group Lifecycle) removed — securitygroup API was STKBILL-only
// and has no STKCNSL equivalent.

// ─── Phase 3: Instance Full Lifecycle ───────────────────────────────────────

func TestPhase3_InstanceLifecycle(t *testing.T) {
	client := setupClient(t)
	instanceSvc := instance.NewService(client)
	volumeSvc := volume.NewService(client)
	snapshotSvc := snapshot.NewService(client)
	networkSvc := network.NewService(client)

	// Auto-detect resources
	t.Log("=== Resource Detection ===")
	regionSlug, cloudProviderSlug := detectRegion(t, client)
	projectSlug := detectProject(t, client)
	templateSlug := detectTemplate(t, client, regionSlug)
	vmPlanSlug := detectVMPlan(t, client, regionSlug)
	storagePlanSlug, storageCategory := detectStoragePlan(t, client, regionSlug)
	networkPlanSlug := detectNetworkPlan(t, client, regionSlug)
	snapshotPlanSlug := detectSnapshotPlan(t, client, regionSlug)
	billingCycle := env("ZCP_TEST_BILLING_CYCLE", "hourly")
	rootDiskSize := 20
	dataDiskSize := 10
	vmName := testID() + "-vm"
	volName := testID() + "-vol"
	networkName := testID() + "-net"

	// Track resources for cleanup (slugs, not UUIDs)
	var vmSlug, vmID, volSlug, snapSlug, networkSlug string
	tagCreated := false // tracks whether we created a tag (cleanup uses key-based delete)

	// Cleanup function — always runs, even on failure. Each dependent resource
	// gets its own deadline so a slow prior deletion cannot prevent later cleanup.
	t.Cleanup(func() {
		withCleanupContext := func(timeout time.Duration, fn func(context.Context)) {
			cleanCtx, cancel := context.WithTimeout(context.Background(), timeout)
			defer cancel()
			fn(cleanCtx)
		}

		t.Log("=== Cleanup ===")
		if snapSlug != "" {
			withCleanupContext(5*time.Minute, func(cleanCtx context.Context) {
				t.Logf("  deleting snapshot %s", snapSlug)
				if err := snapshotSvc.Delete(cleanCtx, snapSlug); err != nil {
					t.Errorf("cleanup deleting snapshot %s: %v", snapSlug, err)
				} else if err := waitForSnapshotAbsentResult(cleanCtx, snapshotSvc, regionSlug, projectSlug, snapSlug); err != nil {
					t.Errorf("cleanup waiting for snapshot %s removal: %v", snapSlug, err)
				} else {
					snapSlug = ""
				}
			})
		}
		if tagCreated && vmSlug != "" {
			withCleanupContext(time.Minute, func(cleanCtx context.Context) {
				t.Logf("  deleting tag lifecycle-test from %s", vmSlug)
				if err := instanceSvc.DeleteTag(cleanCtx, vmSlug, "lifecycle-test"); err != nil {
					t.Errorf("cleanup deleting tag: %v", err)
				}
			})
		}
		if volSlug != "" {
			withCleanupContext(5*time.Minute, func(cleanCtx context.Context) {
				t.Logf("  detaching and deleting volume %s", volSlug)
				if err := deleteVolumeAfterDetach(cleanCtx, volumeSvc, regionSlug, projectSlug, volSlug); err != nil {
					t.Errorf("cleanup removing volume %s: %v", volSlug, err)
				} else {
					volSlug = ""
				}
			})
		}
		vmRemoved := vmSlug == ""
		if vmSlug != "" {
			withCleanupContext(10*time.Minute, func(cleanCtx context.Context) {
				if vmID != "" {
					currentSlug, err := instanceSlugByID(cleanCtx, instanceSvc, regionSlug, projectSlug, vmID)
					if err != nil {
						t.Errorf("cleanup resolving instance ID %s: %v", vmID, err)
						return
					}
					if currentSlug == "" {
						vmSlug = ""
						vmRemoved = true
						return
					}
					vmSlug = currentSlug
				}
				t.Logf("  cancelling instance %s", vmSlug)
				if err := instanceSvc.Delete(cleanCtx, vmSlug, false, false); err != nil {
					t.Errorf("cleanup cancelling instance %s: %v", vmSlug, err)
				} else if err := waitForInstanceAbsentResult(cleanCtx, instanceSvc, regionSlug, projectSlug, vmSlug); err != nil {
					t.Errorf("cleanup waiting for instance %s removal: %v", vmSlug, err)
				} else {
					vmSlug = ""
					vmRemoved = true
				}
			})
		}
		if networkSlug != "" && vmRemoved {
			withCleanupContext(5*time.Minute, func(cleanCtx context.Context) {
				t.Logf("  deleting network %s", networkSlug)
				if err := networkSvc.Delete(cleanCtx, networkSlug); err != nil {
					t.Errorf("cleanup deleting network %s: %v", networkSlug, err)
				} else if err := waitForNetworkAbsentResult(cleanCtx, networkSvc, regionSlug, projectSlug, networkSlug); err != nil {
					t.Errorf("cleanup waiting for network %s removal: %v", networkSlug, err)
				} else {
					networkSlug = ""
				}
			})
		}
		t.Log("  cleanup complete")
	})

	// ── Step 1: Create isolated network ─────────────────────────────────
	t.Log("=== Step 1: Create Isolated Network ===")
	networkCtx := testContext(t, 5*time.Minute)
	net, err := networkSvc.Create(networkCtx, network.CreateRequest{
		Name:          networkName,
		CloudProvider: cloudProviderSlug,
		Project:       projectSlug,
		Region:        regionSlug,
		Type:          "Isolated",
		NetworkPlan:   networkPlanSlug,
		BillingCycle:  billingCycle,
	})
	if err != nil {
		// A failed response can still follow a successful asynchronous create. Find
		// only this run's uniquely named network so cleanup can remove it.
		if createdSlug, lookupErr := networkSlugByName(testContext(t, time.Minute), networkSvc, regionSlug, projectSlug, networkName); lookupErr != nil {
			t.Logf("  could not check whether network %q was created: %v", networkName, lookupErr)
		} else {
			networkSlug = createdSlug
		}
		t.Fatalf("creating isolated network: %v", err)
	}
	networkSlug = net.Slug
	if networkSlug == "" {
		if createdSlug, lookupErr := networkSlugByName(testContext(t, time.Minute), networkSvc, regionSlug, projectSlug, networkName); lookupErr != nil {
			t.Fatalf("network create returned an empty slug and lookup failed: %v", lookupErr)
		} else if createdSlug == "" {
			t.Fatal("network create returned an empty slug")
		} else {
			networkSlug = createdSlug
		}
	}
	t.Logf("  created isolated network: slug=%s", networkSlug)

	// ── Step 2: Create Instance ─────────────────────────────────────────
	t.Log("=== Step 2: Create Instance ===")
	createCtx := testContext(t, 10*time.Minute)
	pw := "TestP@ss1234!"
	vm, err := instanceSvc.Create(createCtx, instance.CreateRequest{
		Name:             vmName,
		Hostname:         vmName,
		CloudProvider:    cloudProviderSlug,
		Project:          projectSlug,
		Region:           regionSlug,
		BootSource:       "image",
		Server:           "cloud-compute",
		Template:         templateSlug,
		Plan:             vmPlanSlug,
		BlockstoragePlan: storagePlanSlug,
		BlockstorageCustomPlan: &instance.BlockstorageCustomPlan{
			Storage: rootDiskSize,
		},
		BillingCycle:    billingCycle,
		StorageCategory: storageCategory,
		IsPublic:        false,
		NetworkType:     "Isolated",
		Networks:        []string{networkSlug},
		Password:        &pw,
		Addons:          []string{},
	})
	if err != nil {
		if createdSlug, lookupErr := instanceSlugByName(testContext(t, time.Minute), instanceSvc, regionSlug, projectSlug, vmName); lookupErr != nil {
			t.Logf("  could not check whether instance %q was created: %v", vmName, lookupErr)
		} else {
			vmSlug = createdSlug
		}
		t.Fatalf("creating instance: %v", err)
	}
	vmSlug = vm.Slug
	if vmSlug == "" {
		if createdSlug, lookupErr := instanceSlugByName(testContext(t, time.Minute), instanceSvc, regionSlug, projectSlug, vmName); lookupErr != nil {
			t.Fatalf("instance create returned an empty slug and lookup failed: %v", lookupErr)
		} else if createdSlug == "" {
			t.Fatal("instance create returned an empty slug")
		} else {
			vmSlug = createdSlug
		}
	}
	t.Logf("  created instance: slug=%s name=%s state=%s", vm.Slug, vm.Name, vm.State)

	// Wait for Running state
	waitCtx := testContext(t, 10*time.Minute)
	waitForInstance(t, instanceSvc, waitCtx, vmSlug, "Running", "running")

	// ── Step 2: Verify Instance in List ─────────────────────────────────
	t.Log("=== Step 2: Verify Instance in List ===")
	listCtx := testContext(t, 30*time.Second)
	vms, err := instanceSvc.List(listCtx, regionSlug, projectSlug)
	if err != nil {
		t.Fatalf("listing instances: %v", err)
	}
	found := false
	for _, v := range vms {
		if v.Slug == vmSlug {
			found = true
			t.Logf("  found instance in list: name=%s state=%s ip=%s", v.Name, v.State, instance.StringVal(v.PrivateIP))
			break
		}
	}
	if !found {
		t.Errorf("instance %s not found in list", vmSlug)
	}

	// ── Step 3: Get Instance State ──────────────────────────────────────
	t.Log("=== Step 3: Get Instance State ===")
	statusCtx := testContext(t, 30*time.Second)
	vmDetail, err := instanceSvc.Get(statusCtx, vmSlug)
	if err != nil {
		t.Fatalf("getting instance: %v", err)
	}
	t.Logf("  instance state: %s", vmDetail.State)
	if !strings.EqualFold(vmDetail.State, "Running") && !strings.EqualFold(vmDetail.State, "running") {
		t.Errorf("expected Running, got %s", vmDetail.State)
	}
	if vmDetail.ID == "" {
		t.Fatal("instance get returned an empty ID")
	}
	vm.ID = vmDetail.ID
	vmID = vmDetail.ID

	// ── Step 4: Rename Instance (ChangeHostname) ────────────────────────
	t.Log("=== Step 4: Rename Instance ===")
	renameCtx := testContext(t, 10*time.Minute)
	newName := vmName + "-renamed"
	_, err = instanceSvc.ChangeHostname(renameCtx, vmSlug, instance.ChangeLabelRequest{
		Name:     vmDetail.Name,
		Hostname: newName,
	})
	if err != nil {
		t.Fatalf("renaming instance: %v", err)
	}
	renamedVM := waitForInstanceHostname(t, instanceSvc, renameCtx, regionSlug, projectSlug, vmDetail.ID, newName, &vmSlug)
	if renamedVM.Slug == "" {
		t.Fatal("hostname change returned an empty instance slug")
	}
	vmSlug = renamedVM.Slug
	t.Logf("  renamed to: %s (slug=%s)", newName, vmSlug)

	// ── Step 5: Tag Instance ────────────────────────────────────────────
	t.Log("=== Step 5: Tag Instance ===")
	tagCtx := testContext(t, 30*time.Second)
	_, err = instanceSvc.CreateTag(tagCtx, vmSlug, instance.TagRequest{
		Key: "lifecycle-test", Value: "true",
	})
	if err != nil {
		t.Fatalf("creating tag: %v", err)
	}
	tagCreated = true
	t.Logf("  created tag: key=lifecycle-test value=true on %s", vmSlug)

	// ── Step 6: List Addons (replaces old ListNetworks) ─────────────────
	t.Log("=== Step 6: List Instance Addons ===")
	addonCtx := testContext(t, 30*time.Second)
	addons, err := instanceSvc.ListAddons(addonCtx, vmSlug)
	if err != nil {
		t.Fatalf("listing addons: %v", err)
	}
	t.Logf("  instance has %d addon(s)", len(addons))
	for _, a := range addons {
		t.Logf("    addon: %s slug=%s status=%v", a.Name, a.Slug, a.Status)
	}

	// ── Step 7: Create Data Volume ──────────────────────────────────────
	t.Log("=== Step 7: Create Data Volume ===")
	volCtx := testContext(t, 2*time.Minute)
	vol, err := volumeSvc.Create(volCtx, volume.CreateRequest{
		Name:            volName,
		Project:         projectSlug,
		CloudProvider:   cloudProviderSlug,
		Region:          regionSlug,
		BillingCycle:    billingCycle,
		StorageCategory: storageCategory,
		Plan:            storagePlanSlug,
		IsCustomPlan:    true,
		CustomPlan:      &volume.CustomPlanStorage{Storage: dataDiskSize},
	})
	if err != nil {
		if createdSlug, lookupErr := volumeSlugByName(testContext(t, time.Minute), volumeSvc, regionSlug, projectSlug, volName); lookupErr != nil {
			t.Logf("  could not check whether volume %q was created: %v", volName, lookupErr)
		} else {
			volSlug = createdSlug
		}
		t.Fatalf("creating volume: %v", err)
	}
	volSlug = vol.Slug
	if volSlug == "" {
		if createdSlug, lookupErr := volumeSlugByName(testContext(t, time.Minute), volumeSvc, regionSlug, projectSlug, volName); lookupErr != nil {
			t.Fatalf("volume create returned an empty slug and lookup failed: %v", lookupErr)
		} else if createdSlug == "" {
			t.Fatal("creating volume returned an empty slug")
		} else {
			volSlug = createdSlug
		}
	}
	t.Logf("  volume create response: slug=%s name=%s", vol.Slug, vol.Name)

	// Volume creation is async — poll until it appears in list
	t.Log("  waiting for volume to appear in list...")
	volumeVisible := false
	for i := 0; i < 12; i++ {
		time.Sleep(10 * time.Second)
		vols, err := volumeSvc.List(volCtx, regionSlug, projectSlug)
		if err != nil {
			t.Fatalf("listing volumes while waiting: %v", err)
		}
		for _, v := range vols {
			if v.Slug == volSlug {
				volumeVisible = true
				t.Logf("  volume ready: slug=%s", v.Slug)
				break
			}
		}
		if volumeVisible {
			break
		}
		t.Log("    still waiting...")
	}
	if !volumeVisible {
		t.Fatal("volume never appeared in list after 2 minutes")
	}

	// ── Step 8: Attach Volume to Instance ───────────────────────────────
	t.Log("=== Step 8: Attach Volume to Instance ===")
	attachCtx := testContext(t, 2*time.Minute)
	resp, err := volumeSvc.Attach(attachCtx, volSlug, vmSlug)
	if err != nil {
		t.Fatalf("attaching volume: %v", err)
	}
	t.Logf("  attached volume %s to instance %s: status=%s message=%s", volSlug, vmSlug, resp.Status, resp.Message)

	// ── Step 9: List Volumes — Verify Attachment ────────────────────────
	t.Log("=== Step 9: Verify Volume Attachment ===")
	attachmentCtx := testContext(t, 2*time.Minute)
	waitForVolumeAttached(t, volumeSvc, attachmentCtx, regionSlug, projectSlug, volSlug, vm.ID)

	// ── Step 10: Create Volume Snapshot ──────────────────────────────────
	t.Log("=== Step 10: Create Volume Snapshot ===")
	snapCtx := testContext(t, 2*time.Minute)

	// Snapshot only the data volume created by this test. Never infer a root
	// volume from the account-wide list, which could select another user's VM.
	snapName := testID() + "-snap"
	snap, err := snapshotSvc.Create(snapCtx, volSlug, snapshot.CreateRequest{
		Name:          snapName,
		Plan:          snapshotPlanSlug,
		Service:       "VM Snapshot",
		Project:       projectSlug,
		CloudProvider: cloudProviderSlug,
		Region:        regionSlug,
		BillingCycle:  billingCycle,
	})
	if err != nil {
		if createdSlug, lookupErr := snapshotSlugByName(testContext(t, time.Minute), snapshotSvc, regionSlug, projectSlug, snapName); lookupErr != nil {
			t.Logf("  could not check whether snapshot %q was created: %v", snapName, lookupErr)
		} else {
			snapSlug = createdSlug
		}
		t.Fatalf("creating snapshot: %v", err)
	}
	snapSlug = snap.Slug
	if snapSlug == "" {
		if createdSlug, lookupErr := snapshotSlugByName(testContext(t, time.Minute), snapshotSvc, regionSlug, projectSlug, snapName); lookupErr != nil {
			t.Fatalf("snapshot create returned an empty slug and lookup failed: %v", lookupErr)
		} else if createdSlug == "" {
			t.Fatal("creating snapshot returned an empty slug")
		} else {
			snapSlug = createdSlug
		}
	}
	t.Logf("  snapshot create response: slug=%s name=%s", snap.Slug, snap.Name)

	// Snapshot creation is async — poll until it appears in list
	t.Log("  waiting for snapshot to appear...")
	snapshotVisible := false
	for i := 0; i < 12; i++ {
		time.Sleep(10 * time.Second)
		snaps, err := snapshotSvc.List(snapCtx, regionSlug, projectSlug)
		if err != nil {
			t.Fatalf("listing snapshots while waiting: %v", err)
		}
		for _, s := range snaps {
			if s.Slug == snapSlug {
				snapshotVisible = true
				t.Logf("  snapshot ready: slug=%s", s.Slug)
				break
			}
		}
		if snapshotVisible {
			break
		}
		t.Log("    still waiting...")
	}

	// ── Step 11: Verify Snapshot in List ─────────────────────────────────
	t.Log("=== Step 11: Verify Snapshot in List ===")
	if snapshotVisible {
		t.Logf("  snapshot %s confirmed in list", snapSlug)
	} else {
		t.Fatal("snapshot never appeared in list after 2 minutes")
	}

	// ── Step 12: Stop Instance and Revert Snapshot ───────────────────────
	t.Log("=== Step 12: Stop Instance for Snapshot Revert ===")
	stopForRevertCtx := testContext(t, 10*time.Minute)
	if _, err := instanceSvc.Stop(stopForRevertCtx, vmSlug); err != nil {
		t.Fatalf("stopping instance before reverting snapshot: %v", err)
	}
	waitForInstance(t, instanceSvc, stopForRevertCtx, vmSlug, "Stopped", "stopped")
	t.Log("=== Step 12: Revert Snapshot ===")
	revertCtx := testContext(t, 60*time.Second)
	_, err = snapshotSvc.Revert(revertCtx, volSlug, snapSlug)
	if err != nil {
		t.Fatalf("reverting snapshot %s: %v", snapSlug, err)
	}
	t.Logf("  reverted snapshot %s on volume %s", snapSlug, volSlug)
	time.Sleep(5 * time.Second)

	// ── Step 13: Delete Tag ──────────────────────────────────────────────
	t.Log("=== Step 13: Delete Tag ===")
	tagDelCtx := testContext(t, 30*time.Second)
	if err := instanceSvc.DeleteTag(tagDelCtx, vmSlug, "lifecycle-test"); err != nil {
		t.Fatalf("deleting tag: %v", err)
	}
	t.Logf("  deleted tag lifecycle-test from %s", vmSlug)
	tagCreated = false // prevent cleanup double-delete

	// ── Step 14: Detach Volume ───────────────────────────────────────────
	t.Log("=== Step 14: Detach Volume ===")
	detachCtx := testContext(t, 2*time.Minute)
	_, err = volumeSvc.Detach(detachCtx, volSlug)
	if err != nil {
		t.Fatalf("detaching volume: %v", err)
	}
	t.Logf("  detached volume %s", volSlug)
	waitForVolumeDetached(t, volumeSvc, detachCtx, regionSlug, projectSlug, volSlug)

	// ── Step 15: Start Instance ──────────────────────────────────────────
	t.Log("=== Step 15: Start Instance ===")
	startCtx := testContext(t, 10*time.Minute)
	_, err = instanceSvc.Start(startCtx, vmSlug)
	if err != nil {
		t.Fatalf("starting instance: %v", err)
	}
	waitForInstance(t, instanceSvc, startCtx, vmSlug, "Running", "running")

	// ── Step 16: Stop Instance (for cleanup) ─────────────────────────────
	t.Log("=== Step 16: Stop Instance (for cleanup) ===")
	stop2Ctx := testContext(t, 10*time.Minute)
	_, err = instanceSvc.Stop(stop2Ctx, vmSlug)
	if err != nil {
		t.Fatalf("stopping instance for cleanup: %v", err)
	}
	waitForInstance(t, instanceSvc, stop2Ctx, vmSlug, "Stopped", "stopped")

	// ── Step 17: Delete Snapshot ─────────────────────────────────────────
	t.Log("=== Step 17: Delete Snapshot ===")
	deleteSnapshotCtx := testContext(t, 5*time.Minute)
	if err := snapshotSvc.Delete(deleteSnapshotCtx, snapSlug); err != nil {
		t.Fatalf("deleting snapshot %s: %v", snapSlug, err)
	}
	waitForSnapshotAbsent(t, snapshotSvc, deleteSnapshotCtx, regionSlug, projectSlug, snapSlug)
	t.Logf("  deleted snapshot %s", snapSlug)
	snapSlug = ""

	// ── Step 18: Delete Data Volume ──────────────────────────────────────
	t.Log("=== Step 18: Delete Data Volume ===")
	deleteVolumeCtx := testContext(t, 5*time.Minute)
	if err := volumeSvc.Delete(deleteVolumeCtx, volSlug); err != nil {
		t.Fatalf("deleting data volume %s: %v", volSlug, err)
	}
	waitForVolumeAbsent(t, volumeSvc, deleteVolumeCtx, regionSlug, projectSlug, volSlug)
	t.Logf("  deleted data volume %s", volSlug)
	volSlug = ""

	// ── Step 19: Cancel Instance ─────────────────────────────────────────
	t.Log("=== Step 19: Cancel Instance ===")
	deleteVMCtx := testContext(t, 10*time.Minute)
	if err := instanceSvc.Delete(deleteVMCtx, vmSlug, false, false); err != nil {
		t.Fatalf("cancelling instance %s: %v", vmSlug, err)
	}
	waitForInstanceAbsent(t, instanceSvc, deleteVMCtx, regionSlug, projectSlug, vmSlug)
	t.Logf("  cancellation requested for instance %s", vmSlug)
	vmSlug = ""

	// ── Step 20: Delete Isolated Network ─────────────────────────────────
	t.Log("=== Step 20: Delete Isolated Network ===")
	deleteNetworkCtx := testContext(t, 5*time.Minute)
	if err := networkSvc.Delete(deleteNetworkCtx, networkSlug); err != nil {
		t.Fatalf("deleting isolated network %s: %v", networkSlug, err)
	}
	waitForNetworkAbsent(t, networkSvc, deleteNetworkCtx, regionSlug, projectSlug, networkSlug)
	t.Logf("  deleted isolated network %s", networkSlug)
	networkSlug = ""

	if !t.Failed() {
		t.Log("")
		t.Log("========================================")
		t.Log("  FULL LIFECYCLE TEST PASSED")
		t.Log("========================================")
	}
}

// ─── Phase 4: Parallel Read-Only Smoke Tests ────────────────────────────────

func TestPhase4_ReadOnlySmoke(t *testing.T) {
	client := setupClient(t)

	regionSlug, _ := detectRegion(t, client)
	projectSlug := detectProject(t, client)

	tests := []struct {
		name string
		fn   func(t *testing.T)
	}{
		{"InstanceList", func(t *testing.T) {
			ctx := testContext(t, 30*time.Second)
			vms, err := instance.NewService(client).List(ctx, regionSlug, projectSlug)
			if err != nil {
				t.Fatalf("instance list: %v", err)
			}
			t.Logf("found %d instances", len(vms))
		}},
		{"VolumeList", func(t *testing.T) {
			ctx := testContext(t, 30*time.Second)
			vols, err := volume.NewService(client).List(ctx, regionSlug, projectSlug)
			if err != nil {
				t.Fatalf("volume list: %v", err)
			}
			t.Logf("found %d volumes", len(vols))
		}},
		{"SnapshotList", func(t *testing.T) {
			ctx := testContext(t, 30*time.Second)
			snaps, err := snapshot.NewService(client).List(ctx, regionSlug, projectSlug)
			if err != nil {
				t.Fatalf("snapshot list: %v", err)
			}
			t.Logf("found %d snapshots", len(snaps))
		}},
		{"TemplateList", func(t *testing.T) {
			ctx := testContext(t, 30*time.Second)
			tmpls, err := template.NewService(client).List(ctx, regionSlug)
			if err != nil {
				t.Fatalf("template list: %v", err)
			}
			t.Logf("found %d templates", len(tmpls))
		}},
		{"VMPlans", func(t *testing.T) {
			ctx := testContext(t, 30*time.Second)
			plans, err := plan.NewService(client).List(ctx, plan.ServiceVM, regionSlug)
			if err != nil {
				t.Fatalf("VM plans: %v", err)
			}
			t.Logf("found %d VM plans", len(plans))
		}},
		{"BlockStoragePlans", func(t *testing.T) {
			ctx := testContext(t, 30*time.Second)
			plans, err := plan.NewService(client).List(ctx, plan.ServiceBlockStorage, regionSlug)
			if err != nil {
				t.Fatalf("block storage plans: %v", err)
			}
			t.Logf("found %d block storage plans", len(plans))
		}},
		{"NetworkList", func(t *testing.T) {
			ctx := testContext(t, 30*time.Second)
			nets, err := network.NewService(client).List(ctx, regionSlug, projectSlug)
			if err != nil {
				t.Fatalf("network list: %v", err)
			}
			t.Logf("found %d networks", len(nets))
		}},
		{"RegionList", func(t *testing.T) {
			ctx := testContext(t, 30*time.Second)
			regions, err := region.NewService(client).List(ctx)
			if err != nil {
				t.Fatalf("region list: %v", err)
			}
			t.Logf("found %d regions", len(regions))
		}},
		{"SSHKeyList", func(t *testing.T) {
			ctx := testContext(t, 30*time.Second)
			keys, err := sshkey.NewService(client).List(ctx)
			if err != nil {
				t.Fatalf("ssh key list: %v", err)
			}
			t.Logf("found %d SSH keys", len(keys))
		}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			tc.fn(t)
		})
	}
}

// ─── Full Summary Runner ────────────────────────────────────────────────────

func TestLifecycleSummary(t *testing.T) {
	t.Log("")
	t.Log("╔═══════════════════════════════════════════════════╗")
	t.Log("║    ZCP FULL LIFECYCLE INTEGRATION TEST SUITE      ║")
	t.Log("╠═══════════════════════════════════════════════════╣")
	t.Log("║  Phase 1: SSH Key       create → list → delete    ║")
	t.Log("║  Phase 3: Instance      create → tag → volume →   ║")
	t.Log("║           snapshot → stop → start                  ║")
	t.Log("║  Phase 4: Read-only     smoke tests (parallel)    ║")
	t.Log("╚═══════════════════════════════════════════════════╝")
	t.Log("")
	t.Log("Run with: go test -tags integration -v -timeout 30m ./tests/integration/")
	t.Log("Set ZCP_TEST_DEBUG=1 for HTTP debug output")
	t.Log("")

	_ = setupClient(t)
}
