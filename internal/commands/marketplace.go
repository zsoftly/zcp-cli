package commands

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/spf13/cobra"
	"github.com/zsoftly/zcp-cli/internal/api/marketplace"
)

// NewMarketplaceCmd returns the 'marketplace' cobra command.
func NewMarketplaceCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "marketplace",
		Aliases: []string{"apps"},
		Short:   "Browse marketplace applications",
	}
	cmd.AddCommand(newMarketplaceListCmd())
	return cmd
}

func newMarketplaceListCmd() *cobra.Command {
	var (
		region  string
		include string
	)

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List marketplace applications",
		Example: `  zcp marketplace list
  zcp marketplace list --region zone-my-bangsarsouth
  zcp marketplace list --include versions
  zcp marketplace list --output json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runMarketplaceList(cmd, region, include)
		},
	}
	cmd.Flags().StringVar(&region, "region", "", "Filter by region slug")
	cmd.Flags().StringVar(&include, "include", "", "Include related data (e.g. versions)")
	return cmd
}

func runMarketplaceList(cmd *cobra.Command, region, include string) error {
	_, client, printer, err := buildClientAndPrinter(cmd)
	if err != nil {
		return err
	}

	svc := marketplace.NewService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	apps, err := svc.ListApps(ctx, region, include)
	if err != nil {
		return fmt.Errorf("marketplace list: %w", err)
	}

	if len(apps) == 0 {
		printer.Fprintf("No marketplace apps found\n")
		return nil
	}

	headers := []string{"NAME", "SLUG", "CATEGORY", "FEATURED", "DESCRIPTION", "URL"}
	rows := make([][]string, 0, len(apps))
	for _, app := range apps {
		desc := app.ShortDescription
		if len(desc) > 60 {
			desc = desc[:57] + "..."
		}
		rows = append(rows, []string{
			app.Name,
			app.Slug,
			app.Category,
			strconv.FormatBool(app.IsFeatured),
			desc,
			app.URL,
		})
	}
	return printer.PrintTable(headers, rows)
}
