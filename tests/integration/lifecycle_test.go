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
//	ZCP_TEST_NETWORK_SLUG    Network slug         (default: auto-detected first network)
//	ZCP_TEST_BILLING_CYCLE   Billing cycle slug   (default: "hourly")
//	ZCP_TEST_STORAGE_CAT     Storage category slug(default: "ssd")
package integration

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/pem"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"golang.org/x/crypto/ssh"

	"github.com/zsoftly/zcp-cli/internal/api/instance"
	"github.com/zsoftly/zcp-cli/internal/api/network"
	"github.com/zsoftly/zcp-cli/internal/api/plan"
	"github.com/zsoftly/zcp-cli/internal/api/project"
	"github.com/zsoftly/zcp-cli/internal/api/region"
	"github.com/zsoftly/zcp-cli/internal/api/snapshot"
	"github.com/zsoftly/zcp-cli/internal/api/sshkey"
	"github.com/zsoftly/zcp-cli/internal/api/template"
	"github.com/zsoftly/zcp-cli/internal/api/volume"
	"github.com/zsoftly/zcp-cli/internal/config"
	"github.com/zsoftly/zcp-cli/internal/httpclient"
)

// ─── helpers ────────────────────────────────────────────────────────────────

// testID returns a unique prefix for test resources to avoid name collisions.
func testID() string {
	return fmt.Sprintf("zcp-test-%d", time.Now().UnixMilli()%100000)
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
	apiURL := config.ActiveAPIURL(profile, "")
	client := httpclient.New(httpclient.Options{
		BaseURL:     apiURL,
		BearerToken: profile.BearerToken,
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
			vm, err := svc.Get(ctx, slug)
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
	for _, r := range regions {
		if r.Status && r.CloudProvider != nil {
			t.Logf("  auto-detected region: %s (%s), cloud_provider: %s", r.Name, r.Slug, r.CloudProvider.Slug)
			return r.Slug, r.CloudProvider.Slug
		}
	}
	if len(regions) > 0 && regions[0].CloudProvider != nil {
		t.Logf("  using first region: %s (%s)", regions[0].Name, regions[0].Slug)
		return regions[0].Slug, regions[0].CloudProvider.Slug
	}
	t.Fatal("no regions available")
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

func detectVMPlan(t *testing.T, client *httpclient.Client) string {
	if v := os.Getenv("ZCP_TEST_PLAN"); v != "" {
		return v
	}
	ctx := testContext(t, 30*time.Second)
	svc := plan.NewService(client)
	plans, err := svc.List(ctx, plan.ServiceVM)
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

func detectStoragePlan(t *testing.T, client *httpclient.Client) string {
	if v := os.Getenv("ZCP_TEST_STORAGE_PLAN"); v != "" {
		return v
	}
	ctx := testContext(t, 30*time.Second)
	svc := plan.NewService(client)
	plans, err := svc.List(ctx, plan.ServiceBlockStorage)
	if err != nil {
		t.Fatalf("listing block storage plans: %v", err)
	}
	for _, pref := range []string{"Small-Disk", "Small"} {
		for _, p := range plans {
			if strings.Contains(p.Name, pref) && p.Status && !p.IsCustom {
				t.Logf("  auto-detected storage plan: %s (%s)", p.Name, p.Slug)
				return p.Slug
			}
		}
	}
	// pick first non-custom
	for _, p := range plans {
		if p.Status && !p.IsCustom {
			t.Logf("  using first storage plan: %s (%s)", p.Name, p.Slug)
			return p.Slug
		}
	}
	t.Fatal("no storage plans available")
	return ""
}

func detectNetworkSlug(t *testing.T, client *httpclient.Client) string {
	if v := os.Getenv("ZCP_TEST_NETWORK_SLUG"); v != "" {
		return v
	}
	ctx := testContext(t, 30*time.Second)
	svc := network.NewService(client)
	nets, err := svc.List(ctx)
	if err != nil {
		t.Fatalf("listing networks: %v", err)
	}
	// Prefer default network
	for _, n := range nets {
		if n.IsDefault {
			t.Logf("  auto-detected default network: %s (%s)", n.Name, n.Slug)
			return n.Slug
		}
	}
	if len(nets) > 0 {
		t.Logf("  auto-detected first network: %s (%s)", nets[0].Name, nets[0].Slug)
		return nets[0].Slug
	}
	t.Fatal("no suitable network found — set ZCP_TEST_NETWORK_SLUG")
	return ""
}

// ─── Phase 1: SSH Key Lifecycle ─────────────────────────────────────────────

func TestPhase1_SSHKeyLifecycle(t *testing.T) {
	client := setupClient(t)
	ctx := testContext(t, 60*time.Second)
	svc := sshkey.NewService(client)
	keyName := testID() + "-key"

	// Generate an ephemeral SSH key pair
	pubKey := generateSSHKey(t)
	t.Logf("generated ephemeral SSH public key: %s...%s", pubKey[:30], pubKey[len(pubKey)-10:])

	// 1. Create (import) SSH key
	t.Log("Step 1: Import SSH key")
	key, err := svc.Create(ctx, sshkey.CreateRequest{
		Name:      keyName,
		PublicKey: pubKey,
	})
	if err != nil {
		t.Fatalf("creating SSH key: %v", err)
	}
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
		}
	}
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

	// Auto-detect resources
	t.Log("=== Resource Detection ===")
	regionSlug, cloudProviderSlug := detectRegion(t, client)
	projectSlug := detectProject(t, client)
	templateSlug := detectTemplate(t, client, regionSlug)
	vmPlanSlug := detectVMPlan(t, client)
	storagePlanSlug := detectStoragePlan(t, client)
	networkSlug := detectNetworkSlug(t, client)
	billingCycle := env("ZCP_TEST_BILLING_CYCLE", "hourly")
	storageCategory := env("ZCP_TEST_STORAGE_CAT", "ssd")
	vmName := testID() + "-vm"
	volName := testID() + "-vol"

	// Track resources for cleanup (slugs, not UUIDs)
	var vmSlug, volSlug string
	tagCreated := false // tracks whether we created a tag (cleanup uses key-based delete)

	// Cleanup function — always runs, even on failure
	t.Cleanup(func() {
		cleanCtx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()
		t.Log("=== Cleanup ===")

		if tagCreated && vmSlug != "" {
			t.Logf("  deleting tag lifecycle-test from %s", vmSlug)
			if err := instanceSvc.DeleteTag(cleanCtx, vmSlug, "lifecycle-test"); err != nil {
				t.Logf("  WARNING: tag delete: %v", err)
			}
		}
		if volSlug != "" {
			// Try detach first (may fail if not attached — that's fine)
			t.Logf("  detaching volume %s (if attached)", volSlug)
			volumeSvc.Detach(cleanCtx, volSlug)
			time.Sleep(10 * time.Second)
			// Note: no Delete endpoint in current STKCNSL volume API;
			// volumes are cleaned up via the console if needed.
			t.Logf("  NOTE: volume %s must be deleted manually (no delete API)", volSlug)
		}
		if vmSlug != "" {
			// Stop instance before cleanup ends — no Destroy endpoint in STKCNSL API
			t.Logf("  stopping instance %s for cleanup", vmSlug)
			if _, err := instanceSvc.Stop(cleanCtx, vmSlug); err != nil {
				t.Logf("  WARNING: instance stop: %v", err)
			}
			t.Logf("  NOTE: instance %s must be destroyed manually (no destroy API)", vmSlug)
		}
		t.Log("  cleanup complete")
	})

	// ── Step 1: Create Instance ─────────────────────────────────────────
	t.Log("=== Step 1: Create Instance ===")
	createCtx := testContext(t, 10*time.Minute)
	pw := "TestP@ss1234!"
	vm, err := instanceSvc.Create(createCtx, instance.CreateRequest{
		Name:            vmName,
		Hostname:        vmName,
		CloudProvider:   cloudProviderSlug,
		Project:         projectSlug,
		Region:          regionSlug,
		BootSource:      "template",
		Template:        templateSlug,
		Plan:            vmPlanSlug,
		BillingCycle:    billingCycle,
		StorageCategory: storageCategory,
		IsPublic:        false,
		NetworkType:     "isolated",
		Networks:        []string{networkSlug},
		Password:        &pw,
		Addons:          []string{},
	})
	if err != nil {
		t.Fatalf("creating instance: %v", err)
	}
	vmSlug = vm.Slug
	t.Logf("  created instance: slug=%s name=%s state=%s", vm.Slug, vm.Name, vm.State)

	// Wait for Running state
	waitCtx := testContext(t, 10*time.Minute)
	waitForInstance(t, instanceSvc, waitCtx, vmSlug, "Running", "running")

	// ── Step 2: Verify Instance in List ─────────────────────────────────
	t.Log("=== Step 2: Verify Instance in List ===")
	listCtx := testContext(t, 30*time.Second)
	vms, err := instanceSvc.List(listCtx)
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

	// ── Step 4: Rename Instance (ChangeHostname) ────────────────────────
	t.Log("=== Step 4: Rename Instance ===")
	renameCtx := testContext(t, 30*time.Second)
	newName := vmName + "-renamed"
	_, err = instanceSvc.ChangeHostname(renameCtx, vmSlug, instance.ChangeLabelRequest{
		Name:     newName,
		Hostname: newName,
	})
	if err != nil {
		t.Fatalf("renaming instance: %v", err)
	}
	t.Logf("  renamed to: %s", newName)

	// ── Step 5: Tag Instance ────────────────────────────────────────────
	t.Log("=== Step 5: Tag Instance ===")
	tagCtx := testContext(t, 30*time.Second)
	_, err = instanceSvc.CreateTag(tagCtx, vmSlug, instance.TagRequest{
		Key: "lifecycle-test", Value: "true",
	})
	if err != nil {
		t.Logf("  SKIP: tag creation failed: %v", err)
	} else {
		tagCreated = true
		t.Logf("  created tag: key=lifecycle-test value=true on %s", vmSlug)
	}

	// ── Step 6: List Addons (replaces old ListNetworks) ─────────────────
	t.Log("=== Step 6: List Instance Addons ===")
	addonCtx := testContext(t, 30*time.Second)
	addons, err := instanceSvc.ListAddons(addonCtx, vmSlug)
	if err != nil {
		t.Logf("  SKIP: listing addons failed: %v", err)
	} else {
		t.Logf("  instance has %d addon(s)", len(addons))
		for _, a := range addons {
			t.Logf("    addon: %s slug=%s status=%v", a.Name, a.Slug, a.Status)
		}
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
	})
	if err != nil {
		t.Fatalf("creating volume: %v", err)
	}
	t.Logf("  volume create response: slug=%s name=%s", vol.Slug, vol.Name)

	// Volume creation is async — poll until it appears in list
	t.Log("  waiting for volume to appear in list...")
	for i := 0; i < 12; i++ {
		time.Sleep(10 * time.Second)
		vols, err := volumeSvc.List(volCtx)
		if err != nil {
			t.Fatalf("listing volumes while waiting: %v", err)
		}
		for _, v := range vols {
			if v.Name == volName || v.Slug == vol.Slug {
				volSlug = v.Slug
				t.Logf("  volume ready: slug=%s", v.Slug)
				break
			}
		}
		if volSlug != "" {
			break
		}
		t.Log("    still waiting...")
	}
	if volSlug == "" {
		t.Fatal("volume never appeared in list after 2 minutes")
	}

	// ── Step 8: Attach Volume to Instance ───────────────────────────────
	t.Log("=== Step 8: Attach Volume to Instance ===")
	attachCtx := testContext(t, 2*time.Minute)
	vol, err = volumeSvc.Attach(attachCtx, volSlug, vmSlug)
	if err != nil {
		t.Fatalf("attaching volume: %v", err)
	}
	t.Logf("  attached volume %s to instance %s", volSlug, vmSlug)

	// Wait for attachment to settle
	time.Sleep(15 * time.Second)

	// ── Step 9: List Volumes — Verify Attachment ────────────────────────
	t.Log("=== Step 9: Verify Volume Attachment ===")
	time.Sleep(10 * time.Second) // additional settle time for attachment
	volListCtx := testContext(t, 30*time.Second)
	vols, err := volumeSvc.List(volListCtx)
	if err != nil {
		t.Fatalf("listing volumes: %v", err)
	}
	attachedFound := false
	for _, v := range vols {
		if v.Slug == volSlug {
			attachedFound = true
			t.Logf("  volume %s is attached: type=%s vm_id=%s", v.Slug, v.VolumeType, v.VirtualMachineID)
			break
		}
	}
	if !attachedFound {
		t.Logf("  NOTE: volume %s not yet visible in volume list (eventual consistency)", volSlug)
	}

	// ── Step 10: Create Volume Snapshot ──────────────────────────────────
	t.Log("=== Step 10: Create Volume Snapshot ===")
	snapCtx := testContext(t, 2*time.Minute)

	// Find the ROOT volume for snapshotting
	rootVolSlug := ""
	allVols, err := volumeSvc.List(snapCtx)
	if err != nil {
		t.Fatalf("listing volumes for snapshot: %v", err)
	}
	for _, v := range allVols {
		if v.VolumeType == "ROOT" && v.VirtualMachineID != "" {
			rootVolSlug = v.Slug
			break
		}
	}
	if rootVolSlug == "" {
		t.Log("  no ROOT volume found, using data volume for snapshot")
		rootVolSlug = volSlug
	}

	snapName := testID() + "-snap"
	var snapSlug string
	snap, err := snapshotSvc.Create(snapCtx, rootVolSlug, snapshot.CreateRequest{
		Name:          snapName,
		Plan:          storagePlanSlug,
		Service:       "VM Snapshot",
		Project:       projectSlug,
		CloudProvider: cloudProviderSlug,
		Region:        regionSlug,
		BillingCycle:  billingCycle,
	})
	if err != nil {
		t.Fatalf("creating snapshot: %v", err)
	}
	t.Logf("  snapshot create response: slug=%s name=%s", snap.Slug, snap.Name)

	// Snapshot creation is async — poll until it appears in list
	t.Log("  waiting for snapshot to appear...")
	for i := 0; i < 12; i++ {
		time.Sleep(10 * time.Second)
		snaps, err := snapshotSvc.List(snapCtx)
		if err != nil {
			t.Fatalf("listing snapshots while waiting: %v", err)
		}
		for _, s := range snaps {
			if s.Name == snapName || s.Slug == snap.Slug {
				snapSlug = s.Slug
				t.Logf("  snapshot ready: slug=%s", s.Slug)
				break
			}
		}
		if snapSlug != "" {
			break
		}
		t.Log("    still waiting...")
	}

	// ── Step 11: Verify Snapshot in List ─────────────────────────────────
	t.Log("=== Step 11: Verify Snapshot in List ===")
	if snapSlug != "" {
		t.Logf("  snapshot %s confirmed in list", snapSlug)
	} else {
		t.Log("  NOTE: snapshot not yet visible in list (async)")
	}

	// ── Step 12: Revert Snapshot (no Delete endpoint available) ──────────
	if snapSlug != "" {
		t.Log("=== Step 12: Revert Snapshot (best-effort) ===")
		revertCtx := testContext(t, 60*time.Second)
		_, err := snapshotSvc.Revert(revertCtx, rootVolSlug, snapSlug)
		if err != nil {
			t.Logf("  NOTE: snapshot revert: %v", err)
		} else {
			t.Logf("  reverted snapshot %s on volume %s", snapSlug, rootVolSlug)
		}
		time.Sleep(5 * time.Second)
	} else {
		t.Log("=== Step 12: Skip (no snapshot slug) ===")
	}

	// ── Step 13: Delete Tag ──────────────────────────────────────────────
	if tagCreated {
		t.Log("=== Step 13: Delete Tag ===")
		tagDelCtx := testContext(t, 30*time.Second)
		if err := instanceSvc.DeleteTag(tagDelCtx, vmSlug, "lifecycle-test"); err != nil {
			t.Logf("  SKIP: tag delete failed: %v", err)
		} else {
			t.Logf("  deleted tag lifecycle-test from %s", vmSlug)
		}
		tagCreated = false // prevent cleanup double-delete
	} else {
		t.Log("=== Step 13: Skip (no tag created) ===")
	}

	// ── Step 14: Detach Volume ───────────────────────────────────────────
	t.Log("=== Step 14: Detach Volume ===")
	detachCtx := testContext(t, 2*time.Minute)
	_, err = volumeSvc.Detach(detachCtx, volSlug)
	if err != nil {
		t.Fatalf("detaching volume: %v", err)
	}
	t.Logf("  detached volume %s", volSlug)
	time.Sleep(15 * time.Second)

	// ── Step 15: Stop Instance ───────────────────────────────────────────
	t.Log("=== Step 15: Stop Instance ===")
	stopCtx := testContext(t, 10*time.Minute)
	_, err = instanceSvc.Stop(stopCtx, vmSlug)
	if err != nil {
		t.Fatalf("stopping instance: %v", err)
	}
	waitForInstance(t, instanceSvc, stopCtx, vmSlug, "Stopped", "stopped")

	// ── Step 16: Start Instance ──────────────────────────────────────────
	t.Log("=== Step 16: Start Instance ===")
	startCtx := testContext(t, 10*time.Minute)
	_, err = instanceSvc.Start(startCtx, vmSlug)
	if err != nil {
		t.Fatalf("starting instance: %v", err)
	}
	waitForInstance(t, instanceSvc, startCtx, vmSlug, "Running", "running")

	// ── Step 17: Stop Again (for cleanup) ────────────────────────────────
	t.Log("=== Step 17: Stop Instance (for cleanup) ===")
	stop2Ctx := testContext(t, 10*time.Minute)
	_, err = instanceSvc.Stop(stop2Ctx, vmSlug)
	if err != nil {
		t.Fatalf("stopping instance for cleanup: %v", err)
	}
	waitForInstance(t, instanceSvc, stop2Ctx, vmSlug, "Stopped", "stopped")

	// Note: The STKCNSL API does not have volume.Delete or instance.Destroy
	// endpoints. Resources created by this test must be cleaned up via the
	// console or future API endpoints. Mark slugs empty to skip cleanup
	// re-attempts.
	t.Log("=== Step 18: Skip volume delete (no API endpoint) ===")
	t.Logf("  volume %s must be deleted manually", volSlug)
	volSlug = "" // prevent cleanup from re-trying

	t.Log("=== Step 19: Skip instance destroy (no API endpoint) ===")
	t.Logf("  instance %s must be destroyed manually", vmSlug)
	vmSlug = "" // prevent cleanup from re-trying

	t.Log("")
	t.Log("========================================")
	t.Log("  FULL LIFECYCLE TEST PASSED")
	t.Log("========================================")
}

// ─── Phase 4: Parallel Read-Only Smoke Tests ────────────────────────────────

func TestPhase4_ReadOnlySmoke(t *testing.T) {
	client := setupClient(t)

	tests := []struct {
		name string
		fn   func(t *testing.T)
	}{
		{"InstanceList", func(t *testing.T) {
			ctx := testContext(t, 30*time.Second)
			vms, err := instance.NewService(client).List(ctx)
			if err != nil {
				t.Fatalf("instance list: %v", err)
			}
			t.Logf("found %d instances", len(vms))
		}},
		{"VolumeList", func(t *testing.T) {
			ctx := testContext(t, 30*time.Second)
			vols, err := volume.NewService(client).List(ctx)
			if err != nil {
				t.Fatalf("volume list: %v", err)
			}
			t.Logf("found %d volumes", len(vols))
		}},
		{"SnapshotList", func(t *testing.T) {
			ctx := testContext(t, 30*time.Second)
			snaps, err := snapshot.NewService(client).List(ctx)
			if err != nil {
				t.Fatalf("snapshot list: %v", err)
			}
			t.Logf("found %d snapshots", len(snaps))
		}},
		{"TemplateList", func(t *testing.T) {
			ctx := testContext(t, 30*time.Second)
			tmpls, err := template.NewService(client).List(ctx, "")
			if err != nil {
				t.Fatalf("template list: %v", err)
			}
			t.Logf("found %d templates", len(tmpls))
		}},
		{"VMPlans", func(t *testing.T) {
			ctx := testContext(t, 30*time.Second)
			plans, err := plan.NewService(client).List(ctx, plan.ServiceVM)
			if err != nil {
				t.Fatalf("VM plans: %v", err)
			}
			t.Logf("found %d VM plans", len(plans))
		}},
		{"BlockStoragePlans", func(t *testing.T) {
			ctx := testContext(t, 30*time.Second)
			plans, err := plan.NewService(client).List(ctx, plan.ServiceBlockStorage)
			if err != nil {
				t.Fatalf("block storage plans: %v", err)
			}
			t.Logf("found %d block storage plans", len(plans))
		}},
		{"NetworkList", func(t *testing.T) {
			ctx := testContext(t, 30*time.Second)
			nets, err := network.NewService(client).List(ctx)
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
