package commands

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/zsoftly/zcp-cli/pkg/api/apierrors"
	"github.com/zsoftly/zcp-cli/pkg/api/backup"
)

// NewBackupCmd returns the 'backup' cobra command for block storage backups.
func NewBackupCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "backup",
		Short: "Manage block storage backups",
	}
	cmd.AddCommand(newBackupListCmd())
	cmd.AddCommand(newBackupCreateCmd())
	cmd.AddCommand(newBackupDeleteCmd())
	return cmd
}

func newBackupListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List block storage backups",
		Example: `  zcp backup list
  zcp backup list --output json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			_, client, printer, err := buildClientAndPrinter(cmd)
			if err != nil {
				return err
			}
			svc := backup.NewService(client)
			ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
			defer cancel()

			region, project := scopedRegionProject(cmd)
			backups, err := svc.List(ctx, region, project)
			if err != nil {
				return fmt.Errorf("backup list: %w", err)
			}

			headers := []string{"SLUG", "NAME", "VOLUME ID", "INTERVAL", "CREATED"}
			rows := make([][]string, 0, len(backups))
			for _, b := range backups {
				rows = append(rows, []string{
					b.Slug,
					b.Name,
					b.BlockstorageID,
					b.Interval,
					b.CreatedAt,
				})
			}
			return printer.PrintTable(headers, rows)
		},
	}
	return cmd
}

func newBackupCreateCmd() *cobra.Command {
	var blockstorageSlug, interval, cloudProvider, region, billingCycle, plan, pseudoService, project string
	var at, immediate int

	// TODO(disabled-plan): `backup-1` is a real plan but backup plans are not yet
	// enabled in the catalog (`zcp plan backup` returns []). Keep the example/help
	// as-is — it works once backup plans are enabled.
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a block storage backup",
		Example: `  zcp backup create --volume root-1234 --interval dailyAt --at 1 --immediate 1 --region yow-1 --billing-cycle hourly --plan backup-1 --project default
  zcp backup create --volume root-1234 --interval dailyAt --at 1 --immediate 0 --region yow-1 --billing-cycle hourly --plan backup-1 --project default`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if blockstorageSlug == "" {
				return fmt.Errorf("--volume is required")
			}
			if interval == "" {
				return fmt.Errorf("--interval is required")
			}
			cloudProvider = resolveCloudProvider(cmd, cloudProvider)
			if cloudProvider == "" {
				return fmt.Errorf("could not determine cloud provider — run 'zcp auth validate' to detect it, or pass --cloud-provider (see 'zcp cloud-provider list')")
			}
			region = resolveRegion(region)
			if region == "" {
				return fmt.Errorf("--region is required")
			}
			if billingCycle == "" {
				return fmt.Errorf("--billing-cycle is required")
			}
			if plan == "" {
				return fmt.Errorf("--plan is required")
			}
			project = resolveProject(project)
			if project == "" {
				return fmt.Errorf("--project is required")
			}
			_, client, printer, err := buildClientAndPrinter(cmd)
			if err != nil {
				return err
			}
			svc := backup.NewService(client)
			ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
			defer cancel()

			if pseudoService == "" {
				pseudoService = "Virtual Machine Backup"
			}

			req := backup.CreateRequest{
				Interval:      interval,
				At:            at,
				Immediate:     immediate,
				CloudProvider: cloudProvider,
				Region:        region,
				BillingCycle:  billingCycle,
				Plan:          plan,
				PseudoService: pseudoService,
				Project:       project,
			}
			bak, err := svc.Create(ctx, blockstorageSlug, req)
			if err != nil {
				return fmt.Errorf("backup create: %w", err)
			}

			headers := []string{"SLUG", "NAME", "VOLUME ID", "INTERVAL", "CREATED"}
			rows := [][]string{{
				bak.Slug,
				bak.Name,
				bak.BlockstorageID,
				bak.Interval,
				bak.CreatedAt,
			}}
			return printer.PrintTable(headers, rows)
		},
	}
	cmd.Flags().StringVar(&blockstorageSlug, "volume", "", "Block storage volume slug (required)")
	cmd.Flags().StringVar(&interval, "interval", "", "Backup interval, e.g. dailyAt (required)")
	cmd.Flags().IntVar(&at, "at", 1, "Hour at which the backup triggers (e.g. 1 for 1 AM)")
	cmd.Flags().IntVar(&immediate, "immediate", 0, "Run backup immediately: 1 for yes, 0 for no")
	cmd.Flags().StringVar(&cloudProvider, "cloud-provider", "", "Cloud provider slug (optional; auto-detected, override only)")
	cmd.Flags().StringVar(&region, "region", "", "Region slug (required)")
	cmd.Flags().StringVar(&billingCycle, "billing-cycle", "", "Billing cycle slug, e.g. hourly (required)")
	cmd.Flags().StringVar(&plan, "plan", "", "Plan slug, e.g. backup-1 (required)")
	cmd.Flags().StringVar(&pseudoService, "pseudo-service", "Virtual Machine Backup", "Service type for the backup")
	cmd.Flags().StringVar(&project, "project", "", "Project slug (required)")
	return cmd
}

func newBackupDeleteCmd() *cobra.Command {
	var yes bool

	cmd := &cobra.Command{
		Use:   "delete <backup-slug>",
		Short: "Permanently delete a block storage backup schedule",
		Args:  exactArgs(1),
		Example: `  zcp backup delete bk-001001-0001
  zcp backup delete bk-001001-0001 --yes`,
		RunE: func(cmd *cobra.Command, args []string) error {
			slug := args[0]
			if !yes && !autoApproved(cmd) {
				fmt.Fprintf(os.Stderr, "Delete backup %q? This cannot be undone. [y/N]: ", slug)
				scanner := bufio.NewScanner(os.Stdin)
				scanner.Scan()
				answer := strings.TrimSpace(strings.ToLower(scanner.Text()))
				if answer != "y" && answer != "yes" {
					fmt.Fprintln(os.Stderr, "Aborted.")
					return nil
				}
			}
			_, client, _, err := buildClientAndPrinter(cmd)
			if err != nil {
				return err
			}
			svc := backup.NewService(client)
			ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
			defer cancel()
			if err := svc.Delete(ctx, slug); err != nil {
				if apierrors.IsResourceNotFound(err) {
					fmt.Fprintf(os.Stderr, "Backup %q not found — already deleted.\n", slug)
					return nil
				}
				return fmt.Errorf("backup delete: %w", err)
			}
			fmt.Fprintf(os.Stdout, "Backup %q deleted.\n", slug)
			return nil
		},
	}
	cmd.Flags().BoolVar(&yes, "yes", false, "Skip confirmation prompt")
	return cmd
}
