package commands

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/spf13/cobra"
	"github.com/zsoftly/zcp-cli/internal/api/host"
)

// NewHostCmd returns the 'host' cobra command.
func NewHostCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "host",
		Short: "Manage hypervisor hosts (admin only)",
	}
	cmd.AddCommand(newHostListCmd())
	return cmd
}

func newHostListCmd() *cobra.Command {
	var hostUUID string

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List hypervisor hosts",
		Example: `  zcp host list
  zcp host list --uuid <host-uuid>
  zcp host list --output json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			_, client, printer, err := buildClientAndPrinter(cmd)
			if err != nil {
				return err
			}
			svc := host.NewService(client)
			ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
			defer cancel()

			hosts, err := svc.List(ctx, hostUUID)
			if err != nil {
				return fmt.Errorf("host list: %w", err)
			}

			headers := []string{"UUID", "NAME", "HYPERVISOR", "POD", "CPU CORES", "VMs", "ACTIVE"}
			rows := make([][]string, 0, len(hosts))
			for _, h := range hosts {
				rows = append(rows, []string{
					h.UUID,
					h.Name,
					h.Hypervisor,
					h.PodName,
					h.CPUCores,
					h.VMCount,
					strconv.FormatBool(h.IsActive),
				})
			}
			return printer.PrintTable(headers, rows)
		},
	}
	cmd.Flags().StringVar(&hostUUID, "uuid", "", "Filter by host UUID")
	return cmd
}
