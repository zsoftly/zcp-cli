package commands

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/zsoftly/zcp-cli/internal/api/snapshot"
)

// NewSnapshotCmd returns the 'snapshot' cobra command.
func NewSnapshotCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "snapshot",
		Short: "Manage volume snapshots",
	}
	cmd.AddCommand(newSnapshotListCmd())
	cmd.AddCommand(newSnapshotCreateCmd())
	cmd.AddCommand(newSnapshotDeleteCmd())
	return cmd
}

func newSnapshotListCmd() *cobra.Command {
	var zoneUUID, snapshotUUID string

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List volume snapshots",
		Example: `  zcp snapshot list
  zcp snapshot list --zone <uuid>
  zcp snapshot list --output json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			_, client, printer, err := buildClientAndPrinter(cmd)
			if err != nil {
				return err
			}
			svc := snapshot.NewService(client)
			ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
			defer cancel()

			snapshots, err := svc.List(ctx, zoneUUID, snapshotUUID)
			if err != nil {
				return fmt.Errorf("snapshot list: %w", err)
			}

			headers := []string{"UUID", "NAME", "STATUS", "VOLUME", "ZONE", "TIME"}
			rows := make([][]string, 0, len(snapshots))
			for _, s := range snapshots {
				rows = append(rows, []string{
					s.UUID,
					s.Name,
					s.Status,
					s.VolumeUUID,
					s.ZoneUUID,
					s.SnapshotTime,
				})
			}
			return printer.PrintTable(headers, rows)
		},
	}
	cmd.Flags().StringVar(&zoneUUID, "zone", "", "Filter by zone UUID")
	cmd.Flags().StringVar(&snapshotUUID, "uuid", "", "Filter by snapshot UUID")
	return cmd
}

func newSnapshotCreateCmd() *cobra.Command {
	var volumeUUID, zoneUUID, name string

	cmd := &cobra.Command{
		Use:     "create",
		Short:   "Create a volume snapshot",
		Example: `  zcp snapshot create --volume <uuid> --zone <uuid> --name my-snapshot`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if volumeUUID == "" {
				return fmt.Errorf("--volume is required")
			}
			if name == "" {
				return fmt.Errorf("--name is required")
			}
			profile, client, printer, err := buildClientAndPrinter(cmd)
			if err != nil {
				return err
			}
			zoneUUID = resolveZone(profile, zoneUUID)
			if zoneUUID == "" {
				return errNoZone()
			}
			svc := snapshot.NewService(client)
			ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
			defer cancel()

			req := snapshot.CreateRequest{
				Name:       name,
				VolumeUUID: volumeUUID,
				ZoneUUID:   zoneUUID,
			}
			snap, err := svc.Create(ctx, req)
			if err != nil {
				return fmt.Errorf("snapshot create: %w", err)
			}

			headers := []string{"UUID", "NAME", "STATUS", "VOLUME", "ZONE", "TIME"}
			rows := [][]string{{
				snap.UUID,
				snap.Name,
				snap.Status,
				snap.VolumeUUID,
				snap.ZoneUUID,
				snap.SnapshotTime,
			}}
			return printer.PrintTable(headers, rows)
		},
	}
	cmd.Flags().StringVar(&volumeUUID, "volume", "", "Volume UUID to snapshot (required)")
	cmd.Flags().StringVar(&zoneUUID, "zone", "", "Zone UUID (overrides default zone)")
	cmd.Flags().StringVar(&name, "name", "", "Snapshot name (required)")
	return cmd
}

func newSnapshotDeleteCmd() *cobra.Command {
	var yes bool

	cmd := &cobra.Command{
		Use:   "delete <uuid>",
		Short: "Delete a snapshot permanently",
		Args:  cobra.ExactArgs(1),
		Example: `  zcp snapshot delete <uuid>
  zcp snapshot delete <uuid> --yes`,
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
			svc := snapshot.NewService(client)
			ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
			defer cancel()

			if err := svc.Delete(ctx, uuid); err != nil {
				return fmt.Errorf("snapshot delete: %w", err)
			}

			printer.Fprintf("Snapshot %q deleted.\n", uuid)
			return nil
		},
	}
	cmd.Flags().BoolVar(&yes, "yes", false, "Skip confirmation prompt")
	return cmd
}
