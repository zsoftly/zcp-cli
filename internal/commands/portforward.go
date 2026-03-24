package commands

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strconv"
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

func validatePortNumber(port, flagName string) error {
	if port == "" {
		return nil
	}
	n, err := strconv.Atoi(port)
	if err != nil || n < 1 || n > 65535 {
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
	var zoneUUID, ipUUID, instanceUUID string

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List port forwarding rules",
		Example: `  zcp portforward list --zone <uuid>
  zcp portforward list --zone <uuid> --ip <uuid>
  zcp portforward list --zone <uuid> --instance <uuid>`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPortForwardList(cmd, zoneUUID, ipUUID, instanceUUID)
		},
	}
	cmd.Flags().StringVar(&zoneUUID, "zone", "", "Zone UUID (overrides default zone)")
	cmd.Flags().StringVar(&ipUUID, "ip", "", "Filter by IP address UUID")
	cmd.Flags().StringVar(&instanceUUID, "instance", "", "Filter by VM UUID")
	return cmd
}

func runPortForwardList(cmd *cobra.Command, zoneUUID, ipUUID, instanceUUID string) error {
	profile, client, printer, err := buildClientAndPrinter(cmd)
	if err != nil {
		return err
	}
	zoneUUID = resolveZone(profile, zoneUUID)
	if zoneUUID == "" {
		return errNoZone()
	}

	svc := portforward.NewService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	rules, err := svc.List(ctx, zoneUUID, "", instanceUUID, ipUUID)
	if err != nil {
		return fmt.Errorf("portforward list: %w", err)
	}

	headers := []string{"UUID", "PROTOCOL", "PUBLIC PORT", "PRIVATE PORT", "INSTANCE", "IP", "STATUS"}
	rows := make([][]string, 0, len(rules))
	for _, r := range rules {
		publicPort := formatPFPorts(r.PublicPort, r.PublicEndPort)
		privatePort := formatPFPorts(r.PrivatePort, r.PrivateEndPort)
		rows = append(rows, []string{
			r.UUID,
			r.Protocol,
			publicPort,
			privatePort,
			r.VirtualMachineName,
			r.IPAddressUUID,
			r.Status,
		})
	}
	return printer.PrintTable(headers, rows)
}

func newPortForwardCreateCmd() *cobra.Command {
	var ipUUID, protocol, publicPort, publicEndPort, privatePort, privateEndPort, instanceUUID, networkUUID string

	cmd := &cobra.Command{
		Use:     "create",
		Short:   "Create a port forwarding rule",
		Example: `  zcp portforward create --ip <uuid> --protocol tcp --public-port 8080 --private-port 80 --instance <uuid>`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if ipUUID == "" {
				return fmt.Errorf("--ip is required")
			}
			if protocol == "" {
				return fmt.Errorf("--protocol is required")
			}
			if err := validatePortForwardProtocol(protocol); err != nil {
				return err
			}
			if publicPort == "" {
				return fmt.Errorf("--public-port is required")
			}
			if privatePort == "" {
				return fmt.Errorf("--private-port is required")
			}
			if instanceUUID == "" {
				return fmt.Errorf("--instance is required")
			}
			for _, check := range []struct{ port, flag string }{
				{publicPort, "--public-port"},
				{publicEndPort, "--public-end-port"},
				{privatePort, "--private-port"},
				{privateEndPort, "--private-end-port"},
			} {
				if err := validatePortNumber(check.port, check.flag); err != nil {
					return err
				}
			}
			return runPortForwardCreate(cmd, portforward.CreateRequest{
				IPAddressUUID:      ipUUID,
				Protocol:           strings.ToUpper(protocol),
				PublicPort:         publicPort,
				PublicEndPort:      publicEndPort,
				PrivatePort:        privatePort,
				PrivateEndPort:     privateEndPort,
				VirtualMachineUUID: instanceUUID,
				NetworkUUID:        networkUUID,
			})
		},
	}
	cmd.Flags().StringVar(&ipUUID, "ip", "", "IP address UUID (required)")
	cmd.Flags().StringVar(&protocol, "protocol", "", "Protocol: tcp or udp (required)")
	cmd.Flags().StringVar(&publicPort, "public-port", "", "Public port number (required)")
	cmd.Flags().StringVar(&publicEndPort, "public-end-port", "", "Public end port for range")
	cmd.Flags().StringVar(&privatePort, "private-port", "", "Private port number (required)")
	cmd.Flags().StringVar(&privateEndPort, "private-end-port", "", "Private end port for range")
	cmd.Flags().StringVar(&instanceUUID, "instance", "", "VM UUID (required)")
	cmd.Flags().StringVar(&networkUUID, "network", "", "Network UUID")
	return cmd
}

func runPortForwardCreate(cmd *cobra.Command, req portforward.CreateRequest) error {
	_, client, printer, err := buildClientAndPrinter(cmd)
	if err != nil {
		return err
	}

	svc := portforward.NewService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	rule, err := svc.Create(ctx, req)
	if err != nil {
		return fmt.Errorf("portforward create: %w", err)
	}

	headers := []string{"FIELD", "VALUE"}
	rows := [][]string{
		{"UUID", rule.UUID},
		{"Protocol", rule.Protocol},
		{"Public Port", formatPFPorts(rule.PublicPort, rule.PublicEndPort)},
		{"Private Port", formatPFPorts(rule.PrivatePort, rule.PrivateEndPort)},
		{"Instance", rule.VirtualMachineName},
		{"IP Address UUID", rule.IPAddressUUID},
		{"Status", rule.Status},
	}
	return printer.PrintTable(headers, rows)
}

func newPortForwardDeleteCmd() *cobra.Command {
	var yes bool

	cmd := &cobra.Command{
		Use:   "delete <uuid>",
		Short: "Delete a port forwarding rule",
		Args:  cobra.ExactArgs(1),
		Example: `  zcp portforward delete <uuid>
  zcp portforward delete <uuid> --yes`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPortForwardDelete(cmd, args[0], yes)
		},
	}
	cmd.Flags().BoolVar(&yes, "yes", false, "Skip confirmation prompt")
	return cmd
}

func runPortForwardDelete(cmd *cobra.Command, uuid string, yes bool) error {
	if !yes {
		fmt.Fprintf(os.Stderr, "Delete port forwarding rule %q? [y/N]: ", uuid)
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

	if err := svc.Delete(ctx, uuid); err != nil {
		return fmt.Errorf("portforward delete: %w", err)
	}

	printer.Fprintf("Port forwarding rule %q deleted.\n", uuid)
	return nil
}

// formatPFPorts returns a human-readable port range string for port forwarding rules.
func formatPFPorts(start, end string) string {
	if start == "" {
		return ""
	}
	if end == "" || end == start {
		return start
	}
	return start + "-" + end
}
