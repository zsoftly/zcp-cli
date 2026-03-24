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
		Short: "Manage egress rules",
	}
	cmd.AddCommand(newEgressListCmd())
	cmd.AddCommand(newEgressCreateCmd())
	cmd.AddCommand(newEgressDeleteCmd())
	return cmd
}

func newEgressListCmd() *cobra.Command {
	var zoneUUID, networkUUID string

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List egress rules",
		Example: `  zcp egress list --zone <uuid>
  zcp egress list --zone <uuid> --network <uuid>`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runEgressList(cmd, zoneUUID, networkUUID)
		},
	}
	cmd.Flags().StringVar(&zoneUUID, "zone", "", "Zone UUID (overrides default zone)")
	cmd.Flags().StringVar(&networkUUID, "network", "", "Filter by network UUID")
	return cmd
}

func runEgressList(cmd *cobra.Command, zoneUUID, networkUUID string) error {
	profile, client, printer, err := buildClientAndPrinter(cmd)
	if err != nil {
		return err
	}
	zoneUUID = resolveZone(profile, zoneUUID)
	if zoneUUID == "" {
		return errNoZone()
	}

	svc := egress.NewService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	rules, err := svc.List(ctx, zoneUUID, "", networkUUID)
	if err != nil {
		return fmt.Errorf("egress list: %w", err)
	}

	headers := []string{"UUID", "PROTOCOL", "PORTS", "NETWORK", "STATUS"}
	rows := make([][]string, 0, len(rules))
	for _, r := range rules {
		ports := formatEgressPorts(r.StartPort, r.EndPort)
		rows = append(rows, []string{
			r.UUID,
			r.Protocol,
			ports,
			r.NetworkUUID,
			r.Status,
		})
	}
	return printer.PrintTable(headers, rows)
}

func newEgressCreateCmd() *cobra.Command {
	var networkUUID, protocol, startPort, endPort, cidr, icmpType, icmpCode string

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create an egress rule",
		Example: `  zcp egress create --network <uuid> --protocol tcp --start-port 443
  zcp egress create --network <uuid> --protocol all`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if networkUUID == "" {
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
				NetworkUUID: networkUUID,
				Protocol:    proto,
				StartPort:   startPort,
				EndPort:     endPort,
				CIDRList:    cidr,
				ICMPType:    icmpType,
				ICMPCode:    icmpCode,
			})
		},
	}
	cmd.Flags().StringVar(&networkUUID, "network", "", "Network UUID (required)")
	cmd.Flags().StringVar(&protocol, "protocol", "", "Protocol: tcp, udp, icmp, or all (required)")
	cmd.Flags().StringVar(&startPort, "start-port", "", "Start port number")
	cmd.Flags().StringVar(&endPort, "end-port", "", "End port number")
	cmd.Flags().StringVar(&cidr, "cidr", "", "Comma-separated CIDR list")
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
		{"UUID", rule.UUID},
		{"Protocol", rule.Protocol},
		{"Ports", formatEgressPorts(rule.StartPort, rule.EndPort)},
		{"Network UUID", rule.NetworkUUID},
		{"Status", rule.Status},
	}
	return printer.PrintTable(headers, rows)
}

func newEgressDeleteCmd() *cobra.Command {
	var yes bool

	cmd := &cobra.Command{
		Use:   "delete <uuid>",
		Short: "Delete an egress rule",
		Args:  cobra.ExactArgs(1),
		Example: `  zcp egress delete <uuid>
  zcp egress delete <uuid> --yes`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runEgressDelete(cmd, args[0], yes)
		},
	}
	cmd.Flags().BoolVar(&yes, "yes", false, "Skip confirmation prompt")
	return cmd
}

func runEgressDelete(cmd *cobra.Command, uuid string, yes bool) error {
	if !yes {
		fmt.Fprintf(os.Stderr, "Delete egress rule %q? [y/N]: ", uuid)
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

	if err := svc.Delete(ctx, uuid); err != nil {
		return fmt.Errorf("egress delete: %w", err)
	}

	printer.Fprintf("Egress rule %q deleted.\n", uuid)
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
