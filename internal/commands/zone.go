package commands

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/spf13/cobra"
	"github.com/zsoftly/zcp-cli/internal/api/zone"
	"github.com/zsoftly/zcp-cli/internal/config"
)

// NewZoneCmd returns the 'zone' cobra command.
func NewZoneCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "zone",
		Short: "Manage availability zones",
	}
	cmd.AddCommand(newZoneListCmd())
	cmd.AddCommand(newZoneUseCmd())
	return cmd
}

func newZoneListCmd() *cobra.Command {
	var zoneUUID string

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List availability zones",
		Example: `  zcp zone list
  zcp zone list --zone <uuid>
  zcp zone list --output json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runZoneList(cmd, zoneUUID)
		},
	}
	cmd.Flags().StringVar(&zoneUUID, "zone", "", "Filter by zone UUID")
	return cmd
}

func runZoneList(cmd *cobra.Command, zoneUUID string) error {
	_, client, printer, err := buildClientAndPrinter(cmd)
	if err != nil {
		return err
	}

	svc := zone.NewService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	zones, err := svc.List(ctx, zoneUUID)
	if err != nil {
		return fmt.Errorf("zone list: %w", err)
	}

	headers := []string{"UUID", "NAME", "COUNTRY", "ACTIVE"}
	rows := make([][]string, 0, len(zones))
	for _, z := range zones {
		rows = append(rows, []string{
			z.UUID,
			z.Name,
			z.CountryName,
			strconv.FormatBool(z.IsActive),
		})
	}
	return printer.PrintTable(headers, rows)
}

func newZoneUseCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "use <uuid>",
		Short: "Set the default zone for the active profile",
		Long: `Set the default zone for the active profile.

Once set, all commands that require --zone will use this value automatically.
You can still override it per-command with --zone <uuid>.

To clear the default zone, pass an empty string:
  zcp zone use ""`,
		Example: `  zcp zone use abc123-zone-uuid
  zcp zone use ""`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			profileName, _ := cmd.Root().PersistentFlags().GetString("profile")

			cfg, err := config.Load()
			if err != nil {
				return fmt.Errorf("loading config: %w", err)
			}

			name := profileName
			if name == "" {
				name = cfg.ActiveProfile
			}
			if name == "" {
				return fmt.Errorf("no active profile — run: zcp profile add")
			}

			p, ok := cfg.Profiles[name]
			if !ok {
				return fmt.Errorf("profile %q not found", name)
			}

			p.DefaultZone = args[0]
			cfg.Profiles[name] = p

			if err := config.Save(cfg); err != nil {
				return fmt.Errorf("saving config: %w", err)
			}

			if args[0] == "" {
				fmt.Fprintf(cmd.OutOrStdout(), "Default zone cleared for profile %q\n", name)
			} else {
				fmt.Fprintf(cmd.OutOrStdout(), "Default zone set to %q for profile %q\n", args[0], name)
			}
			return nil
		},
	}
}
