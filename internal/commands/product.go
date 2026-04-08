package commands

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/spf13/cobra"
	"github.com/zsoftly/zcp-cli/internal/api/product"
)

// NewProductCmd returns the 'product' cobra command.
func NewProductCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "product",
		Short: "View products and product categories",
	}
	cmd.AddCommand(newProductCategoriesCmd())
	cmd.AddCommand(newProductListCmd())
	return cmd
}

func newProductCategoriesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "categories",
		Short: "List product categories",
		Example: `  zcp product categories
  zcp product categories --output json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runProductCategories(cmd)
		},
	}
	return cmd
}

func runProductCategories(cmd *cobra.Command) error {
	_, client, printer, err := buildClientAndPrinter(cmd)
	if err != nil {
		return err
	}

	svc := product.NewService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	categories, err := svc.ListCategories(ctx)
	if err != nil {
		return fmt.Errorf("product categories: %w", err)
	}

	if len(categories) == 0 {
		printer.Fprintf("No product categories found\n")
		return nil
	}

	headers := []string{"ID", "NAME", "SLUG", "DESCRIPTION", "STATUS", "CREATED"}
	rows := make([][]string, 0, len(categories))
	for _, c := range categories {
		rows = append(rows, []string{
			c.ID,
			c.Name,
			c.Slug,
			c.Description,
			strconv.FormatBool(c.Status),
			c.CreatedAt,
		})
	}
	return printer.PrintTable(headers, rows)
}

func newProductListCmd() *cobra.Command {
	var (
		cardType string
		cardSlug string
		include  string
	)

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all products",
		Example: `  zcp product list
  zcp product list --card-type RateCard --card-slug default
  zcp product list --include product_category
  zcp product list --output json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runProductList(cmd, cardType, cardSlug, include)
		},
	}
	cmd.Flags().StringVar(&cardType, "card-type", "", "Card type filter (e.g. RateCard)")
	cmd.Flags().StringVar(&cardSlug, "card-slug", "", "Card slug filter (e.g. default)")
	cmd.Flags().StringVar(&include, "include", "", "Include related data (e.g. product_category)")
	return cmd
}

func runProductList(cmd *cobra.Command, cardType, cardSlug, include string) error {
	_, client, printer, err := buildClientAndPrinter(cmd)
	if err != nil {
		return err
	}

	svc := product.NewService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	products, err := svc.ListAll(ctx, cardType, cardSlug, include)
	if err != nil {
		return fmt.Errorf("product list: %w", err)
	}

	if len(products) == 0 {
		printer.Fprintf("No products found\n")
		return nil
	}

	headers := []string{"ID", "NAME", "SLUG", "DESCRIPTION", "STATUS", "CATEGORY ID", "CREATED"}
	rows := make([][]string, 0, len(products))
	for _, p := range products {
		categoryID := p.ProductCategoryID
		if p.ProductCategory != nil {
			categoryID = p.ProductCategory.Name
		}
		rows = append(rows, []string{
			p.ID,
			p.Name,
			p.Slug,
			p.Description,
			strconv.FormatBool(p.Status),
			categoryID,
			p.CreatedAt,
		})
	}
	return printer.PrintTable(headers, rows)
}
