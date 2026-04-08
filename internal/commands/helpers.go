// Package commands implements ZCP CLI cobra commands.
package commands

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
	"github.com/zsoftly/zcp-cli/internal/config"
	"github.com/zsoftly/zcp-cli/internal/httpclient"
	"github.com/zsoftly/zcp-cli/internal/output"
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
	debugFlag, _ := cmd.Root().PersistentFlags().GetBool("debug")
	noColor, _ := cmd.Root().PersistentFlags().GetBool("no-color")
	pager, _ := cmd.Root().PersistentFlags().GetBool("pager")

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

// resolveZone returns flagZone if set, otherwise the profile's default zone.
// If neither is set it returns "" and the caller is responsible for the error.
func resolveZone(profile *config.Profile, flagZone string) string {
	if flagZone != "" {
		return flagZone
	}
	if profile != nil {
		return profile.DefaultZone
	}
	return ""
}

// errNoZone is the standard error shown when --zone is missing and no default is set.
func errNoZone() error {
	return fmt.Errorf("--zone is required (or set a default: zcp zone use <uuid>)")
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
