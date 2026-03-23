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
func buildClientAndPrinter(cmd *cobra.Command) (*config.GlobalFlags, *httpclient.Client, *output.Printer, error) {
	// Read global persistent flags from root
	profileName, _ := cmd.Root().PersistentFlags().GetString("profile")
	outputFmt, _ := cmd.Root().PersistentFlags().GetString("output")
	apiURL, _ := cmd.Root().PersistentFlags().GetString("api-url")
	timeoutSec, _ := cmd.Root().PersistentFlags().GetInt("timeout")
	debugFlag, _ := cmd.Root().PersistentFlags().GetBool("debug")
	noColor, _ := cmd.Root().PersistentFlags().GetBool("no-color")
	pager, _ := cmd.Root().PersistentFlags().GetBool("pager")

	flags := &config.GlobalFlags{
		Profile: profileName,
		Output:  outputFmt,
		APIURL:  apiURL,
		Timeout: timeoutSec,
		Debug:   debugFlag,
		NoColor: noColor,
	}

	cfg, err := config.Load()
	if err != nil {
		return flags, nil, nil, fmt.Errorf("loading config: %w", err)
	}

	profile, err := config.ResolveProfile(cfg, profileName)
	if err != nil {
		return flags, nil, nil, err
	}

	baseURL := config.ActiveAPIURL(profile, apiURL)
	opts := httpclient.Options{
		BaseURL:   baseURL,
		APIKey:    profile.APIKey,
		SecretKey: profile.SecretKey,
		Timeout:   time.Duration(timeoutSec) * time.Second,
		Debug:     debugFlag,
		DebugOut:  os.Stderr,
	}

	client := httpclient.New(opts)
	printer := output.NewPrinter(os.Stdout, output.ParseFormat(outputFmt), noColor)
	printer.SetPager(pager)

	return flags, client, printer, nil
}

// getTimeout reads the --timeout persistent flag value from the command's root.
func getTimeout(cmd *cobra.Command) int {
	t, err := cmd.Root().PersistentFlags().GetInt("timeout")
	if err != nil || t <= 0 {
		return 30
	}
	return t
}
