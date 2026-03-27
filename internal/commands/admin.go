package commands

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/spf13/cobra"
	"github.com/zsoftly/zcp-cli/internal/api/host"
	"github.com/zsoftly/zcp-cli/internal/api/invoice"
	"github.com/zsoftly/zcp-cli/internal/api/quota"
)

// NewAdminCmd returns the 'admin' cobra command.
// Admin commands require elevated API credentials.
func NewAdminCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "admin",
		Short: "Administrative operations (requires admin credentials)",
		Long: `Administrative operations for platform operators.

These commands access privileged APIs and require admin-level API credentials.
Customer accounts will receive authorization errors.`,
	}
	cmd.AddCommand(newAdminHostCmd())
	cmd.AddCommand(newAdminQuotaCmd())
	cmd.AddCommand(newAdminInvoiceCmd())
	return cmd
}

// ── Host ─────────────────────────────────────────────────────────────────────

func newAdminHostCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "host",
		Short: "Manage hypervisor hosts",
	}
	cmd.AddCommand(newAdminHostListCmd())
	return cmd
}

func newAdminHostListCmd() *cobra.Command {
	var hostUUID string

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List hypervisor hosts",
		Example: `  zcp admin host list
  zcp admin host list --host <uuid>
  zcp admin host list --output json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runAdminHostList(cmd, hostUUID)
		},
	}
	cmd.Flags().StringVar(&hostUUID, "host", "", "Filter by host UUID (optional)")
	return cmd
}

func runAdminHostList(cmd *cobra.Command, hostUUID string) error {
	_, client, printer, err := buildClientAndPrinter(cmd)
	if err != nil {
		return err
	}

	svc := host.NewService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	hosts, err := svc.List(ctx, hostUUID)
	if err != nil {
		return fmt.Errorf("admin host list: %w", err)
	}

	headers := []string{"UUID", "NAME", "HYPERVISOR", "POD", "CPU CORES", "CPU USED", "MEMORY TOTAL", "MEM USED %", "VMs"}
	rows := make([][]string, 0, len(hosts))
	for _, h := range hosts {
		rows = append(rows, []string{
			h.UUID,
			h.Name,
			h.Hypervisor,
			h.PodName,
			h.CPUCores,
			h.CPUUsed,
			h.MemoryTotal,
			h.MemoryUsedPercentage,
			h.VMCount,
		})
	}
	return printer.PrintTable(headers, rows)
}

// ── Quota ─────────────────────────────────────────────────────────────────────

func newAdminQuotaCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "quota",
		Short: "View resource quotas",
	}
	cmd.AddCommand(newAdminQuotaListCmd())
	return cmd
}

func newAdminQuotaListCmd() *cobra.Command {
	var domainUUID string

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List resource quotas",
		Example: `  zcp admin quota list
  zcp admin quota list --domain <uuid>
  zcp admin quota list --output json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runAdminQuotaList(cmd, domainUUID)
		},
	}
	cmd.Flags().StringVar(&domainUUID, "domain", "", "Filter by domain UUID (optional)")
	return cmd
}

func runAdminQuotaList(cmd *cobra.Command, domainUUID string) error {
	_, client, printer, err := buildClientAndPrinter(cmd)
	if err != nil {
		return err
	}

	svc := quota.NewService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	quotas, err := svc.List(ctx, domainUUID)
	if err != nil {
		return fmt.Errorf("admin quota list: %w", err)
	}

	headers := []string{"QUOTA TYPE", "UNIT", "USED", "AVAILABLE", "MAXIMUM", "DOMAIN"}
	rows := make([][]string, 0, len(quotas))
	for _, q := range quotas {
		rows = append(rows, []string{
			q.QuotaType,
			q.UnitType,
			q.UsedLimit,
			q.AvailableLimit,
			q.MaximumLimit,
			q.DomainUUID,
		})
	}
	return printer.PrintTable(headers, rows)
}

// ── Invoice ───────────────────────────────────────────────────────────────────

func newAdminInvoiceCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "invoice",
		Short: "Manage invoices",
	}
	cmd.AddCommand(newAdminInvoiceListCmd())
	cmd.AddCommand(newAdminInvoiceGenerateCmd())
	return cmd
}

func newAdminInvoiceListCmd() *cobra.Command {
	var email, period, status string

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List invoices",
		Example: `  zcp admin invoice list
  zcp admin invoice list --email client@example.com
  zcp admin invoice list --period 2025-01 --status paid
  zcp admin invoice list --output json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runAdminInvoiceList(cmd, email, period, status)
		},
	}
	cmd.Flags().StringVar(&email, "email", "", "Filter by client email (optional)")
	cmd.Flags().StringVar(&period, "period", "", "Filter by billing period (optional, format: YYYY-MM)")
	cmd.Flags().StringVar(&status, "status", "", "Filter by invoice status (optional)")
	return cmd
}

func runAdminInvoiceList(cmd *cobra.Command, email, period, status string) error {
	_, client, printer, err := buildClientAndPrinter(cmd)
	if err != nil {
		return err
	}

	svc := invoice.NewService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	invoices, err := svc.List(ctx, email, status, period)
	if err != nil {
		return fmt.Errorf("admin invoice list: %w", err)
	}

	headers := []string{"NUMBER", "CLIENT", "PERIOD", "TOTAL", "CURRENCY", "DATE"}
	rows := make([][]string, 0, len(invoices))
	for _, inv := range invoices {
		rows = append(rows, []string{
			inv.InvoiceNumber,
			inv.ClientEmail,
			inv.BillPeriod,
			fmt.Sprintf("%.2f", inv.TotalCost),
			inv.Currency,
			inv.GeneratedDate,
		})
	}
	return printer.PrintTable(headers, rows)
}

func newAdminInvoiceGenerateCmd() *cobra.Command {
	var number string

	cmd := &cobra.Command{
		Use:     "generate",
		Short:   "Generate an invoice by invoice number",
		Example: `  zcp admin invoice generate --number 12345`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if number == "" {
				return fmt.Errorf("--number is required")
			}
			return runAdminInvoiceGenerate(cmd, number)
		},
	}
	cmd.Flags().StringVar(&number, "number", "", "Invoice number (required)")
	return cmd
}

func runAdminInvoiceGenerate(cmd *cobra.Command, number string) error {
	_, client, printer, err := buildClientAndPrinter(cmd)
	if err != nil {
		return err
	}

	svc := invoice.NewService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	resp, err := svc.Generate(ctx, number)
	if err != nil {
		return fmt.Errorf("admin invoice generate: %w", err)
	}

	printer.Fprintf("Invoice %d: %s (status: %s)\n",
		resp.InvoiceNumber,
		resp.Message,
		strconv.FormatBool(resp.Status),
	)
	return nil
}
