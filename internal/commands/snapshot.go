package commands

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/zsoftly/zcp-cli/internal/api/apierrors"
	"github.com/zsoftly/zcp-cli/internal/api/snapshot"
)

// NewSnapshotCmd returns the 'snapshot' cobra command for block storage snapshots.
func NewSnapshotCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "snapshot",
		Short: "Manage block storage snapshots",
	}
	cmd.AddCommand(newSnapshotListCmd())
	cmd.AddCommand(newSnapshotCreateCmd())
	cmd.AddCommand(newSnapshotRevertCmd())
	cmd.AddCommand(newSnapshotDeleteCmd())
	return cmd
}

func newSnapshotListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List block storage snapshots",
		Example: `  zcp snapshot list
  zcp snapshot list --output json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			_, client, printer, err := buildClientAndPrinter(cmd)
			if err != nil {
				return err
			}
			svc := snapshot.NewService(client)
			ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
			defer cancel()

			snapshots, err := svc.List(ctx)
			if err != nil {
				return fmt.Errorf("snapshot list: %w", err)
			}

			headers := []string{"SLUG", "NAME", "VOLUME ID", "SERVICE", "CREATED"}
			rows := make([][]string, 0, len(snapshots))
			for _, s := range snapshots {
				rows = append(rows, []string{
					s.Slug,
					s.Name,
					s.BlockstorageID,
					s.ServiceDisplayName,
					s.CreatedAt,
				})
			}
			return printer.PrintTable(headers, rows)
		},
	}
	return cmd
}

func newSnapshotCreateCmd() *cobra.Command {
	var blockstorageSlug, name, plan, project, cloudProvider, region, billingCycle, coupon string

	cmd := &cobra.Command{
		Use:     "create",
		Short:   "Create a block storage snapshot",
		Example: `  zcp snapshot create --volume root-1234 --name my-snapshot --plan snapshot-per-gb --cloud-provider nimbo --region yow-1 --billing-cycle hourly --project my-project`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if blockstorageSlug == "" {
				return fmt.Errorf("--volume is required")
			}
			if name == "" {
				return fmt.Errorf("--name is required")
			}
			if plan == "" {
				return fmt.Errorf("--plan is required")
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
			project = resolveProject(project)
			if project == "" {
				return fmt.Errorf("--project is required")
			}
			_, client, printer, err := buildClientAndPrinter(cmd)
			if err != nil {
				return err
			}
			svc := snapshot.NewService(client)
			ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
			defer cancel()

			req := snapshot.CreateRequest{
				Name:          name,
				Plan:          plan,
				Service:       "Block Storage Snapshot",
				CloudProvider: cloudProvider,
				Region:        region,
				BillingCycle:  billingCycle,
				Project:       project,
				Coupon:        coupon,
			}
			snap, err := svc.Create(ctx, blockstorageSlug, req)
			if err != nil {
				return fmt.Errorf("snapshot create: %w", err)
			}

			headers := []string{"SLUG", "NAME", "VOLUME ID", "SERVICE", "CREATED"}
			rows := [][]string{{
				snap.Slug,
				snap.Name,
				snap.BlockstorageID,
				snap.ServiceDisplayName,
				snap.CreatedAt,
			}}
			return printer.PrintTable(headers, rows)
		},
	}
	cmd.Flags().StringVar(&blockstorageSlug, "volume", "", "Block storage volume slug to snapshot (required)")
	cmd.Flags().StringVar(&name, "name", "", "Snapshot name (required)")
	cmd.Flags().StringVar(&plan, "plan", "", "Plan slug, e.g. snapshot-per-gb (required)")
	cmd.Flags().StringVar(&cloudProvider, "cloud-provider", "", "Cloud provider slug (required)")
	cmd.Flags().StringVar(&region, "region", "", "Region slug (required)")
	cmd.Flags().StringVar(&billingCycle, "billing-cycle", "", "Billing cycle slug, e.g. hourly (required)")
	cmd.Flags().StringVar(&project, "project", "", "Project slug (required)")
	cmd.Flags().StringVar(&coupon, "coupon", "", "Coupon code")
	return cmd
}

func newSnapshotRevertCmd() *cobra.Command {
	var yes bool
	var blockstorageSlug string

	cmd := &cobra.Command{
		Use:   "revert <snapshot-slug>",
		Short: "Revert a block storage volume to a snapshot state (DESTRUCTIVE)",
		Args:  cobra.ExactArgs(1),
		Example: `  zcp snapshot revert ss-001001-0001 --volume bs-001001-0042
  zcp snapshot revert ss-001001-0001 --volume bs-001001-0042 --yes`,
		RunE: func(cmd *cobra.Command, args []string) error {
			snapshotSlug := args[0]
			if blockstorageSlug == "" {
				return fmt.Errorf("--volume is required")
			}
			if !yes && !autoApproved(cmd) {
				fmt.Fprintf(os.Stderr, "WARNING: Reverting snapshot %q on volume %q will discard all changes since the snapshot. This cannot be undone. [y/N]: ", snapshotSlug, blockstorageSlug)
				scanner := bufio.NewScanner(os.Stdin)
				scanner.Scan()
				answer := strings.TrimSpace(strings.ToLower(scanner.Text()))
				if answer != "y" && answer != "yes" {
					fmt.Fprintln(os.Stderr, "Aborted.")
					return nil
				}
			}
			_, client, printer, err := buildClientAndPrinter(cmd)
			if err != nil {
				return err
			}
			svc := snapshot.NewService(client)
			ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
			defer cancel()

			snap, err := svc.Revert(ctx, blockstorageSlug, snapshotSlug)
			if err != nil {
				return fmt.Errorf("snapshot revert: %w", err)
			}

			headers := []string{"SLUG", "NAME", "VOLUME ID", "SERVICE", "CREATED"}
			rows := [][]string{{
				snap.Slug,
				snap.Name,
				snap.BlockstorageID,
				snap.ServiceDisplayName,
				snap.CreatedAt,
			}}
			return printer.PrintTable(headers, rows)
		},
	}
	cmd.Flags().BoolVar(&yes, "yes", false, "Skip confirmation prompt")
	cmd.Flags().StringVar(&blockstorageSlug, "volume", "", "Block storage volume slug (required)")
	return cmd
}

func newSnapshotDeleteCmd() *cobra.Command {
	var yes bool

	cmd := &cobra.Command{
		Use:   "delete <snapshot-slug>",
		Short: "Permanently delete a block storage snapshot",
		Args:  cobra.ExactArgs(1),
		Example: `  zcp snapshot delete ss-001001-0001
  zcp snapshot delete ss-001001-0001 --yes`,
		RunE: func(cmd *cobra.Command, args []string) error {
			slug := args[0]
			if !yes && !autoApproved(cmd) {
				fmt.Fprintf(os.Stderr, "Delete snapshot %q? This cannot be undone. [y/N]: ", slug)
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
			svc := snapshot.NewService(client)
			ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
			defer cancel()
			if err := svc.Delete(ctx, slug); err != nil {
				if apierrors.IsResourceNotFound(err) {
					fmt.Fprintf(os.Stderr, "Snapshot %q not found — already deleted.\n", slug)
					return nil
				}
				return fmt.Errorf("snapshot delete: %w", err)
			}
			fmt.Fprintf(os.Stdout, "Snapshot %q deleted.\n", slug)
			return nil
		},
	}
	cmd.Flags().BoolVar(&yes, "yes", false, "Skip confirmation prompt")
	return cmd
}
