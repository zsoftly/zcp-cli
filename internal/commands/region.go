package commands

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"github.com/zsoftly/zcp-cli/pkg/api/region"
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

	// PROVIDER and COMING SOON are intentionally omitted: the provider name
	// (e.g. "Cloud Stack", "Ceph") leaks backend technology and must not be
	// exposed to users.
	headers := []string{"SLUG", "NAME", "COUNTRY", "CONTINENT", "STATUS"}
	rows := make([][]string, 0, len(regions))
	for _, r := range regions {
		status := "active"
		if !r.Status {
			status = "inactive"
		}
		rows = append(rows, []string{
			r.Slug,
			r.Name,
			r.Country,
			r.ContinentName,
			status,
		})
	}
	return printer.PrintTable(headers, rows)
}
