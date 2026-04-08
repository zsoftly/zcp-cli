package commands

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"github.com/zsoftly/zcp-cli/internal/api/acl"
)

// NewACLCmd returns the 'acl' cobra command.
func NewACLCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "acl",
		Short: "Manage Network ACLs",
	}
	cmd.AddCommand(newACLListCmd())
	cmd.AddCommand(newACLCreateRuleCmd())
	cmd.AddCommand(newACLReplaceCmd())
	return cmd
}

func newACLListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "list <vpc-slug>",
		Short:   "List network ACLs for a VPC",
		Args:    cobra.ExactArgs(1),
		Example: `  zcp acl list <vpc-slug>`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runACLList(cmd, args[0])
		},
	}
	return cmd
}

func runACLList(cmd *cobra.Command, vpcSlug string) error {
	_, client, printer, err := buildClientAndPrinter(cmd)
	if err != nil {
		return err
	}

	svc := acl.NewService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	acls, err := svc.List(ctx, vpcSlug)
	if err != nil {
		return fmt.Errorf("acl list: %w", err)
	}

	headers := []string{"SLUG", "NAME", "DESCRIPTION", "VPC", "STATUS"}
	rows := make([][]string, 0, len(acls))
	for _, a := range acls {
		rows = append(rows, []string{
			a.Slug,
			a.Name,
			a.Description,
			a.VPCSlug,
			a.Status,
		})
	}
	return printer.PrintTable(headers, rows)
}

func newACLCreateRuleCmd() *cobra.Command {
	var protocol, cidrList, trafficType, action string
	var startPort, endPort, number, icmpCode, icmpType int

	cmd := &cobra.Command{
		Use:   "create-rule <vpc-slug>",
		Short: "Create a network ACL rule in a VPC",
		Args:  cobra.ExactArgs(1),
		Example: `  zcp acl create-rule <vpc-slug> --protocol tcp --action allow --start-port 80 --end-port 80 --cidr 0.0.0.0/0
  zcp acl create-rule <vpc-slug> --protocol icmp --action deny --icmp-type 8 --icmp-code 0`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if protocol == "" {
				return fmt.Errorf("--protocol is required")
			}
			if action == "" {
				return fmt.Errorf("--action is required")
			}
			return runACLCreateRule(cmd, args[0], acl.ACLRuleCreateRequest{
				Protocol:    protocol,
				CIDRList:    cidrList,
				StartPort:   startPort,
				EndPort:     endPort,
				TrafficType: trafficType,
				Action:      action,
				Number:      number,
				ICMPCode:    icmpCode,
				ICMPType:    icmpType,
			})
		},
	}
	cmd.Flags().StringVar(&protocol, "protocol", "", "Protocol (tcp, udp, icmp, all) (required)")
	cmd.Flags().StringVar(&cidrList, "cidr", "", "CIDR list (e.g. 0.0.0.0/0)")
	cmd.Flags().IntVar(&startPort, "start-port", 0, "Start port")
	cmd.Flags().IntVar(&endPort, "end-port", 0, "End port")
	cmd.Flags().StringVar(&trafficType, "traffic-type", "", "Traffic type (ingress, egress)")
	cmd.Flags().StringVar(&action, "action", "", "Action (allow, deny) (required)")
	cmd.Flags().IntVar(&number, "number", 0, "Rule number (ordering)")
	cmd.Flags().IntVar(&icmpCode, "icmp-code", 0, "ICMP code")
	cmd.Flags().IntVar(&icmpType, "icmp-type", 0, "ICMP type")
	return cmd
}

func runACLCreateRule(cmd *cobra.Command, vpcSlug string, req acl.ACLRuleCreateRequest) error {
	_, client, printer, err := buildClientAndPrinter(cmd)
	if err != nil {
		return err
	}

	svc := acl.NewService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	rule, err := svc.CreateRule(ctx, vpcSlug, req)
	if err != nil {
		return fmt.Errorf("acl create-rule: %w", err)
	}

	headers := []string{"FIELD", "VALUE"}
	rows := [][]string{
		{"Slug", rule.Slug},
		{"Protocol", rule.Protocol},
		{"Action", rule.Action},
		{"CIDR List", rule.CIDRList},
		{"Start Port", fmt.Sprintf("%d", rule.StartPort)},
		{"End Port", fmt.Sprintf("%d", rule.EndPort)},
		{"Traffic Type", rule.TrafficType},
		{"Number", fmt.Sprintf("%d", rule.Number)},
	}
	return printer.PrintTable(headers, rows)
}

func newACLReplaceCmd() *cobra.Command {
	var networkSlug, aclSlug string

	cmd := &cobra.Command{
		Use:     "replace",
		Short:   "Replace the ACL on a network",
		Example: `  zcp acl replace --network <network-slug> --acl <acl-slug>`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if networkSlug == "" {
				return fmt.Errorf("--network is required")
			}
			if aclSlug == "" {
				return fmt.Errorf("--acl is required")
			}
			return runACLReplace(cmd, networkSlug, aclSlug)
		},
	}
	cmd.Flags().StringVar(&networkSlug, "network", "", "Network slug (required)")
	cmd.Flags().StringVar(&aclSlug, "acl", "", "ACL slug (required)")
	return cmd
}

func runACLReplace(cmd *cobra.Command, networkSlug, aclSlug string) error {
	_, client, printer, err := buildClientAndPrinter(cmd)
	if err != nil {
		return err
	}

	svc := acl.NewService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	if err := svc.ReplaceNetworkACL(ctx, networkSlug, aclSlug); err != nil {
		return fmt.Errorf("acl replace: %w", err)
	}

	printer.Fprintf("ACL replaced on network %q.\n", networkSlug)
	return nil
}
