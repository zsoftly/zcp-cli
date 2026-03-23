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
	cmd.AddCommand(newIPReleaseCmd())

	natCmd := &cobra.Command{Use: "static-nat", Short: "Manage static NAT on IP addresses"}
	natCmd.AddCommand(newIPStaticNATEnableCmd())
	natCmd.AddCommand(newIPStaticNATDisableCmd())
	cmd.AddCommand(natCmd)

	return cmd
}

func newIPListCmd() *cobra.Command {
	var zoneUUID, networkUUID string

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List public IP addresses",
		Example: `  zcp ip list --zone <uuid>
  zcp ip list --zone <uuid> --network <uuid>`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if zoneUUID == "" {
				return fmt.Errorf("--zone is required")
			}
			return runIPList(cmd, zoneUUID, networkUUID)
		},
	}
	cmd.Flags().StringVar(&zoneUUID, "zone", "", "Zone UUID (required)")
	cmd.Flags().StringVar(&networkUUID, "network", "", "Filter by network UUID")
	return cmd
}

func runIPList(cmd *cobra.Command, zoneUUID, networkUUID string) error {
	_, client, printer, err := buildClientAndPrinter(cmd)
	if err != nil {
		return err
	}

	svc := ipaddress.NewService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	ips, err := svc.List(ctx, zoneUUID, networkUUID)
	if err != nil {
		return fmt.Errorf("ip list: %w", err)
	}

	headers := []string{"UUID", "PUBLIC IP", "STATE", "NETWORK", "SOURCE NAT"}
	rows := make([][]string, 0, len(ips))
	for _, ip := range ips {
		sourceNAT := "false"
		if ip.IsSourceNAT {
			sourceNAT = "true"
		}
		rows = append(rows, []string{
			ip.UUID,
			ip.PublicIPAddress,
			ip.State,
			ip.NetworkUUID,
			sourceNAT,
		})
	}
	return printer.PrintTable(headers, rows)
}

func newIPAllocateCmd() *cobra.Command {
	var networkUUID, networkType string

	cmd := &cobra.Command{
		Use:   "allocate",
		Short: "Allocate a new public IP address",
		Example: `  zcp ip allocate --network <uuid>
  zcp ip allocate --network <uuid> --type Isolated`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if networkUUID == "" {
				return fmt.Errorf("--network is required")
			}
			if networkType == "" {
				networkType = "Isolated"
			}
			return runIPAllocate(cmd, networkUUID, networkType)
		},
	}
	cmd.Flags().StringVar(&networkUUID, "network", "", "Network UUID (required)")
	cmd.Flags().StringVar(&networkType, "type", "Isolated", "Network type (Isolated or VPC)")
	return cmd
}

func runIPAllocate(cmd *cobra.Command, networkUUID, networkType string) error {
	_, client, printer, err := buildClientAndPrinter(cmd)
	if err != nil {
		return err
	}

	svc := ipaddress.NewService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	ip, err := svc.Acquire(ctx, networkUUID, networkType)
	if err != nil {
		return fmt.Errorf("ip allocate: %w", err)
	}

	headers := []string{"FIELD", "VALUE"}
	rows := [][]string{
		{"UUID", ip.UUID},
		{"Public IP", ip.PublicIPAddress},
		{"State", ip.State},
		{"Network UUID", ip.NetworkUUID},
		{"Zone UUID", ip.ZoneUUID},
		{"Source NAT", fmt.Sprintf("%v", ip.IsSourceNAT)},
	}
	return printer.PrintTable(headers, rows)
}

func newIPReleaseCmd() *cobra.Command {
	var yes bool

	cmd := &cobra.Command{
		Use:   "release <uuid>",
		Short: "Release a public IP address",
		Args:  cobra.ExactArgs(1),
		Example: `  zcp ip release <uuid>
  zcp ip release <uuid> --yes`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runIPRelease(cmd, args[0], yes)
		},
	}
	cmd.Flags().BoolVar(&yes, "yes", false, "Skip confirmation prompt")
	return cmd
}

func runIPRelease(cmd *cobra.Command, uuid string, yes bool) error {
	if !yes {
		fmt.Fprintf(os.Stderr, "Release IP address %q? This action cannot be undone. [y/N]: ", uuid)
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

	if err := svc.Release(ctx, uuid); err != nil {
		return fmt.Errorf("ip release: %w", err)
	}

	printer.Fprintf("IP address %q released.\n", uuid)
	return nil
}

func newIPStaticNATEnableCmd() *cobra.Command {
	var instanceUUID, networkUUID string

	cmd := &cobra.Command{
		Use:     "enable <ip-uuid>",
		Short:   "Enable static NAT on an IP address",
		Args:    cobra.ExactArgs(1),
		Example: `  zcp ip static-nat enable <ip-uuid> --instance <uuid> --network <uuid>`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if instanceUUID == "" {
				return fmt.Errorf("--instance is required")
			}
			if networkUUID == "" {
				return fmt.Errorf("--network is required")
			}
			return runIPStaticNATEnable(cmd, args[0], instanceUUID, networkUUID)
		},
	}
	cmd.Flags().StringVar(&instanceUUID, "instance", "", "VM UUID to associate (required)")
	cmd.Flags().StringVar(&networkUUID, "network", "", "Network UUID (required)")
	return cmd
}

func runIPStaticNATEnable(cmd *cobra.Command, ipUUID, vmUUID, networkUUID string) error {
	_, client, printer, err := buildClientAndPrinter(cmd)
	if err != nil {
		return err
	}

	svc := ipaddress.NewService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	cfg, err := svc.EnableStaticNAT(ctx, ipUUID, vmUUID, networkUUID)
	if err != nil {
		return fmt.Errorf("ip static-nat enable: %w", err)
	}

	headers := []string{"FIELD", "VALUE"}
	rows := [][]string{
		{"IP Address UUID", cfg.IPAddressUUID},
		{"VM UUID", cfg.VMUUID},
		{"VM Name", cfg.VMName},
		{"Network UUID", cfg.NetworkUUID},
		{"Status", cfg.Status},
	}
	return printer.PrintTable(headers, rows)
}

func newIPStaticNATDisableCmd() *cobra.Command {
	var yes bool

	cmd := &cobra.Command{
		Use:   "disable <ip-uuid>",
		Short: "Disable static NAT on an IP address",
		Args:  cobra.ExactArgs(1),
		Example: `  zcp ip static-nat disable <ip-uuid>
  zcp ip static-nat disable <ip-uuid> --yes`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runIPStaticNATDisable(cmd, args[0], yes)
		},
	}
	cmd.Flags().BoolVar(&yes, "yes", false, "Skip confirmation prompt")
	return cmd
}

func runIPStaticNATDisable(cmd *cobra.Command, ipUUID string, yes bool) error {
	if !yes {
		fmt.Fprintf(os.Stderr, "Disable static NAT for IP address %q? [y/N]: ", ipUUID)
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

	if err := svc.DisableStaticNAT(ctx, ipUUID); err != nil {
		return fmt.Errorf("ip static-nat disable: %w", err)
	}

	printer.Fprintf("Static NAT disabled for IP address %q.\n", ipUUID)
	return nil
}
