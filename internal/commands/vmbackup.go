package commands

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"github.com/zsoftly/zcp-cli/internal/api/vmbackup"
)

// NewVMBackupCmd returns the 'vm-backup' cobra command.
func NewVMBackupCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "vm-backup",
		Short: "Manage VM backups",
	}
	cmd.AddCommand(newVMBackupListCmd())
	cmd.AddCommand(newVMBackupCreateCmd())
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

	backups, err := svc.List(ctx)
	if err != nil {
		return fmt.Errorf("vm-backup list: %w", err)
	}

	headers := []string{"ID", "NAME", "SLUG", "STATE", "SERVICE", "VM ID", "CREATED"}
	rows := make([][]string, 0, len(backups))
	for _, b := range backups {
		rows = append(rows, []string{
			b.ID,
			b.Name,
			b.Slug,
			b.State,
			b.ServiceName,
			b.VirtualMachineID,
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
		psudoService  string
		project       string
		isVMSnapshot  bool
		coupon        string
	)

	cmd := &cobra.Command{
		Use:   "create <vm-slug>",
		Short: "Create a VM backup",
		Args:  cobra.ExactArgs(1),
		Example: `  zcp vm-backup create my-vm --interval daily --cloud-provider zcp --region yow-1 --billing-cycle hourly --plan backup-basic --psudo-service vm-backup --project default-60
  zcp vm-backup create my-vm --interval daily --immediate 1 --cloud-provider zcp --region yow-1 --billing-cycle hourly --plan backup-basic --psudo-service vm-backup --project default-60`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if cloudProvider == "" {
				return fmt.Errorf("--cloud-provider is required")
			}
			if region == "" {
				return fmt.Errorf("--region is required")
			}
			if billingCycle == "" {
				return fmt.Errorf("--billing-cycle is required")
			}
			if plan == "" {
				return fmt.Errorf("--plan is required")
			}
			if psudoService == "" {
				return fmt.Errorf("--psudo-service is required")
			}
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
				PsudoService:  psudoService,
				Project:       project,
				IsVMSnapshot:  isVMSnapshot,
			}
			if coupon != "" {
				req.Coupon = &coupon
			}
			return runVMBackupCreate(cmd, args[0], req)
		},
	}
	cmd.Flags().StringVar(&interval, "interval", "daily", "Backup interval (e.g. daily, weekly)")
	cmd.Flags().IntVar(&at, "at", 0, "Hour of day for scheduled backup (0-23)")
	cmd.Flags().IntVar(&immediate, "immediate", 0, "Run backup immediately (1=yes, 0=no)")
	cmd.Flags().StringVar(&cloudProvider, "cloud-provider", "", "Cloud provider slug (required)")
	cmd.Flags().StringVar(&region, "region", "", "Region slug (required)")
	cmd.Flags().StringVar(&billingCycle, "billing-cycle", "", "Billing cycle slug (required)")
	cmd.Flags().StringVar(&plan, "plan", "", "Backup plan slug (required)")
	cmd.Flags().StringVar(&psudoService, "psudo-service", "", "Pseudo service name (required)")
	cmd.Flags().StringVar(&project, "project", "", "Project slug (required)")
	cmd.Flags().BoolVar(&isVMSnapshot, "vm-snapshot", false, "Create as VM snapshot")
	cmd.Flags().StringVar(&coupon, "coupon", "", "Coupon code")
	return cmd
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

	fmt.Fprintf(cmd.ErrOrStderr(), "VM backup created: %s — %s\n", resp.Status, resp.Message)
	return nil
}
