package commands

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"github.com/zsoftly/zcp-cli/pkg/api/storagecategory"
)

// NewStorageCategoryCmd returns the 'storage-category' cobra command.
func NewStorageCategoryCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "storage-category",
		Short: "Manage storage categories",
	}
	cmd.AddCommand(newStorageCategoryListCmd())
	return cmd
}

// ─── List ───────────────────────────────────────────────────────────────────

func newStorageCategoryListCmd() *cobra.Command {
	var region string
	cmd := &cobra.Command{
		Use:     "list",
		Short:   "List storage categories",
		Example: `  zcp storage-category list --region yow-1`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runStorageCategoryList(cmd, region)
		},
	}
	cmd.Flags().StringVar(&region, "region", "", "Region slug (required; or set ZCP_REGION)")
	return cmd
}

func runStorageCategoryList(cmd *cobra.Command, region string) error {
	_, client, printer, err := buildClientAndPrinter(cmd)
	if err != nil {
		return err
	}

	region, err = requireRegion(cmd, region)
	if err != nil {
		return err
	}

	svc := storagecategory.NewService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	categories, err := svc.List(ctx, region)
	if err != nil {
		return fmt.Errorf("storage-category list: %w", err)
	}

	headers := []string{"ID", "NAME", "SLUG", "STATUS", "CREATED"}
	rows := make([][]string, 0, len(categories))
	for _, c := range categories {
		rows = append(rows, []string{
			c.ID,
			c.Name,
			c.Slug,
			fmt.Sprintf("%v", c.Status),
			c.CreatedAt,
		})
	}
	return printer.PrintTable(headers, rows)
}
