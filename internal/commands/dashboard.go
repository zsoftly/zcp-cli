package commands

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/spf13/cobra"
	"github.com/zsoftly/zcp-cli/internal/api/dashboard"
)

// NewDashboardCmd returns the 'dashboard' cobra command.
func NewDashboardCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "dashboard",
		Short: "Account dashboard and service management",
		Long: `View account service counts and manage service cancellations.

The dashboard command provides a quick overview of active resources
in your account and allows you to submit service cancellation requests.`,
	}
	cmd.AddCommand(newDashboardSummaryCmd())
	cmd.AddCommand(newDashboardCancelCmd())
	return cmd
}

// ── Summary ─────────────────────────────────────────────────────────────────

func newDashboardSummaryCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "summary",
		Short: "Show a summary of active service counts",
		Example: `  zcp dashboard summary
  zcp dashboard summary --output json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDashboardSummary(cmd)
		},
	}
}

func runDashboardSummary(cmd *cobra.Command) error {
	_, client, printer, err := buildClientAndPrinter(cmd)
	if err != nil {
		return err
	}

	svc := dashboard.NewService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	counts, err := svc.GetServiceCounts(ctx)
	if err != nil {
		return fmt.Errorf("dashboard summary: %w", err)
	}

	headers := []string{"SERVICE", "COUNT"}
	rows := [][]string{
		{"Instances", strconv.Itoa(counts.Instance)},
		{"Kubernetes", strconv.Itoa(counts.Kubernetes)},
		{"Volumes", strconv.Itoa(counts.Volume)},
		{"Snapshots", strconv.Itoa(counts.Snapshot)},
		{"Networks", strconv.Itoa(counts.Network)},
		{"VPCs", strconv.Itoa(counts.VPC)},
		{"Public IPs", strconv.Itoa(counts.PublicIP)},
		{"Firewalls", strconv.Itoa(counts.Firewall)},
		{"Load Balancers", strconv.Itoa(counts.LoadBalancer)},
		{"VPNs", strconv.Itoa(counts.VPN)},
		{"SSH Keys", strconv.Itoa(counts.SSHKey)},
		{"Templates", strconv.Itoa(counts.Template)},
	}
	return printer.PrintTable(headers, rows)
}

// ── Cancel ──────────────────────────────────────────────────────────────────

func newDashboardCancelCmd() *cobra.Command {
	var serviceSlug string

	cmd := &cobra.Command{
		Use:   "cancel-service",
		Short: "Submit a service cancellation request",
		Example: `  zcp dashboard cancel-service --slug vm-abc-123
  zcp dashboard cancel-service --slug k8s-cluster-456`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if serviceSlug == "" {
				return fmt.Errorf("--slug is required")
			}
			return runDashboardCancel(cmd, serviceSlug)
		},
	}
	cmd.Flags().StringVar(&serviceSlug, "slug", "", "Service slug to cancel (required)")
	return cmd
}

func runDashboardCancel(cmd *cobra.Command, serviceSlug string) error {
	_, client, printer, err := buildClientAndPrinter(cmd)
	if err != nil {
		return err
	}

	svc := dashboard.NewService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	resp, err := svc.CancelService(ctx, serviceSlug, "not_needed_anymore")
	if err != nil {
		return fmt.Errorf("dashboard cancel-service: %w", err)
	}

	printer.Fprintf("Service %s: %s\n", serviceSlug, resp.Message)
	return nil
}
