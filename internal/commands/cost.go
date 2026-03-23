package commands

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/spf13/cobra"
	"github.com/zsoftly/zcp-cli/internal/api/cost"
)

// NewCostCmd returns the 'cost' cobra command.
func NewCostCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cost",
		Short: "View pricing, currencies, and tax information",
	}
	cmd.AddCommand(newCostCurrencyCmd())
	cmd.AddCommand(newCostTaxCmd())
	return cmd
}

func newCostCurrencyCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "currency",
		Short: "List supported billing currencies and their rates",
		Example: `  zcp cost currency
  zcp cost currency --output json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCostCurrency(cmd)
		},
	}
	return cmd
}

func runCostCurrency(cmd *cobra.Command) error {
	_, client, printer, err := buildClientAndPrinter(cmd)
	if err != nil {
		return err
	}

	svc := cost.NewService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	resp, err := svc.ListCurrencies(ctx)
	if err != nil {
		return fmt.Errorf("cost currency: %w", err)
	}

	headers := []string{"UUID", "CURRENCY", "SYMBOL", "RATE", "DEFAULT"}
	rows := make([][]string, 0, len(resp.ListMultiCurrency))
	for _, c := range resp.ListMultiCurrency {
		rows = append(rows, []string{
			c.UUID,
			c.Currency,
			c.CurrencySymbol,
			fmt.Sprintf("%.4f", c.Cost),
			strconv.FormatBool(c.IsDefaultCurrency),
		})
	}
	return printer.PrintTable(headers, rows)
}

func newCostTaxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "tax",
		Short: "Show tax configuration for the organization",
		Example: `  zcp cost tax
  zcp cost tax --output json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCostTax(cmd)
		},
	}
	return cmd
}

func runCostTax(cmd *cobra.Command) error {
	_, client, printer, err := buildClientAndPrinter(cmd)
	if err != nil {
		return err
	}

	svc := cost.NewService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	taxes, err := svc.GetTax(ctx)
	if err != nil {
		return fmt.Errorf("cost tax: %w", err)
	}

	headers := []string{"NAME", "TAX %", "ORG TAX", "INDIVIDUAL TAX"}
	rows := make([][]string, 0, len(taxes))
	for _, t := range taxes {
		rows = append(rows, []string{
			t.Name,
			fmt.Sprintf("%.2f", t.TaxPercentage),
			fmt.Sprintf("%.2f", t.OrganizationTax),
			fmt.Sprintf("%.2f", t.IndividualTax),
		})
	}
	return printer.PrintTable(headers, rows)
}
