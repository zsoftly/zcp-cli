package commands

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"github.com/zsoftly/zcp-cli/internal/api/region"
)

// NewRegionCmd returns the 'region' cobra command.
func NewRegionCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "region",
		Short:   "Manage regions",
		Aliases: []string{"regions"},
	}
	cmd.AddCommand(newRegionListCmd())
	return cmd
}

func newRegionListCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "list",
		Short:   "List available regions",
		Example: `  zcp region list`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runRegionList(cmd)
		},
	}
}

func runRegionList(cmd *cobra.Command) error {
	_, client, printer, err := buildClientAndPrinter(cmd)
	if err != nil {
		return err
	}

	svc := region.NewService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	regions, err := svc.List(ctx)
	if err != nil {
		return fmt.Errorf("region list: %w", err)
	}

	headers := []string{"SLUG", "NAME", "COUNTRY", "CONTINENT", "PROVIDER", "STATUS", "COMING SOON"}
	rows := make([][]string, 0, len(regions))
	for _, r := range regions {
		provider := ""
		if r.CloudProvider != nil {
			provider = r.CloudProvider.DisplayName
		}
		status := "active"
		if !r.Status {
			status = "inactive"
		}
		comingSoon := ""
		if r.IsComingSoon {
			comingSoon = "yes"
		}
		rows = append(rows, []string{
			r.Slug,
			r.Name,
			r.Country,
			r.ContinentName,
			provider,
			status,
			comingSoon,
		})
	}
	return printer.PrintTable(headers, rows)
}
