package commands

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/zsoftly/zcp-cli/internal/output"
	"github.com/zsoftly/zcp-cli/pkg/api/permission"
)

// NewPermissionCmd returns the 'permission' cobra command.
func NewPermissionCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "permission",
		Short: "View the assignable permission catalog (for building roles)",
	}
	cmd.AddCommand(newPermissionListCmd())
	return cmd
}

func newPermissionListCmd() *cobra.Command {
	var category string

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List permissions in the catalog",
		Long: `List the account's assignable permissions, grouped by category.

Use the SLUG values with 'zcp role create/update --permission <slug>'.`,
		Example: `  zcp permission list
  zcp permission list --category "Virtual Machine"
  zcp permission list --output json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPermissionList(cmd, category)
		},
	}
	cmd.Flags().StringVar(&category, "category", "", "Only show permissions in this category (case-insensitive)")
	return cmd
}

func runPermissionList(cmd *cobra.Command, category string) error {
	_, client, printer, err := buildClientAndPrinter(cmd)
	if err != nil {
		return err
	}

	svc := permission.NewService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	perms, err := svc.List(ctx)
	if err != nil {
		return fmt.Errorf("permission list: %w", err)
	}

	if category != "" {
		filtered := perms[:0:0]
		for _, p := range perms {
			if strings.EqualFold(p.Category, category) {
				filtered = append(filtered, p)
			}
		}
		perms = filtered
	}

	if printer.Format() == output.FormatJSON || printer.Format() == output.FormatYAML {
		return printer.Print(perms)
	}

	// Stable, category-grouped order so related permissions sit together.
	sort.SliceStable(perms, func(i, j int) bool {
		if perms[i].Category != perms[j].Category {
			return perms[i].Category < perms[j].Category
		}
		return perms[i].Slug < perms[j].Slug
	})

	headers := []string{"CATEGORY", "SLUG", "NAME", "DESCRIPTION"}
	rows := make([][]string, 0, len(perms))
	for _, p := range perms {
		rows = append(rows, []string{p.Category, p.Slug, p.Name, p.Description})
	}
	return printer.PrintTable(headers, rows)
}
