package commands

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"github.com/zsoftly/zcp-cli/internal/api/template"
)

// NewTemplateCmd returns the 'template' cobra command.
func NewTemplateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "template",
		Short: "Manage VM templates",
	}
	cmd.AddCommand(newTemplateListCmd())
	return cmd
}

func newTemplateListCmd() *cobra.Command {
	var zoneUUID string

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List available templates",
		Example: `  zcp template list
  zcp template list --zone <uuid>
  zcp template list --output json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			_, client, printer, err := buildClientAndPrinter(cmd)
			if err != nil {
				return err
			}
			svc := template.NewService(client)
			ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
			defer cancel()

			templates, err := svc.List(ctx, zoneUUID, "")
			if err != nil {
				return fmt.Errorf("template list: %w", err)
			}

			headers := []string{"UUID", "NAME", "OS CATEGORY", "FORMAT", "ZONE", "ACTIVE"}
			rows := make([][]string, 0, len(templates))
			for _, t := range templates {
				rows = append(rows, []string{
					t.UUID, t.Name, t.OsCategoryName, t.Format, t.ZoneName, t.IsActive,
				})
			}
			return printer.PrintTable(headers, rows)
		},
	}
	cmd.Flags().StringVar(&zoneUUID, "zone", "", "Filter by zone UUID")
	return cmd
}
