package commands

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"github.com/zsoftly/zcp-cli/internal/api/store"
)

// NewStoreCmd returns the 'store' cobra command.
func NewStoreCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "store",
		Short: "Manage store items and checkout",
	}
	cmd.AddCommand(newStoreListCmd())
	cmd.AddCommand(newStoreCheckoutCmd())
	return cmd
}

func newStoreListCmd() *cobra.Command {
	var (
		sort  string
		page  int
		limit int
	)

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List store items",
		Example: `  zcp store list
  zcp store list --sort -created_at
  zcp store list --page 1 --limit 10
  zcp store list --output json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runStoreList(cmd, sort, page, limit)
		},
	}
	cmd.Flags().StringVar(&sort, "sort", "-created_at", "Sort order (e.g. -created_at)")
	cmd.Flags().IntVar(&page, "page", 1, "Page number")
	cmd.Flags().IntVar(&limit, "limit", 0, "Items per page (0 = all)")
	return cmd
}

func runStoreList(cmd *cobra.Command, sort string, page, limit int) error {
	_, client, printer, err := buildClientAndPrinter(cmd)
	if err != nil {
		return err
	}

	svc := store.NewService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	items, total, err := svc.ListItems(ctx, sort, page, limit)
	if err != nil {
		return fmt.Errorf("store list: %w", err)
	}

	if len(items) == 0 {
		printer.Fprintf("No store items found (total: %d)\n", total)
		return nil
	}

	headers := []string{"ID", "NAME", "SLUG", "DESCRIPTION", "STATUS", "CREATED"}
	rows := make([][]string, 0, len(items))
	for _, item := range items {
		rows = append(rows, []string{
			item.ID,
			item.Name,
			item.Slug,
			item.Description,
			item.Status,
			item.CreatedAt,
		})
	}
	return printer.PrintTable(headers, rows)
}

func newStoreCheckoutCmd() *cobra.Command {
	var (
		productSlug  string
		description  string
		quantity     int
		billingCycle string
		coupon       string
	)

	cmd := &cobra.Command{
		Use:   "checkout",
		Short: "Purchase a store product",
		Example: `  zcp store checkout --product product-002 --description "Testing" --quantity 1
  zcp store checkout --product product-002 --description "Order" --quantity 2 --billing-cycle monthly
  zcp store checkout --product product-002 --description "Order" --quantity 1 --coupon SAVE10`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runStoreCheckout(cmd, productSlug, description, quantity, billingCycle, coupon)
		},
	}
	cmd.Flags().StringVar(&productSlug, "product", "", "Product slug (required)")
	cmd.Flags().StringVar(&description, "description", "", "Order description (required)")
	cmd.Flags().IntVar(&quantity, "quantity", 1, "Quantity to purchase")
	cmd.Flags().StringVar(&billingCycle, "billing-cycle", "monthly", "Billing cycle (e.g. hourly, monthly)")
	cmd.Flags().StringVar(&coupon, "coupon", "", "Coupon code (optional)")
	cmd.MarkFlagRequired("product")
	cmd.MarkFlagRequired("description")
	return cmd
}

func runStoreCheckout(cmd *cobra.Command, productSlug, description string, quantity int, billingCycle, coupon string) error {
	_, client, printer, err := buildClientAndPrinter(cmd)
	if err != nil {
		return err
	}

	svc := store.NewService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	req := store.CheckoutRequest{
		Service: "Store",
		Products: []store.CheckoutProduct{
			{
				Description: description,
				Product:     productSlug,
				Quantity:    quantity,
				Status:      "Completed",
			},
		},
		BillingCycle: billingCycle,
	}
	if coupon != "" {
		req.Coupon = &coupon
	}

	if err := svc.Checkout(ctx, req); err != nil {
		return fmt.Errorf("store checkout: %w", err)
	}

	printer.Fprintf("Checkout completed successfully for product %q (quantity: %d)\n", productSlug, quantity)
	return nil
}
