// Package commands implements ZCP CLI cobra commands.
package commands

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/zsoftly/zcp-cli/internal/config"
	"github.com/zsoftly/zcp-cli/internal/output"
	"github.com/zsoftly/zcp-cli/pkg/api/cloudprovider"
	"github.com/zsoftly/zcp-cli/pkg/httpclient"
)

// buildClientAndPrinter is a helper used by all read commands to:
// 1. Load config and resolve the active profile
// 2. Build an httpclient using profile credentials
// 3. Build an output.Printer using the --output flag
// Returns the resolved Profile so callers can read profile defaults (e.g. DefaultZone).
func buildClientAndPrinter(cmd *cobra.Command) (*config.Profile, *httpclient.Client, *output.Printer, error) {
	// Read global persistent flags from root
	profileName, _ := cmd.Root().PersistentFlags().GetString("profile")
	outputFmt, _ := cmd.Root().PersistentFlags().GetString("output")
	apiURL, _ := cmd.Root().PersistentFlags().GetString("api-url")
	timeoutSec, _ := cmd.Root().PersistentFlags().GetInt("timeout")
	noColor, _ := cmd.Root().PersistentFlags().GetBool("no-color")
	pager, _ := cmd.Root().PersistentFlags().GetBool("pager")
	debugFlag := debugEnabled(cmd)

	// Apply environment variable overrides for global flags
	if envOutput := os.Getenv("ZCP_OUTPUT"); envOutput != "" {
		outputFmt = envOutput
	}

	cfg, err := config.Load()
	if err != nil {
		return nil, nil, nil, fmt.Errorf("loading config: %w", err)
	}

	profile, err := config.ResolveProfile(cfg, profileName)
	if err != nil {
		return nil, nil, nil, err
	}

	baseURL := config.ActiveAPIURL(profile, apiURL)
	opts := httpclient.Options{
		BaseURL:     baseURL,
		BearerToken: profile.BearerToken,
		Timeout:     time.Duration(timeoutSec) * time.Second,
		Debug:       debugFlag,
		DebugOut:    os.Stderr,
	}

	client := httpclient.New(opts)
	printer := output.NewPrinter(os.Stdout, output.ParseFormat(outputFmt), noColor)
	printer.SetPager(pager)

	return profile, client, printer, nil
}

func debugEnabled(cmd *cobra.Command) bool {
	debugFlag, _ := cmd.Root().PersistentFlags().GetBool("debug")
	if v := strings.ToLower(os.Getenv("ZCP_DEBUG")); v == "true" || v == "1" || v == "yes" {
		debugFlag = true
	}
	return debugFlag
}

// resolveProject returns flagProject if set, otherwise the ZCP_PROJECT env var.
func resolveProject(flagProject string) string {
	if flagProject != "" {
		return flagProject
	}
	return os.Getenv("ZCP_PROJECT")
}

// resolveRegion returns flagRegion if set, otherwise the ZCP_REGION env var.
func resolveRegion(flagRegion string) string {
	if flagRegion != "" {
		return flagRegion
	}
	return os.Getenv("ZCP_REGION")
}

// profileScopeDefaults returns the active profile's stored region/project
// defaults for the profile selected by --profile/ZCP_PROFILE. It mirrors the
// fallback the root scope gate applies, so the gate and the command layer always
// resolve region/project from the same source.
func profileScopeDefaults(cmd *cobra.Command) (region, project string) {
	profileName, _ := cmd.Root().PersistentFlags().GetString("profile")
	return config.ScopeDefaults(profileName)
}

// scopedRegionProject returns the resolved region and project for a
// region/project-scoped command, using the precedence flag > env > active
// profile default — the SAME precedence the root scope gate enforces. The gate
// guarantees at least one source is set before the command runs; consulting the
// profile here ensures a configured user (who satisfied the gate via a stored
// default) actually gets that region/project as a filter instead of an empty,
// unscoped listing.
func scopedRegionProject(cmd *cobra.Command) (region, project string) {
	r, _ := cmd.Flags().GetString("region")
	p, _ := cmd.Flags().GetString("project")
	region, project = strings.TrimSpace(resolveRegion(r)), strings.TrimSpace(resolveProject(p))
	if region == "" || project == "" {
		pr, pp := profileScopeDefaults(cmd)
		pr, pp = strings.TrimSpace(pr), strings.TrimSpace(pp)
		if region == "" {
			region = pr
		}
		if project == "" {
			project = pp
		}
	}
	return region, project
}

// requireRegion resolves a region using flag > ZCP_REGION > active profile
// default and errors if none is set. Region-specific catalog listings (plans,
// templates, images, storage categories) must never run unscoped, or they
// return entries from other regions that are invalid for — and will fail to
// deploy in — the target region. It consults the profile default so it agrees
// with the root scope gate (which accepts the same fallback); without that a
// configured user would pass the gate yet be rejected here.
func requireRegion(cmd *cobra.Command, flagRegion string) (string, error) {
	region := strings.TrimSpace(resolveRegion(flagRegion))
	if region == "" {
		region, _ = profileScopeDefaults(cmd)
		region = strings.TrimSpace(region)
	}
	if region == "" {
		return "", fmt.Errorf("--region is required (or set ZCP_REGION, or `zcp profile add` a default) — " +
			"this catalog is region-specific and entries from another region will not deploy here. " +
			"See 'zcp region list'")
	}
	return region, nil
}

// cloudProviderFlagOrEnv returns flagCloudProvider if set, otherwise the
// ZCP_CLOUD_PROVIDER env var. It does NOT consult the stored profile default —
// used by commands that supply their own service-specific default (e.g. object
// storage defaults to "ceph").
func cloudProviderFlagOrEnv(flagCloudProvider string) string {
	if flagCloudProvider != "" {
		return flagCloudProvider
	}
	return os.Getenv("ZCP_CLOUD_PROVIDER")
}

// resolveCloudProvider resolves the cloud-provider slug for create commands using
// the precedence flag > ZCP_CLOUD_PROVIDER > the slug auto-detected and stored on
// the active profile (see detectCloudProvider). Returns "" if none can be found,
// in which case the caller errors with guidance to run `zcp auth validate`.
func resolveCloudProvider(cmd *cobra.Command, flagCloudProvider string) string {
	if v := cloudProviderFlagOrEnv(flagCloudProvider); v != "" {
		return v
	}
	profileName, _ := cmd.Root().PersistentFlags().GetString("profile")
	cfg, err := config.Load()
	if err != nil {
		return ""
	}
	p, err := config.ResolveProfile(cfg, profileName)
	if err != nil {
		return ""
	}
	return p.CloudProvider
}

// computeServiceName is the catalog service that identifies the primary
// infrastructure ("compute") cloud provider. Verified against the production
// /cloud-providers endpoint, which returns three active providers:
//   - "nimbo" (display "Cloud Stack"): serves "Virtual Machine" plus 19 more —
//     Block Storage, Network, VPC, Kubernetes, Load Balancer, IP Address,
//     snapshots, ISO, autoscale, monitoring, storage tiers. This is the slug
//     used by every create command except the two below.
//   - "ceph" (display "Ceph"): serves only "Object Storage" — the object-storage
//     command defaults to it directly.
//   - "dns" (display "Dns"): serves only "Dns Domain" — the dns command defaults
//     to it directly.
//
// Picking the provider that advertises "Virtual Machine" deterministically
// selects the compute provider regardless of how many others exist.
const computeServiceName = "Virtual Machine"

// detectCloudProvider fetches the account's cloud providers and stores the
// primary compute provider's slug on the named profile, so future create
// commands need not ask for it. The compute provider is the active provider
// whose service catalog includes "Virtual Machine"; if none advertises it but
// exactly one provider is active, that one is used. It is a no-op (returning the
// existing value) when the profile already has a cloud provider, and returns ""
// without error when it cannot be determined (env-only profile absent from the
// config file, or no unambiguous match).
func detectCloudProvider(ctx context.Context, client *httpclient.Client, cfg *config.Config, profileName string) (string, error) {
	prof, ok := cfg.Profiles[profileName]
	if !ok {
		return "", nil
	}
	if prof.CloudProvider != "" {
		return prof.CloudProvider, nil
	}

	providers, err := cloudprovider.NewService(client).List(ctx)
	if err != nil {
		return "", err
	}

	var active []cloudprovider.CloudProvider
	for _, p := range providers {
		if p.Status {
			active = append(active, p)
		}
	}

	chosen := ""
	for _, p := range active {
		for _, svc := range p.Services {
			if svc == computeServiceName {
				chosen = p.Slug
				break
			}
		}
		if chosen != "" {
			break
		}
	}
	if chosen == "" && len(active) == 1 {
		chosen = active[0].Slug
	}
	if chosen == "" {
		return "", nil
	}

	// Re-load immediately before the mutate-and-save so we persist onto the
	// newest on-disk config rather than the snapshot taken when the command
	// started — this narrows the window in which a concurrent `zcp` write to a
	// different profile could be clobbered (Save rewrites the whole file).
	save := cfg
	if fresh, ferr := config.Load(); ferr == nil {
		if _, ok := fresh.Profiles[profileName]; ok {
			save = fresh
		}
	}
	prof = save.Profiles[profileName]
	prof.CloudProvider = chosen
	save.Profiles[profileName] = prof
	if err := config.Save(save); err != nil {
		return "", err
	}
	// Reflect the change in the caller's in-memory config too.
	cfg.Profiles[profileName] = prof
	return chosen, nil
}

// getTimeout reads the --timeout persistent flag value from the command's root.
func getTimeout(cmd *cobra.Command) int {
	t, err := cmd.Root().PersistentFlags().GetInt("timeout")
	if err != nil || t <= 0 {
		return 30
	}
	return t
}

// autoApproved returns true if the global --auto-approve / -y flag is set.
func autoApproved(cmd *cobra.Command) bool {
	v, _ := cmd.Root().PersistentFlags().GetBool("auto-approve")
	return v
}

// confirmAction prompts the user for confirmation unless --auto-approve is set.
// Returns true if the action should proceed, false if cancelled.
func confirmAction(cmd *cobra.Command, format string, args ...interface{}) bool {
	if autoApproved(cmd) {
		return true
	}
	msg := fmt.Sprintf(format, args...)
	fmt.Fprintf(cmd.ErrOrStderr(), "%s [y/N]: ", msg)
	var confirm string
	fmt.Fscanln(cmd.InOrStdin(), &confirm)
	return confirm == "y" || confirm == "Y"
}

// looksLikeUUID reports whether s has the canonical 8-4-4-4-12 UUID shape.
func looksLikeUUID(s string) bool {
	if len(s) != 36 {
		return false
	}
	for i, c := range s {
		switch i {
		case 8, 13, 18, 23:
			if c != '-' {
				return false
			}
		default:
			isHex := (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')
			if !isHex {
				return false
			}
		}
	}
	return true
}
