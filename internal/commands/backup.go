package commands

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"github.com/zsoftly/zcp-cli/internal/api/backup"
)

// NewBackupCmd returns the 'backup' cobra command for block storage backups.
func NewBackupCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "backup",
		Short: "Manage block storage backups",
	}
	cmd.AddCommand(newBackupListCmd())
	cmd.AddCommand(newBackupCreateCmd())
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

			backups, err := svc.List(ctx)
			if err != nil {
				return fmt.Errorf("backup list: %w", err)
			}

			headers := []string{"SLUG", "NAME", "VOLUME ID", "INTERVAL", "SERVICE", "CREATED"}
			rows := make([][]string, 0, len(backups))
			for _, b := range backups {
				rows = append(rows, []string{
					b.Slug,
					b.Name,
					b.BlockstorageID,
					b.Interval,
					b.ServiceDisplayName,
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

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a block storage backup",
		Example: `  zcp backup create --volume root-1234 --interval dailyAt --at 1 --immediate 1 --cloud-provider zcp --region yow-1 --billing-cycle hourly --plan backup-1 --project my-project
  zcp backup create --volume root-1234 --interval dailyAt --at 1 --immediate 0 --cloud-provider zcp --region yow-1 --billing-cycle hourly --plan backup-1 --project my-project`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if blockstorageSlug == "" {
				return fmt.Errorf("--volume is required")
			}
			if interval == "" {
				return fmt.Errorf("--interval is required")
			}
			cloudProvider = resolveCloudProvider(cloudProvider)
			if cloudProvider == "" {
				return fmt.Errorf("--cloud-provider is required")
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

			headers := []string{"SLUG", "NAME", "VOLUME ID", "INTERVAL", "SERVICE", "CREATED"}
			rows := [][]string{{
				bak.Slug,
				bak.Name,
				bak.BlockstorageID,
				bak.Interval,
				bak.ServiceDisplayName,
				bak.CreatedAt,
			}}
			return printer.PrintTable(headers, rows)
		},
	}
	cmd.Flags().StringVar(&blockstorageSlug, "volume", "", "Block storage volume slug (required)")
	cmd.Flags().StringVar(&interval, "interval", "", "Backup interval, e.g. dailyAt (required)")
	cmd.Flags().IntVar(&at, "at", 1, "Hour at which the backup triggers (e.g. 1 for 1 AM)")
	cmd.Flags().IntVar(&immediate, "immediate", 0, "Run backup immediately: 1 for yes, 0 for no")
	cmd.Flags().StringVar(&cloudProvider, "cloud-provider", "", "Cloud provider slug (required)")
	cmd.Flags().StringVar(&region, "region", "", "Region slug (required)")
	cmd.Flags().StringVar(&billingCycle, "billing-cycle", "", "Billing cycle slug, e.g. hourly (required)")
	cmd.Flags().StringVar(&plan, "plan", "", "Plan slug, e.g. backup-1 (required)")
	cmd.Flags().StringVar(&pseudoService, "pseudo-service", "Virtual Machine Backup", "Service type for the backup")
	cmd.Flags().StringVar(&project, "project", "", "Project slug (required)")
	return cmd
}
