package commands

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/spf13/cobra"
	"github.com/zsoftly/zcp-cli/internal/api/plan"
)

// NewPlanCmd returns the 'plan' cobra command with subcommands for each
// STKCNSL service type.
func NewPlanCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "plan",
		Short: "List service plans and pricing",
		Long: `List available service plans and pricing from the ZCP catalog.

Each subcommand queries a specific service type and displays the plans
with their resource attributes and pricing.`,
	}
	cmd.AddCommand(newPlanVMCmd())
	cmd.AddCommand(newPlanRouterCmd())
	cmd.AddCommand(newPlanStorageCmd())
	cmd.AddCommand(newPlanLBCmd())
	cmd.AddCommand(newPlanK8sCmd())
	cmd.AddCommand(newPlanIPCmd())
	cmd.AddCommand(newPlanVMSnapshotCmd())
	cmd.AddCommand(newPlanTemplateCmd())
	cmd.AddCommand(newPlanISOCmd())
	cmd.AddCommand(newPlanBackupCmd())
	return cmd
}

// ---------------------------------------------------------------------------
// Virtual Machine
// ---------------------------------------------------------------------------

func newPlanVMCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "vm",
		Short: "List Virtual Machine plans",
		Example: `  zcp plan vm
  zcp plan vm --output json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			_, client, printer, err := buildClientAndPrinter(cmd)
			if err != nil {
				return err
			}
			svc := plan.NewService(client)
			ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
			defer cancel()

			plans, err := svc.List(ctx, plan.ServiceVM)
			if err != nil {
				return fmt.Errorf("plan vm: %w", err)
			}

			headers := []string{"ID", "NAME", "CPU", "MEMORY", "STORAGE", "HOURLY", "MONTHLY", "ACTIVE"}
			rows := make([][]string, 0, len(plans))
			for _, p := range plans {
				rows = append(rows, []string{
					p.ID,
					p.Name,
					p.Attribute.FormattedCPU.String(),
					p.Attribute.FormattedMemory,
					p.Attribute.FormattedStorage,
					formatPrice(p.HourlyPrice),
					formatPrice(p.MonthlyPrice),
					strconv.FormatBool(p.Status),
				})
			}
			return printer.PrintTable(headers, rows)
		},
	}
}

// ---------------------------------------------------------------------------
// Virtual Router
// ---------------------------------------------------------------------------

func newPlanRouterCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "router",
		Short: "List Virtual Router plans",
		Example: `  zcp plan router
  zcp plan router --output json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			_, client, printer, err := buildClientAndPrinter(cmd)
			if err != nil {
				return err
			}
			svc := plan.NewService(client)
			ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
			defer cancel()

			plans, err := svc.List(ctx, plan.ServiceVirtualRouter)
			if err != nil {
				return fmt.Errorf("plan router: %w", err)
			}

			headers := []string{"ID", "NAME", "CPU", "MEMORY", "NETWORK RATE", "HOURLY", "MONTHLY", "ACTIVE"}
			rows := make([][]string, 0, len(plans))
			for _, p := range plans {
				rows = append(rows, []string{
					p.ID,
					p.Name,
					p.Attribute.CPU.String(),
					p.Attribute.FormattedMemory,
					p.Attribute.NetworkRate,
					formatPrice(p.HourlyPrice),
					formatPrice(p.MonthlyPrice),
					strconv.FormatBool(p.Status),
				})
			}
			return printer.PrintTable(headers, rows)
		},
	}
}

// ---------------------------------------------------------------------------
// Block Storage
// ---------------------------------------------------------------------------

func newPlanStorageCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "storage",
		Short: "List Block Storage plans",
		Example: `  zcp plan storage
  zcp plan storage --output json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			_, client, printer, err := buildClientAndPrinter(cmd)
			if err != nil {
				return err
			}
			svc := plan.NewService(client)
			ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
			defer cancel()

			plans, err := svc.List(ctx, plan.ServiceBlockStorage)
			if err != nil {
				return fmt.Errorf("plan storage: %w", err)
			}

			headers := []string{"ID", "NAME", "SIZE", "HOURLY", "MONTHLY", "CUSTOM", "ACTIVE"}
			rows := make([][]string, 0, len(plans))
			for _, p := range plans {
				rows = append(rows, []string{
					p.ID,
					p.Name,
					p.Attribute.FormattedSize,
					formatPrice(p.HourlyPrice),
					formatPrice(p.MonthlyPrice),
					strconv.FormatBool(p.IsCustom),
					strconv.FormatBool(p.Status),
				})
			}
			return printer.PrintTable(headers, rows)
		},
	}
}

// ---------------------------------------------------------------------------
// Load Balancer
// ---------------------------------------------------------------------------

func newPlanLBCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "lb",
		Short: "List Load Balancer plans",
		Example: `  zcp plan lb
  zcp plan lb --output json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			_, client, printer, err := buildClientAndPrinter(cmd)
			if err != nil {
				return err
			}
			svc := plan.NewService(client)
			ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
			defer cancel()

			plans, err := svc.List(ctx, plan.ServiceLoadBalancer)
			if err != nil {
				return fmt.Errorf("plan lb: %w", err)
			}

			headers := []string{"ID", "NAME", "TAG", "HOURLY", "MONTHLY", "ACTIVE"}
			rows := make([][]string, 0, len(plans))
			for _, p := range plans {
				rows = append(rows, []string{
					p.ID,
					p.Name,
					p.ParsedTag(),
					formatPrice(p.HourlyPrice),
					formatPrice(p.MonthlyPrice),
					strconv.FormatBool(p.Status),
				})
			}
			return printer.PrintTable(headers, rows)
		},
	}
}

// ---------------------------------------------------------------------------
// Kubernetes
// ---------------------------------------------------------------------------

func newPlanK8sCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "kubernetes",
		Short:   "List Kubernetes plans",
		Aliases: []string{"k8s"},
		Example: `  zcp plan kubernetes
  zcp plan k8s --output json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			_, client, printer, err := buildClientAndPrinter(cmd)
			if err != nil {
				return err
			}
			svc := plan.NewService(client)
			ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
			defer cancel()

			plans, err := svc.List(ctx, plan.ServiceKubernetes)
			if err != nil {
				return fmt.Errorf("plan kubernetes: %w", err)
			}

			headers := []string{"ID", "NAME", "CPU", "MEMORY", "HOURLY", "MONTHLY", "ACTIVE"}
			rows := make([][]string, 0, len(plans))
			for _, p := range plans {
				rows = append(rows, []string{
					p.ID,
					p.Name,
					p.Attribute.FormattedCPU.String(),
					p.Attribute.FormattedMemory,
					formatPrice(p.HourlyPrice),
					formatPrice(p.MonthlyPrice),
					strconv.FormatBool(p.Status),
				})
			}
			return printer.PrintTable(headers, rows)
		},
	}
}

// ---------------------------------------------------------------------------
// IP Address
// ---------------------------------------------------------------------------

func newPlanIPCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "ip",
		Short: "List IP Address plans",
		Example: `  zcp plan ip
  zcp plan ip --output json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			_, client, printer, err := buildClientAndPrinter(cmd)
			if err != nil {
				return err
			}
			svc := plan.NewService(client)
			ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
			defer cancel()

			plans, err := svc.List(ctx, plan.ServiceIPAddress)
			if err != nil {
				return fmt.Errorf("plan ip: %w", err)
			}

			headers := []string{"ID", "NAME", "TAG", "HOURLY", "MONTHLY", "ACTIVE"}
			rows := make([][]string, 0, len(plans))
			for _, p := range plans {
				rows = append(rows, []string{
					p.ID,
					p.Name,
					p.ParsedTag(),
					formatPrice(p.HourlyPrice),
					formatPrice(p.MonthlyPrice),
					strconv.FormatBool(p.Status),
				})
			}
			return printer.PrintTable(headers, rows)
		},
	}
}

// ---------------------------------------------------------------------------
// VM Snapshot
// ---------------------------------------------------------------------------

func newPlanVMSnapshotCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "vm-snapshot",
		Short: "List VM Snapshot plans",
		Example: `  zcp plan vm-snapshot
  zcp plan vm-snapshot --output json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			_, client, printer, err := buildClientAndPrinter(cmd)
			if err != nil {
				return err
			}
			svc := plan.NewService(client)
			ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
			defer cancel()

			plans, err := svc.List(ctx, plan.ServiceVMSnapshot)
			if err != nil {
				return fmt.Errorf("plan vm-snapshot: %w", err)
			}

			headers := []string{"ID", "NAME", "HOURLY", "MONTHLY", "ACTIVE"}
			rows := make([][]string, 0, len(plans))
			for _, p := range plans {
				rows = append(rows, []string{
					p.ID,
					p.Name,
					formatPrice(p.HourlyPrice),
					formatPrice(p.MonthlyPrice),
					strconv.FormatBool(p.Status),
				})
			}
			return printer.PrintTable(headers, rows)
		},
	}
}

// ---------------------------------------------------------------------------
// My Template
// ---------------------------------------------------------------------------

func newPlanTemplateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "template",
		Short: "List My Template plans",
		Example: `  zcp plan template
  zcp plan template --output json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			_, client, printer, err := buildClientAndPrinter(cmd)
			if err != nil {
				return err
			}
			svc := plan.NewService(client)
			ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
			defer cancel()

			plans, err := svc.List(ctx, plan.ServiceMyTemplate)
			if err != nil {
				return fmt.Errorf("plan template: %w", err)
			}

			headers := []string{"ID", "NAME", "TAG", "HOURLY", "MONTHLY", "ACTIVE"}
			rows := make([][]string, 0, len(plans))
			for _, p := range plans {
				rows = append(rows, []string{
					p.ID,
					p.Name,
					p.ParsedTag(),
					formatPrice(p.HourlyPrice),
					formatPrice(p.MonthlyPrice),
					strconv.FormatBool(p.Status),
				})
			}
			return printer.PrintTable(headers, rows)
		},
	}
}

// ---------------------------------------------------------------------------
// ISO
// ---------------------------------------------------------------------------

func newPlanISOCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "iso",
		Short: "List ISO plans",
		Example: `  zcp plan iso
  zcp plan iso --output json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			_, client, printer, err := buildClientAndPrinter(cmd)
			if err != nil {
				return err
			}
			svc := plan.NewService(client)
			ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
			defer cancel()

			plans, err := svc.List(ctx, plan.ServiceISO)
			if err != nil {
				return fmt.Errorf("plan iso: %w", err)
			}

			headers := []string{"ID", "NAME", "TAG", "HOURLY", "MONTHLY", "ACTIVE"}
			rows := make([][]string, 0, len(plans))
			for _, p := range plans {
				rows = append(rows, []string{
					p.ID,
					p.Name,
					p.ParsedTag(),
					formatPrice(p.HourlyPrice),
					formatPrice(p.MonthlyPrice),
					strconv.FormatBool(p.Status),
				})
			}
			return printer.PrintTable(headers, rows)
		},
	}
}

// ---------------------------------------------------------------------------
// Backups
// ---------------------------------------------------------------------------

func newPlanBackupCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "backup",
		Short: "List Backup plans",
		Example: `  zcp plan backup
  zcp plan backup --output json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			_, client, printer, err := buildClientAndPrinter(cmd)
			if err != nil {
				return err
			}
			svc := plan.NewService(client)
			ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
			defer cancel()

			plans, err := svc.List(ctx, plan.ServiceBackups)
			if err != nil {
				return fmt.Errorf("plan backup: %w", err)
			}

			headers := []string{"ID", "NAME", "TAG", "HOURLY", "MONTHLY", "ACTIVE"}
			rows := make([][]string, 0, len(plans))
			for _, p := range plans {
				rows = append(rows, []string{
					p.ID,
					p.Name,
					p.ParsedTag(),
					formatPrice(p.HourlyPrice),
					formatPrice(p.MonthlyPrice),
					strconv.FormatBool(p.Status),
				})
			}
			return printer.PrintTable(headers, rows)
		},
	}
}

// formatPrice renders a float price as a string with up to 4 decimal places,
// trimming trailing zeros for readability.
func formatPrice(v float64) string {
	return strconv.FormatFloat(v, 'f', -1, 64)
}
