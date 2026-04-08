package commands

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/spf13/cobra"
	"github.com/zsoftly/zcp-cli/internal/api/billingcycle"
)

// NewBillingCycleCmd returns the 'billing-cycle' cobra command.
func NewBillingCycleCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "billing-cycle",
		Short: "Manage billing cycles",
	}
	cmd.AddCommand(newBillingCycleListCmd())
	return cmd
}

// ─── List ───────────────────────────────────────────────────────────────────

func newBillingCycleListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "list",
		Short:   "List billing cycles",
		Example: `  zcp billing-cycle list`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runBillingCycleList(cmd)
		},
	}
	return cmd
}

func runBillingCycleList(cmd *cobra.Command) error {
	_, client, printer, err := buildClientAndPrinter(cmd)
	if err != nil {
		return err
	}

	svc := billingcycle.NewService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	cycles, err := svc.List(ctx)
	if err != nil {
		return fmt.Errorf("billing-cycle list: %w", err)
	}

	headers := []string{"ID", "NAME", "SLUG", "DURATION", "UNIT", "ENABLED", "CREATED"}
	rows := make([][]string, 0, len(cycles))
	for _, c := range cycles {
		rows = append(rows, []string{
			c.ID,
			c.Name,
			c.Slug,
			strconv.Itoa(c.Duration),
			c.Unit,
			fmt.Sprintf("%v", c.IsEnabled),
			c.CreatedAt,
		})
	}
	return printer.PrintTable(headers, rows)
}
