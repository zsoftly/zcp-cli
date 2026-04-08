package commands

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"github.com/zsoftly/zcp-cli/internal/api/monitoring"
)

// NewMonitoringCmd returns the 'monitoring' cobra command.
func NewMonitoringCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "monitoring",
		Short: "View resource monitoring and VM metrics",
	}
	cmd.AddCommand(newMonitoringGlobalCmd())
	cmd.AddCommand(newMonitoringCPUCmd())
	cmd.AddCommand(newMonitoringMemoryCmd())
	cmd.AddCommand(newMonitoringDiskCmd())
	cmd.AddCommand(newMonitoringDiskIOCmd())
	cmd.AddCommand(newMonitoringNetworkCmd())
	cmd.AddCommand(newMonitoringChartsCmd())
	return cmd
}

func newMonitoringGlobalCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "global",
		Short:   "Show global resource monitoring overview",
		Example: `  zcp monitoring global`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runMonitoringGlobal(cmd)
		},
	}
	return cmd
}

func runMonitoringGlobal(cmd *cobra.Command) error {
	_, client, printer, err := buildClientAndPrinter(cmd)
	if err != nil {
		return err
	}

	svc := monitoring.NewService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	resources, err := svc.Global(ctx)
	if err != nil {
		return fmt.Errorf("monitoring global: %w", err)
	}

	headers := []string{"NAME", "TOTAL", "USED", "FREE", "UNIT", "USAGE %"}
	rows := make([][]string, 0, len(resources))
	for _, r := range resources {
		rows = append(rows, []string{
			r.Name,
			fmt.Sprintf("%.1f", r.Total),
			fmt.Sprintf("%.1f", r.Used),
			fmt.Sprintf("%.1f", r.Free),
			r.Unit,
			fmt.Sprintf("%.1f%%", r.Percentage),
		})
	}
	return printer.PrintTable(headers, rows)
}

func newMonitoringCPUCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cpu <vm-slug>",
		Short: "Show CPU usage metrics for a VM",
		Args:  cobra.ExactArgs(1),
		Example: `  zcp monitoring cpu my-vm-slug
  zcp monitoring cpu my-vm-slug --output json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runMonitoringCPU(cmd, args[0])
		},
	}
	return cmd
}

func runMonitoringCPU(cmd *cobra.Command, vmSlug string) error {
	_, client, printer, err := buildClientAndPrinter(cmd)
	if err != nil {
		return err
	}

	svc := monitoring.NewService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	points, err := svc.CPUUsage(ctx, vmSlug)
	if err != nil {
		return fmt.Errorf("monitoring cpu: %w", err)
	}

	headers := []string{"TIMESTAMP", "VALUE", "UNIT"}
	rows := make([][]string, 0, len(points))
	for _, p := range points {
		rows = append(rows, []string{
			p.Timestamp,
			fmt.Sprintf("%.2f", p.Value),
			p.Unit,
		})
	}
	return printer.PrintTable(headers, rows)
}

func newMonitoringMemoryCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "memory <vm-slug>",
		Short: "Show memory usage metrics for a VM",
		Args:  cobra.ExactArgs(1),
		Example: `  zcp monitoring memory my-vm-slug
  zcp monitoring memory my-vm-slug --output json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runMonitoringMemory(cmd, args[0])
		},
	}
	return cmd
}

func runMonitoringMemory(cmd *cobra.Command, vmSlug string) error {
	_, client, printer, err := buildClientAndPrinter(cmd)
	if err != nil {
		return err
	}

	svc := monitoring.NewService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	points, err := svc.MemoryUsage(ctx, vmSlug)
	if err != nil {
		return fmt.Errorf("monitoring memory: %w", err)
	}

	headers := []string{"TIMESTAMP", "VALUE", "UNIT"}
	rows := make([][]string, 0, len(points))
	for _, p := range points {
		rows = append(rows, []string{
			p.Timestamp,
			fmt.Sprintf("%.2f", p.Value),
			p.Unit,
		})
	}
	return printer.PrintTable(headers, rows)
}

func newMonitoringDiskCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "disk <vm-slug>",
		Short: "Show disk read/write metrics for a VM",
		Args:  cobra.ExactArgs(1),
		Example: `  zcp monitoring disk my-vm-slug
  zcp monitoring disk my-vm-slug --output json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runMonitoringDisk(cmd, args[0])
		},
	}
	return cmd
}

func runMonitoringDisk(cmd *cobra.Command, vmSlug string) error {
	_, client, printer, err := buildClientAndPrinter(cmd)
	if err != nil {
		return err
	}

	svc := monitoring.NewService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	points, err := svc.DiskReadWrite(ctx, vmSlug)
	if err != nil {
		return fmt.Errorf("monitoring disk: %w", err)
	}

	headers := []string{"TIMESTAMP", "READ", "WRITE", "UNIT"}
	rows := make([][]string, 0, len(points))
	for _, p := range points {
		rows = append(rows, []string{
			p.Timestamp,
			fmt.Sprintf("%.2f", p.Read),
			fmt.Sprintf("%.2f", p.Write),
			p.Unit,
		})
	}
	return printer.PrintTable(headers, rows)
}

func newMonitoringDiskIOCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "disk-io <vm-slug>",
		Short: "Show disk IO read/write metrics for a VM",
		Args:  cobra.ExactArgs(1),
		Example: `  zcp monitoring disk-io my-vm-slug
  zcp monitoring disk-io my-vm-slug --output json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runMonitoringDiskIO(cmd, args[0])
		},
	}
	return cmd
}

func runMonitoringDiskIO(cmd *cobra.Command, vmSlug string) error {
	_, client, printer, err := buildClientAndPrinter(cmd)
	if err != nil {
		return err
	}

	svc := monitoring.NewService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	points, err := svc.DiskIOReadWrite(ctx, vmSlug)
	if err != nil {
		return fmt.Errorf("monitoring disk-io: %w", err)
	}

	headers := []string{"TIMESTAMP", "READ", "WRITE", "UNIT"}
	rows := make([][]string, 0, len(points))
	for _, p := range points {
		rows = append(rows, []string{
			p.Timestamp,
			fmt.Sprintf("%.2f", p.Read),
			fmt.Sprintf("%.2f", p.Write),
			p.Unit,
		})
	}
	return printer.PrintTable(headers, rows)
}

func newMonitoringNetworkCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "network <vm-slug>",
		Short: "Show network traffic metrics for a VM",
		Args:  cobra.ExactArgs(1),
		Example: `  zcp monitoring network my-vm-slug
  zcp monitoring network my-vm-slug --output json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runMonitoringNetwork(cmd, args[0])
		},
	}
	return cmd
}

func runMonitoringNetwork(cmd *cobra.Command, vmSlug string) error {
	_, client, printer, err := buildClientAndPrinter(cmd)
	if err != nil {
		return err
	}

	svc := monitoring.NewService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	points, err := svc.NetworkTraffic(ctx, vmSlug)
	if err != nil {
		return fmt.Errorf("monitoring network: %w", err)
	}

	headers := []string{"TIMESTAMP", "INCOMING", "OUTGOING", "UNIT"}
	rows := make([][]string, 0, len(points))
	for _, p := range points {
		rows = append(rows, []string{
			p.Timestamp,
			fmt.Sprintf("%.2f", p.Incoming),
			fmt.Sprintf("%.2f", p.Outgoing),
			p.Unit,
		})
	}
	return printer.PrintTable(headers, rows)
}

func newMonitoringChartsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "charts",
		Short:   "Show monitoring charts data",
		Example: `  zcp monitoring charts`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runMonitoringCharts(cmd)
		},
	}
	return cmd
}

func runMonitoringCharts(cmd *cobra.Command) error {
	_, client, printer, err := buildClientAndPrinter(cmd)
	if err != nil {
		return err
	}

	svc := monitoring.NewService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	result, err := svc.Charts(ctx)
	if err != nil {
		return fmt.Errorf("monitoring charts: %w", err)
	}

	// Response schema is undefined — always output as raw JSON
	return printer.Print(result)
}
