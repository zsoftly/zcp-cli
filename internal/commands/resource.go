package commands

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"github.com/zsoftly/zcp-cli/internal/api/resource"
)

// NewResourceCmd returns the 'resource' cobra command.
func NewResourceCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "resource",
		Short: "View cloud resource limits and availability",
	}
	cmd.AddCommand(newResourceAvailableCmd())
	return cmd
}

func newResourceAvailableCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "available",
		Short: "List available resource limits for your account",
		Example: `  zcp resource available
  zcp resource available --output json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			_, client, printer, err := buildClientAndPrinter(cmd)
			if err != nil {
				return err
			}
			svc := resource.NewService(client)
			ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
			defer cancel()

			resources, err := svc.ListAvailable(ctx)
			if err != nil {
				return fmt.Errorf("resource available: %w", err)
			}

			headers := []string{"RESOURCE TYPE", "USED", "AVAILABLE", "MAXIMUM"}
			rows := make([][]string, 0, len(resources))
			for _, r := range resources {
				rows = append(rows, []string{
					r.ResourceType, r.UsedLimit, r.AvailableLimit, r.MaximumLimit,
				})
			}
			return printer.PrintTable(headers, rows)
		},
	}
	return cmd
}
