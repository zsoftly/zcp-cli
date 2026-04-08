package commands

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/zsoftly/zcp-cli/internal/api/loadbalancer"
)

var validLBAlgorithms = map[string]bool{
	"roundrobin": true,
	"leastconn":  true,
	"source":     true,
}

// NewLoadBalancerCmd returns the 'loadbalancer' cobra command.
func NewLoadBalancerCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "loadbalancer",
		Short: "Manage load balancers",
	}
	cmd.AddCommand(newLBListCmd())
	cmd.AddCommand(newLBCreateCmd())
	cmd.AddCommand(newLBCreateRuleCmd())
	cmd.AddCommand(newLBAttachVMCmd())
	return cmd
}

func newLBListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List load balancers",
		Example: `  zcp loadbalancer list
  zcp loadbalancer list -o json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runLBList(cmd)
		},
	}
	return cmd
}

func runLBList(cmd *cobra.Command) error {
	_, client, printer, err := buildClientAndPrinter(cmd)
	if err != nil {
		return err
	}

	svc := loadbalancer.NewService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	lbs, err := svc.List(ctx)
	if err != nil {
		return fmt.Errorf("loadbalancer list: %w", err)
	}

	headers := []string{"SLUG", "NAME", "STATE", "IP", "REGION", "PROJECT", "CREATED"}
	rows := make([][]string, 0, len(lbs))
	for _, lb := range lbs {
		ip := ""
		if lb.IPAddress != nil {
			ip = lb.IPAddress.IPAddress
		}
		region := ""
		if lb.Region != nil {
			region = lb.Region.Name
		}
		project := ""
		if lb.Project != nil {
			project = lb.Project.Name
		}
		rows = append(rows, []string{
			lb.Slug,
			lb.Name,
			lb.State,
			ip,
			region,
			project,
			lb.CreatedAt,
		})
	}
	return printer.PrintTable(headers, rows)
}

func newLBCreateCmd() *cobra.Command {
	var (
		name          string
		cloudProvider string
		project       string
		region        string
		network       string
		plan          string
		billingCycle  string
		acquireNewIP  bool
		ipAddress     string
	)

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new load balancer",
		Example: `  zcp loadbalancer create --name my-lb --cloud-provider nimbo --project default-33 --region ixg-belagavi --network d-net-test --plan load-balancer --billing-cycle hourly --acquire-new-ip
  zcp loadbalancer create --name my-lb --cloud-provider nimbo --project default-33 --region ixg-belagavi --network d-net-test --plan load-balancer --billing-cycle monthly --ip existing-ip-slug`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if name == "" {
				return fmt.Errorf("--name is required")
			}
			if cloudProvider == "" {
				return fmt.Errorf("--cloud-provider is required")
			}
			if project == "" {
				return fmt.Errorf("--project is required")
			}
			if region == "" {
				return fmt.Errorf("--region is required")
			}
			if network == "" {
				return fmt.Errorf("--network is required")
			}
			if plan == "" {
				plan = "load-balancer"
			}
			if billingCycle == "" {
				return fmt.Errorf("--billing-cycle is required")
			}

			req := loadbalancer.CreateRequest{
				Name:          name,
				CloudProvider: cloudProvider,
				Project:       project,
				Region:        region,
				Network:       network,
				Plan:          plan,
				BillingCycle:  billingCycle,
				AcquireNewIP:  acquireNewIP,
				Rules:         []loadbalancer.CreateRuleSpec{},
			}
			if ipAddress != "" {
				req.IPAddress = &ipAddress
				req.AcquireNewIP = false
			}

			return runLBCreate(cmd, req)
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "Load balancer name (required)")
	cmd.Flags().StringVar(&cloudProvider, "cloud-provider", "", "Cloud provider slug (required)")
	cmd.Flags().StringVar(&project, "project", "", "Project slug (required)")
	cmd.Flags().StringVar(&region, "region", "", "Region slug (required)")
	cmd.Flags().StringVar(&network, "network", "", "Network slug (required)")
	cmd.Flags().StringVar(&plan, "plan", "load-balancer", "Plan slug")
	cmd.Flags().StringVar(&billingCycle, "billing-cycle", "", "Billing cycle: hourly, monthly, quarterly, yearly (required)")
	cmd.Flags().BoolVar(&acquireNewIP, "acquire-new-ip", true, "Acquire a new public IP for the load balancer")
	cmd.Flags().StringVar(&ipAddress, "ip", "", "Existing IP address slug (overrides --acquire-new-ip)")
	return cmd
}

func runLBCreate(cmd *cobra.Command, req loadbalancer.CreateRequest) error {
	_, client, printer, err := buildClientAndPrinter(cmd)
	if err != nil {
		return err
	}

	svc := loadbalancer.NewService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	lb, err := svc.Create(ctx, req)
	if err != nil {
		return fmt.Errorf("loadbalancer create: %w", err)
	}

	headers := []string{"FIELD", "VALUE"}
	ip := ""
	if lb.IPAddress != nil {
		ip = lb.IPAddress.IPAddress
	}
	rows := [][]string{
		{"Slug", lb.Slug},
		{"Name", lb.Name},
		{"State", lb.State},
		{"IP", ip},
		{"Created", lb.CreatedAt},
	}
	return printer.PrintTable(headers, rows)
}

func newLBCreateRuleCmd() *cobra.Command {
	var (
		name                string
		publicPort          string
		privatePort         string
		protocol            string
		algorithm           string
		stickyMethod        string
		enableTLS           bool
		enableProxyProtocol bool
		vmSlugs             []string
	)

	cmd := &cobra.Command{
		Use:   "create-rule <lb-slug>",
		Short: "Create a rule on an existing load balancer",
		Args:  cobra.ExactArgs(1),
		Example: `  zcp loadbalancer create-rule my-lb --name web-rule --public-port 80 --private-port 8080 --protocol tcp --algorithm roundrobin
  zcp loadbalancer create-rule my-lb --name ssl-rule --public-port 443 --private-port 8443 --protocol tcp --algorithm leastconn --vm vm-slug-1 --vm vm-slug-2`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if name == "" {
				return fmt.Errorf("--name is required")
			}
			if publicPort == "" {
				return fmt.Errorf("--public-port is required")
			}
			if privatePort == "" {
				return fmt.Errorf("--private-port is required")
			}
			if algorithm == "" {
				return fmt.Errorf("--algorithm is required")
			}
			if !validLBAlgorithms[algorithm] {
				return fmt.Errorf("--algorithm must be one of: roundrobin, leastconn, source")
			}
			if protocol == "" {
				protocol = "tcp"
			}

			vms := make([]loadbalancer.VMAttachment, 0, len(vmSlugs))
			for _, slug := range vmSlugs {
				vms = append(vms, loadbalancer.VMAttachment{Slug: slug})
			}

			rule := loadbalancer.CreateRuleSpec{
				Name:                name,
				PublicPort:          publicPort,
				PrivatePort:         privatePort,
				Protocol:            protocol,
				Algorithm:           algorithm,
				StickyMethod:        stickyMethod,
				EnableTLSProtocol:   enableTLS,
				EnableProxyProtocol: enableProxyProtocol,
				VirtualMachines:     vms,
			}

			return runLBCreateRule(cmd, args[0], loadbalancer.CreateRuleRequest{Rules: []loadbalancer.CreateRuleSpec{rule}})
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "Rule name (required)")
	cmd.Flags().StringVar(&publicPort, "public-port", "", "Public port (required)")
	cmd.Flags().StringVar(&privatePort, "private-port", "", "Private port (required)")
	cmd.Flags().StringVar(&protocol, "protocol", "tcp", "Protocol (e.g. tcp, udp)")
	cmd.Flags().StringVar(&algorithm, "algorithm", "", "Algorithm: roundrobin, leastconn, or source (required)")
	cmd.Flags().StringVar(&stickyMethod, "sticky-method", "", "Sticky session method (e.g. AppCookie)")
	cmd.Flags().BoolVar(&enableTLS, "enable-tls", false, "Enable TLS protocol")
	cmd.Flags().BoolVar(&enableProxyProtocol, "enable-proxy-protocol", false, "Enable proxy protocol")
	cmd.Flags().StringArrayVar(&vmSlugs, "vm", nil, "VM slug to attach (can be repeated)")
	return cmd
}

func runLBCreateRule(cmd *cobra.Command, lbSlug string, req loadbalancer.CreateRuleRequest) error {
	_, client, printer, err := buildClientAndPrinter(cmd)
	if err != nil {
		return err
	}

	svc := loadbalancer.NewService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	if err := svc.CreateRule(ctx, lbSlug, req); err != nil {
		return fmt.Errorf("loadbalancer create-rule: %w", err)
	}

	printer.Fprintf("Rule created on load balancer %q.\n", lbSlug)
	return nil
}

func newLBAttachVMCmd() *cobra.Command {
	var (
		cloudProvider string
		region        string
		project       string
		vmSlugs       []string
	)

	cmd := &cobra.Command{
		Use:   "attach-vm <lb-slug> <rule-id>",
		Short: "Attach VMs to a load balancer rule",
		Args:  cobra.ExactArgs(2),
		Example: `  zcp loadbalancer attach-vm my-lb rule-123 --vm vm-slug-1 --vm vm-slug-2 --cloud-provider nimbo --region ixg-belagavi --project default-33
  zcp loadbalancer attach-vm my-lb rule-123 --vm vm-slug-1 --cloud-provider nimbo --region ixg-belagavi --project default-33 --yes`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(vmSlugs) == 0 {
				return fmt.Errorf("at least one --vm is required")
			}
			if cloudProvider == "" {
				return fmt.Errorf("--cloud-provider is required")
			}
			if region == "" {
				return fmt.Errorf("--region is required")
			}
			if project == "" {
				return fmt.Errorf("--project is required")
			}

			yes, _ := cmd.Flags().GetBool("yes")
			if !yes && !autoApproved(cmd) {
				fmt.Fprintf(os.Stderr, "Attach %d VM(s) to rule %q on load balancer %q? [y/N]: ", len(vmSlugs), args[1], args[0])
				scanner := bufio.NewScanner(os.Stdin)
				scanner.Scan()
				answer := strings.TrimSpace(strings.ToLower(scanner.Text()))
				if answer != "y" && answer != "yes" {
					fmt.Fprintln(os.Stderr, "Aborted.")
					return nil
				}
			}

			req := loadbalancer.AttachVMRequest{
				VirtualMachines: vmSlugs,
				CloudProvider:   cloudProvider,
				Region:          region,
				Project:         project,
			}

			return runLBAttachVM(cmd, args[0], args[1], req)
		},
	}
	cmd.Flags().StringArrayVar(&vmSlugs, "vm", nil, "VM slug to attach (can be repeated, required)")
	cmd.Flags().StringVar(&cloudProvider, "cloud-provider", "", "Cloud provider slug (required)")
	cmd.Flags().StringVar(&region, "region", "", "Region slug (required)")
	cmd.Flags().StringVar(&project, "project", "", "Project slug (required)")
	cmd.Flags().Bool("yes", false, "Skip confirmation prompt")
	return cmd
}

func runLBAttachVM(cmd *cobra.Command, lbSlug, ruleID string, req loadbalancer.AttachVMRequest) error {
	_, client, printer, err := buildClientAndPrinter(cmd)
	if err != nil {
		return err
	}

	svc := loadbalancer.NewService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	if err := svc.AttachVM(ctx, lbSlug, ruleID, req); err != nil {
		return fmt.Errorf("loadbalancer attach-vm: %w", err)
	}

	printer.Fprintf("Attached %d VM(s) to rule %q on load balancer %q.\n", len(req.VirtualMachines), ruleID, lbSlug)
	return nil
}
