package commands

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/spf13/cobra"
	"github.com/zsoftly/zcp-cli/internal/api/billing"
)

// NewBillingCmd returns the 'billing' cobra command group.
func NewBillingCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "billing",
		Short: "View billing, costs, usage, invoices, subscriptions, and payments",
	}
	cmd.AddCommand(newBillingBalanceCmd())
	cmd.AddCommand(newBillingCostsCmd())
	cmd.AddCommand(newBillingMonthlyUsageCmd())
	cmd.AddCommand(newBillingServiceCountsCmd())
	cmd.AddCommand(newBillingCreditLimitCmd())
	cmd.AddCommand(newBillingInvoicesCmd())
	cmd.AddCommand(newBillingInvoiceCountCmd())
	cmd.AddCommand(newBillingUsageCmd())
	cmd.AddCommand(newBillingFreeCreditsCmd())
	cmd.AddCommand(newBillingSubscriptionsCmd())
	cmd.AddCommand(newBillingContractsCmd())
	cmd.AddCommand(newBillingTrialsCmd())
	cmd.AddCommand(newBillingCancelRequestsCmd())
	cmd.AddCommand(newBillingCancelServiceCmd())
	cmd.AddCommand(newBillingPaymentsCmd())
	cmd.AddCommand(newBillingCouponsCmd())
	cmd.AddCommand(newBillingRedeemCouponCmd())
	cmd.AddCommand(newBillingBudgetAlertCmd())
	cmd.AddCommand(newBillingBudgetAlertSetCmd())
	return cmd
}

// --- balance ---

func newBillingBalanceCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "balance",
		Short: "Show account balance summary",
		Example: `  zcp billing balance
  zcp billing balance --output json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runBillingBalance(cmd)
		},
	}
}

func runBillingBalance(cmd *cobra.Command) error {
	_, client, printer, err := buildClientAndPrinter(cmd)
	if err != nil {
		return err
	}

	svc := billing.NewService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	bal, err := svc.GetBalance(ctx)
	if err != nil {
		return fmt.Errorf("billing balance: %w", err)
	}

	headers := []string{"FIELD", "VALUE"}
	rows := [][]string{
		{"Available Balance", fmt.Sprintf("%.2f", bal.AvailableBalance)},
		{"Available Net Balance", fmt.Sprintf("%.2f", bal.AvailableNetBalance)},
		{"Deposited", fmt.Sprintf("%.2f", bal.Deposited)},
		{"Charged", fmt.Sprintf("%.2f", bal.Charged)},
		{"Current Usage", fmt.Sprintf("%.2f", bal.CurrentUsage)},
		{"Hourly Usage", fmt.Sprintf("%.6f", bal.HourlyUsage)},
		{"Current Hourly Rate", fmt.Sprintf("%.7f", bal.CurrentHourlyRate)},
		{"All-Time Usage", fmt.Sprintf("%.2f", bal.AllTimeUsage)},
		{"Current Month Usage", fmt.Sprintf("%.2f", bal.CurrentMonthUsage)},
		{"Estimated Hourly Usage", fmt.Sprintf("%.4f", bal.EstimatedHourlyUsage)},
		{"Free Credits", fmt.Sprintf("%.2f", bal.AvailableFreeCredits)},
		{"Unpaid Invoices", fmt.Sprintf("%.2f", bal.UnpaidInvoices)},
		{"Subscription Amount", fmt.Sprintf("%.4f", bal.SubscriptionAmount)},
	}
	return printer.PrintTable(headers, rows)
}

// --- costs ---

func newBillingCostsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "costs",
		Short: "Show per-service cost breakdown",
		Example: `  zcp billing costs
  zcp billing costs --output json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runBillingCosts(cmd)
		},
	}
}

func runBillingCosts(cmd *cobra.Command) error {
	_, client, printer, err := buildClientAndPrinter(cmd)
	if err != nil {
		return err
	}

	svc := billing.NewService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	costs, err := svc.ListServiceCosts(ctx)
	if err != nil {
		return fmt.Errorf("billing costs: %w", err)
	}

	headers := []string{"SERVICE", "DISPLAY NAME", "TOTAL COST"}
	rows := make([][]string, 0, len(costs))
	for _, c := range costs {
		if c.TotalCost > 0 {
			rows = append(rows, []string{
				c.Name,
				c.DisplayName,
				fmt.Sprintf("%.4f", c.TotalCost),
			})
		}
	}
	// If no costs with value, show all
	if len(rows) == 0 {
		for _, c := range costs {
			rows = append(rows, []string{
				c.Name,
				c.DisplayName,
				fmt.Sprintf("%.4f", c.TotalCost),
			})
		}
	}
	return printer.PrintTable(headers, rows)
}

// --- monthly-usage ---

func newBillingMonthlyUsageCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "monthly-usage",
		Short: "Show month-by-month usage history",
		Example: `  zcp billing monthly-usage
  zcp billing monthly-usage --output json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runBillingMonthlyUsage(cmd)
		},
	}
}

func runBillingMonthlyUsage(cmd *cobra.Command) error {
	_, client, printer, err := buildClientAndPrinter(cmd)
	if err != nil {
		return err
	}

	svc := billing.NewService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	usage, err := svc.ListMonthlyUsage(ctx)
	if err != nil {
		return fmt.Errorf("billing monthly-usage: %w", err)
	}

	headers := []string{"MONTH", "YEAR", "COST"}
	rows := make([][]string, 0, len(usage))
	for _, u := range usage {
		rows = append(rows, []string{
			u.Month,
			u.Year,
			u.Cost.String(),
		})
	}
	return printer.PrintTable(headers, rows)
}

// --- service-counts ---

func newBillingServiceCountsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "service-counts",
		Short: "Show active service counts by type",
		Example: `  zcp billing service-counts
  zcp billing service-counts --output json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runBillingServiceCounts(cmd)
		},
	}
}

func runBillingServiceCounts(cmd *cobra.Command) error {
	_, client, printer, err := buildClientAndPrinter(cmd)
	if err != nil {
		return err
	}

	svc := billing.NewService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	counts, err := svc.GetServiceCounts(ctx)
	if err != nil {
		return fmt.Errorf("billing service-counts: %w", err)
	}

	// Sort keys for deterministic output
	keys := make([]string, 0, len(counts))
	for k := range counts {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	headers := []string{"SERVICE", "COUNT"}
	rows := make([][]string, 0, len(counts))
	for _, k := range keys {
		rows = append(rows, []string{
			k,
			fmt.Sprintf("%d", counts[k]),
		})
	}
	return printer.PrintTable(headers, rows)
}

// --- credit-limit ---

func newBillingCreditLimitCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "credit-limit",
		Short: "Show account credit limit",
		Example: `  zcp billing credit-limit
  zcp billing credit-limit --output json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runBillingCreditLimit(cmd)
		},
	}
}

func runBillingCreditLimit(cmd *cobra.Command) error {
	_, client, printer, err := buildClientAndPrinter(cmd)
	if err != nil {
		return err
	}

	svc := billing.NewService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	limit, err := svc.GetCreditLimit(ctx)
	if err != nil {
		return fmt.Errorf("billing credit-limit: %w", err)
	}

	headers := []string{"CREDIT LIMIT", "USAGE AMOUNT", "AVAILABLE TO SPEND"}
	rows := [][]string{
		{
			limit.Limit,
			fmt.Sprintf("%.2f", limit.UsageAmount),
			fmt.Sprintf("%.2f", limit.AvailableToSpend),
		},
	}
	return printer.PrintTable(headers, rows)
}

// --- invoices ---

func newBillingInvoicesCmd() *cobra.Command {
	var page int

	cmd := &cobra.Command{
		Use:   "invoices",
		Short: "List billing invoices",
		Example: `  zcp billing invoices
  zcp billing invoices --page 2
  zcp billing invoices --output json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runBillingInvoices(cmd, page)
		},
	}
	cmd.Flags().IntVar(&page, "page", 0, "Page number for paginated results")
	return cmd
}

func runBillingInvoices(cmd *cobra.Command, page int) error {
	_, client, printer, err := buildClientAndPrinter(cmd)
	if err != nil {
		return err
	}

	svc := billing.NewService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	invoices, total, err := svc.ListInvoices(ctx, page)
	if err != nil {
		return fmt.Errorf("billing invoices: %w", err)
	}

	headers := []string{"NUMBER", "AMOUNT", "TAX", "STATUS", "TYPE", "DATE", "PAID AT", "PAYMENT METHOD"}
	rows := make([][]string, 0, len(invoices))
	for _, inv := range invoices {
		rows = append(rows, []string{
			inv.CustomNumber,
			inv.SubAmount,
			fmt.Sprintf("%.2f", inv.TaxAmount),
			inv.Status,
			inv.Type,
			inv.InvoiceAt,
			inv.PaidAt,
			inv.PaymentMethods,
		})
	}
	if err := printer.PrintTable(headers, rows); err != nil {
		return err
	}
	printer.Fprintf("Total invoices: %d\n", total)
	return nil
}

// --- invoices-count ---

func newBillingInvoiceCountCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "invoices-count",
		Short:   "Show total number of invoices",
		Example: `  zcp billing invoices-count`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runBillingInvoiceCount(cmd)
		},
	}
}

func runBillingInvoiceCount(cmd *cobra.Command) error {
	_, client, printer, err := buildClientAndPrinter(cmd)
	if err != nil {
		return err
	}

	svc := billing.NewService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	count, err := svc.GetInvoiceCount(ctx)
	if err != nil {
		return fmt.Errorf("billing invoices-count: %w", err)
	}

	headers := []string{"INVOICE COUNT"}
	rows := [][]string{{fmt.Sprintf("%d", count)}}
	return printer.PrintTable(headers, rows)
}

// --- usage ---

func newBillingUsageCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "usage",
		Short: "Show detailed account usage",
		Example: `  zcp billing usage
  zcp billing usage --output json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runBillingUsage(cmd)
		},
	}
}

func runBillingUsage(cmd *cobra.Command) error {
	_, client, printer, err := buildClientAndPrinter(cmd)
	if err != nil {
		return err
	}

	svc := billing.NewService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	result, err := svc.GetAccountUsage(ctx)
	if err != nil {
		return fmt.Errorf("billing usage: %w", err)
	}

	return printer.Print(result)
}

// --- free-credits ---

func newBillingFreeCreditsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "free-credits",
		Short: "Show available free credits",
		Example: `  zcp billing free-credits
  zcp billing free-credits --output json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runBillingFreeCredits(cmd)
		},
	}
}

func runBillingFreeCredits(cmd *cobra.Command) error {
	_, client, printer, err := buildClientAndPrinter(cmd)
	if err != nil {
		return err
	}

	svc := billing.NewService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	result, err := svc.GetFreeCredits(ctx)
	if err != nil {
		return fmt.Errorf("billing free-credits: %w", err)
	}

	return printer.Print(result)
}

// --- subscriptions ---

func newBillingSubscriptionsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "subscriptions",
		Short: "View active and inactive subscriptions",
	}
	cmd.AddCommand(newBillingSubscriptionsActiveCmd())
	cmd.AddCommand(newBillingSubscriptionsInactiveCmd())
	return cmd
}

func newBillingSubscriptionsActiveCmd() *cobra.Command {
	var page int

	cmd := &cobra.Command{
		Use:   "active",
		Short: "List active service subscriptions",
		Example: `  zcp billing subscriptions active
  zcp billing subscriptions active --page 2
  zcp billing subscriptions active --output json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runBillingSubscriptionsActive(cmd, page)
		},
	}
	cmd.Flags().IntVar(&page, "page", 0, "Page number for paginated results")
	return cmd
}

func runBillingSubscriptionsActive(cmd *cobra.Command, page int) error {
	_, client, printer, err := buildClientAndPrinter(cmd)
	if err != nil {
		return err
	}

	svc := billing.NewService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	subs, total, err := svc.ListActiveSubscriptions(ctx, page)
	if err != nil {
		return fmt.Errorf("billing subscriptions active: %w", err)
	}

	headers := []string{"NAME", "PRODUCT", "PRICE", "TOTAL USAGE", "BILLING CYCLE", "PROJECT", "RENEW AT"}
	rows := make([][]string, 0, len(subs))
	for _, sub := range subs {
		rows = append(rows, []string{
			sub.Name,
			sub.ProductDisplayName,
			sub.Price,
			sub.TotalUsage,
			sub.BillingCycle.Name,
			sub.Project.Name,
			sub.RenewAt,
		})
	}
	if err := printer.PrintTable(headers, rows); err != nil {
		return err
	}
	printer.Fprintf("Total active subscriptions: %d\n", total)
	return nil
}

func newBillingSubscriptionsInactiveCmd() *cobra.Command {
	var page int

	cmd := &cobra.Command{
		Use:   "inactive",
		Short: "List inactive service subscriptions",
		Example: `  zcp billing subscriptions inactive
  zcp billing subscriptions inactive --output json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runBillingSubscriptionsInactive(cmd, page)
		},
	}
	cmd.Flags().IntVar(&page, "page", 0, "Page number for paginated results")
	return cmd
}

func runBillingSubscriptionsInactive(cmd *cobra.Command, page int) error {
	_, client, printer, err := buildClientAndPrinter(cmd)
	if err != nil {
		return err
	}

	svc := billing.NewService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	subs, total, err := svc.ListInactiveSubscriptions(ctx, page)
	if err != nil {
		return fmt.Errorf("billing subscriptions inactive: %w", err)
	}

	headers := []string{"NAME", "PRODUCT", "PRICE", "TOTAL USAGE", "BILLING CYCLE", "PROJECT"}
	rows := make([][]string, 0, len(subs))
	for _, sub := range subs {
		rows = append(rows, []string{
			sub.Name,
			sub.ProductDisplayName,
			sub.Price,
			sub.TotalUsage,
			sub.BillingCycle.Name,
			sub.Project.Name,
		})
	}
	if err := printer.PrintTable(headers, rows); err != nil {
		return err
	}
	printer.Fprintf("Total inactive subscriptions: %d\n", total)
	return nil
}

// --- contracts ---

func newBillingContractsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "contracts",
		Short: "List service contracts",
		Example: `  zcp billing contracts
  zcp billing contracts --output json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runBillingContracts(cmd)
		},
	}
}

func runBillingContracts(cmd *cobra.Command) error {
	_, client, printer, err := buildClientAndPrinter(cmd)
	if err != nil {
		return err
	}

	svc := billing.NewService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	result, err := svc.ListServiceContracts(ctx)
	if err != nil {
		return fmt.Errorf("billing contracts: %w", err)
	}

	return printer.Print(result)
}

// --- trials ---

func newBillingTrialsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "trials",
		Short: "List active free trials",
		Example: `  zcp billing trials
  zcp billing trials --output json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runBillingTrials(cmd)
		},
	}
}

func runBillingTrials(cmd *cobra.Command) error {
	_, client, printer, err := buildClientAndPrinter(cmd)
	if err != nil {
		return err
	}

	svc := billing.NewService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	result, err := svc.ListServiceTrials(ctx)
	if err != nil {
		return fmt.Errorf("billing trials: %w", err)
	}

	return printer.Print(result)
}

// --- cancel-requests ---

func newBillingCancelRequestsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "cancel-requests",
		Short: "List scheduled service cancellation requests",
		Example: `  zcp billing cancel-requests
  zcp billing cancel-requests --output json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runBillingCancelRequests(cmd)
		},
	}
}

func runBillingCancelRequests(cmd *cobra.Command) error {
	_, client, printer, err := buildClientAndPrinter(cmd)
	if err != nil {
		return err
	}

	svc := billing.NewService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	result, err := svc.ListCancelRequests(ctx)
	if err != nil {
		return fmt.Errorf("billing cancel-requests: %w", err)
	}

	return printer.Print(result)
}

// --- cancel-service ---

func newBillingCancelServiceCmd() *cobra.Command {
	var serviceName, reason, cancelType, description string
	cmd := &cobra.Command{
		Use:   "cancel-service <subscription-slug>",
		Short: "Submit a cancellation request for a service",
		Example: `  zcp billing cancel-service demo-prj-vm --service "Virtual Machine" --reason not_needed_anymore
  zcp billing cancel-service root-4153 --service "Block Storage" --reason not_needed_anymore --type Immediate`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if serviceName == "" {
				return fmt.Errorf("--service is required (e.g. 'Virtual Machine', 'Block Storage', 'IP Address')")
			}
			if reason == "" {
				reason = "not_needed_anymore"
			}
			if cancelType == "" {
				cancelType = "Immediate"
			}
			return runBillingCancelService(cmd, args[0], serviceName, reason, cancelType, description)
		},
	}
	cmd.Flags().StringVar(&serviceName, "service", "", "Service type (e.g. 'Virtual Machine', 'Block Storage', 'IP Address', 'Object Storage')")
	cmd.Flags().StringVar(&reason, "reason", "not_needed_anymore", "Reason: limit_expenses, not_needed_anymore, better_offer, not_satisfied, switch_product, other")
	cmd.Flags().StringVar(&cancelType, "type", "Immediate", "Cancel type: Immediate or 'End of billing period'")
	cmd.Flags().StringVar(&description, "description", "", "Additional description (optional)")
	return cmd
}

func runBillingCancelService(cmd *cobra.Command, slug, serviceName, reason, cancelType, description string) error {
	_, client, printer, err := buildClientAndPrinter(cmd)
	if err != nil {
		return err
	}

	svc := billing.NewService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	req := billing.CancelServiceRequest{
		ServiceName: serviceName,
		Reason:      reason,
		Type:        cancelType,
		Description: description,
	}
	if err := svc.CancelService(ctx, slug, req); err != nil {
		return fmt.Errorf("billing cancel-service: %w", err)
	}

	printer.Fprintf("Cancellation request submitted for %s (%s)\n", slug, serviceName)
	return nil
}

// --- payments ---

func newBillingPaymentsCmd() *cobra.Command {
	var page int

	cmd := &cobra.Command{
		Use:   "payments",
		Short: "List payment transactions",
		Example: `  zcp billing payments
  zcp billing payments --page 2
  zcp billing payments --output json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runBillingPayments(cmd, page)
		},
	}
	cmd.Flags().IntVar(&page, "page", 0, "Page number for paginated results")
	return cmd
}

func runBillingPayments(cmd *cobra.Command, page int) error {
	_, client, printer, err := buildClientAndPrinter(cmd)
	if err != nil {
		return err
	}

	svc := billing.NewService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	result, err := svc.ListPayments(ctx, page)
	if err != nil {
		return fmt.Errorf("billing payments: %w", err)
	}

	return printer.Print(result)
}

// --- coupons ---

func newBillingCouponsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "coupons",
		Short: "List coupons associated with the account",
		Example: `  zcp billing coupons
  zcp billing coupons --output json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runBillingCoupons(cmd)
		},
	}
}

func runBillingCoupons(cmd *cobra.Command) error {
	_, client, printer, err := buildClientAndPrinter(cmd)
	if err != nil {
		return err
	}

	svc := billing.NewService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	result, err := svc.ListCoupons(ctx)
	if err != nil {
		return fmt.Errorf("billing coupons: %w", err)
	}

	return printer.Print(result)
}

// --- redeem-coupon ---

func newBillingRedeemCouponCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "redeem-coupon <code>",
		Short:   "Apply a coupon code to the account",
		Example: `  zcp billing redeem-coupon SAVE50`,
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runBillingRedeemCoupon(cmd, args[0])
		},
	}
}

func runBillingRedeemCoupon(cmd *cobra.Command, code string) error {
	_, client, printer, err := buildClientAndPrinter(cmd)
	if err != nil {
		return err
	}

	svc := billing.NewService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	result, err := svc.RedeemCoupon(ctx, code)
	if err != nil {
		return fmt.Errorf("billing redeem-coupon: %w", err)
	}

	return printer.Print(result)
}

// --- budget-alert ---

func newBillingBudgetAlertCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "budget-alert",
		Short: "Show current budget alert settings",
		Example: `  zcp billing budget-alert
  zcp billing budget-alert --output json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runBillingBudgetAlert(cmd)
		},
	}
}

func runBillingBudgetAlert(cmd *cobra.Command) error {
	_, client, printer, err := buildClientAndPrinter(cmd)
	if err != nil {
		return err
	}

	svc := billing.NewService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	result, err := svc.GetBudgetAlert(ctx)
	if err != nil {
		return fmt.Errorf("billing budget-alert: %w", err)
	}

	return printer.Print(result)
}

// --- budget-alert-set ---

func newBillingBudgetAlertSetCmd() *cobra.Command {
	var amount, threshold float64
	var enabled bool

	cmd := &cobra.Command{
		Use:   "budget-alert-set",
		Short: "Configure budget alert settings",
		Example: `  zcp billing budget-alert-set --amount 500 --threshold 80 --enabled
  zcp billing budget-alert-set --amount 1000 --threshold 90 --enabled=false`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runBillingBudgetAlertSet(cmd, amount, threshold, enabled)
		},
	}
	cmd.Flags().Float64Var(&amount, "amount", 0, "Budget amount (required)")
	cmd.Flags().Float64Var(&threshold, "threshold", 0, "Alert threshold percentage (required)")
	cmd.Flags().BoolVar(&enabled, "enabled", true, "Enable or disable the alert")
	cmd.MarkFlagRequired("amount")
	cmd.MarkFlagRequired("threshold")
	return cmd
}

func runBillingBudgetAlertSet(cmd *cobra.Command, amount, threshold float64, enabled bool) error {
	_, client, printer, err := buildClientAndPrinter(cmd)
	if err != nil {
		return err
	}

	svc := billing.NewService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	result, err := svc.SetBudgetAlert(ctx, billing.SetBudgetAlertRequest{
		Amount:    amount,
		Threshold: threshold,
		IsEnabled: enabled,
	})
	if err != nil {
		return fmt.Errorf("billing budget-alert-set: %w", err)
	}

	return printer.Print(result)
}
