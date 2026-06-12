package commands

import (
	"bufio"
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/zsoftly/zcp-cli/pkg/api/acl"
	"github.com/zsoftly/zcp-cli/pkg/api/apierrors"
)

// NewACLCmd returns the 'acl' cobra command.
func NewACLCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "acl",
		Short: "Manage Network ACLs",
	}
	cmd.AddCommand(newACLListCmd())
	cmd.AddCommand(newACLCreateCmd())
	cmd.AddCommand(newACLReplaceCmd())
	cmd.AddCommand(newACLDeleteCmd())
	cmd.AddCommand(newACLRulesCmd())
	cmd.AddCommand(newACLCreateRuleCmd())
	cmd.AddCommand(newACLUpdateRuleCmd())
	cmd.AddCommand(newACLDeleteRuleCmd())
	return cmd
}

func newACLListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "list <vpc-slug>",
		Short:   "List network ACLs for a VPC",
		Args:    cobra.ExactArgs(1),
		Example: `  zcp acl list my-vpc`,
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

	headers := []string{"ID", "NAME", "DESCRIPTION"}
	rows := make([][]string, 0, len(acls))
	for _, a := range acls {
		rows = append(rows, []string{
			a.ID,
			a.Name,
			a.Description,
		})
	}
	return printer.PrintTable(headers, rows)
}

func newACLCreateCmd() *cobra.Command {
	var name, description string

	cmd := &cobra.Command{
		Use:     "create <vpc-slug>",
		Short:   "Create a network ACL in a VPC",
		Args:    cobra.ExactArgs(1),
		Example: `  zcp acl create my-vpc --name allow-web --description "Allow HTTP traffic"`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if name == "" {
				return fmt.Errorf("--name is required")
			}
			return runACLCreate(cmd, args[0], acl.ACLCreateRequest{
				Name:        name,
				Description: description,
				VPC:         args[0],
			})
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "ACL name (required)")
	cmd.Flags().StringVar(&description, "description", "", "ACL description")
	return cmd
}

func runACLCreate(cmd *cobra.Command, vpcSlug string, req acl.ACLCreateRequest) error {
	_, client, _, err := buildClientAndPrinter(cmd)
	if err != nil {
		return err
	}

	svc := acl.NewService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	if err := svc.Create(ctx, vpcSlug, req); err != nil {
		return fmt.Errorf("acl create: %w", err)
	}

	fmt.Fprintf(cmd.ErrOrStderr(), "ACL %q created in VPC %q.\n", req.Name, vpcSlug)
	return nil
}

func newACLReplaceCmd() *cobra.Command {
	var networkSlug, aclRef, vpcSlug string

	cmd := &cobra.Command{
		Use:   "replace",
		Short: "Replace the ACL on a network",
		Long: `Replace the network ACL on a VPC network (tier).

--acl accepts the ACL ID directly, or an ACL name when --vpc is given
(names are resolved against the VPC's ACL list).`,
		Example: `  zcp acl replace --network web-tier --acl 5290f39f-5f56-4ca3-b2b5-05a464a081df
  zcp acl replace --network web-tier --acl web-acl --vpc my-vpc`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if networkSlug == "" {
				return fmt.Errorf("--network is required")
			}
			if aclRef == "" {
				return fmt.Errorf("--acl is required")
			}
			return runACLReplace(cmd, networkSlug, aclRef, vpcSlug)
		},
	}
	cmd.Flags().StringVar(&networkSlug, "network", "", "Network slug (required)")
	cmd.Flags().StringVar(&aclRef, "acl", "", "ACL ID, or ACL name when --vpc is given (required)")
	cmd.Flags().StringVar(&vpcSlug, "vpc", "", "VPC slug, used to resolve an ACL name to its ID")
	return cmd
}

func runACLReplace(cmd *cobra.Command, networkSlug, aclRef, vpcSlug string) error {
	_, client, printer, err := buildClientAndPrinter(cmd)
	if err != nil {
		return err
	}

	svc := acl.NewService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	aclID := aclRef
	if vpcSlug != "" {
		if aclID, err = svc.Resolve(ctx, vpcSlug, aclRef); err != nil {
			return fmt.Errorf("acl replace: %w", err)
		}
	} else if !looksLikeUUID(aclRef) {
		return fmt.Errorf("acl replace: %q does not look like an ACL ID — pass --vpc <vpc-slug> to resolve an ACL by name", aclRef)
	}

	if err := svc.ReplaceNetworkACL(ctx, networkSlug, aclID); err != nil {
		return fmt.Errorf("acl replace: %w", err)
	}

	printer.Fprintf("ACL replaced on network %q.\n", networkSlug)
	return nil
}

func newACLDeleteCmd() *cobra.Command {
	var yes bool

	cmd := &cobra.Command{
		Use:   "delete <vpc-slug> <acl-name-or-id>",
		Short: "Delete a network ACL from a VPC",
		Args:  cobra.ExactArgs(2),
		Example: `  zcp acl delete my-vpc web-acl
  zcp acl delete my-vpc 5290f39f-5f56-4ca3-b2b5-05a464a081df --yes`,
		RunE: func(cmd *cobra.Command, args []string) error {
			vpcSlug, aclRef := args[0], args[1]
			if !yes && !autoApproved(cmd) {
				fmt.Fprintf(cmd.ErrOrStderr(), "Delete ACL %q from VPC %q? Networks using it must be moved to another ACL first. [y/N]: ", aclRef, vpcSlug)
				scanner := bufio.NewScanner(cmd.InOrStdin())
				scanner.Scan()
				answer := strings.TrimSpace(strings.ToLower(scanner.Text()))
				if answer != "y" && answer != "yes" {
					fmt.Fprintln(cmd.ErrOrStderr(), "Aborted.")
					return nil
				}
			}
			_, client, _, err := buildClientAndPrinter(cmd)
			if err != nil {
				return err
			}
			svc := acl.NewService(client)
			ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
			defer cancel()
			aclID, err := svc.Resolve(ctx, vpcSlug, aclRef)
			if err != nil {
				return fmt.Errorf("acl delete: %w", err)
			}
			if err := svc.Delete(ctx, vpcSlug, aclID); err != nil {
				if apierrors.IsResourceNotFound(err) {
					fmt.Fprintf(cmd.ErrOrStderr(), "ACL %q not found — already deleted.\n", aclRef)
					return nil
				}
				return fmt.Errorf("acl delete: %w", err)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "ACL %q deleted from VPC %q.\n", aclRef, vpcSlug)
			return nil
		},
	}
	cmd.Flags().BoolVar(&yes, "yes", false, "Skip confirmation prompt")
	return cmd
}

func newACLRulesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "rules <vpc-slug> <acl-name-or-id>",
		Short:   "List the rules inside a network ACL",
		Args:    cobra.ExactArgs(2),
		Example: `  zcp acl rules my-vpc web-acl`,
		RunE: func(cmd *cobra.Command, args []string) error {
			_, client, printer, err := buildClientAndPrinter(cmd)
			if err != nil {
				return err
			}
			svc := acl.NewService(client)
			ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
			defer cancel()

			aclID, err := svc.Resolve(ctx, args[0], args[1])
			if err != nil {
				return fmt.Errorf("acl rules: %w", err)
			}
			rules, err := svc.ListRules(ctx, args[0], aclID)
			if err != nil {
				return fmt.Errorf("acl rules: %w", err)
			}

			headers := []string{"ID", "NUMBER", "ACTION", "TRAFFIC", "PROTOCOL", "PORTS", "CIDR", "STATE"}
			rows := make([][]string, 0, len(rules))
			for _, r := range rules {
				ports := ""
				if r.StartPort != "" {
					ports = r.StartPort + "-" + r.EndPort
				}
				rows = append(rows, []string{
					r.ID,
					strconv.Itoa(r.Number),
					r.Action,
					r.TrafficType,
					r.Protocol,
					ports,
					r.CIDRList,
					r.State,
				})
			}
			return printer.PrintTable(headers, rows)
		},
	}
	return cmd
}

// aclRuleFlags holds the shared flag values for create-rule and update-rule.
type aclRuleFlags struct {
	protocol, cidrList, action, trafficType, description, protocolNumber string
	number, startPort, endPort, icmpType, icmpCode                       int
}

func addACLRuleFlags(cmd *cobra.Command, f *aclRuleFlags) {
	cmd.Flags().IntVar(&f.number, "number", 0, "Rule number (evaluation order)")
	cmd.Flags().StringVar(&f.protocol, "protocol", "", "Protocol: tcp, udp, icmp, all, or protocol_number (required)")
	cmd.Flags().StringVar(&f.protocolNumber, "protocol-number", "", "IP protocol number (with --protocol protocol_number)")
	cmd.Flags().StringVar(&f.cidrList, "cidr", "", "Comma-separated CIDR list (required), e.g. 10.30.1.0/24,10.30.2.0/24")
	cmd.Flags().IntVar(&f.startPort, "start-port", 0, "Start port (required for tcp/udp)")
	cmd.Flags().IntVar(&f.endPort, "end-port", 0, "End port (required for tcp/udp)")
	cmd.Flags().IntVar(&f.icmpType, "icmp-type", -1, "ICMP type (icmp only, -1 for all)")
	cmd.Flags().IntVar(&f.icmpCode, "icmp-code", -1, "ICMP code (icmp only, -1 for all)")
	cmd.Flags().StringVar(&f.action, "action", "allow", "Action: allow or deny")
	cmd.Flags().StringVar(&f.trafficType, "traffic-type", "ingress", "Traffic type: ingress or egress")
	cmd.Flags().StringVar(&f.description, "description", "", "Rule description")
}

// buildACLRuleRequest validates the shared rule flags and builds the request.
func buildACLRuleRequest(cmd *cobra.Command, f *aclRuleFlags) (acl.RuleCreateRequest, error) {
	var req acl.RuleCreateRequest
	switch f.protocol {
	case "tcp", "udp":
		if !cmd.Flags().Changed("start-port") || !cmd.Flags().Changed("end-port") {
			return req, fmt.Errorf("--start-port and --end-port are required for protocol %s", f.protocol)
		}
		if f.startPort < 1 || f.startPort > 65535 || f.endPort < 1 || f.endPort > 65535 {
			return req, fmt.Errorf("ports must be between 1 and 65535 (got %d-%d)", f.startPort, f.endPort)
		}
		if f.endPort < f.startPort {
			return req, fmt.Errorf("--end-port (%d) must not be lower than --start-port (%d)", f.endPort, f.startPort)
		}
	case "icmp", "all":
	case "protocol_number":
		if f.protocolNumber == "" {
			return req, fmt.Errorf("--protocol-number is required for protocol protocol_number")
		}
	case "":
		return req, fmt.Errorf("--protocol is required (tcp, udp, icmp, all, or protocol_number)")
	default:
		return req, fmt.Errorf("--protocol must be tcp, udp, icmp, all, or protocol_number (got %q)", f.protocol)
	}
	if f.cidrList == "" {
		return req, fmt.Errorf("--cidr is required (comma-separated CIDR list)")
	}
	if f.action != "allow" && f.action != "deny" {
		return req, fmt.Errorf("--action must be allow or deny (got %q)", f.action)
	}
	if f.trafficType != "ingress" && f.trafficType != "egress" {
		return req, fmt.Errorf("--traffic-type must be ingress or egress (got %q)", f.trafficType)
	}

	req = acl.RuleCreateRequest{
		Number:         f.number,
		Description:    f.description,
		Protocol:       f.protocol,
		ProtocolNumber: f.protocolNumber,
		CIDRList:       f.cidrList,
		Action:         f.action,
		TrafficType:    f.trafficType,
	}
	if f.protocol == "tcp" || f.protocol == "udp" {
		req.StartPort, req.EndPort = &f.startPort, &f.endPort
	}
	if f.protocol == "icmp" {
		req.ICMPType, req.ICMPCode = &f.icmpType, &f.icmpCode
	}
	return req, nil
}

func newACLCreateRuleCmd() *cobra.Command {
	var f aclRuleFlags

	cmd := &cobra.Command{
		Use:   "create-rule <vpc-slug> <acl-name-or-id>",
		Short: "Add a rule to a network ACL",
		Args:  cobra.ExactArgs(2),
		Example: `  zcp acl create-rule my-vpc web-acl --number 1 --protocol tcp --start-port 80 --end-port 80 --cidr 0.0.0.0/0
  zcp acl create-rule my-vpc web-acl --number 2 --protocol icmp --cidr 10.30.1.0/24,10.30.2.0/24
  zcp acl create-rule my-vpc web-acl --number 3 --protocol all --cidr 0.0.0.0/0 --traffic-type egress`,
		RunE: func(cmd *cobra.Command, args []string) error {
			req, err := buildACLRuleRequest(cmd, &f)
			if err != nil {
				return err
			}

			_, client, printer, err := buildClientAndPrinter(cmd)
			if err != nil {
				return err
			}
			svc := acl.NewService(client)
			ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
			defer cancel()

			aclID, err := svc.Resolve(ctx, args[0], args[1])
			if err != nil {
				return fmt.Errorf("acl create-rule: %w", err)
			}
			if err := svc.CreateRule(ctx, args[0], aclID, req); err != nil {
				return fmt.Errorf("acl create-rule: %w", err)
			}

			printer.Fprintf("Rule added to ACL %q in VPC %q.\n", args[1], args[0])
			return nil
		},
	}
	addACLRuleFlags(cmd, &f)
	return cmd
}

func newACLUpdateRuleCmd() *cobra.Command {
	var f aclRuleFlags

	cmd := &cobra.Command{
		Use:   "update-rule <vpc-slug> <acl-name-or-id> <rule-id>",
		Short: "Update a rule in a network ACL in place",
		Long: `Update a rule in a network ACL. The rule ID is preserved.

All rule fields must be provided (the API replaces the whole rule); use
"zcp acl rules" to see the current values and the rule ID.`,
		Args: cobra.ExactArgs(3),
		Example: `  zcp acl rules my-vpc web-acl
  zcp acl update-rule my-vpc web-acl <rule-id> --number 3 --protocol icmp --cidr 10.30.1.0/24,10.30.2.0/24,10.30.3.0/24`,
		RunE: func(cmd *cobra.Command, args []string) error {
			req, err := buildACLRuleRequest(cmd, &f)
			if err != nil {
				return err
			}

			_, client, printer, err := buildClientAndPrinter(cmd)
			if err != nil {
				return err
			}
			svc := acl.NewService(client)
			ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
			defer cancel()

			aclID, err := svc.Resolve(ctx, args[0], args[1])
			if err != nil {
				return fmt.Errorf("acl update-rule: %w", err)
			}
			if err := svc.UpdateRule(ctx, args[0], aclID, args[2], req); err != nil {
				return fmt.Errorf("acl update-rule: %w", err)
			}

			printer.Fprintf("Rule %q updated in ACL %q.\n", args[2], args[1])
			return nil
		},
	}
	addACLRuleFlags(cmd, &f)
	return cmd
}

func newACLDeleteRuleCmd() *cobra.Command {
	var yes bool

	cmd := &cobra.Command{
		Use:     "delete-rule <vpc-slug> <acl-name-or-id> <rule-id>",
		Short:   "Delete a rule from a network ACL",
		Args:    cobra.ExactArgs(3),
		Example: `  zcp acl delete-rule my-vpc web-acl 71b2bf4d-dffc-4956-9ea3-8befedc5b0a1 --yes`,
		RunE: func(cmd *cobra.Command, args []string) error {
			vpcSlug, aclRef, ruleID := args[0], args[1], args[2]
			if !yes && !autoApproved(cmd) {
				fmt.Fprintf(cmd.ErrOrStderr(), "Delete rule %q from ACL %q? [y/N]: ", ruleID, aclRef)
				scanner := bufio.NewScanner(cmd.InOrStdin())
				scanner.Scan()
				answer := strings.TrimSpace(strings.ToLower(scanner.Text()))
				if answer != "y" && answer != "yes" {
					fmt.Fprintln(cmd.ErrOrStderr(), "Aborted.")
					return nil
				}
			}
			_, client, _, err := buildClientAndPrinter(cmd)
			if err != nil {
				return err
			}
			svc := acl.NewService(client)
			ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
			defer cancel()

			aclID, err := svc.Resolve(ctx, vpcSlug, aclRef)
			if err != nil {
				return fmt.Errorf("acl delete-rule: %w", err)
			}
			if err := svc.DeleteRule(ctx, vpcSlug, aclID, ruleID); err != nil {
				if apierrors.IsResourceNotFound(err) {
					fmt.Fprintf(cmd.ErrOrStderr(), "Rule %q not found in ACL %q — already deleted.\n", ruleID, aclRef)
					return nil
				}
				return fmt.Errorf("acl delete-rule: %w", err)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Rule %q deleted from ACL %q.\n", ruleID, aclRef)
			return nil
		},
	}
	cmd.Flags().BoolVar(&yes, "yes", false, "Skip confirmation prompt")
	return cmd
}
