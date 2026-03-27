package commands

import (
	"context"
	"fmt"
	"strconv"
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
	cmd.AddCommand(newResourceQuotaCmd())
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

func newResourceQuotaCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "quota",
		Short: "List resource quota limits for your account",
		Example: `  zcp resource quota
  zcp resource quota --output json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			_, client, printer, err := buildClientAndPrinter(cmd)
			if err != nil {
				return err
			}
			svc := resource.NewService(client)
			ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
			defer cancel()

			quotas, err := svc.ListQuota(ctx, "")
			if err != nil {
				return fmt.Errorf("resource quota: %w", err)
			}

			headers := []string{"QUOTA TYPE", "UNIT", "USED", "AVAILABLE", "MAXIMUM"}
			rows := make([][]string, 0, len(quotas))
			for _, q := range quotas {
				rows = append(rows, []string{
					q.QuotaType,
					q.UnitType,
					strconv.FormatInt(q.UsedLimit, 10),
					strconv.FormatInt(q.AvailableLimit, 10),
					strconv.FormatInt(q.MaximumLimit, 10),
				})
			}
			return printer.PrintTable(headers, rows)
		},
	}
	return cmd
}
