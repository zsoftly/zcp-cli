package commands

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/zsoftly/zcp-cli/internal/output"
	"github.com/zsoftly/zcp-cli/pkg/api/apierrors"
	"github.com/zsoftly/zcp-cli/pkg/api/loadbalancer"
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
	cmd.AddCommand(newLBDeleteCmd())
	cmd.AddCommand(newLBCreateRuleCmd())
	cmd.AddCommand(newLBDeleteRuleCmd())
	cmd.AddCommand(newLBAttachVMCmd())
	cmd.AddCommand(newLBDetachVMCmd())
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

	region, project := scopedRegionProject(cmd)
	lbs, err := svc.List(ctx, region, project)
	if err != nil {
		return fmt.Errorf("loadbalancer list: %w", err)
	}
	if printer.Format() == output.FormatJSON || printer.Format() == output.FormatYAML {
		return printer.Print(lbs)
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
		ruleName      string
		publicPort    string
		privatePort   string
		protocol      string
		algorithm     string
		stickyMethod  string
		enableTLS     bool
		enableProxy   bool
		vmSlugs       []string
	)

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new load balancer",
		Example: `  zcp loadbalancer create --name my-lb --project default --region yul-1 --network my-network --plan lb-yul --billing-cycle hourly --acquire-new-ip --public-port 80 --private-port 8080 --algorithm roundrobin
  zcp loadbalancer create --name my-lb --project default --region yul-1 --network my-network --plan lb-yul --billing-cycle monthly --ip existing-ip-slug --rule-name web --public-port 443 --private-port 8443 --algorithm leastconn --vm vm-slug-1 --vm vm-slug-2`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if name == "" {
				return fmt.Errorf("--name is required")
			}
			cloudProvider = resolveCloudProvider(cmd, cloudProvider)
			if cloudProvider == "" {
				return fmt.Errorf("could not determine cloud provider — run 'zcp auth validate' to detect it, or pass --cloud-provider (see 'zcp cloud-provider list')")
			}
			project = resolveProject(project)
			if project == "" {
				return fmt.Errorf("--project is required")
			}
			region = resolveRegion(region)
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
			if publicPort == "" {
				return fmt.Errorf("--public-port is required because the API requires an initial load balancer rule")
			}
			if privatePort == "" {
				return fmt.Errorf("--private-port is required because the API requires an initial load balancer rule")
			}
			if algorithm == "" {
				return fmt.Errorf("--algorithm is required because the API requires an initial load balancer rule")
			}
			if !validLBAlgorithms[algorithm] {
				return fmt.Errorf("--algorithm must be one of: roundrobin, leastconn, source")
			}
			// protocol defaults to "tcp" via the flag definition below.
			if ruleName == "" {
				ruleName = name + "-rule"
			}

			vms := make([]loadbalancer.VMAttachment, 0, len(vmSlugs))
			for _, slug := range vmSlugs {
				vms = append(vms, loadbalancer.VMAttachment{Slug: slug})
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
				Rules: []loadbalancer.CreateRuleSpec{
					{
						Name:                ruleName,
						PublicPort:          publicPort,
						PrivatePort:         privatePort,
						Protocol:            protocol,
						Algorithm:           algorithm,
						StickyMethod:        stickyMethod,
						EnableTLSProtocol:   enableTLS,
						EnableProxyProtocol: enableProxy,
						VirtualMachines:     vms,
					},
				},
			}
			if ipAddress != "" {
				req.IPAddress = &ipAddress
				req.AcquireNewIP = false
			}

			return runLBCreate(cmd, req)
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "Load balancer name (required)")
	cmd.Flags().StringVar(&cloudProvider, "cloud-provider", "", "Cloud provider slug (optional; auto-detected, override only)")
	cmd.Flags().StringVar(&project, "project", "", "Project slug (required)")
	cmd.Flags().StringVar(&region, "region", "", "Region slug (required)")
	cmd.Flags().StringVar(&network, "network", "", "Network slug (required)")
	cmd.Flags().StringVar(&plan, "plan", "load-balancer", "Plan slug")
	cmd.Flags().StringVar(&billingCycle, "billing-cycle", "", "Billing cycle: hourly, monthly, quarterly, yearly (required)")
	cmd.Flags().BoolVar(&acquireNewIP, "acquire-new-ip", true, "Acquire a new public IP for the load balancer")
	cmd.Flags().StringVar(&ipAddress, "ip", "", "Existing IP address slug (overrides --acquire-new-ip)")
	cmd.Flags().StringVar(&ruleName, "rule-name", "", "Initial rule name (defaults to <name>-rule)")
	cmd.Flags().StringVar(&publicPort, "public-port", "", "Initial rule public port (required)")
	cmd.Flags().StringVar(&privatePort, "private-port", "", "Initial rule private port (required)")
	cmd.Flags().StringVar(&protocol, "protocol", "tcp", "Initial rule protocol")
	cmd.Flags().StringVar(&algorithm, "algorithm", "", "Initial rule algorithm: roundrobin, leastconn, or source (required)")
	cmd.Flags().StringVar(&stickyMethod, "sticky-method", "", "Initial rule sticky session method")
	cmd.Flags().BoolVar(&enableTLS, "enable-tls", false, "Enable TLS protocol on the initial rule")
	cmd.Flags().BoolVar(&enableProxy, "enable-proxy-protocol", false, "Enable proxy protocol on the initial rule")
	cmd.Flags().StringArrayVar(&vmSlugs, "vm", nil, "VM slug to attach to the initial rule (can be repeated)")
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

func newLBDeleteCmd() *cobra.Command {
	var yes bool

	cmd := &cobra.Command{
		Use:   "delete <lb-slug>",
		Short: "Permanently delete a load balancer",
		Args:  exactArgs(1),
		Example: `  zcp loadbalancer delete my-lb
  zcp loadbalancer delete my-lb --yes`,
		RunE: func(cmd *cobra.Command, args []string) error {
			slug := args[0]
			if !yes && !autoApproved(cmd) {
				fmt.Fprintf(os.Stderr, "Delete load balancer %q? This cannot be undone. [y/N]: ", slug)
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
			svc := loadbalancer.NewService(client)
			ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
			defer cancel()
			if err := svc.Delete(ctx, slug); err != nil {
				if apierrors.IsResourceNotFound(err) {
					fmt.Fprintf(os.Stderr, "Load balancer %q not found — already deleted.\n", slug)
					return nil
				}
				return fmt.Errorf("loadbalancer delete: %w", err)
			}
			printer.Fprintf("Load balancer %q deleted.\n", slug)
			return nil
		},
	}
	cmd.Flags().BoolVar(&yes, "yes", false, "Skip confirmation prompt")
	return cmd
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
		Args:  exactArgs(1),
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

func newLBDeleteRuleCmd() *cobra.Command {
	var yes bool

	cmd := &cobra.Command{
		Use:   "delete-rule <lb-slug> <rule-id>",
		Short: "Delete a rule from a load balancer",
		Args:  exactArgs(2),
		Example: `  zcp loadbalancer delete-rule my-lb rule-123
  zcp loadbalancer delete-rule my-lb rule-123 --yes`,
		RunE: func(cmd *cobra.Command, args []string) error {
			lbSlug, ruleID := args[0], args[1]
			if !yes && !autoApproved(cmd) {
				fmt.Fprintf(os.Stderr, "Delete rule %q from load balancer %q? [y/N]: ", ruleID, lbSlug)
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
			svc := loadbalancer.NewService(client)
			ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
			defer cancel()
			if err := svc.DeleteRule(ctx, lbSlug, ruleID); err != nil {
				if apierrors.IsResourceNotFound(err) {
					fmt.Fprintf(os.Stderr, "Load balancer rule %q not found — already deleted.\n", ruleID)
					return nil
				}
				return fmt.Errorf("loadbalancer delete-rule: %w", err)
			}
			printer.Fprintf("Rule %q deleted from load balancer %q.\n", ruleID, lbSlug)
			return nil
		},
	}
	cmd.Flags().BoolVar(&yes, "yes", false, "Skip confirmation prompt")
	return cmd
}

func newLBDetachVMCmd() *cobra.Command {
	var vmSlug string
	var yes bool

	cmd := &cobra.Command{
		Use:   "detach-vm <lb-slug> <rule-id>",
		Short: "Detach a VM from a load balancer rule",
		Args:  exactArgs(2),
		Example: `  zcp loadbalancer detach-vm my-lb rule-123 --vm vm-slug-1
  zcp loadbalancer detach-vm my-lb rule-123 --vm vm-slug-1 --yes`,
		RunE: func(cmd *cobra.Command, args []string) error {
			lbSlug, ruleID := args[0], args[1]
			if vmSlug == "" {
				return fmt.Errorf("--vm is required")
			}
			if !yes && !autoApproved(cmd) {
				fmt.Fprintf(os.Stderr, "Detach VM %q from rule %q on load balancer %q? [y/N]: ", vmSlug, ruleID, lbSlug)
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
			svc := loadbalancer.NewService(client)
			ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
			defer cancel()
			if err := svc.DetachVM(ctx, lbSlug, ruleID, vmSlug); err != nil {
				if apierrors.IsResourceNotFound(err) {
					fmt.Fprintf(os.Stderr, "VM %q not found on load balancer rule — already detached.\n", vmSlug)
					return nil
				}
				return fmt.Errorf("loadbalancer detach-vm: %w", err)
			}
			printer.Fprintf("VM %q detached from rule %q on load balancer %q.\n", vmSlug, ruleID, lbSlug)
			return nil
		},
	}
	cmd.Flags().StringVar(&vmSlug, "vm", "", "VM slug to detach (required)")
	cmd.Flags().BoolVar(&yes, "yes", false, "Skip confirmation prompt")
	return cmd
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
		Args:  exactArgs(2),
		Example: `  zcp loadbalancer attach-vm my-lb rule-123 --vm vm-slug-1 --vm vm-slug-2 --region yul-1 --project default
  zcp loadbalancer attach-vm my-lb rule-123 --vm vm-slug-1 --region yul-1 --project default --yes`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(vmSlugs) == 0 {
				return fmt.Errorf("at least one --vm is required")
			}
			cloudProvider = resolveCloudProvider(cmd, cloudProvider)
			if cloudProvider == "" {
				return fmt.Errorf("could not determine cloud provider — run 'zcp auth validate' to detect it, or pass --cloud-provider (see 'zcp cloud-provider list')")
			}
			region = resolveRegion(region)
			if region == "" {
				return fmt.Errorf("--region is required")
			}
			project = resolveProject(project)
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
	cmd.Flags().StringVar(&cloudProvider, "cloud-provider", "", "Cloud provider slug (optional; auto-detected, override only)")
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
