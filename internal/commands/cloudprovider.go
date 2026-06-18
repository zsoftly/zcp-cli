package commands

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"github.com/zsoftly/zcp-cli/pkg/api/cloudprovider"
)

// NewCloudProviderCmd returns the 'cloud-provider' cobra command.
func NewCloudProviderCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cloud-provider",
		Short: "Manage cloud providers",
	}
	cmd.AddCommand(newCloudProviderListCmd())
	return cmd
}

// ─── List ───────────────────────────────────────────────────────────────────

func newCloudProviderListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "list",
		Short:   "List cloud providers",
		Example: `  zcp cloud-provider list`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runCloudProviderList(cmd)
		},
	}
	return cmd
}

func runCloudProviderList(cmd *cobra.Command) error {
	_, client, printer, err := buildClientAndPrinter(cmd)
	if err != nil {
		return err
	}

	svc := cloudprovider.NewService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	providers, err := svc.List(ctx)
	if err != nil {
		return fmt.Errorf("cloud-provider list: %w", err)
	}

	// DISPLAY NAME is intentionally omitted: it can surface backend technology
	// names (e.g. "Cloud Stack", "Ceph"). SLUG is the value used by
	// --cloud-provider, so it is kept; the human label is not exposed.
	headers := []string{"ID", "NAME", "SLUG", "STATUS", "MULTI-REGION", "CREATED"}
	rows := make([][]string, 0, len(providers))
	for _, p := range providers {
		rows = append(rows, []string{
			p.ID,
			p.Name,
			p.Slug,
			fmt.Sprintf("%v", p.Status),
			fmt.Sprintf("%v", p.IsMultiRegionSetup),
			p.CreatedAt,
		})
	}
	return printer.PrintTable(headers, rows)
}
