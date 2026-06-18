package commands

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
	"github.com/zsoftly/zcp-cli/internal/config"
	"github.com/zsoftly/zcp-cli/internal/output"
	"github.com/zsoftly/zcp-cli/pkg/api/region"
	"github.com/zsoftly/zcp-cli/pkg/httpclient"
)

// NewAuthCmd returns the 'auth' cobra command.
func NewAuthCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "auth",
		Short: "Authentication operations",
	}
	cmd.AddCommand(newAuthValidateCmd())
	return cmd
}

func newAuthValidateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "validate",
		Short: "Validate API credentials for the active profile",
		Long: `Validates credentials by making a test API call (region list).
If the call succeeds, the credentials are valid.`,
		Example: `  zcp auth validate
  zcp auth validate --profile prod`,
		RunE: func(cmd *cobra.Command, args []string) error {
			profileName, _ := cmd.Root().PersistentFlags().GetString("profile")
			apiURL, _ := cmd.Root().PersistentFlags().GetString("api-url")
			timeoutSec, _ := cmd.Root().PersistentFlags().GetInt("timeout")
			debugFlag, _ := cmd.Root().PersistentFlags().GetBool("debug")
			noColor, _ := cmd.Root().PersistentFlags().GetBool("no-color")

			cfg, err := config.Load()
			if err != nil {
				return err
			}

			profile, err := config.ResolveProfile(cfg, profileName)
			if err != nil {
				return err
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
			svc := region.NewService(client)

			ctx, cancel := context.WithTimeout(context.Background(), opts.Timeout)
			defer cancel()

			printer := output.NewPrinter(os.Stdout, output.FormatTable, noColor)

			switch {
			case os.Getenv("ZCP_BEARER_TOKEN") != "" && profile.Name != "":
				fmt.Fprintf(os.Stdout, "Validating ZCP_BEARER_TOKEN (overrides profile %q) against %s...\n", profile.Name, baseURL)
			case os.Getenv("ZCP_BEARER_TOKEN") != "":
				fmt.Fprintf(os.Stdout, "Validating ZCP_BEARER_TOKEN against %s...\n", baseURL)
			default:
				fmt.Fprintf(os.Stdout, "Validating credentials for profile %q against %s...\n", profile.Name, baseURL)
			}

			_, err = svc.List(ctx)
			if err != nil {
				fmt.Fprintln(os.Stderr, "Validation FAILED:", err)
				return fmt.Errorf("credential validation failed")
			}

			printer.Fprintf("Credentials are valid.\n")

			// Auto-detect and persist the account's cloud provider so create
			// commands no longer need --cloud-provider. Best-effort: never fail
			// validation over this.
			if slug, derr := detectCloudProvider(ctx, client, cfg, profile.Name); derr == nil && slug != "" {
				fmt.Fprintf(os.Stdout, "Cloud provider detected and saved to profile %q: %s\n", profile.Name, slug)
			}
			return nil
		},
	}
}
