package commands

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/zsoftly/zcp-cli/pkg/api/apierrors"
	"github.com/zsoftly/zcp-cli/pkg/api/vmbackup"
)

// NewVMBackupCmd returns the 'vm-backup' cobra command.
func NewVMBackupCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "vm-backup",
		Short: "Manage VM backups",
	}
	cmd.AddCommand(newVMBackupListCmd())
	cmd.AddCommand(newVMBackupCreateCmd())
	cmd.AddCommand(newVMBackupDeleteCmd())
	return cmd
}

// ─── List ───────────────────────────────────────────────────────────────────

func newVMBackupListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "list",
		Short:   "List VM backups",
		Example: `  zcp vm-backup list`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runVMBackupList(cmd)
		},
	}
	return cmd
}

func runVMBackupList(cmd *cobra.Command) error {
	_, client, printer, err := buildClientAndPrinter(cmd)
	if err != nil {
		return err
	}

	svc := vmbackup.NewService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	region, project := scopedRegionProject(cmd)
	backups, err := svc.List(ctx, region, project)
	if err != nil {
		return fmt.Errorf("vm-backup list: %w", err)
	}

	headers := []string{"SLUG", "NAME", "VM", "INTERVAL", "AT", "SCHEDULED AT", "CREATED"}
	rows := make([][]string, 0, len(backups))
	for _, b := range backups {
		rows = append(rows, []string{
			b.Slug,
			b.Name,
			b.VMSlug(),
			b.Interval,
			strconv.Itoa(b.At),
			b.ScheduledAt,
			b.CreatedAt,
		})
	}
	return printer.PrintTable(headers, rows)
}

// ─── Create ─────────────────────────────────────────────────────────────────

func newVMBackupCreateCmd() *cobra.Command {
	var (
		interval      string
		at            int
		immediate     int
		cloudProvider string
		region        string
		billingCycle  string
		plan          string
		pseudoService string
		project       string
		isVMSnapshot  bool
		coupon        string
	)

	// Backup plans are region-specific (`zcp plan backup --region yul-1` returns
	// backup-yul; yow-1 returns backup-yow), verified against the live catalog.
	cmd := &cobra.Command{
		Use:   "create <vm-slug>",
		Short: "Create a VM backup",
		Args:  exactArgs(1),
		Example: `  zcp vm-backup create my-vm --interval dailyAt --region yul-1 --billing-cycle hourly --plan backup-yul --pseudo-service vm-backup --project default-9
  zcp vm-backup create my-vm --interval dailyAt --immediate 1 --region yul-1 --billing-cycle hourly --plan backup-yul --pseudo-service vm-backup --project default-9`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validateBackupInterval(interval); err != nil {
				return err
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
			if pseudoService == "" {
				return fmt.Errorf("--pseudo-service is required")
			}
			project = resolveProject(project)
			if project == "" {
				return fmt.Errorf("--project is required")
			}
			if at < 0 || at > 23 {
				return fmt.Errorf("--at must be between 0 and 23 (hour of day)")
			}
			if immediate != 0 && immediate != 1 {
				return fmt.Errorf("--immediate must be 0 or 1")
			}
			req := vmbackup.CreateRequest{
				Interval:      interval,
				At:            at,
				Immediate:     immediate,
				CloudProvider: cloudProvider,
				Region:        region,
				BillingCycle:  billingCycle,
				Plan:          plan,
				PseudoService: pseudoService,
				Project:       project,
				IsVMSnapshot:  isVMSnapshot,
			}
			if coupon != "" {
				req.Coupon = &coupon
			}
			return runVMBackupCreate(cmd, args[0], req)
		},
	}
	cmd.Flags().StringVar(&interval, "interval", "dailyAt", "Backup interval: dailyAt or hourlyAt")
	cmd.Flags().IntVar(&at, "at", 0, "Hour of day for scheduled backup (0-23)")
	cmd.Flags().IntVar(&immediate, "immediate", 0, "Run backup immediately (1=yes, 0=no)")
	cmd.Flags().StringVar(&cloudProvider, "cloud-provider", "", "Cloud provider slug (optional; auto-detected, override only)")
	cmd.Flags().StringVar(&region, "region", "", "Region slug (required)")
	cmd.Flags().StringVar(&billingCycle, "billing-cycle", "", "Billing cycle slug (required)")
	cmd.Flags().StringVar(&plan, "plan", "", "Backup plan slug (required)")
	cmd.Flags().StringVar(&pseudoService, "pseudo-service", "", "Pseudo service name (required)")
	cmd.Flags().StringVar(&project, "project", "", "Project slug (required)")
	cmd.Flags().BoolVar(&isVMSnapshot, "vm-snapshot", false, "Create as VM snapshot")
	cmd.Flags().StringVar(&coupon, "coupon", "", "Coupon code")
	return cmd
}

func newVMBackupDeleteCmd() *cobra.Command {
	var yes bool

	cmd := &cobra.Command{
		Use:   "delete <backup-slug>",
		Short: "Delete a VM backup schedule",
		Args:  exactArgs(1),
		Long: `Delete a VM backup schedule.

This submits an immediate service-cancellation request via
POST /billing/service-cancel-requests/{slug}, the same workflow 'instance
delete' uses, because the direct DELETE endpoint for VM backups only
supports PUT and always rejects deletion. Deletion runs asynchronously in
the background: a successful response means the request was accepted, not
that the schedule is already gone.`,
		Example: `  zcp vm-backup delete vmb-001001-0001
  zcp vm-backup delete vmb-001001-0001 --yes`,
		RunE: func(cmd *cobra.Command, args []string) error {
			slug := args[0]
			if !yes && !autoApproved(cmd) {
				fmt.Fprintf(os.Stderr, "Delete VM backup schedule %q? This cannot be undone. [y/N]: ", slug)
				scanner := bufio.NewScanner(os.Stdin)
				scanner.Scan()
				answer := strings.TrimSpace(strings.ToLower(scanner.Text()))
				if answer != "y" && answer != "yes" {
					fmt.Fprintln(os.Stderr, "Aborted.")
					return nil
				}
			}
			return runVMBackupDelete(cmd, slug)
		},
	}
	cmd.Flags().BoolVar(&yes, "yes", false, "Skip confirmation prompt")
	return cmd
}

func runVMBackupDelete(cmd *cobra.Command, slug string) error {
	_, client, _, err := buildClientAndPrinter(cmd)
	if err != nil {
		return err
	}
	svc := vmbackup.NewService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	if err := svc.Delete(ctx, slug); err != nil {
		if apierrors.IsResourceNotFound(err) {
			fmt.Fprintf(os.Stderr, "VM backup %q not found. Already deleted.\n", slug)
			return nil
		}
		return fmt.Errorf("vm-backup delete: %w", err)
	}
	fmt.Fprintf(os.Stdout, "Deletion requested for VM backup %q. The schedule is being removed in the background.\n", slug)
	return nil
}

func runVMBackupCreate(cmd *cobra.Command, vmSlug string, req vmbackup.CreateRequest) error {
	_, client, _, err := buildClientAndPrinter(cmd)
	if err != nil {
		return err
	}

	svc := vmbackup.NewService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	resp, err := svc.Create(ctx, vmSlug, req)
	if err != nil {
		return fmt.Errorf("vm-backup create: %w", err)
	}

	fmt.Fprintf(cmd.ErrOrStderr(), "VM backup created: %s. %s\n", resp.Status, resp.Message)
	fmt.Fprintln(cmd.ErrOrStderr(), "Run 'zcp vm-backup list' to see the schedule slug.")
	return nil
}
