package commands

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/zsoftly/zcp-cli/internal/api/vmsnapshot"
)

// NewVMSnapshotCmd returns the 'vm-snapshot' cobra command.
func NewVMSnapshotCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "vm-snapshot",
		Short: "Manage VM snapshots (whole-machine snapshots)",
	}
	cmd.AddCommand(newVMSnapshotListCmd())
	cmd.AddCommand(newVMSnapshotCreateCmd())
	cmd.AddCommand(newVMSnapshotDeleteCmd())
	cmd.AddCommand(newVMSnapshotRevertCmd())
	return cmd
}

func newVMSnapshotListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List VM snapshots",
		Example: `  zcp vm-snapshot list
  zcp vm-snapshot list --output json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			_, client, printer, err := buildClientAndPrinter(cmd)
			if err != nil {
				return err
			}
			svc := vmsnapshot.NewService(client)
			ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
			defer cancel()

			snapshots, err := svc.List(ctx)
			if err != nil {
				return fmt.Errorf("vm-snapshot list: %w", err)
			}

			headers := []string{"SLUG", "NAME", "STATE", "VM ID", "REGION", "CREATED"}
			rows := make([][]string, 0, len(snapshots))
			for _, s := range snapshots {
				rows = append(rows, []string{
					s.Slug,
					s.Name,
					s.State,
					s.VirtualMachineID,
					s.RegionID,
					s.CreatedAt,
				})
			}
			return printer.PrintTable(headers, rows)
		},
	}
	return cmd
}

func newVMSnapshotCreateCmd() *cobra.Command {
	var vmSlug, name, plan, billingCycle, project, cloudProvider, region, service string
	var memory bool
	var coupon string

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a VM snapshot",
		Example: `  zcp vm-snapshot create --vm my-vm --name my-snap --plan basic --billing-cycle monthly --project proj-1 --cloud-provider cp-1 --region rgn-1 --service svc-1
  zcp vm-snapshot create --vm my-vm --name my-snap --plan basic --billing-cycle monthly --project proj-1 --cloud-provider cp-1 --region rgn-1 --service svc-1 --memory`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if name == "" {
				return fmt.Errorf("--name is required")
			}
			if vmSlug == "" {
				return fmt.Errorf("--vm is required")
			}
			_, client, printer, err := buildClientAndPrinter(cmd)
			if err != nil {
				return err
			}
			svc := vmsnapshot.NewService(client)
			ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
			defer cancel()

			req := vmsnapshot.CreateRequest{
				Name:          name,
				BillingCycle:  billingCycle,
				Plan:          plan,
				IsMemory:      memory,
				IsVMSnapshot:  true,
				Project:       project,
				CloudProvider: cloudProvider,
				Region:        region,
				Service:       service,
			}
			if coupon != "" {
				req.Coupon = &coupon
			}
			resp, err := svc.Create(ctx, vmSlug, req)
			if err != nil {
				return fmt.Errorf("vm-snapshot create: %w", err)
			}

			printer.Fprintf("VM snapshot created (status: %s, message: %s)\n", resp.Status, resp.Message)
			return nil
		},
	}
	cmd.Flags().StringVar(&vmSlug, "vm", "", "VM slug to snapshot (required)")
	cmd.Flags().StringVar(&name, "name", "", "Snapshot name (required)")
	cmd.Flags().StringVar(&plan, "plan", "", "Plan slug")
	cmd.Flags().StringVar(&billingCycle, "billing-cycle", "", "Billing cycle slug")
	cmd.Flags().StringVar(&project, "project", "", "Project slug")
	cmd.Flags().StringVar(&cloudProvider, "cloud-provider", "", "Cloud provider slug")
	cmd.Flags().StringVar(&region, "region", "", "Region slug")
	cmd.Flags().StringVar(&service, "service", "", "Service slug")
	cmd.Flags().BoolVar(&memory, "memory", false, "Include memory state in snapshot")
	cmd.Flags().StringVar(&coupon, "coupon", "", "Optional coupon code")
	return cmd
}

func newVMSnapshotDeleteCmd() *cobra.Command {
	var yes bool

	cmd := &cobra.Command{
		Use:   "delete <slug>",
		Short: "Delete a VM snapshot permanently",
		Args:  cobra.ExactArgs(1),
		Example: `  zcp vm-snapshot delete <slug>
  zcp vm-snapshot delete <slug> --yes`,
		RunE: func(cmd *cobra.Command, args []string) error {
			slug := args[0]
			if !yes && !autoApproved(cmd) {
				fmt.Fprintf(os.Stdout, "Are you sure you want to delete %q? This cannot be undone. [y/N]: ", slug)
				var answer string
				fmt.Scanln(&answer)
				if strings.ToLower(strings.TrimSpace(answer)) != "y" {
					fmt.Fprintln(os.Stdout, "Aborted.")
					return nil
				}
			}
			_, client, printer, err := buildClientAndPrinter(cmd)
			if err != nil {
				return err
			}
			svc := vmsnapshot.NewService(client)
			ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
			defer cancel()

			if err := svc.Delete(ctx, slug); err != nil {
				return fmt.Errorf("vm-snapshot delete: %w", err)
			}

			printer.Fprintf("VM snapshot %q deleted.\n", slug)
			return nil
		},
	}
	cmd.Flags().BoolVar(&yes, "yes", false, "Skip confirmation prompt")
	return cmd
}

func newVMSnapshotRevertCmd() *cobra.Command {
	var yes bool

	cmd := &cobra.Command{
		Use:   "revert <slug>",
		Short: "Revert a VM to a snapshot state (DESTRUCTIVE)",
		Args:  cobra.ExactArgs(1),
		Example: `  zcp vm-snapshot revert <slug>
  zcp vm-snapshot revert <slug> --yes`,
		RunE: func(cmd *cobra.Command, args []string) error {
			slug := args[0]
			if !yes && !autoApproved(cmd) {
				fmt.Fprintf(os.Stdout, "WARNING: Reverting to snapshot %q will discard all VM state since the snapshot was taken. This cannot be undone. [y/N]: ", slug)
				var answer string
				fmt.Scanln(&answer)
				if strings.ToLower(strings.TrimSpace(answer)) != "y" {
					fmt.Fprintln(os.Stdout, "Aborted.")
					return nil
				}
			}
			_, client, printer, err := buildClientAndPrinter(cmd)
			if err != nil {
				return err
			}
			svc := vmsnapshot.NewService(client)
			ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
			defer cancel()

			resp, err := svc.Revert(ctx, slug)
			if err != nil {
				return fmt.Errorf("vm-snapshot revert: %w", err)
			}

			printer.Fprintf("VM snapshot %q reverted (status: %s, message: %s)\n", slug, resp.Status, resp.Message)
			return nil
		},
	}
	cmd.Flags().BoolVar(&yes, "yes", false, "Skip confirmation prompt")
	return cmd
}
