package commands

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/zsoftly/zcp-cli/internal/api/firewall"
)

var validFirewallProtocols = map[string]bool{"tcp": true, "udp": true, "icmp": true, "all": true}

func validateFirewallProtocol(protocol string) error {
	if !validFirewallProtocols[strings.ToLower(protocol)] {
		return fmt.Errorf("invalid protocol %q: must be tcp, udp, icmp, or all", protocol)
	}
	return nil
}

// NewFirewallCmd returns the 'firewall' cobra command.
func NewFirewallCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "firewall",
		Short: "Manage firewall rules",
	}
	cmd.AddCommand(newFirewallListCmd())
	cmd.AddCommand(newFirewallCreateCmd())
	cmd.AddCommand(newFirewallDeleteCmd())
	return cmd
}

func newFirewallListCmd() *cobra.Command {
	var zoneUUID, ipUUID string

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List firewall rules",
		Example: `  zcp firewall list --zone <uuid>
  zcp firewall list --zone <uuid> --ip <uuid>`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runFirewallList(cmd, zoneUUID, ipUUID)
		},
	}
	cmd.Flags().StringVar(&zoneUUID, "zone", "", "Zone UUID (overrides default zone)")
	cmd.Flags().StringVar(&ipUUID, "ip", "", "Filter by IP address UUID")
	return cmd
}

func runFirewallList(cmd *cobra.Command, zoneUUID, ipUUID string) error {
	profile, client, printer, err := buildClientAndPrinter(cmd)
	if err != nil {
		return err
	}
	zoneUUID = resolveZone(profile, zoneUUID)
	if zoneUUID == "" {
		return errNoZone()
	}

	svc := firewall.NewService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	rules, err := svc.List(ctx, zoneUUID, "", ipUUID)
	if err != nil {
		// Kong returns error instead of empty list when account has no IPs
		if strings.Contains(err.Error(), "Invalid IpAddress") {
			rules = nil
		} else {
			return fmt.Errorf("firewall list: %w", err)
		}
	}

	headers := []string{"UUID", "PROTOCOL", "PORTS", "CIDR", "IP ADDRESS", "STATUS"}
	rows := make([][]string, 0, len(rules))
	for _, r := range rules {
		ports := formatPorts(r.StartPort, r.EndPort)
		rows = append(rows, []string{
			r.UUID,
			r.Protocol,
			ports,
			r.CIDRList,
			r.IPAddressUUID,
			r.Status,
		})
	}
	return printer.PrintTable(headers, rows)
}

func newFirewallCreateCmd() *cobra.Command {
	var ipUUID, protocol, startPort, endPort, cidr, icmpType, icmpCode string

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a firewall rule",
		Example: `  zcp firewall create --ip <uuid> --protocol tcp --start-port 80 --end-port 80
  zcp firewall create --ip <uuid> --protocol icmp`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if ipUUID == "" {
				return fmt.Errorf("--ip is required")
			}
			if protocol == "" {
				return fmt.Errorf("--protocol is required")
			}
			if err := validateFirewallProtocol(protocol); err != nil {
				return err
			}
			proto := strings.ToUpper(protocol)
			if (proto == "TCP" || proto == "UDP") && startPort == "" {
				fmt.Fprintln(os.Stderr, "Warning: no ports specified for TCP/UDP rule; all ports will be affected.")
			}
			return runFirewallCreate(cmd, firewall.CreateRequest{
				IPAddressUUID: ipUUID,
				Protocol:      proto,
				StartPort:     startPort,
				EndPort:       endPort,
				CIDRList:      cidr,
				ICMPType:      icmpType,
				ICMPCode:      icmpCode,
			})
		},
	}
	cmd.Flags().StringVar(&ipUUID, "ip", "", "IP address UUID (required)")
	cmd.Flags().StringVar(&protocol, "protocol", "", "Protocol: tcp, udp, icmp, or all (required)")
	cmd.Flags().StringVar(&startPort, "start-port", "", "Start port number")
	cmd.Flags().StringVar(&endPort, "end-port", "", "End port number")
	cmd.Flags().StringVar(&cidr, "cidr", "", "Comma-separated CIDR list (e.g. 0.0.0.0/0)")
	cmd.Flags().StringVar(&icmpType, "icmp-type", "", "ICMP type (ICMP protocol only)")
	cmd.Flags().StringVar(&icmpCode, "icmp-code", "", "ICMP code (ICMP protocol only)")
	return cmd
}

func runFirewallCreate(cmd *cobra.Command, req firewall.CreateRequest) error {
	_, client, printer, err := buildClientAndPrinter(cmd)
	if err != nil {
		return err
	}

	svc := firewall.NewService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	rule, err := svc.Create(ctx, req)
	if err != nil {
		return fmt.Errorf("firewall create: %w", err)
	}

	headers := []string{"FIELD", "VALUE"}
	rows := [][]string{
		{"UUID", rule.UUID},
		{"Protocol", rule.Protocol},
		{"Ports", formatPorts(rule.StartPort, rule.EndPort)},
		{"CIDR", rule.CIDRList},
		{"IP Address UUID", rule.IPAddressUUID},
		{"Status", rule.Status},
	}
	return printer.PrintTable(headers, rows)
}

func newFirewallDeleteCmd() *cobra.Command {
	var yes bool

	cmd := &cobra.Command{
		Use:   "delete <uuid>",
		Short: "Delete a firewall rule",
		Args:  cobra.ExactArgs(1),
		Example: `  zcp firewall delete <uuid>
  zcp firewall delete <uuid> --yes`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runFirewallDelete(cmd, args[0], yes)
		},
	}
	cmd.Flags().BoolVar(&yes, "yes", false, "Skip confirmation prompt")
	return cmd
}

func runFirewallDelete(cmd *cobra.Command, uuid string, yes bool) error {
	if !yes {
		fmt.Fprintf(os.Stderr, "Delete firewall rule %q? [y/N]: ", uuid)
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

	svc := firewall.NewService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	if err := svc.Delete(ctx, uuid); err != nil {
		return fmt.Errorf("firewall delete: %w", err)
	}

	printer.Fprintf("Firewall rule %q deleted.\n", uuid)
	return nil
}

// formatPorts returns a human-readable ports string from start/end port values.
func formatPorts(start, end string) string {
	if start == "" && end == "" {
		return "all"
	}
	if end == "" || end == start {
		return start
	}
	return start + "-" + end
}
