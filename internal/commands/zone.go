package commands

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/spf13/cobra"
	"github.com/zsoftly/zcp-cli/internal/api/zone"
)

// NewZoneCmd returns the 'zone' cobra command.
func NewZoneCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "zone",
		Short: "Manage availability zones",
	}
	cmd.AddCommand(newZoneListCmd())
	return cmd
}

func newZoneListCmd() *cobra.Command {
	var zoneUUID string

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List availability zones",
		Example: `  zcp zone list
  zcp zone list --zone <uuid>
  zcp zone list --output json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runZoneList(cmd, zoneUUID)
		},
	}
	cmd.Flags().StringVar(&zoneUUID, "zone", "", "Filter by zone UUID")
	return cmd
}

func runZoneList(cmd *cobra.Command, zoneUUID string) error {
	_, client, printer, err := buildClientAndPrinter(cmd)
	if err != nil {
		return err
	}

	svc := zone.NewService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	zones, err := svc.List(ctx, zoneUUID)
	if err != nil {
		return fmt.Errorf("zone list: %w", err)
	}

	headers := []string{"UUID", "NAME", "COUNTRY", "ACTIVE"}
	rows := make([][]string, 0, len(zones))
	for _, z := range zones {
		rows = append(rows, []string{
			z.UUID,
			z.Name,
			z.CountryName,
			strconv.FormatBool(z.IsActive),
		})
	}
	return printer.PrintTable(headers, rows)
}
