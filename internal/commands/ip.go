package commands

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/zsoftly/zcp-cli/internal/api/ipaddress"
)

// NewIPCmd returns the 'ip' cobra command.
func NewIPCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ip",
		Short: "Manage public IP addresses",
	}
	cmd.AddCommand(newIPListCmd())
	cmd.AddCommand(newIPAllocateCmd())

	natCmd := &cobra.Command{Use: "static-nat", Short: "Manage static NAT on IP addresses"}
	natCmd.AddCommand(newIPStaticNATEnableCmd())
	cmd.AddCommand(natCmd)

	vpnCmd := &cobra.Command{Use: "vpn", Short: "Manage remote access VPN on IP addresses"}
	vpnCmd.AddCommand(newIPVPNListCmd())
	vpnCmd.AddCommand(newIPVPNEnableCmd())
	vpnCmd.AddCommand(newIPVPNDisableCmd())
	cmd.AddCommand(vpnCmd)

	return cmd
}

func newIPListCmd() *cobra.Command {
	var vpcSlug string

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List public IP addresses",
		Example: `  zcp ip list
  zcp ip list --vpc <slug>`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runIPList(cmd, vpcSlug)
		},
	}
	cmd.Flags().StringVar(&vpcSlug, "vpc", "", "Filter by VPC slug")
	return cmd
}

func runIPList(cmd *cobra.Command, vpcSlug string) error {
	_, client, printer, err := buildClientAndPrinter(cmd)
	if err != nil {
		return err
	}

	svc := ipaddress.NewService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	ips, err := svc.List(ctx, vpcSlug)
	if err != nil {
		return fmt.Errorf("ip list: %w", err)
	}

	headers := []string{"SLUG", "IP ADDRESS", "STRATEGY", "VM", "NETWORK ID", "VPC ID"}
	rows := make([][]string, 0, len(ips))
	for _, ip := range ips {
		rows = append(rows, []string{
			ip.Slug,
			ip.IPAddress,
			ip.Strategy,
			ip.VirtualMachineName,
			ip.NetworkID,
			ip.VPCID,
		})
	}
	return printer.PrintTable(headers, rows)
}

func newIPAllocateCmd() *cobra.Command {
	var vpc, network, plan, billingCycle string

	cmd := &cobra.Command{
		Use:   "allocate",
		Short: "Allocate a new public IP address",
		Example: `  zcp ip allocate --plan ip-plan --billing-cycle hourly
  zcp ip allocate --plan ip-plan --billing-cycle hourly --network <slug>
  zcp ip allocate --plan ip-plan --billing-cycle hourly --vpc <slug>`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if plan == "" {
				return fmt.Errorf("--plan is required")
			}
			if billingCycle == "" {
				return fmt.Errorf("--billing-cycle is required")
			}
			return runIPAllocate(cmd, ipaddress.CreateRequest{
				VPC:          vpc,
				Network:      network,
				Plan:         plan,
				BillingCycle: billingCycle,
			})
		},
	}
	cmd.Flags().StringVar(&vpc, "vpc", "", "VPC slug")
	cmd.Flags().StringVar(&network, "network", "", "Network slug")
	cmd.Flags().StringVar(&plan, "plan", "", "IP plan slug (required)")
	cmd.Flags().StringVar(&billingCycle, "billing-cycle", "", "Billing cycle slug (required, e.g. hourly, monthly)")
	return cmd
}

func runIPAllocate(cmd *cobra.Command, req ipaddress.CreateRequest) error {
	_, client, printer, err := buildClientAndPrinter(cmd)
	if err != nil {
		return err
	}

	svc := ipaddress.NewService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	ip, err := svc.Allocate(ctx, req)
	if err != nil {
		return fmt.Errorf("ip allocate: %w", err)
	}

	headers := []string{"FIELD", "VALUE"}
	rows := [][]string{
		{"Slug", ip.Slug},
		{"IP Address", ip.IPAddress},
		{"Strategy", ip.Strategy},
		{"Network ID", ip.NetworkID},
		{"VPC ID", ip.VPCID},
		{"Region ID", ip.RegionID},
		{"Created At", ip.CreatedAt},
	}
	return printer.PrintTable(headers, rows)
}

func newIPStaticNATEnableCmd() *cobra.Command {
	var vmSlug string

	cmd := &cobra.Command{
		Use:     "enable <ip-slug>",
		Short:   "Enable static NAT on an IP address",
		Args:    cobra.ExactArgs(1),
		Example: `  zcp ip static-nat enable <ip-slug> --instance <vm-slug>`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if vmSlug == "" {
				return fmt.Errorf("--instance is required")
			}
			return runIPStaticNATEnable(cmd, args[0], vmSlug)
		},
	}
	cmd.Flags().StringVar(&vmSlug, "instance", "", "VM slug to associate (required)")
	return cmd
}

func runIPStaticNATEnable(cmd *cobra.Command, ipSlug, vmSlug string) error {
	_, client, printer, err := buildClientAndPrinter(cmd)
	if err != nil {
		return err
	}

	svc := ipaddress.NewService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	ip, err := svc.EnableStaticNAT(ctx, ipSlug, vmSlug)
	if err != nil {
		return fmt.Errorf("ip static-nat enable: %w", err)
	}

	headers := []string{"FIELD", "VALUE"}
	rows := [][]string{
		{"Slug", ip.Slug},
		{"IP Address", ip.IPAddress},
		{"Strategy", ip.Strategy},
		{"VM", ip.VirtualMachineName},
		{"Network ID", ip.NetworkID},
	}
	return printer.PrintTable(headers, rows)
}

// ─── Remote Access VPN ───────────────────────────────────────────────────────

func newIPVPNListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "list <ip-slug>",
		Short:   "List remote access VPNs on an IP address",
		Args:    cobra.ExactArgs(1),
		Example: `  zcp ip vpn list <ip-slug>`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runIPVPNList(cmd, args[0])
		},
	}
	return cmd
}

func runIPVPNList(cmd *cobra.Command, ipSlug string) error {
	_, client, printer, err := buildClientAndPrinter(cmd)
	if err != nil {
		return err
	}

	svc := ipaddress.NewService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	vpns, err := svc.ListRemoteAccessVPNs(ctx, ipSlug)
	if err != nil {
		return fmt.Errorf("ip vpn list: %w", err)
	}

	headers := []string{"ID", "PUBLIC IP", "STATE", "CREATED AT"}
	rows := make([][]string, 0, len(vpns))
	for _, v := range vpns {
		rows = append(rows, []string{
			v.ID,
			v.PublicIP,
			v.State,
			v.CreatedAt,
		})
	}
	return printer.PrintTable(headers, rows)
}

func newIPVPNEnableCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "enable <ip-slug>",
		Short:   "Enable remote access VPN on an IP address",
		Args:    cobra.ExactArgs(1),
		Example: `  zcp ip vpn enable <ip-slug>`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runIPVPNEnable(cmd, args[0])
		},
	}
	return cmd
}

func runIPVPNEnable(cmd *cobra.Command, ipSlug string) error {
	_, client, printer, err := buildClientAndPrinter(cmd)
	if err != nil {
		return err
	}

	svc := ipaddress.NewService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	vpn, err := svc.EnableRemoteAccessVPN(ctx, ipSlug)
	if err != nil {
		return fmt.Errorf("ip vpn enable: %w", err)
	}

	headers := []string{"FIELD", "VALUE"}
	rows := [][]string{
		{"ID", vpn.ID},
		{"Public IP", vpn.PublicIP},
		{"State", vpn.State},
		{"Created At", vpn.CreatedAt},
	}
	return printer.PrintTable(headers, rows)
}

func newIPVPNDisableCmd() *cobra.Command {
	var yes bool

	cmd := &cobra.Command{
		Use:   "disable <ip-slug> <vpn-id>",
		Short: "Disable remote access VPN on an IP address",
		Args:  cobra.ExactArgs(2),
		Example: `  zcp ip vpn disable <ip-slug> <vpn-id>
  zcp ip vpn disable <ip-slug> <vpn-id> --yes`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runIPVPNDisable(cmd, args[0], args[1], yes)
		},
	}
	cmd.Flags().BoolVar(&yes, "yes", false, "Skip confirmation prompt")
	return cmd
}

func runIPVPNDisable(cmd *cobra.Command, ipSlug, vpnID string, yes bool) error {
	if !yes && !autoApproved(cmd) {
		fmt.Fprintf(os.Stderr, "Disable remote access VPN %q on IP %q? [y/N]: ", vpnID, ipSlug)
		scanner := bufio.NewScanner(os.Stdin)
		scanner.Scan()
		answer := strings.TrimSpace(strings.ToLower(scanner.Text()))
		if answer != "y" && answer != "yes" {
			fmt.Fprintln(os.Stderr, "Aborted.")
			return nil
		}
	}

	_, client, printer, err := buildClientAndPrinter(cmd)
	if err != nil {
		return err
	}

	svc := ipaddress.NewService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	if err := svc.DisableRemoteAccessVPN(ctx, ipSlug, vpnID); err != nil {
		return fmt.Errorf("ip vpn disable: %w", err)
	}

	printer.Fprintf("Remote access VPN %q disabled on IP %q.\n", vpnID, ipSlug)
	return nil
}
