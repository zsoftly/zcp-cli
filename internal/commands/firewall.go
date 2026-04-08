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
	var ipSlug string

	cmd := &cobra.Command{
		Use:     "list",
		Short:   "List firewall rules",
		Example: `  zcp firewall list --ip <ip-slug>`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if ipSlug == "" {
				return fmt.Errorf("--ip is required")
			}
			return runFirewallList(cmd, ipSlug)
		},
	}
	cmd.Flags().StringVar(&ipSlug, "ip", "", "IP address slug (required)")
	return cmd
}

func runFirewallList(cmd *cobra.Command, ipSlug string) error {
	_, client, printer, err := buildClientAndPrinter(cmd)
	if err != nil {
		return err
	}

	svc := firewall.NewService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	rules, err := svc.List(ctx, ipSlug)
	if err != nil {
		return fmt.Errorf("firewall list: %w", err)
	}

	headers := []string{"ID", "PROTOCOL", "PORTS", "CIDR", "STATE"}
	rows := make([][]string, 0, len(rules))
	for _, r := range rules {
		ports := formatFWPorts(fmt.Sprintf("%v", r.StartPort), fmt.Sprintf("%v", r.EndPort))
		rows = append(rows, []string{
			r.ID,
			r.Protocol,
			ports,
			r.CIDRList,
			r.State,
		})
	}
	return printer.PrintTable(headers, rows)
}

func newFirewallCreateCmd() *cobra.Command {
	var ipSlug, protocol, cidr, destCIDR string
	var startPort, endPort string

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a firewall rule",
		Example: `  zcp firewall create --ip <ip-slug> --protocol tcp --start-port 80 --end-port 80
  zcp firewall create --ip <ip-slug> --protocol tcp --start-port 80 --end-port 80 --cidr 0.0.0.0/0
  zcp firewall create --ip <ip-slug> --protocol icmp`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if ipSlug == "" {
				return fmt.Errorf("--ip is required")
			}
			if protocol == "" {
				return fmt.Errorf("--protocol is required")
			}
			if err := validateFirewallProtocol(protocol); err != nil {
				return err
			}
			proto := strings.ToLower(protocol)
			if (proto == "tcp" || proto == "udp") && startPort == "" {
				fmt.Fprintln(os.Stderr, "Warning: no ports specified for TCP/UDP rule; all ports will be affected.")
			}
			return runFirewallCreate(cmd, ipSlug, firewall.CreateRequest{
				Protocol:            proto,
				CIDRList:            cidr,
				DestinationCIDRList: destCIDR,
				StartPort:           startPort,
				EndPort:             endPort,
			})
		},
	}
	cmd.Flags().StringVar(&ipSlug, "ip", "", "IP address slug (required)")
	cmd.Flags().StringVar(&protocol, "protocol", "", "Protocol: tcp, udp, icmp, or all (required)")
	cmd.Flags().StringVar(&startPort, "start-port", "", "Start port number")
	cmd.Flags().StringVar(&endPort, "end-port", "", "End port number")
	cmd.Flags().StringVar(&cidr, "cidr", "", "Source CIDR list (e.g. 0.0.0.0/0)")
	cmd.Flags().StringVar(&destCIDR, "dest-cidr", "", "Destination CIDR list")
	return cmd
}

func runFirewallCreate(cmd *cobra.Command, ipSlug string, req firewall.CreateRequest) error {
	_, client, printer, err := buildClientAndPrinter(cmd)
	if err != nil {
		return err
	}

	svc := firewall.NewService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	rule, err := svc.Create(ctx, ipSlug, req)
	if err != nil {
		return fmt.Errorf("firewall create: %w", err)
	}

	headers := []string{"FIELD", "VALUE"}
	rows := [][]string{
		{"ID", rule.ID},
		{"Protocol", rule.Protocol},
		{"Ports", formatFWPorts(fmt.Sprintf("%v", rule.StartPort), fmt.Sprintf("%v", rule.EndPort))},
		{"CIDR", rule.CIDRList},
		{"State", rule.State},
	}
	return printer.PrintTable(headers, rows)
}

func newFirewallDeleteCmd() *cobra.Command {
	var ipSlug string
	var yes bool

	cmd := &cobra.Command{
		Use:   "delete <rule-id>",
		Short: "Delete a firewall rule",
		Args:  cobra.ExactArgs(1),
		Example: `  zcp firewall delete <rule-id> --ip <ip-slug>
  zcp firewall delete <rule-id> --ip <ip-slug> --yes`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if ipSlug == "" {
				return fmt.Errorf("--ip is required")
			}
			return runFirewallDelete(cmd, ipSlug, args[0], yes)
		},
	}
	cmd.Flags().StringVar(&ipSlug, "ip", "", "IP address slug (required)")
	cmd.Flags().BoolVar(&yes, "yes", false, "Skip confirmation prompt")
	return cmd
}

func runFirewallDelete(cmd *cobra.Command, ipSlug, ruleID string, yes bool) error {
	if !yes && !autoApproved(cmd) {
		fmt.Fprintf(os.Stderr, "Delete firewall rule %q on IP %q? [y/N]: ", ruleID, ipSlug)
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

	if err := svc.Delete(ctx, ipSlug, ruleID); err != nil {
		return fmt.Errorf("firewall delete: %w", err)
	}

	printer.Fprintf("Firewall rule %q deleted.\n", ruleID)
	return nil
}

// formatFWPorts returns a human-readable ports string from start/end port values.
func formatFWPorts(start, end string) string {
	if start == "" || start == "<nil>" || start == "null" {
		start = ""
	}
	if end == "" || end == "<nil>" || end == "null" {
		end = ""
	}
	if start == "" && end == "" {
		return "all"
	}
	if end == "" || end == start {
		return start
	}
	return start + "-" + end
}

// formatPorts returns a human-readable ports string from start/end port string values.
// Retained for backward compatibility with other commands that may reference it.
func formatPorts(start, end string) string {
	if start == "" && end == "" {
		return "all"
	}
	if end == "" || end == start {
		return start
	}
	return start + "-" + end
}
