package commands

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/zsoftly/zcp-cli/internal/api/snapshotpolicy"
)

// NewSnapshotPolicyCmd returns the 'snapshot-policy' cobra command.
func NewSnapshotPolicyCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "snapshot-policy",
		Short: "Manage automated snapshot policies",
	}
	cmd.AddCommand(newSnapshotPolicyListCmd())
	cmd.AddCommand(newSnapshotPolicyCreateCmd())
	cmd.AddCommand(newSnapshotPolicyDeleteCmd())
	return cmd
}

func newSnapshotPolicyListCmd() *cobra.Command {
	var volumeUUID, policyUUID string

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List snapshot policies",
		Example: `  zcp snapshot-policy list --volume <uuid>
  zcp snapshot-policy list --volume <uuid> --output json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if volumeUUID == "" {
				return fmt.Errorf("--volume is required")
			}
			_, client, printer, err := buildClientAndPrinter(cmd)
			if err != nil {
				return err
			}
			svc := snapshotpolicy.NewService(client)
			ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
			defer cancel()

			policies, err := svc.List(ctx, volumeUUID, policyUUID)
			if err != nil {
				return fmt.Errorf("snapshot-policy list: %w", err)
			}

			headers := []string{"UUID", "VOLUME", "INTERVAL", "TIME", "MAX SNAPSHOTS", "STATUS"}
			rows := make([][]string, 0, len(policies))
			for _, p := range policies {
				rows = append(rows, []string{
					p.UUID,
					p.VolumeUUID,
					p.IntervalType,
					p.ScheduleTime,
					p.MaximumSnapshots,
					p.Status,
				})
			}
			return printer.PrintTable(headers, rows)
		},
	}
	cmd.Flags().StringVar(&volumeUUID, "volume", "", "Filter by volume UUID")
	cmd.Flags().StringVar(&policyUUID, "uuid", "", "Filter by policy UUID")
	return cmd
}

func newSnapshotPolicyCreateCmd() *cobra.Command {
	var volumeUUID, intervalType, timer, dayOfWeek, dayOfMonth, timezone, maxSnapshots string

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create an automated snapshot policy",
		Example: `  zcp snapshot-policy create --volume <uuid> --interval daily --time 02:00 --timezone UTC --max-snapshots 7
  zcp snapshot-policy create --volume <uuid> --interval weekly --time 02:00 --timezone UTC --max-snapshots 4 --day-of-week mon
  zcp snapshot-policy create --volume <uuid> --interval monthly --time 02:00 --timezone UTC --max-snapshots 3 --day-of-month 1`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if volumeUUID == "" {
				return fmt.Errorf("--volume is required")
			}
			if intervalType == "" {
				return fmt.Errorf("--interval is required (hourly|daily|weekly|monthly)")
			}
			if timer == "" {
				return fmt.Errorf("--time is required (HH:MM)")
			}
			if timezone == "" {
				return fmt.Errorf("--timezone is required")
			}
			if maxSnapshots == "" {
				return fmt.Errorf("--max-snapshots is required")
			}
			_, client, printer, err := buildClientAndPrinter(cmd)
			if err != nil {
				return err
			}
			svc := snapshotpolicy.NewService(client)
			ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
			defer cancel()

			req := snapshotpolicy.CreateRequest{
				VolumeUUID:       volumeUUID,
				IntervalType:     intervalType,
				Timer:            timer,
				DayOfWeek:        dayOfWeek,
				DayOfMonth:       dayOfMonth,
				TimeZone:         timezone,
				MaximumSnapshots: maxSnapshots,
			}
			policy, err := svc.Create(ctx, req)
			if err != nil {
				return fmt.Errorf("snapshot-policy create: %w", err)
			}

			headers := []string{"UUID", "VOLUME", "INTERVAL", "TIME", "MAX SNAPSHOTS", "STATUS"}
			rows := [][]string{{
				policy.UUID,
				policy.VolumeUUID,
				policy.IntervalType,
				policy.ScheduleTime,
				policy.MaximumSnapshots,
				policy.Status,
			}}
			return printer.PrintTable(headers, rows)
		},
	}
	cmd.Flags().StringVar(&volumeUUID, "volume", "", "Volume UUID (required)")
	cmd.Flags().StringVar(&intervalType, "interval", "", "Schedule interval: hourly, daily, weekly, or monthly (required)")
	cmd.Flags().StringVar(&timer, "time", "", "Schedule time in HH:MM format (required)")
	cmd.Flags().StringVar(&timezone, "timezone", "", "Timezone (e.g. UTC, America/New_York) (required)")
	cmd.Flags().StringVar(&maxSnapshots, "max-snapshots", "", "Maximum number of snapshots to retain (required)")
	cmd.Flags().StringVar(&dayOfWeek, "day-of-week", "", "Day of week for weekly policies (mon-sun)")
	cmd.Flags().StringVar(&dayOfMonth, "day-of-month", "", "Day of month for monthly policies (1-31)")
	return cmd
}

func newSnapshotPolicyDeleteCmd() *cobra.Command {
	var yes bool

	cmd := &cobra.Command{
		Use:   "delete <uuid>",
		Short: "Delete a snapshot policy",
		Args:  cobra.ExactArgs(1),
		Example: `  zcp snapshot-policy delete <uuid>
  zcp snapshot-policy delete <uuid> --yes`,
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
			svc := snapshotpolicy.NewService(client)
			ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
			defer cancel()

			if err := svc.Delete(ctx, uuid); err != nil {
				return fmt.Errorf("snapshot-policy delete: %w", err)
			}

			printer.Fprintf("Snapshot policy %q deleted.\n", uuid)
			return nil
		},
	}
	cmd.Flags().BoolVar(&yes, "yes", false, "Skip confirmation prompt")
	return cmd
}
