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
	"github.com/zsoftly/zcp-cli/internal/api/securitygroup"
)

var validSGProtocols = map[string]bool{"tcp": true, "udp": true, "icmp": true, "all": true}

func validateSGProtocol(protocol string) error {
	if !validSGProtocols[strings.ToLower(protocol)] {
		return fmt.Errorf("invalid protocol %q: must be tcp, udp, icmp, or all", protocol)
	}
	return nil
}

// NewSecurityGroupCmd returns the 'security-group' cobra command.
func NewSecurityGroupCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "security-group",
		Short: "Manage security groups and their rules",
	}
	cmd.AddCommand(newSGListCmd())
	cmd.AddCommand(newSGGetCmd())
	cmd.AddCommand(newSGCreateCmd())
	cmd.AddCommand(newSGDeleteCmd())
	cmd.AddCommand(newSGRuleCmd())
	return cmd
}

func newSGListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List security groups",
		Example: `  zcp security-group list
  zcp security-group list --output json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSGList(cmd)
		},
	}
	return cmd
}

func runSGList(cmd *cobra.Command) error {
	_, client, printer, err := buildClientAndPrinter(cmd)
	if err != nil {
		return err
	}

	svc := securitygroup.NewService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	groups, err := svc.List(ctx, "")
	if err != nil {
		return fmt.Errorf("security-group list: %w", err)
	}

	headers := []string{"UUID", "NAME", "DESCRIPTION", "INBOUND RULES", "OUTBOUND RULES", "STATUS"}
	rows := make([][]string, 0, len(groups))
	for _, sg := range groups {
		rows = append(rows, []string{
			sg.UUID,
			sg.Name,
			sg.Description,
			strconv.Itoa(len(sg.FirewallRules)),
			strconv.Itoa(len(sg.EgressRules)),
			sg.Status,
		})
	}
	return printer.PrintTable(headers, rows)
}

func newSGGetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get <uuid>",
		Short: "Get details of a security group including its rules",
		Args:  cobra.ExactArgs(1),
		Example: `  zcp security-group get <uuid>
  zcp security-group get <uuid> --output json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSGGet(cmd, args[0])
		},
	}
	return cmd
}

func runSGGet(cmd *cobra.Command, uuid string) error {
	_, client, printer, err := buildClientAndPrinter(cmd)
	if err != nil {
		return err
	}

	svc := securitygroup.NewService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	sg, err := svc.Get(ctx, uuid)
	if err != nil {
		return fmt.Errorf("security-group get: %w", err)
	}

	// Print group details
	detailHeaders := []string{"FIELD", "VALUE"}
	detailRows := [][]string{
		{"UUID", sg.UUID},
		{"Name", sg.Name},
		{"Description", sg.Description},
		{"Status", sg.Status},
		{"Active", strconv.FormatBool(sg.IsActive)},
	}
	if err := printer.PrintTable(detailHeaders, detailRows); err != nil {
		return err
	}

	// Print inbound rules
	printer.Fprintf("\nInbound Rules:\n")
	if len(sg.FirewallRules) == 0 {
		printer.Fprintf("  (none)\n")
	} else {
		fwHeaders := []string{"UUID", "PROTOCOL", "PORTS", "CIDR"}
		fwRows := make([][]string, 0, len(sg.FirewallRules))
		for _, r := range sg.FirewallRules {
			fwRows = append(fwRows, []string{
				r.UUID,
				r.Protocol,
				formatPorts(r.StartPort, r.EndPort),
				r.CIDRList,
			})
		}
		if err := printer.PrintTable(fwHeaders, fwRows); err != nil {
			return err
		}
	}

	// Print outbound rules
	printer.Fprintf("\nOutbound Rules:\n")
	if len(sg.EgressRules) == 0 {
		printer.Fprintf("  (none)\n")
	} else {
		egHeaders := []string{"UUID", "PROTOCOL", "PORTS"}
		egRows := make([][]string, 0, len(sg.EgressRules))
		for _, r := range sg.EgressRules {
			egRows = append(egRows, []string{
				r.UUID,
				r.Protocol,
				formatPorts(r.StartPort, r.EndPort),
			})
		}
		if err := printer.PrintTable(egHeaders, egRows); err != nil {
			return err
		}
	}

	return nil
}

func newSGCreateCmd() *cobra.Command {
	var name, description string

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a security group",
		Example: `  zcp security-group create --name my-sg
  zcp security-group create --name my-sg --description "Web tier security group"`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if name == "" {
				return fmt.Errorf("--name is required")
			}
			return runSGCreate(cmd, securitygroup.CreateGroupRequest{
				Name:        name,
				Description: description,
			})
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "Security group name (required)")
	cmd.Flags().StringVar(&description, "description", "", "Security group description (optional)")
	return cmd
}

func runSGCreate(cmd *cobra.Command, req securitygroup.CreateGroupRequest) error {
	_, client, printer, err := buildClientAndPrinter(cmd)
	if err != nil {
		return err
	}

	svc := securitygroup.NewService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	sg, err := svc.Create(ctx, req)
	if err != nil {
		return fmt.Errorf("security-group create: %w", err)
	}

	headers := []string{"FIELD", "VALUE"}
	rows := [][]string{
		{"UUID", sg.UUID},
		{"Name", sg.Name},
		{"Description", sg.Description},
		{"Status", sg.Status},
	}
	return printer.PrintTable(headers, rows)
}

func newSGDeleteCmd() *cobra.Command {
	var yes bool

	cmd := &cobra.Command{
		Use:   "delete <uuid>",
		Short: "Delete a security group and all its rules",
		Args:  cobra.ExactArgs(1),
		Example: `  zcp security-group delete <uuid>
  zcp security-group delete <uuid> --yes`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSGDelete(cmd, args[0], yes)
		},
	}
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "Skip confirmation prompt")
	return cmd
}

func runSGDelete(cmd *cobra.Command, uuid string, yes bool) error {
	if !yes && !autoApproved(cmd) {
		fmt.Fprintf(os.Stderr, "WARNING: deleting security group %q will also delete all its rules.\n", uuid)
		fmt.Fprintf(os.Stderr, "Delete security group %q? [y/N]: ", uuid)
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

	svc := securitygroup.NewService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	if err := svc.Delete(ctx, uuid); err != nil {
		return fmt.Errorf("security-group delete: %w", err)
	}

	// Verify deletion — Kong may return 204 even when delete silently fails
	time.Sleep(2 * time.Second)
	if _, err := svc.Get(ctx, uuid); err == nil {
		fmt.Fprintln(os.Stderr, "WARNING: security group may not have been deleted (e.g. in use by an instance).")
		return fmt.Errorf("security group %q still exists after delete — check dependencies", uuid)
	}

	printer.Fprintf("Security group %q deleted.\n", uuid)
	return nil
}

func newSGRuleCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "rule",
		Short: "Manage security group rules",
	}
	cmd.AddCommand(newSGRuleAddFirewallCmd())
	cmd.AddCommand(newSGRuleAddEgressCmd())
	cmd.AddCommand(newSGRuleDeleteCmd())
	return cmd
}

func newSGRuleAddFirewallCmd() *cobra.Command {
	var sgUUID, protocol, startPort, endPort, cidr, icmpType, icmpCode string

	cmd := &cobra.Command{
		Use:   "add-inbound",
		Short: "Add an inbound (firewall) rule to a security group",
		Example: `  zcp security-group rule add-inbound --group <sg-uuid> --protocol tcp --start-port 80 --end-port 80
  zcp security-group rule add-inbound --group <sg-uuid> --protocol tcp --start-port 443 --end-port 443 --cidr 0.0.0.0/0
  zcp security-group rule add-inbound --group <sg-uuid> --protocol icmp`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if sgUUID == "" {
				return fmt.Errorf("--group is required")
			}
			if protocol == "" {
				return fmt.Errorf("--protocol is required")
			}
			if err := validateSGProtocol(protocol); err != nil {
				return err
			}
			proto := strings.ToUpper(protocol)
			if (proto == "TCP" || proto == "UDP") && startPort == "" {
				fmt.Fprintln(os.Stderr, "Warning: no ports specified for TCP/UDP rule; all ports will be affected.")
			}
			return runSGRuleAddFirewall(cmd, securitygroup.CreateFirewallRuleRequest{
				SecurityGroupUUID: sgUUID,
				Protocol:          proto,
				StartPort:         startPort,
				EndPort:           endPort,
				CIDRList:          cidr,
				ICMPType:          icmpType,
				ICMPCode:          icmpCode,
			})
		},
	}
	cmd.Flags().StringVar(&sgUUID, "group", "", "Security group UUID (required)")
	cmd.Flags().StringVar(&protocol, "protocol", "", "Protocol: tcp, udp, icmp, or all (required)")
	cmd.Flags().StringVar(&startPort, "start-port", "", "Start port number")
	cmd.Flags().StringVar(&endPort, "end-port", "", "End port number")
	cmd.Flags().StringVar(&cidr, "cidr", "", "CIDR list (e.g. 0.0.0.0/0)")
	cmd.Flags().StringVar(&icmpType, "icmp-type", "", "ICMP type (ICMP protocol only)")
	cmd.Flags().StringVar(&icmpCode, "icmp-code", "", "ICMP code (ICMP protocol only)")
	return cmd
}

func runSGRuleAddFirewall(cmd *cobra.Command, req securitygroup.CreateFirewallRuleRequest) error {
	_, client, printer, err := buildClientAndPrinter(cmd)
	if err != nil {
		return err
	}

	svc := securitygroup.NewService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	sg, err := svc.CreateFirewallRule(ctx, req)
	if err != nil {
		return fmt.Errorf("security-group rule add-inbound: %w", err)
	}

	headers := []string{"FIELD", "VALUE"}
	rows := [][]string{
		{"Security Group UUID", sg.UUID},
		{"Name", sg.Name},
		{"Inbound Rules", strconv.Itoa(len(sg.FirewallRules))},
		{"Outbound Rules", strconv.Itoa(len(sg.EgressRules))},
	}
	return printer.PrintTable(headers, rows)
}

func newSGRuleAddEgressCmd() *cobra.Command {
	var sgUUID, protocol, startPort, endPort, icmpType, icmpCode string

	cmd := &cobra.Command{
		Use:   "add-egress",
		Short: "Add an outbound (egress) rule to a security group",
		Example: `  zcp security-group rule add-egress --group <sg-uuid> --protocol tcp --start-port 443 --end-port 443
  zcp security-group rule add-egress --group <sg-uuid> --protocol all`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if sgUUID == "" {
				return fmt.Errorf("--group is required")
			}
			if protocol == "" {
				return fmt.Errorf("--protocol is required")
			}
			if err := validateSGProtocol(protocol); err != nil {
				return err
			}
			proto := strings.ToUpper(protocol)
			if (proto == "TCP" || proto == "UDP") && startPort == "" {
				fmt.Fprintln(os.Stderr, "Warning: no ports specified for TCP/UDP rule; all ports will be affected.")
			}
			return runSGRuleAddEgress(cmd, securitygroup.CreateEgressRuleRequest{
				SecurityGroupUUID: sgUUID,
				Protocol:          proto,
				StartPort:         startPort,
				EndPort:           endPort,
				ICMPType:          icmpType,
				ICMPCode:          icmpCode,
			})
		},
	}
	cmd.Flags().StringVar(&sgUUID, "group", "", "Security group UUID (required)")
	cmd.Flags().StringVar(&protocol, "protocol", "", "Protocol: tcp, udp, icmp, or all (required)")
	cmd.Flags().StringVar(&startPort, "start-port", "", "Start port number")
	cmd.Flags().StringVar(&endPort, "end-port", "", "End port number")
	cmd.Flags().StringVar(&icmpType, "icmp-type", "", "ICMP type (ICMP protocol only)")
	cmd.Flags().StringVar(&icmpCode, "icmp-code", "", "ICMP code (ICMP protocol only)")
	return cmd
}

func runSGRuleAddEgress(cmd *cobra.Command, req securitygroup.CreateEgressRuleRequest) error {
	_, client, printer, err := buildClientAndPrinter(cmd)
	if err != nil {
		return err
	}

	svc := securitygroup.NewService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	sg, err := svc.CreateEgressRule(ctx, req)
	if err != nil {
		return fmt.Errorf("security-group rule add-egress: %w", err)
	}

	headers := []string{"FIELD", "VALUE"}
	rows := [][]string{
		{"Security Group UUID", sg.UUID},
		{"Name", sg.Name},
		{"Inbound Rules", strconv.Itoa(len(sg.FirewallRules))},
		{"Outbound Rules", strconv.Itoa(len(sg.EgressRules))},
	}
	return printer.PrintTable(headers, rows)
}

func newSGRuleDeleteCmd() *cobra.Command {
	var sgUUID, ruleUUID, ruleType string
	var yes bool

	cmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete a rule from a security group",
		Example: `  zcp security-group rule delete --group <sg-uuid> --rule <rule-uuid> --type firewall
  zcp security-group rule delete --group <sg-uuid> --rule <rule-uuid> --type egress --yes`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if sgUUID == "" {
				return fmt.Errorf("--group is required")
			}
			if ruleUUID == "" {
				return fmt.Errorf("--rule is required")
			}
			if ruleType == "" {
				return fmt.Errorf("--type is required")
			}
			if ruleType != "firewall" && ruleType != "egress" {
				return fmt.Errorf("invalid --type %q: must be firewall or egress", ruleType)
			}
			return runSGRuleDelete(cmd, sgUUID, ruleType, ruleUUID, yes)
		},
	}
	cmd.Flags().StringVar(&sgUUID, "group", "", "Security group UUID (required)")
	cmd.Flags().StringVar(&ruleUUID, "rule", "", "Rule UUID to delete (required)")
	cmd.Flags().StringVar(&ruleType, "type", "", "Rule type: firewall or egress (required)")
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "Skip confirmation prompt")
	return cmd
}

func runSGRuleDelete(cmd *cobra.Command, sgUUID, ruleType, ruleUUID string, yes bool) error {
	if !yes && !autoApproved(cmd) {
		fmt.Fprintf(os.Stderr, "Delete %s rule %q from security group %q? [y/N]: ", ruleType, ruleUUID, sgUUID)
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

	svc := securitygroup.NewService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	if err := svc.DeleteRule(ctx, sgUUID, ruleType, ruleUUID); err != nil {
		return fmt.Errorf("security-group rule delete: %w", err)
	}

	printer.Fprintf("Rule %q deleted from security group %q.\n", ruleUUID, sgUUID)
	return nil
}
