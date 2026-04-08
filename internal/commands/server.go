package commands

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"github.com/zsoftly/zcp-cli/internal/api/server"
)

// NewServerCmd returns the 'server' cobra command.
func NewServerCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "server",
		Short: "Manage servers",
	}
	cmd.AddCommand(newServerListCmd())
	return cmd
}

// ─── List ───────────────────────────────────────────────────────────────────

func newServerListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "list",
		Short:   "List servers",
		Example: `  zcp server list`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runServerList(cmd)
		},
	}
	return cmd
}

func runServerList(cmd *cobra.Command) error {
	_, client, printer, err := buildClientAndPrinter(cmd)
	if err != nil {
		return err
	}

	svc := server.NewService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	servers, err := svc.List(ctx)
	if err != nil {
		return fmt.Errorf("server list: %w", err)
	}

	headers := []string{"ID", "NAME", "SLUG", "DESCRIPTION", "STATUS", "CREATED"}
	rows := make([][]string, 0, len(servers))
	for _, s := range servers {
		rows = append(rows, []string{
			s.ID,
			s.Name,
			s.Slug,
			s.Description,
			fmt.Sprintf("%v", s.Status),
			s.CreatedAt,
		})
	}
	return printer.PrintTable(headers, rows)
}
