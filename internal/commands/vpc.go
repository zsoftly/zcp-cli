package commands

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/zsoftly/zcp-cli/internal/api/vpc"
)

// NewVPCCmd returns the 'vpc' cobra command.
func NewVPCCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "vpc",
		Short: "Manage Virtual Private Clouds",
	}
	cmd.AddCommand(newVPCListCmd())
	cmd.AddCommand(newVPCGetCmd())
	cmd.AddCommand(newVPCCreateCmd())
	cmd.AddCommand(newVPCUpdateCmd())
	cmd.AddCommand(newVPCDeleteCmd())
	cmd.AddCommand(newVPCRestartCmd())
	cmd.AddCommand(newVPCACLListCmd())
	cmd.AddCommand(newVPCACLCreateRuleCmd())
	cmd.AddCommand(newVPCACLReplaceCmd())
	cmd.AddCommand(newVPCVPNGatewayCmd())
	return cmd
}

func newVPCListCmd() *cobra.Command {
	var zoneSlug string

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List VPCs",
		Example: `  zcp vpc list
  zcp vpc list --zone <slug>
  zcp vpc list --output json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runVPCList(cmd, zoneSlug)
		},
	}
	cmd.Flags().StringVar(&zoneSlug, "zone", "", "Filter by zone slug")
	return cmd
}

func runVPCList(cmd *cobra.Command, zoneSlug string) error {
	_, client, printer, err := buildClientAndPrinter(cmd)
	if err != nil {
		return err
	}

	svc := vpc.NewService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	vpcs, err := svc.List(ctx, zoneSlug)
	if err != nil {
		return fmt.Errorf("vpc list: %w", err)
	}

	headers := []string{"SLUG", "NAME", "CIDR", "STATUS", "ZONE"}
	rows := make([][]string, 0, len(vpcs))
	for _, v := range vpcs {
		rows = append(rows, []string{
			v.Slug,
			v.Name,
			v.CIDR,
			v.Status,
			v.ZoneName,
		})
	}
	return printer.PrintTable(headers, rows)
}

func newVPCGetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "get <slug>",
		Short:   "Get details of a VPC",
		Args:    cobra.ExactArgs(1),
		Example: `  zcp vpc get <slug>`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runVPCGet(cmd, args[0])
		},
	}
	return cmd
}

func runVPCGet(cmd *cobra.Command, slug string) error {
	_, client, printer, err := buildClientAndPrinter(cmd)
	if err != nil {
		return err
	}

	svc := vpc.NewService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	v, err := svc.Get(ctx, slug)
	if err != nil {
		return fmt.Errorf("vpc get: %w", err)
	}

	headers := []string{"FIELD", "VALUE"}
	rows := [][]string{
		{"Slug", v.Slug},
		{"Name", v.Name},
		{"Description", v.Description},
		{"CIDR", v.CIDR},
		{"Status", v.Status},
		{"Zone Name", v.ZoneName},
		{"Domain Name", v.DomainName},
	}
	return printer.PrintTable(headers, rows)
}

func newVPCCreateCmd() *cobra.Command {
	var zoneSlug, name, offeringSlug, cidr, description, networkDomain, lbProvider string

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new VPC",
		Example: `  zcp vpc create --zone <slug> --name my-vpc --offering <slug> --cidr 10.0.0.0/8
  zcp vpc create --zone <slug> --name my-vpc --offering <slug> --cidr 10.0.0.0/8 --description "Production VPC"`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if name == "" {
				return fmt.Errorf("--name is required")
			}
			if offeringSlug == "" {
				return fmt.Errorf("--offering is required")
			}
			if cidr == "" {
				return fmt.Errorf("--cidr is required")
			}
			if !strings.Contains(cidr, "/") {
				return fmt.Errorf("--cidr must be a valid CIDR (e.g. 10.0.0.0/8)")
			}
			if zoneSlug == "" {
				return fmt.Errorf("--zone is required")
			}
			return runVPCCreate(cmd, vpc.CreateRequest{
				Name:                       name,
				ZoneSlug:                   zoneSlug,
				VPCOfferingSlug:            offeringSlug,
				CIDR:                       cidr,
				Description:                description,
				NetworkDomain:              networkDomain,
				PublicLoadBalancerProvider: lbProvider,
			})
		},
	}
	cmd.Flags().StringVar(&zoneSlug, "zone", "", "Zone slug (required)")
	cmd.Flags().StringVar(&name, "name", "", "VPC name (required)")
	cmd.Flags().StringVar(&offeringSlug, "offering", "", "VPC offering slug (required)")
	cmd.Flags().StringVar(&cidr, "cidr", "", "CIDR block (required, e.g. 10.0.0.0/8)")
	cmd.Flags().StringVar(&description, "description", "", "VPC description")
	cmd.Flags().StringVar(&networkDomain, "network-domain", "", "Network domain")
	cmd.Flags().StringVar(&lbProvider, "lb-provider", "", "Public load balancer provider")
	return cmd
}

func runVPCCreate(cmd *cobra.Command, req vpc.CreateRequest) error {
	_, client, printer, err := buildClientAndPrinter(cmd)
	if err != nil {
		return err
	}

	svc := vpc.NewService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	v, err := svc.Create(ctx, req)
	if err != nil {
		return fmt.Errorf("vpc create: %w", err)
	}

	headers := []string{"FIELD", "VALUE"}
	rows := [][]string{
		{"Slug", v.Slug},
		{"Name", v.Name},
		{"CIDR", v.CIDR},
		{"Status", v.Status},
		{"Zone Name", v.ZoneName},
	}
	return printer.PrintTable(headers, rows)
}

func newVPCUpdateCmd() *cobra.Command {
	var name, description string

	cmd := &cobra.Command{
		Use:     "update <slug>",
		Short:   "Update a VPC",
		Args:    cobra.ExactArgs(1),
		Example: `  zcp vpc update <slug> --name new-name --description "Updated description"`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if name == "" {
				return fmt.Errorf("--name is required")
			}
			return runVPCUpdate(cmd, args[0], vpc.UpdateRequest{
				Name:        name,
				Description: description,
			})
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "New VPC name (required)")
	cmd.Flags().StringVar(&description, "description", "", "New VPC description")
	return cmd
}

func runVPCUpdate(cmd *cobra.Command, slug string, req vpc.UpdateRequest) error {
	_, client, printer, err := buildClientAndPrinter(cmd)
	if err != nil {
		return err
	}

	svc := vpc.NewService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	v, err := svc.Update(ctx, slug, req)
	if err != nil {
		return fmt.Errorf("vpc update: %w", err)
	}

	headers := []string{"FIELD", "VALUE"}
	rows := [][]string{
		{"Slug", v.Slug},
		{"Name", v.Name},
		{"Description", v.Description},
		{"Status", v.Status},
	}
	return printer.PrintTable(headers, rows)
}

func newVPCDeleteCmd() *cobra.Command {
	var yes bool

	cmd := &cobra.Command{
		Use:   "delete <slug>",
		Short: "Delete a VPC",
		Args:  cobra.ExactArgs(1),
		Example: `  zcp vpc delete <slug>
  zcp vpc delete <slug> --yes`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runVPCDelete(cmd, args[0], yes)
		},
	}
	cmd.Flags().BoolVar(&yes, "yes", false, "Skip confirmation prompt")
	return cmd
}

func runVPCDelete(cmd *cobra.Command, slug string, yes bool) error {
	if !yes && !autoApproved(cmd) {
		fmt.Fprintf(os.Stderr, "Delete VPC %q? This action cannot be undone. [y/N]: ", slug)
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

	svc := vpc.NewService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	if err := svc.Delete(ctx, slug); err != nil {
		return fmt.Errorf("vpc delete: %w", err)
	}

	// Verify deletion
	time.Sleep(2 * time.Second)
	if _, err := svc.Get(ctx, slug); err == nil {
		fmt.Fprintln(os.Stderr, "WARNING: VPC may not have been deleted (e.g. has active network tiers).")
		fmt.Fprintln(os.Stderr, "         Delete all network tiers first, then retry.")
		return fmt.Errorf("vpc %q still exists after delete — check dependencies", slug)
	}

	printer.Fprintf("VPC %q deleted.\n", slug)
	return nil
}

func newVPCRestartCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "restart <slug>",
		Short:   "Restart a VPC",
		Args:    cobra.ExactArgs(1),
		Example: `  zcp vpc restart <slug>`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runVPCRestart(cmd, args[0])
		},
	}
	return cmd
}

func runVPCRestart(cmd *cobra.Command, slug string) error {
	_, client, printer, err := buildClientAndPrinter(cmd)
	if err != nil {
		return err
	}

	svc := vpc.NewService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	v, err := svc.Restart(ctx, slug)
	if err != nil {
		return fmt.Errorf("vpc restart: %w", err)
	}

	headers := []string{"FIELD", "VALUE"}
	rows := [][]string{
		{"Slug", v.Slug},
		{"Name", v.Name},
		{"Status", v.Status},
	}
	return printer.PrintTable(headers, rows)
}

// ─── VPC ACL subcommands ─────────────────────────────────────────────────────

func newVPCACLListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "acl-list <vpc-slug>",
		Short:   "List network ACLs for a VPC",
		Args:    cobra.ExactArgs(1),
		Example: `  zcp vpc acl-list <vpc-slug>`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runVPCACLList(cmd, args[0])
		},
	}
	return cmd
}

func runVPCACLList(cmd *cobra.Command, vpcSlug string) error {
	_, client, printer, err := buildClientAndPrinter(cmd)
	if err != nil {
		return err
	}

	svc := vpc.NewService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	acls, err := svc.ListACLs(ctx, vpcSlug)
	if err != nil {
		return fmt.Errorf("vpc acl-list: %w", err)
	}

	headers := []string{"SLUG", "NAME", "DESCRIPTION", "STATUS"}
	rows := make([][]string, 0, len(acls))
	for _, a := range acls {
		rows = append(rows, []string{
			a.Slug,
			a.Name,
			a.Description,
			a.Status,
		})
	}
	return printer.PrintTable(headers, rows)
}

func newVPCACLCreateRuleCmd() *cobra.Command {
	var protocol, cidrList, trafficType, action string
	var startPort, endPort, number, icmpCode, icmpType int

	cmd := &cobra.Command{
		Use:   "acl-create-rule <vpc-slug>",
		Short: "Create a network ACL rule in a VPC",
		Args:  cobra.ExactArgs(1),
		Example: `  zcp vpc acl-create-rule <vpc-slug> --protocol tcp --action allow --start-port 80 --end-port 80 --cidr 0.0.0.0/0
  zcp vpc acl-create-rule <vpc-slug> --protocol icmp --action deny --icmp-type 8 --icmp-code 0`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if protocol == "" {
				return fmt.Errorf("--protocol is required")
			}
			if action == "" {
				return fmt.Errorf("--action is required")
			}
			return runVPCACLCreateRule(cmd, args[0], vpc.ACLRuleCreateRequest{
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

func runVPCACLCreateRule(cmd *cobra.Command, vpcSlug string, req vpc.ACLRuleCreateRequest) error {
	_, client, printer, err := buildClientAndPrinter(cmd)
	if err != nil {
		return err
	}

	svc := vpc.NewService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	rule, err := svc.CreateACLRule(ctx, vpcSlug, req)
	if err != nil {
		return fmt.Errorf("vpc acl-create-rule: %w", err)
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

func newVPCACLReplaceCmd() *cobra.Command {
	var networkSlug, aclSlug string

	cmd := &cobra.Command{
		Use:     "acl-replace",
		Short:   "Replace the ACL on a network",
		Example: `  zcp vpc acl-replace --network <network-slug> --acl <acl-slug>`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if networkSlug == "" {
				return fmt.Errorf("--network is required")
			}
			if aclSlug == "" {
				return fmt.Errorf("--acl is required")
			}
			return runVPCACLReplace(cmd, networkSlug, aclSlug)
		},
	}
	cmd.Flags().StringVar(&networkSlug, "network", "", "Network slug (required)")
	cmd.Flags().StringVar(&aclSlug, "acl", "", "ACL slug (required)")
	return cmd
}

func runVPCACLReplace(cmd *cobra.Command, networkSlug, aclSlug string) error {
	_, client, printer, err := buildClientAndPrinter(cmd)
	if err != nil {
		return err
	}

	svc := vpc.NewService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	req := map[string]string{"aclSlug": aclSlug}
	if err := svc.ReplaceNetworkACL(ctx, networkSlug, req); err != nil {
		return fmt.Errorf("vpc acl-replace: %w", err)
	}

	printer.Fprintf("ACL replaced on network %q.\n", networkSlug)
	return nil
}

// ─── VPC VPN Gateway subcommands ─────────────────────────────────────────────

func newVPCVPNGatewayCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "vpn-gateway",
		Short: "Manage VPN gateways for a VPC",
	}
	cmd.AddCommand(newVPCVPNGatewayListCmd())
	cmd.AddCommand(newVPCVPNGatewayCreateCmd())
	cmd.AddCommand(newVPCVPNGatewayDeleteCmd())
	return cmd
}

func newVPCVPNGatewayListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "list <vpc-slug>",
		Short:   "List VPN gateways for a VPC",
		Args:    cobra.ExactArgs(1),
		Example: `  zcp vpc vpn-gateway list <vpc-slug>`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runVPCVPNGatewayList(cmd, args[0])
		},
	}
	return cmd
}

func runVPCVPNGatewayList(cmd *cobra.Command, vpcSlug string) error {
	_, client, printer, err := buildClientAndPrinter(cmd)
	if err != nil {
		return err
	}

	svc := vpc.NewService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	gateways, err := svc.ListVPNGateways(ctx, vpcSlug)
	if err != nil {
		return fmt.Errorf("vpc vpn-gateway list: %w", err)
	}

	headers := []string{"SLUG", "PUBLIC IP", "VPC SLUG", "STATUS"}
	rows := make([][]string, 0, len(gateways))
	for _, g := range gateways {
		rows = append(rows, []string{
			g.Slug,
			g.PublicIP,
			g.VPCSlug,
			g.Status,
		})
	}
	return printer.PrintTable(headers, rows)
}

func newVPCVPNGatewayCreateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "create <vpc-slug>",
		Short:   "Create a VPN gateway for a VPC",
		Args:    cobra.ExactArgs(1),
		Example: `  zcp vpc vpn-gateway create <vpc-slug>`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runVPCVPNGatewayCreate(cmd, args[0])
		},
	}
	return cmd
}

func runVPCVPNGatewayCreate(cmd *cobra.Command, vpcSlug string) error {
	_, client, printer, err := buildClientAndPrinter(cmd)
	if err != nil {
		return err
	}

	svc := vpc.NewService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	g, err := svc.CreateVPNGateway(ctx, vpcSlug)
	if err != nil {
		return fmt.Errorf("vpc vpn-gateway create: %w", err)
	}

	headers := []string{"FIELD", "VALUE"}
	rows := [][]string{
		{"Slug", g.Slug},
		{"Public IP", g.PublicIP},
		{"VPC Slug", g.VPCSlug},
		{"Zone Name", g.ZoneName},
		{"Status", g.Status},
	}
	return printer.PrintTable(headers, rows)
}

func newVPCVPNGatewayDeleteCmd() *cobra.Command {
	var yes bool

	cmd := &cobra.Command{
		Use:   "delete <vpc-slug> <gateway-id>",
		Short: "Delete a VPN gateway from a VPC",
		Args:  cobra.ExactArgs(2),
		Example: `  zcp vpc vpn-gateway delete <vpc-slug> <gateway-id>
  zcp vpc vpn-gateway delete <vpc-slug> <gateway-id> --yes`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runVPCVPNGatewayDelete(cmd, args[0], args[1], yes)
		},
	}
	cmd.Flags().BoolVar(&yes, "yes", false, "Skip confirmation prompt")
	return cmd
}

func runVPCVPNGatewayDelete(cmd *cobra.Command, vpcSlug, gatewayID string, yes bool) error {
	if !yes && !autoApproved(cmd) {
		fmt.Fprintf(os.Stderr, "Delete VPN gateway %q from VPC %q? This action cannot be undone. [y/N]: ", gatewayID, vpcSlug)
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

	svc := vpc.NewService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	if err := svc.DeleteVPNGateway(ctx, vpcSlug, gatewayID); err != nil {
		return fmt.Errorf("vpc vpn-gateway delete: %w", err)
	}

	printer.Fprintf("VPN gateway %q deleted from VPC %q.\n", gatewayID, vpcSlug)
	return nil
}
