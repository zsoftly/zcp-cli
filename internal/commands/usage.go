package commands

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"github.com/zsoftly/zcp-cli/internal/api/usage"
)

// NewUsageCmd returns the 'usage' cobra command.
func NewUsageCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "usage",
		Short: "View usage and consumption data",
	}
	cmd.AddCommand(newUsageConsumptionCmd())
	cmd.AddCommand(newUsageReportCmd())
	cmd.AddCommand(newUsageStatusCmd())
	cmd.AddCommand(newUsageCreditCmd())
	return cmd
}

func newUsageConsumptionCmd() *cobra.Command {
	var period, customer string

	cmd := &cobra.Command{
		Use:   "consumption",
		Short: "List usage consumption for a billing period",
		Example: `  zcp usage consumption --period 2025-01
  zcp usage consumption --period 2025-01 --customer user@example.com`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if period == "" {
				return fmt.Errorf("--period is required (format: YYYY-MM)")
			}
			return runUsageConsumption(cmd, period, customer)
		},
	}
	cmd.Flags().StringVar(&period, "period", "", "Billing period (required, format: YYYY-MM)")
	cmd.Flags().StringVar(&customer, "customer", "", "Filter by customer email (optional)")
	return cmd
}

func runUsageConsumption(cmd *cobra.Command, period, customer string) error {
	_, client, printer, err := buildClientAndPrinter(cmd)
	if err != nil {
		return err
	}

	svc := usage.NewService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	result, err := svc.ConsumptionList(ctx, period, customer)
	if err != nil {
		return fmt.Errorf("usage consumption: %w", err)
	}

	// Response schema is undefined — always output as raw JSON
	return printer.Print(result)
}

func newUsageReportCmd() *cobra.Command {
	var from, to, customer string

	cmd := &cobra.Command{
		Use:   "report",
		Short: "List usage report for a date range",
		Example: `  zcp usage report --from 2025-01 --to 2025-03
  zcp usage report --from 2025-01 --to 2025-03 --customer user@example.com`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if from == "" {
				return fmt.Errorf("--from is required (format: YYYY-MM)")
			}
			if to == "" {
				return fmt.Errorf("--to is required (format: YYYY-MM)")
			}
			return runUsageReport(cmd, from, to, customer)
		},
	}
	cmd.Flags().StringVar(&from, "from", "", "Start period (required, format: YYYY-MM)")
	cmd.Flags().StringVar(&to, "to", "", "End period (required, format: YYYY-MM)")
	cmd.Flags().StringVar(&customer, "customer", "", "Filter by customer email (optional)")
	return cmd
}

func runUsageReport(cmd *cobra.Command, from, to, customer string) error {
	_, client, printer, err := buildClientAndPrinter(cmd)
	if err != nil {
		return err
	}

	svc := usage.NewService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	result, err := svc.ReportList(ctx, from, to, customer)
	if err != nil {
		return fmt.Errorf("usage report: %w", err)
	}

	// Response schema is undefined — always output as raw JSON
	return printer.Print(result)
}

func newUsageStatusCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "status",
		Short:   "Show current billing progress status",
		Example: `  zcp usage status`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runUsageStatus(cmd)
		},
	}
	return cmd
}

func runUsageStatus(cmd *cobra.Command) error {
	_, client, printer, err := buildClientAndPrinter(cmd)
	if err != nil {
		return err
	}

	svc := usage.NewService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	result, err := svc.ProgressStatus(ctx)
	if err != nil {
		return fmt.Errorf("usage status: %w", err)
	}

	// Response schema is undefined — always output as raw JSON
	return printer.Print(result)
}

func newUsageCreditCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "credit-balance",
		Short:   "Show your account credit balance",
		Example: `  zcp usage credit-balance`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runUsageCredit(cmd)
		},
	}
	return cmd
}

func runUsageCredit(cmd *cobra.Command) error {
	_, client, printer, err := buildClientAndPrinter(cmd)
	if err != nil {
		return err
	}

	svc := usage.NewService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	bal, err := svc.CreditBalance(ctx)
	if err != nil {
		return fmt.Errorf("usage credit-balance: %w", err)
	}

	headers := []string{"EMAIL", "TYPE", "BALANCE", "CURRENCY"}
	rows := [][]string{
		{
			bal.UserEmail,
			bal.UserType,
			fmt.Sprintf("%.2f", bal.BalanceAmount),
			bal.Type,
		},
	}
	return printer.PrintTable(headers, rows)
}
