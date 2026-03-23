package commands

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
	"github.com/zsoftly/zcp-cli/internal/api/zone"
	"github.com/zsoftly/zcp-cli/internal/config"
	"github.com/zsoftly/zcp-cli/internal/httpclient"
	"github.com/zsoftly/zcp-cli/internal/output"
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
		Long: `Validates credentials by making a test API call (zone list).
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
				BaseURL:   baseURL,
				APIKey:    profile.APIKey,
				SecretKey: profile.SecretKey,
				Timeout:   time.Duration(timeoutSec) * time.Second,
				Debug:     debugFlag,
				DebugOut:  os.Stderr,
			}

			client := httpclient.New(opts)
			svc := zone.NewService(client)

			ctx, cancel := context.WithTimeout(context.Background(), opts.Timeout)
			defer cancel()

			printer := output.NewPrinter(os.Stdout, output.FormatTable, noColor)

			fmt.Fprintf(os.Stdout, "Validating credentials for profile %q against %s...\n", profile.Name, baseURL)

			_, err = svc.List(ctx, "")
			if err != nil {
				fmt.Fprintln(os.Stderr, "Validation FAILED:", err)
				return fmt.Errorf("credential validation failed")
			}

			printer.Fprintf("Credentials are valid.\n")
			return nil
		},
	}
}
