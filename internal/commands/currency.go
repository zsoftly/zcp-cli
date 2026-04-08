package commands

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"github.com/zsoftly/zcp-cli/internal/api/currency"
)

// NewCurrencyCmd returns the 'currency' cobra command.
func NewCurrencyCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "currency",
		Short: "Manage currencies",
	}
	cmd.AddCommand(newCurrencyListCmd())
	return cmd
}

// ─── List ───────────────────────────────────────────────────────────────────

func newCurrencyListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "list",
		Short:   "List currencies",
		Example: `  zcp currency list`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCurrencyList(cmd)
		},
	}
	return cmd
}

func runCurrencyList(cmd *cobra.Command) error {
	_, client, printer, err := buildClientAndPrinter(cmd)
	if err != nil {
		return err
	}

	svc := currency.NewService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	currencies, err := svc.List(ctx)
	if err != nil {
		return fmt.Errorf("currency list: %w", err)
	}

	headers := []string{"ID", "NAME", "SLUG", "CURRENCY NAME", "LOCALE", "DEFAULT", "STATUS", "CREATED"}
	rows := make([][]string, 0, len(currencies))
	for _, c := range currencies {
		rows = append(rows, []string{
			c.ID,
			c.Name,
			c.Slug,
			c.CurrencyName,
			c.Locale,
			fmt.Sprintf("%v", c.Default),
			fmt.Sprintf("%v", c.Status),
			c.CreatedAt,
		})
	}
	return printer.PrintTable(headers, rows)
}
