package commands

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/zsoftly/zcp-cli/internal/api/vmsnapshot"
	"github.com/zsoftly/zcp-cli/internal/api/waiters"
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
	var zoneUUID, snapshotUUID string

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List VM snapshots",
		Example: `  zcp vm-snapshot list
  zcp vm-snapshot list --zone <uuid>
  zcp vm-snapshot list --output json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			_, client, printer, err := buildClientAndPrinter(cmd)
			if err != nil {
				return err
			}
			svc := vmsnapshot.NewService(client)
			ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
			defer cancel()

			snapshots, err := svc.List(ctx, zoneUUID, snapshotUUID)
			if err != nil {
				return fmt.Errorf("vm-snapshot list: %w", err)
			}

			headers := []string{"UUID", "NAME", "STATUS", "CURRENT", "ZONE", "CREATED"}
			rows := make([][]string, 0, len(snapshots))
			for _, s := range snapshots {
				rows = append(rows, []string{
					s.UUID,
					s.Name,
					s.Status,
					strconv.FormatBool(s.IsCurrent),
					s.ZoneUUID,
					s.CreatedAt,
				})
			}
			return printer.PrintTable(headers, rows)
		},
	}
	cmd.Flags().StringVar(&zoneUUID, "zone", "", "Filter by zone UUID")
	cmd.Flags().StringVar(&snapshotUUID, "uuid", "", "Filter by VM snapshot UUID")
	return cmd
}

func newVMSnapshotCreateCmd() *cobra.Command {
	var zoneUUID, name, instanceUUID, description string
	var memory bool
	var wait bool

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a VM snapshot",
		Example: `  zcp vm-snapshot create --zone <uuid> --name my-snap --instance <uuid>
  zcp vm-snapshot create --zone <uuid> --name my-snap --instance <uuid> --description "pre-upgrade" --memory`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if name == "" {
				return fmt.Errorf("--name is required")
			}
			if instanceUUID == "" {
				return fmt.Errorf("--instance is required")
			}
			profile, client, printer, err := buildClientAndPrinter(cmd)
			if err != nil {
				return err
			}
			zoneUUID = resolveZone(profile, zoneUUID)
			if zoneUUID == "" {
				return errNoZone()
			}
			svc := vmsnapshot.NewService(client)
			ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
			defer cancel()

			req := vmsnapshot.CreateRequest{
				Name:               name,
				ZoneUUID:           zoneUUID,
				VirtualMachineUUID: instanceUUID,
				Description:        description,
				SnapshotMemory:     memory,
			}
			snap, err := svc.Create(ctx, req)
			if err != nil {
				return fmt.Errorf("vm-snapshot create: %w", err)
			}

			if wait && snap.JobID != "" {
				fmt.Fprintf(os.Stderr, "Waiting for job %s to complete...\n", snap.JobID)
				waiter := waiters.New(client, waiters.WithProgressWriter(os.Stderr))
				if _, err := waiter.Wait(ctx, snap.JobID); err != nil {
					return fmt.Errorf("wait failed: %w", err)
				}
			}

			headers := []string{"UUID", "NAME", "STATUS", "CURRENT", "ZONE", "JOB ID", "CREATED"}
			rows := [][]string{{
				snap.UUID,
				snap.Name,
				snap.Status,
				strconv.FormatBool(snap.IsCurrent),
				snap.ZoneUUID,
				snap.JobID,
				snap.CreatedAt,
			}}
			return printer.PrintTable(headers, rows)
		},
	}
	cmd.Flags().StringVar(&zoneUUID, "zone", "", "Zone UUID (overrides default zone)")
	cmd.Flags().StringVar(&name, "name", "", "Snapshot name (required)")
	cmd.Flags().StringVar(&instanceUUID, "instance", "", "VM instance UUID (required)")
	cmd.Flags().StringVar(&description, "description", "", "Optional description")
	cmd.Flags().BoolVar(&memory, "memory", false, "Include memory state in snapshot")
	cmd.Flags().BoolVar(&wait, "wait", false, "Wait for async operation to complete")
	return cmd
}

func newVMSnapshotDeleteCmd() *cobra.Command {
	var yes bool

	cmd := &cobra.Command{
		Use:   "delete <uuid>",
		Short: "Delete a VM snapshot permanently",
		Args:  cobra.ExactArgs(1),
		Example: `  zcp vm-snapshot delete <uuid>
  zcp vm-snapshot delete <uuid> --yes`,
		RunE: func(cmd *cobra.Command, args []string) error {
			uuid := args[0]
			if !yes {
				fmt.Fprintf(os.Stdout, "Are you sure you want to delete %q? This cannot be undone. [y/N]: ", uuid)
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

			resp, err := svc.Delete(ctx, uuid)
			if err != nil {
				return fmt.Errorf("vm-snapshot delete: %w", err)
			}

			printer.Fprintf("VM snapshot %q deleted (status: %s)\n", resp.UUID, resp.Status)
			return nil
		},
	}
	cmd.Flags().BoolVar(&yes, "yes", false, "Skip confirmation prompt")
	return cmd
}

func newVMSnapshotRevertCmd() *cobra.Command {
	var yes bool
	var wait bool

	cmd := &cobra.Command{
		Use:   "revert <uuid>",
		Short: "Revert a VM to a snapshot state (DESTRUCTIVE)",
		Args:  cobra.ExactArgs(1),
		Example: `  zcp vm-snapshot revert <uuid>
  zcp vm-snapshot revert <uuid> --yes`,
		RunE: func(cmd *cobra.Command, args []string) error {
			uuid := args[0]
			if !yes {
				fmt.Fprintf(os.Stdout, "WARNING: Reverting to snapshot %q will discard all VM state since the snapshot was taken. This cannot be undone. [y/N]: ", uuid)
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

			snap, err := svc.Revert(ctx, uuid)
			if err != nil {
				return fmt.Errorf("vm-snapshot revert: %w", err)
			}

			if wait && snap.JobID != "" {
				fmt.Fprintf(os.Stderr, "Waiting for job %s to complete...\n", snap.JobID)
				waiter := waiters.New(client, waiters.WithProgressWriter(os.Stderr))
				if _, err := waiter.Wait(ctx, snap.JobID); err != nil {
					return fmt.Errorf("wait failed: %w", err)
				}
			}

			headers := []string{"UUID", "NAME", "STATUS", "CURRENT", "ZONE", "JOB ID"}
			rows := [][]string{{
				snap.UUID,
				snap.Name,
				snap.Status,
				strconv.FormatBool(snap.IsCurrent),
				snap.ZoneUUID,
				snap.JobID,
			}}
			return printer.PrintTable(headers, rows)
		},
	}
	cmd.Flags().BoolVar(&yes, "yes", false, "Skip confirmation prompt")
	cmd.Flags().BoolVar(&wait, "wait", false, "Wait for async operation to complete")
	return cmd
}
