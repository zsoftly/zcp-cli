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
	"github.com/zsoftly/zcp-cli/pkg/api/acl"
	"github.com/zsoftly/zcp-cli/pkg/api/apierrors"
	"github.com/zsoftly/zcp-cli/pkg/api/network"
)

// NewNetworkCmd returns the 'network' cobra command.
func NewNetworkCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "network",
		Short: "Manage networks (isolated, L2, and VPC subnets)",
	}
	cmd.AddCommand(newNetworkListCmd())
	cmd.AddCommand(newNetworkGetCmd())
	cmd.AddCommand(newNetworkCreateCmd())
	cmd.AddCommand(newNetworkUpdateCmd())
	cmd.AddCommand(newNetworkCategoriesCmd())
	cmd.AddCommand(newNetworkDeleteCmd())
	return cmd
}

func newNetworkListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List networks",
		Example: `  zcp network list
  zcp network list --output json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runNetworkList(cmd)
		},
	}
	return cmd
}

func runNetworkList(cmd *cobra.Command) error {
	_, client, printer, err := buildClientAndPrinter(cmd)
	if err != nil {
		return err
	}

	svc := network.NewService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	nets, err := svc.List(ctx)
	if err != nil {
		return fmt.Errorf("network list: %w", err)
	}

	headers := []string{"SLUG", "NAME", "TYPE", "CIDR", "GATEWAY", "STATUS", "ZONE"}
	rows := make([][]string, 0, len(nets))
	for _, n := range nets {
		rows = append(rows, []string{
			n.Slug,
			n.Name,
			n.NetworkType,
			n.CIDR,
			n.Gateway,
			strconv.FormatBool(n.Status),
			n.ZoneSlug,
		})
	}
	return printer.PrintTable(headers, rows)
}

func newNetworkGetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get <slug>",
		Short: "Get details of a network",
		Args:  cobra.ExactArgs(1),
		Example: `  zcp network get web-tier
  zcp network get web-tier --output json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runNetworkGet(cmd, args[0])
		},
	}
	return cmd
}

func runNetworkGet(cmd *cobra.Command, slug string) error {
	_, client, printer, err := buildClientAndPrinter(cmd)
	if err != nil {
		return err
	}

	svc := network.NewService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	d, err := svc.GetDetail(ctx, slug)
	if err != nil {
		return fmt.Errorf("network get: %w", err)
	}

	headers := []string{"FIELD", "VALUE"}
	rows := [][]string{
		{"Slug", d.Slug},
		{"Name", d.Name},
		{"Type", d.Meta.Type},
		{"CIDR", d.Meta.CIDR},
		{"Gateway", d.Meta.Gateway},
		{"Netmask", d.Meta.Netmask},
		{"State", d.Meta.State},
		{"VPC ID", d.Meta.VPCID},
		{"ACL", d.Meta.ACLName},
		{"Zone", d.Meta.ZoneName},
	}
	return printer.PrintTable(headers, rows)
}

func newNetworkCreateCmd() *cobra.Command {
	var name, categorySlug, zoneSlug, gateway, netmask, description string
	var cloudProvider, region, project string
	var vpcSlug, billingCycle, networkPlan, netType, aclRef string

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create an isolated network or a VPC subnet",
		Long: `Create an isolated network, an L2 network, or a VPC subnet (tier).

For VPC subnets, --acl attaches a custom network ACL right after creation
(the API has no attach-at-create parameter, so the network briefly carries
the VPC default ACL before the replacement is applied).`,
		Example: `  zcp network create --name my-net --network-plan inet-yow --billing-cycle hourly --cloud-provider nimbo --region yow-1 --project default
  zcp network create --name my-l2 --network-plan l2net-yow --type L2 --cloud-provider nimbo --region yow-1 --project default
  zcp network create --name web-tier --vpc my-vpc --gateway 10.1.1.1 --netmask 255.255.255.0 --billing-cycle hourly --cloud-provider nimbo --region yow-1 --project default
  zcp network create --name web-tier --vpc my-vpc --acl web-acl --gateway 10.1.1.1 --netmask 255.255.255.0 --billing-cycle hourly --cloud-provider nimbo --region yow-1 --project default`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if name == "" {
				return fmt.Errorf("--name is required")
			}
			cloudProvider = resolveCloudProvider(cloudProvider)
			if cloudProvider == "" {
				return fmt.Errorf("--cloud-provider is required")
			}
			region = resolveRegion(region)
			if region == "" {
				return fmt.Errorf("--region is required")
			}
			project = resolveProject(project)
			if project == "" {
				return fmt.Errorf("--project is required")
			}

			if vpcSlug != "" {
				// VPC subnet (tier). The API requires type "Vpc" (exact case):
				// any other value is silently ignored and the network is
				// created detached from the VPC.
				if netType != "" && netType != "Vpc" {
					return fmt.Errorf("--type cannot be combined with --vpc (VPC subnets are always type Vpc)")
				}
				if gateway == "" || netmask == "" {
					return fmt.Errorf("--gateway and --netmask are required when --vpc is set")
				}
				if billingCycle == "" {
					return fmt.Errorf("--billing-cycle is required when --vpc is set")
				}
				if networkPlan != "" {
					return fmt.Errorf("--network-plan cannot be combined with --vpc (the tier inherits the VPC's network offering)")
				}
				netType = "Vpc"
			} else {
				if aclRef != "" {
					return fmt.Errorf("--acl requires --vpc (network ACLs only apply to VPC subnets)")
				}
				switch {
				case netType == "" || strings.EqualFold(netType, "isolated"):
					netType = "Isolated"
				case strings.EqualFold(netType, "l2"):
					netType = "L2"
				default:
					return fmt.Errorf("--type must be Isolated or L2 (got %q); use --vpc for VPC subnets", netType)
				}
				if networkPlan == "" {
					return fmt.Errorf("--network-plan is required (see: zcp plan network)")
				}
			}

			return runNetworkCreate(cmd, network.CreateRequest{
				Name:          name,
				CategorySlug:  categorySlug,
				ZoneSlug:      zoneSlug,
				Gateway:       gateway,
				Netmask:       netmask,
				Description:   description,
				CloudProvider: cloudProvider,
				Region:        region,
				Project:       project,
				VPC:           vpcSlug,
				BillingCycle:  billingCycle,
				Type:          netType,
				NetworkPlan:   networkPlan,
			}, aclRef)
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "Network name (required)")
	cmd.Flags().StringVar(&aclRef, "acl", "", "Network ACL to attach after creation (VPC subnets only, see: zcp acl list <vpc>)")
	cmd.Flags().StringVar(&networkPlan, "network-plan", "", "Network plan slug (required for isolated/L2 networks, see: zcp plan network)")
	cmd.Flags().StringVar(&vpcSlug, "vpc", "", "VPC slug to create this network in as a subnet (tier)")
	cmd.Flags().StringVar(&billingCycle, "billing-cycle", "", "Billing cycle: hourly, monthly (required with --vpc)")
	cmd.Flags().StringVar(&netType, "type", "", "Network type: Isolated (default) or L2")
	cmd.Flags().StringVar(&categorySlug, "category", "", "Network category slug (legacy, optional)")
	cmd.Flags().StringVar(&zoneSlug, "zone", "", "Zone slug")
	cmd.Flags().StringVar(&gateway, "gateway", "", "Gateway IP (required with --vpc)")
	cmd.Flags().StringVar(&netmask, "netmask", "", "Netmask, e.g. 255.255.255.0 (required with --vpc)")
	cmd.Flags().StringVar(&description, "description", "", "Network description")
	cmd.Flags().StringVar(&cloudProvider, "cloud-provider", "", "Cloud provider slug (required)")
	cmd.Flags().StringVar(&region, "region", "", "Region slug (required)")
	cmd.Flags().StringVar(&project, "project", "", "Project slug (required)")
	return cmd
}

func runNetworkCreate(cmd *cobra.Command, req network.CreateRequest, aclRef string) error {
	_, client, printer, err := buildClientAndPrinter(cmd)
	if err != nil {
		return err
	}

	svc := network.NewService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	n, err := svc.Create(ctx, req)
	if err != nil {
		return fmt.Errorf("network create: %w", err)
	}

	if aclRef != "" {
		aclSvc := acl.NewService(client)
		aclID, rerr := aclSvc.Resolve(ctx, req.VPC, aclRef)
		if rerr == nil {
			rerr = aclSvc.ReplaceNetworkACL(ctx, n.Slug, aclID)
		}
		if rerr != nil {
			return fmt.Errorf("network %q was created, but attaching ACL %q failed: %w (attach manually with: zcp acl replace --network %s --acl %s --vpc %s)",
				n.Slug, aclRef, rerr, n.Slug, aclRef, req.VPC)
		}
	}

	// The create response is sparse (no CIDR/gateway/state) — fetch the
	// provider-side detail, falling back to what we sent if it isn't
	// available yet.
	netType := n.NetworkType
	if netType == "" {
		netType = req.Type
	}
	cidr, gateway, netmask, state, aclName := n.CIDR, req.Gateway, req.Netmask, "", aclRef
	if d, derr := svc.GetDetail(ctx, n.Slug); derr == nil {
		if d.Meta.CIDR != "" {
			cidr = d.Meta.CIDR
		}
		if d.Meta.Gateway != "" {
			gateway = d.Meta.Gateway
		}
		if d.Meta.Netmask != "" {
			netmask = d.Meta.Netmask
		}
		if d.Meta.ACLName != "" {
			aclName = d.Meta.ACLName
		}
		state = d.Meta.State
	}

	headers := []string{"FIELD", "VALUE"}
	rows := [][]string{
		{"Slug", n.Slug},
		{"Name", n.Name},
		{"Type", netType},
		{"VPC", req.VPC},
		{"CIDR", cidr},
		{"Gateway", gateway},
		{"Netmask", netmask},
		{"ACL", aclName},
		{"State", state},
	}
	return printer.PrintTable(headers, rows)
}

func newNetworkUpdateCmd() *cobra.Command {
	var name, description string

	cmd := &cobra.Command{
		Use:   "update <slug>",
		Short: "Update a network",
		Args:  cobra.ExactArgs(1),
		Example: `  zcp network update en-001001-0018 --name new-name
  zcp network update en-001001-0018 --description "Updated description"`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if name == "" && !cmd.Flags().Changed("description") {
				return fmt.Errorf("at least one of --name or --description is required")
			}
			req := network.UpdateRequest{Name: name}
			if cmd.Flags().Changed("description") {
				req.Description = &description
			}
			return runNetworkUpdate(cmd, args[0], req)
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "New network name")
	cmd.Flags().StringVar(&description, "description", "", "New description")
	return cmd
}

func runNetworkUpdate(cmd *cobra.Command, slug string, req network.UpdateRequest) error {
	_, client, printer, err := buildClientAndPrinter(cmd)
	if err != nil {
		return err
	}

	svc := network.NewService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	n, err := svc.Update(ctx, slug, req)
	if err != nil {
		return fmt.Errorf("network update: %w", err)
	}

	headers := []string{"FIELD", "VALUE"}
	rows := [][]string{
		{"Slug", n.Slug},
		{"Name", n.Name},
		{"Type", n.NetworkType},
		{"CIDR", n.CIDR},
		{"Gateway", n.Gateway},
		{"Status", strconv.FormatBool(n.Status)},
	}
	return printer.PrintTable(headers, rows)
}

func newNetworkCategoriesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "categories",
		Short: "List network categories",
		Long: `List network categories (offerings).

Note: the live ZCP API returns an empty list here. Network creation is
driven by network plans instead — see "zcp plan network".`,
		Example: `  zcp network categories
  zcp network categories --output json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runNetworkCategories(cmd)
		},
	}
	return cmd
}

func runNetworkCategories(cmd *cobra.Command) error {
	_, client, printer, err := buildClientAndPrinter(cmd)
	if err != nil {
		return err
	}

	svc := network.NewService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	cats, err := svc.ListCategories(ctx)
	if err != nil {
		return fmt.Errorf("network categories: %w", err)
	}

	headers := []string{"SLUG", "NAME", "DESCRIPTION"}
	rows := make([][]string, 0, len(cats))
	for _, c := range cats {
		rows = append(rows, []string{
			c.Slug,
			c.Name,
			c.Description,
		})
	}
	return printer.PrintTable(headers, rows)
}

func newNetworkDeleteCmd() *cobra.Command {
	var yes bool

	cmd := &cobra.Command{
		Use:   "delete <slug>",
		Short: "Delete a network",
		Args:  cobra.ExactArgs(1),
		Example: `  zcp network delete en-001001-0018
  zcp network delete en-001001-0018 --yes`,
		RunE: func(cmd *cobra.Command, args []string) error {
			slug := args[0]
			if !yes && !autoApproved(cmd) {
				fmt.Fprintf(os.Stderr, "Delete network %q? Its SOURCE-NAT IP will also be released. [y/N]: ", slug)
				scanner := bufio.NewScanner(os.Stdin)
				scanner.Scan()
				answer := strings.TrimSpace(strings.ToLower(scanner.Text()))
				if answer != "y" && answer != "yes" {
					fmt.Fprintln(os.Stderr, "Aborted.")
					return nil
				}
			}
			_, client, _, err := buildClientAndPrinter(cmd)
			if err != nil {
				return err
			}
			svc := network.NewService(client)
			ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
			defer cancel()
			if err := svc.Delete(ctx, slug); err != nil {
				if apierrors.IsResourceNotFound(err) {
					fmt.Fprintf(os.Stderr, "Network %q not found — already deleted.\n", slug)
					return nil
				}
				return fmt.Errorf("network delete: %w", err)
			}
			fmt.Fprintf(os.Stdout, "Network %q deleted.\n", slug)
			return nil
		},
	}
	cmd.Flags().BoolVar(&yes, "yes", false, "Skip confirmation prompt")
	return cmd
}
