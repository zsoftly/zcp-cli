package commands

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/spf13/cobra"
	"github.com/zsoftly/zcp-cli/internal/api/offering"
)

// NewOfferingCmd returns the 'offering' cobra command.
func NewOfferingCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "offering",
		Short: "List resource offerings (compute, storage, network, vpc)",
	}
	cmd.AddCommand(newOfferingComputeCmd())
	cmd.AddCommand(newOfferingStorageCmd())
	cmd.AddCommand(newOfferingNetworkCmd())
	cmd.AddCommand(newOfferingVPCCmd())
	return cmd
}

func newOfferingComputeCmd() *cobra.Command {
	var zoneUUID string

	cmd := &cobra.Command{
		Use:   "compute",
		Short: "List compute offerings (instance sizes)",
		Example: `  zcp offering compute --zone <uuid>
  zcp offering compute --output json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			profile, client, printer, err := buildClientAndPrinter(cmd)
			if err != nil {
				return err
			}
			zoneUUID = resolveZone(profile, zoneUUID)
			if zoneUUID == "" {
				return errNoZone()
			}
			svc := offering.NewService(client)
			ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
			defer cancel()

			offerings, err := svc.ListCompute(ctx, zoneUUID, "")
			if err != nil {
				return fmt.Errorf("offering compute: %w", err)
			}

			headers := []string{"UUID", "NAME", "CPU", "MEMORY", "STORAGE TYPE", "ACTIVE"}
			rows := make([][]string, 0, len(offerings))
			for _, o := range offerings {
				rows = append(rows, []string{
					o.UUID, o.Name, o.CPUCores, o.Memory, o.StorageType,
					strconv.FormatBool(o.IsActive),
				})
			}
			return printer.PrintTable(headers, rows)
		},
	}
	cmd.Flags().StringVar(&zoneUUID, "zone", "", "Zone UUID (overrides default zone)")
	return cmd
}

func newOfferingStorageCmd() *cobra.Command {
	var zoneUUID string

	cmd := &cobra.Command{
		Use:     "storage",
		Short:   "List storage offerings",
		Example: `  zcp offering storage --zone <uuid>`,
		RunE: func(cmd *cobra.Command, args []string) error {
			profile, client, printer, err := buildClientAndPrinter(cmd)
			if err != nil {
				return err
			}
			zoneUUID = resolveZone(profile, zoneUUID)
			if zoneUUID == "" {
				return errNoZone()
			}
			svc := offering.NewService(client)
			ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
			defer cancel()

			offerings, err := svc.ListStorage(ctx, zoneUUID)
			if err != nil {
				return fmt.Errorf("offering storage: %w", err)
			}

			headers := []string{"UUID", "NAME", "DISK SIZE", "STORAGE TYPE", "CUSTOM", "ACTIVE"}
			rows := make([][]string, 0, len(offerings))
			for _, o := range offerings {
				rows = append(rows, []string{
					o.UUID, o.Name, o.DiskSize, o.StorageType,
					strconv.FormatBool(o.IsCustomDisk),
					strconv.FormatBool(o.IsActive),
				})
			}
			return printer.PrintTable(headers, rows)
		},
	}
	cmd.Flags().StringVar(&zoneUUID, "zone", "", "Zone UUID (overrides default zone)")
	return cmd
}

func newOfferingNetworkCmd() *cobra.Command {
	var zoneUUID string

	cmd := &cobra.Command{
		Use:   "network",
		Short: "List network offerings",
		RunE: func(cmd *cobra.Command, args []string) error {
			profile, client, printer, err := buildClientAndPrinter(cmd)
			if err != nil {
				return err
			}
			zoneUUID = resolveZone(profile, zoneUUID)
			if zoneUUID == "" {
				return errNoZone()
			}
			svc := offering.NewService(client)
			ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
			defer cancel()

			offerings, err := svc.ListNetwork(ctx, zoneUUID)
			if err != nil {
				return fmt.Errorf("offering network: %w", err)
			}

			headers := []string{"UUID", "NAME", "DISPLAY TEXT", "GUEST IP TYPE", "ACTIVE"}
			rows := make([][]string, 0, len(offerings))
			for _, o := range offerings {
				rows = append(rows, []string{
					o.UUID, o.Name, o.DisplayText, o.GuestIPType,
					strconv.FormatBool(o.IsActive),
				})
			}
			return printer.PrintTable(headers, rows)
		},
	}
	cmd.Flags().StringVar(&zoneUUID, "zone", "", "Zone UUID (overrides default zone)")
	return cmd
}

func newOfferingVPCCmd() *cobra.Command {
	var zoneUUID string

	cmd := &cobra.Command{
		Use:   "vpc",
		Short: "List VPC offerings",
		RunE: func(cmd *cobra.Command, args []string) error {
			profile, client, printer, err := buildClientAndPrinter(cmd)
			if err != nil {
				return err
			}
			zoneUUID = resolveZone(profile, zoneUUID)
			if zoneUUID == "" {
				return errNoZone()
			}
			svc := offering.NewService(client)
			ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
			defer cancel()

			offerings, err := svc.ListVPC(ctx, zoneUUID)
			if err != nil {
				return fmt.Errorf("offering vpc: %w", err)
			}

			headers := []string{"UUID", "NAME", "DISPLAY TEXT"}
			rows := make([][]string, 0, len(offerings))
			for _, o := range offerings {
				rows = append(rows, []string{o.UUID, o.Name, o.DisplayText})
			}
			return printer.PrintTable(headers, rows)
		},
	}
	cmd.Flags().StringVar(&zoneUUID, "zone", "", "Zone UUID (overrides default zone)")
	return cmd
}
