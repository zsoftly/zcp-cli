package commands

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/zsoftly/zcp-cli/internal/api/egress"
)

var validEgressProtocols = map[string]bool{"tcp": true, "udp": true, "icmp": true, "all": true}

func validateEgressProtocol(protocol string) error {
	if !validEgressProtocols[strings.ToLower(protocol)] {
		return fmt.Errorf("invalid protocol %q: must be tcp, udp, icmp, or all", protocol)
	}
	return nil
}

// NewEgressCmd returns the 'egress' cobra command.
func NewEgressCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "egress",
		Short: "Manage egress firewall rules",
	}
	cmd.AddCommand(newEgressListCmd())
	cmd.AddCommand(newEgressCreateCmd())
	cmd.AddCommand(newEgressDeleteCmd())
	return cmd
}

func newEgressListCmd() *cobra.Command {
	var networkSlug string

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List egress rules for a network",
		Example: `  zcp egress list --network <slug>
  zcp egress list --network <slug> --output json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if networkSlug == "" {
				return fmt.Errorf("--network is required")
			}
			return runEgressList(cmd, networkSlug)
		},
	}
	cmd.Flags().StringVar(&networkSlug, "network", "", "Network slug (required)")
	return cmd
}

func runEgressList(cmd *cobra.Command, networkSlug string) error {
	_, client, printer, err := buildClientAndPrinter(cmd)
	if err != nil {
		return err
	}

	svc := egress.NewService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	rules, err := svc.List(ctx, networkSlug)
	if err != nil {
		return fmt.Errorf("egress list: %w", err)
	}

	headers := []string{"ID", "PROTOCOL", "PORTS", "CIDR", "STATUS"}
	rows := make([][]string, 0, len(rules))
	for _, r := range rules {
		ports := formatEgressPorts(r.StartPort, r.EndPort)
		rows = append(rows, []string{
			r.ID,
			r.Protocol,
			ports,
			r.CIDR,
			r.Status,
		})
	}
	return printer.PrintTable(headers, rows)
}

func newEgressCreateCmd() *cobra.Command {
	var networkSlug, protocol, startPort, endPort, cidr, icmpType, icmpCode string

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create an egress rule",
		Example: `  zcp egress create --network <slug> --protocol tcp --start-port 443
  zcp egress create --network <slug> --protocol all`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if networkSlug == "" {
				return fmt.Errorf("--network is required")
			}
			if protocol == "" {
				return fmt.Errorf("--protocol is required")
			}
			if err := validateEgressProtocol(protocol); err != nil {
				return err
			}
			proto := strings.ToUpper(protocol)
			if (proto == "TCP" || proto == "UDP") && startPort == "" {
				fmt.Fprintln(os.Stderr, "Warning: no ports specified for TCP/UDP rule; all ports will be affected.")
			}
			return runEgressCreate(cmd, egress.CreateRequest{
				NetworkSlug: networkSlug,
				Protocol:    proto,
				StartPort:   startPort,
				EndPort:     endPort,
				CIDR:        cidr,
				ICMPType:    icmpType,
				ICMPCode:    icmpCode,
			})
		},
	}
	cmd.Flags().StringVar(&networkSlug, "network", "", "Network slug (required)")
	cmd.Flags().StringVar(&protocol, "protocol", "", "Protocol: tcp, udp, icmp, or all (required)")
	cmd.Flags().StringVar(&startPort, "start-port", "", "Start port number")
	cmd.Flags().StringVar(&endPort, "end-port", "", "End port number")
	cmd.Flags().StringVar(&cidr, "cidr", "", "CIDR (e.g. 0.0.0.0/0)")
	cmd.Flags().StringVar(&icmpType, "icmp-type", "", "ICMP type (ICMP protocol only)")
	cmd.Flags().StringVar(&icmpCode, "icmp-code", "", "ICMP code (ICMP protocol only)")
	return cmd
}

func runEgressCreate(cmd *cobra.Command, req egress.CreateRequest) error {
	_, client, printer, err := buildClientAndPrinter(cmd)
	if err != nil {
		return err
	}

	svc := egress.NewService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	rule, err := svc.Create(ctx, req)
	if err != nil {
		return fmt.Errorf("egress create: %w", err)
	}

	headers := []string{"FIELD", "VALUE"}
	rows := [][]string{
		{"ID", rule.ID},
		{"Protocol", rule.Protocol},
		{"Ports", formatEgressPorts(rule.StartPort, rule.EndPort)},
		{"CIDR", rule.CIDR},
		{"Status", rule.Status},
	}
	return printer.PrintTable(headers, rows)
}

func newEgressDeleteCmd() *cobra.Command {
	var networkSlug string
	var yes bool

	cmd := &cobra.Command{
		Use:   "delete <id>",
		Short: "Delete an egress rule",
		Args:  cobra.ExactArgs(1),
		Example: `  zcp egress delete 42 --network <slug>
  zcp egress delete 42 --network <slug> --yes`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if networkSlug == "" {
				return fmt.Errorf("--network is required")
			}
			ruleID := args[0]
			return runEgressDelete(cmd, networkSlug, ruleID, yes)
		},
	}
	cmd.Flags().StringVar(&networkSlug, "network", "", "Network slug (required)")
	cmd.Flags().BoolVar(&yes, "yes", false, "Skip confirmation prompt")
	return cmd
}

func runEgressDelete(cmd *cobra.Command, networkSlug string, ruleID string, yes bool) error {
	if !yes && !autoApproved(cmd) {
		fmt.Fprintf(os.Stderr, "Delete egress rule %s from network %q? [y/N]: ", ruleID, networkSlug)
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

	svc := egress.NewService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	if err := svc.Delete(ctx, networkSlug, ruleID); err != nil {
		return fmt.Errorf("egress delete: %w", err)
	}

	printer.Fprintf("Deleted egress rule %s from network %q\n", ruleID, networkSlug)
	return nil
}

// formatEgressPorts returns a human-readable ports string.
func formatEgressPorts(start, end string) string {
	if start == "" && end == "" {
		return "all"
	}
	if end == "" || end == start {
		return start
	}
	return start + "-" + end
}
