package commands

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/zsoftly/zcp-cli/internal/api/volume"
	"github.com/zsoftly/zcp-cli/internal/api/waiters"
)

// NewVolumeCmd returns the 'volume' cobra command.
func NewVolumeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "volume",
		Short: "Manage data volumes",
	}
	cmd.AddCommand(newVolumeListCmd())
	cmd.AddCommand(newVolumeCreateCmd())
	cmd.AddCommand(newVolumeAttachCmd())
	cmd.AddCommand(newVolumeDetachCmd())
	cmd.AddCommand(newVolumeDeleteCmd())
	cmd.AddCommand(newVolumeResizeCmd())
	return cmd
}

func newVolumeListCmd() *cobra.Command {
	var zoneUUID, instanceUUID, volumeUUID string

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List data volumes",
		Example: `  zcp volume list --zone <uuid>
  zcp volume list --zone <uuid> --instance <uuid>
  zcp volume list --zone <uuid> --output json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if zoneUUID == "" {
				return fmt.Errorf("--zone is required")
			}
			_, client, printer, err := buildClientAndPrinter(cmd)
			if err != nil {
				return err
			}
			svc := volume.NewService(client)
			ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
			defer cancel()

			volumes, err := svc.List(ctx, zoneUUID, instanceUUID, volumeUUID)
			if err != nil {
				return fmt.Errorf("volume list: %w", err)
			}

			headers := []string{"UUID", "NAME", "STATUS", "SIZE", "TYPE", "INSTANCE", "ZONE"}
			rows := make([][]string, 0, len(volumes))
			for _, v := range volumes {
				rows = append(rows, []string{
					v.UUID,
					v.Name,
					v.Status,
					v.StorageDiskSize,
					v.VolumeType,
					v.VMInstanceName,
					v.ZoneUUID,
				})
			}
			return printer.PrintTable(headers, rows)
		},
	}
	cmd.Flags().StringVar(&zoneUUID, "zone", "", "Zone UUID (required)")
	cmd.Flags().StringVar(&instanceUUID, "instance", "", "Filter by instance UUID")
	cmd.Flags().StringVar(&volumeUUID, "uuid", "", "Filter by volume UUID")
	return cmd
}

func newVolumeCreateCmd() *cobra.Command {
	var zoneUUID, name, storageOfferingUUID string
	var diskSize int
	var wait bool

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new data volume",
		Example: `  zcp volume create --zone <uuid> --name my-disk --storage-offering <uuid>
  zcp volume create --zone <uuid> --name my-disk --storage-offering <uuid> --disk-size 100`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if zoneUUID == "" {
				return fmt.Errorf("--zone is required")
			}
			if name == "" {
				return fmt.Errorf("--name is required")
			}
			if storageOfferingUUID == "" {
				return fmt.Errorf("--storage-offering is required")
			}
			_, client, printer, err := buildClientAndPrinter(cmd)
			if err != nil {
				return err
			}
			svc := volume.NewService(client)
			ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
			defer cancel()

			req := volume.CreateRequest{
				Name:                name,
				ZoneUUID:            zoneUUID,
				StorageOfferingUUID: storageOfferingUUID,
				DiskSize:            diskSize,
			}
			vol, err := svc.Create(ctx, req)
			if err != nil {
				return fmt.Errorf("volume create: %w", err)
			}

			if wait && vol.JobID != "" {
				fmt.Fprintf(os.Stderr, "Waiting for job %s to complete...\n", vol.JobID)
				waiter := waiters.New(client, waiters.WithProgressWriter(os.Stderr))
				if _, err := waiter.Wait(ctx, vol.JobID); err != nil {
					return fmt.Errorf("wait failed: %w", err)
				}
			}

			headers := []string{"UUID", "NAME", "STATUS", "SIZE", "TYPE", "ZONE", "JOB ID"}
			rows := [][]string{{
				vol.UUID,
				vol.Name,
				vol.Status,
				vol.StorageDiskSize,
				vol.VolumeType,
				vol.ZoneUUID,
				vol.JobID,
			}}
			return printer.PrintTable(headers, rows)
		},
	}
	cmd.Flags().StringVar(&zoneUUID, "zone", "", "Zone UUID (required)")
	cmd.Flags().StringVar(&name, "name", "", "Volume name (required)")
	cmd.Flags().StringVar(&storageOfferingUUID, "storage-offering", "", "Storage offering UUID (required)")
	cmd.Flags().IntVar(&diskSize, "disk-size", 0, "Custom disk size in GB (for custom offerings)")
	cmd.Flags().BoolVar(&wait, "wait", false, "Wait for async operation to complete")
	return cmd
}

func newVolumeAttachCmd() *cobra.Command {
	var instanceUUID string
	var wait bool

	cmd := &cobra.Command{
		Use:     "attach <volume-uuid>",
		Short:   "Attach a volume to an instance",
		Args:    cobra.ExactArgs(1),
		Example: `  zcp volume attach <volume-uuid> --instance <instance-uuid>`,
		RunE: func(cmd *cobra.Command, args []string) error {
			volumeUUID := args[0]
			if instanceUUID == "" {
				return fmt.Errorf("--instance is required")
			}
			_, client, printer, err := buildClientAndPrinter(cmd)
			if err != nil {
				return err
			}
			svc := volume.NewService(client)
			ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
			defer cancel()

			vol, err := svc.Attach(ctx, volumeUUID, instanceUUID)
			if err != nil {
				return fmt.Errorf("volume attach: %w", err)
			}

			if wait && vol.JobID != "" {
				fmt.Fprintf(os.Stderr, "Waiting for job %s to complete...\n", vol.JobID)
				waiter := waiters.New(client, waiters.WithProgressWriter(os.Stderr))
				if _, err := waiter.Wait(ctx, vol.JobID); err != nil {
					return fmt.Errorf("wait failed: %w", err)
				}
			}

			headers := []string{"UUID", "NAME", "STATUS", "INSTANCE", "ZONE"}
			rows := [][]string{{
				vol.UUID,
				vol.Name,
				vol.Status,
				vol.VMInstanceName,
				vol.ZoneUUID,
			}}
			return printer.PrintTable(headers, rows)
		},
	}
	cmd.Flags().StringVar(&instanceUUID, "instance", "", "Instance UUID to attach to (required)")
	cmd.Flags().BoolVar(&wait, "wait", false, "Wait for async operation to complete")
	return cmd
}

func newVolumeDetachCmd() *cobra.Command {
	var wait bool

	cmd := &cobra.Command{
		Use:     "detach <volume-uuid>",
		Short:   "Detach a volume from its instance",
		Args:    cobra.ExactArgs(1),
		Example: `  zcp volume detach <volume-uuid>`,
		RunE: func(cmd *cobra.Command, args []string) error {
			volumeUUID := args[0]
			_, client, printer, err := buildClientAndPrinter(cmd)
			if err != nil {
				return err
			}
			svc := volume.NewService(client)
			ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
			defer cancel()

			vol, err := svc.Detach(ctx, volumeUUID)
			if err != nil {
				return fmt.Errorf("volume detach: %w", err)
			}

			if wait && vol.JobID != "" {
				fmt.Fprintf(os.Stderr, "Waiting for job %s to complete...\n", vol.JobID)
				waiter := waiters.New(client, waiters.WithProgressWriter(os.Stderr))
				if _, err := waiter.Wait(ctx, vol.JobID); err != nil {
					return fmt.Errorf("wait failed: %w", err)
				}
			}

			headers := []string{"UUID", "NAME", "STATUS", "ZONE"}
			rows := [][]string{{
				vol.UUID,
				vol.Name,
				vol.Status,
				vol.ZoneUUID,
			}}
			return printer.PrintTable(headers, rows)
		},
	}
	cmd.Flags().BoolVar(&wait, "wait", false, "Wait for async operation to complete")
	return cmd
}

func newVolumeDeleteCmd() *cobra.Command {
	var yes bool

	cmd := &cobra.Command{
		Use:   "delete <uuid>",
		Short: "Delete a volume permanently",
		Args:  cobra.ExactArgs(1),
		Example: `  zcp volume delete <uuid>
  zcp volume delete <uuid> --yes`,
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
			svc := volume.NewService(client)
			ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
			defer cancel()

			resp, err := svc.Delete(ctx, uuid)
			if err != nil {
				return fmt.Errorf("volume delete: %w", err)
			}

			printer.Fprintf("Volume %q deleted (status: %s)\n", resp.UUID, resp.Status)
			return nil
		},
	}
	cmd.Flags().BoolVar(&yes, "yes", false, "Skip confirmation prompt")
	return cmd
}

func newVolumeResizeCmd() *cobra.Command {
	var storageOfferingUUID string
	var diskSize int
	var shrink bool
	var wait bool

	cmd := &cobra.Command{
		Use:   "resize <uuid>",
		Short: "Resize a volume (change offering or disk size)",
		Args:  cobra.ExactArgs(1),
		Example: `  zcp volume resize <uuid> --storage-offering <uuid>
  zcp volume resize <uuid> --storage-offering <uuid> --disk-size 200
  zcp volume resize <uuid> --storage-offering <uuid> --disk-size 50 --shrink`,
		RunE: func(cmd *cobra.Command, args []string) error {
			uuid := args[0]
			if storageOfferingUUID == "" {
				return fmt.Errorf("--storage-offering is required")
			}
			_, client, printer, err := buildClientAndPrinter(cmd)
			if err != nil {
				return err
			}
			svc := volume.NewService(client)
			ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
			defer cancel()

			vol, err := svc.Resize(ctx, uuid, storageOfferingUUID, diskSize, shrink)
			if err != nil {
				return fmt.Errorf("volume resize: %w", err)
			}

			if wait && vol.JobID != "" {
				fmt.Fprintf(os.Stderr, "Waiting for job %s to complete...\n", vol.JobID)
				waiter := waiters.New(client, waiters.WithProgressWriter(os.Stderr))
				if _, err := waiter.Wait(ctx, vol.JobID); err != nil {
					return fmt.Errorf("wait failed: %w", err)
				}
			}

			headers := []string{"UUID", "NAME", "STATUS", "SIZE", "OFFERING", "ZONE", "JOB ID"}
			rows := [][]string{{
				vol.UUID,
				vol.Name,
				vol.Status,
				vol.StorageDiskSize,
				vol.StorageOfferingName,
				vol.ZoneUUID,
				vol.JobID,
			}}
			return printer.PrintTable(headers, rows)
		},
	}
	cmd.Flags().StringVar(&storageOfferingUUID, "storage-offering", "", "Storage offering UUID (required)")
	cmd.Flags().IntVar(&diskSize, "disk-size", 0, "New disk size in GB")
	cmd.Flags().BoolVar(&shrink, "shrink", false, "Allow shrinking the volume (use with caution)")
	cmd.Flags().BoolVar(&wait, "wait", false, "Wait for async operation to complete")
	return cmd
}
