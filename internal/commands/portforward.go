package commands

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/zsoftly/zcp-cli/internal/api/portforward"
)

var validPortForwardProtocols = map[string]bool{"tcp": true, "udp": true}

func validatePortForwardProtocol(protocol string) error {
	if !validPortForwardProtocols[strings.ToLower(protocol)] {
		return fmt.Errorf("invalid protocol %q: must be tcp or udp", protocol)
	}
	return nil
}

func validatePortNumber(port int, flagName string) error {
	if port == 0 {
		return nil
	}
	if port < 1 || port > 65535 {
		return fmt.Errorf("%s must be a number between 1 and 65535", flagName)
	}
	return nil
}

// NewPortForwardCmd returns the 'portforward' cobra command.
func NewPortForwardCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "portforward",
		Short: "Manage port forwarding rules",
	}
	cmd.AddCommand(newPortForwardListCmd())
	cmd.AddCommand(newPortForwardCreateCmd())
	cmd.AddCommand(newPortForwardDeleteCmd())
	return cmd
}

func newPortForwardListCmd() *cobra.Command {
	var ipSlug string

	cmd := &cobra.Command{
		Use:     "list",
		Short:   "List port forwarding rules",
		Example: `  zcp portforward list --ip <ip-slug>`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if ipSlug == "" {
				return fmt.Errorf("--ip is required")
			}
			return runPortForwardList(cmd, ipSlug)
		},
	}
	cmd.Flags().StringVar(&ipSlug, "ip", "", "IP address slug (required)")
	return cmd
}

func runPortForwardList(cmd *cobra.Command, ipSlug string) error {
	_, client, printer, err := buildClientAndPrinter(cmd)
	if err != nil {
		return err
	}

	svc := portforward.NewService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	rules, err := svc.List(ctx, ipSlug)
	if err != nil {
		return fmt.Errorf("portforward list: %w", err)
	}

	headers := []string{"ID", "PROTOCOL", "PUBLIC PORT", "PRIVATE PORT", "VM", "STATE"}
	rows := make([][]string, 0, len(rules))
	for _, r := range rules {
		publicPort := formatPFPortsInt(r.PublicStartPort, r.PublicEndPort)
		privatePort := formatPFPortsInt(r.PrivateStartPort, r.PrivateEndPort)
		rows = append(rows, []string{
			r.ID,
			r.Protocol,
			publicPort,
			privatePort,
			r.VirtualMachine,
			r.State,
		})
	}
	return printer.PrintTable(headers, rows)
}

func newPortForwardCreateCmd() *cobra.Command {
	var ipSlug, protocol, vmSlug string
	var publicStartPort, publicEndPort, privateStartPort, privateEndPort string

	cmd := &cobra.Command{
		Use:     "create",
		Short:   "Create a port forwarding rule",
		Example: `  zcp portforward create --ip <ip-slug> --protocol tcp --public-port 8080 --private-port 80 --instance <vm-slug>`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if ipSlug == "" {
				return fmt.Errorf("--ip is required")
			}
			if protocol == "" {
				return fmt.Errorf("--protocol is required")
			}
			if err := validatePortForwardProtocol(protocol); err != nil {
				return err
			}
			if publicStartPort == "" {
				return fmt.Errorf("--public-port is required")
			}
			if privateStartPort == "" {
				return fmt.Errorf("--private-port is required")
			}
			if vmSlug == "" {
				return fmt.Errorf("--instance is required")
			}
			for _, check := range []struct {
				port string
				flag string
			}{
				{publicStartPort, "--public-port"},
				{privateStartPort, "--private-port"},
			} {
				if check.port == "" {
					return fmt.Errorf("%s is required", check.flag)
				}
			}
			return runPortForwardCreate(cmd, ipSlug, portforward.CreateRequest{
				Protocol:         strings.ToLower(protocol),
				PublicStartPort:  publicStartPort,
				PublicEndPort:    publicEndPort,
				PrivateStartPort: privateStartPort,
				PrivateEndPort:   privateEndPort,
				VirtualMachine:   vmSlug,
			})
		},
	}
	cmd.Flags().StringVar(&ipSlug, "ip", "", "IP address slug (required)")
	cmd.Flags().StringVar(&protocol, "protocol", "", "Protocol: tcp or udp (required)")
	cmd.Flags().StringVar(&publicStartPort, "public-port", "", "Public port number (required)")
	cmd.Flags().StringVar(&publicEndPort, "public-end-port", "", "Public end port for range")
	cmd.Flags().StringVar(&privateStartPort, "private-port", "", "Private port number (required)")
	cmd.Flags().StringVar(&privateEndPort, "private-end-port", "", "Private end port for range")
	cmd.Flags().StringVar(&vmSlug, "instance", "", "VM slug (required)")
	return cmd
}

func runPortForwardCreate(cmd *cobra.Command, ipSlug string, req portforward.CreateRequest) error {
	_, client, printer, err := buildClientAndPrinter(cmd)
	if err != nil {
		return err
	}

	svc := portforward.NewService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	rule, err := svc.Create(ctx, ipSlug, req)
	if err != nil {
		return fmt.Errorf("portforward create: %w", err)
	}

	headers := []string{"FIELD", "VALUE"}
	rows := [][]string{
		{"ID", rule.ID},
		{"Protocol", rule.Protocol},
		{"Public Port", formatPFPortsInt(rule.PublicStartPort, rule.PublicEndPort)},
		{"Private Port", formatPFPortsInt(rule.PrivateStartPort, rule.PrivateEndPort)},
		{"VM", rule.VirtualMachine},
		{"State", rule.State},
	}
	return printer.PrintTable(headers, rows)
}

func newPortForwardDeleteCmd() *cobra.Command {
	var ipSlug string
	var yes bool

	cmd := &cobra.Command{
		Use:   "delete <rule-id>",
		Short: "Delete a port forwarding rule",
		Args:  cobra.ExactArgs(1),
		Example: `  zcp portforward delete <rule-id> --ip <ip-slug>
  zcp portforward delete <rule-id> --ip <ip-slug> --yes`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if ipSlug == "" {
				return fmt.Errorf("--ip is required")
			}
			return runPortForwardDelete(cmd, ipSlug, args[0], yes)
		},
	}
	cmd.Flags().StringVar(&ipSlug, "ip", "", "IP address slug (required)")
	cmd.Flags().BoolVar(&yes, "yes", false, "Skip confirmation prompt")
	return cmd
}

func runPortForwardDelete(cmd *cobra.Command, ipSlug, ruleID string, yes bool) error {
	if !yes && !autoApproved(cmd) {
		fmt.Fprintf(os.Stderr, "Delete port forwarding rule %q on IP %q? [y/N]: ", ruleID, ipSlug)
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

	svc := portforward.NewService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	if err := svc.Delete(ctx, ipSlug, ruleID); err != nil {
		return fmt.Errorf("portforward delete: %w", err)
	}

	printer.Fprintf("Port forwarding rule %q deleted.\n", ruleID)
	return nil
}

// formatPFPorts returns a human-readable port range string for port forwarding rules (string args).
// Retained for backward compatibility with other commands that may reference it.
func formatPFPorts(start, end string) string {
	if start == "" {
		return ""
	}
	if end == "" || end == start {
		return start
	}
	return start + "-" + end
}

// formatPFPortsInt returns a human-readable port range string for port forwarding rules (int args).
func formatPFPortsInt(start, end string) string {
	if start == "" {
		return ""
	}
	if end == "" || end == start {
		return start
	}
	return start + "-" + end
}
