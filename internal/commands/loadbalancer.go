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
	"github.com/zsoftly/zcp-cli/pkg/api/billing"
	"github.com/zsoftly/zcp-cli/pkg/api/ipaddress"
	"github.com/zsoftly/zcp-cli/pkg/api/loadbalancer"
	"github.com/zsoftly/zcp-cli/pkg/httpclient"
)

// lbIPReleaseWait controls the backoff between --release-ip retries. The LB deletion
// is async, so the IP can still be attached (and un-releasable) for a few seconds after
// the cancel request is accepted. Overridden in tests to avoid real sleeps.
var lbIPReleaseWait = func(attempt int) time.Duration {
	return time.Duration(3*(attempt+1)) * time.Second
}

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
		Example: `  zcp loadbalancer create --name my-lb --project default-9 --region yul-1 --network my-network --plan lb-yul --billing-cycle hourly --acquire-new-ip --public-port 80 --private-port 8080 --algorithm roundrobin
  zcp loadbalancer create --name my-lb --project default-9 --region yul-1 --network my-network --plan lb-yul --billing-cycle monthly --ip existing-ip-slug --rule-name web --public-port 443 --private-port 8443 --algorithm leastconn --vm vm-slug-1 --vm vm-slug-2`,
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
	var yes, releaseIP bool
	var billingCycle string

	cmd := &cobra.Command{
		Use:   "delete <lb-slug>",
		Short: "Permanently delete a load balancer",
		Long: `Permanently delete a load balancer.

This submits an immediate service-cancellation request via
POST /billing/service-cancel-requests/{slug} — the same workflow the CMP Web UI
uses. Deletion runs asynchronously in the background: a successful response means
the request was accepted, not that the LB is already gone. Poll with
'zcp loadbalancer list' to confirm removal.

The load balancer's public IP is a separate, reusable resource (as in the Web UI,
you Choose an existing IP or Acquire a new one), so it is NOT released by default —
you may want to reuse it. Pass --release-ip to also release it after deletion. The
network's source-NAT IP is never released (only a dedicated IP the LB holds); and if
you attached other rules such as port-forwarding to that IP, releasing it removes
those too.
--billing-cycle defaults to hourly; pass --billing-cycle monthly for a monthly-billed LB.`,
		Args: exactArgs(1),
		Example: `  zcp loadbalancer delete my-lb
  zcp loadbalancer delete my-lb --yes
  # also release the load balancer's dedicated public IP:
  zcp loadbalancer delete my-lb --yes --release-ip
  zcp loadbalancer delete my-lb --yes --billing-cycle monthly`,
		RunE: func(cmd *cobra.Command, args []string) error {
			slug := args[0]
			if _, ok := billingCycleUnit(billingCycle); !ok {
				return fmt.Errorf("invalid --billing-cycle %q: use hourly or monthly", billingCycle)
			}
			if !yes && !autoApproved(cmd) {
				ipNote := " Its public IP is kept (reusable); pass --release-ip to free it too."
				if releaseIP {
					ipNote = " Its dedicated public IP will also be released (the network source-NAT IP is never released)."
				}
				fmt.Fprintf(os.Stderr, "Delete load balancer %q? This cannot be undone.%s [y/N]: ", slug, ipNote)
				scanner := bufio.NewScanner(os.Stdin)
				scanner.Scan()
				answer := strings.TrimSpace(strings.ToLower(scanner.Text()))
				if answer != "y" && answer != "yes" {
					fmt.Fprintln(os.Stderr, "Aborted.")
					return nil
				}
			}
			return runLBDelete(cmd, slug, billingCycle, releaseIP)
		},
	}
	cmd.Flags().BoolVar(&yes, "yes", false, "Skip confirmation prompt")
	cmd.Flags().StringVar(&billingCycle, "billing-cycle", "hourly", "The load balancer's billing cycle: hourly or monthly")
	cmd.Flags().BoolVar(&releaseIP, "release-ip", false, "Also release the load balancer's public IP after deletion. The network source-NAT IP is never released, only a dedicated IP the LB holds; if you attached other rules (e.g. port-forwarding) to that IP, releasing it removes them too. Off by default because the IP is a reusable resource you may want to keep")
	return cmd
}

func runLBDelete(cmd *cobra.Command, slug, billingCycle string, releaseIP bool) error {
	_, client, printer, err := buildClientAndPrinter(cmd)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()
	region, project := scopedRegionProject(cmd)

	// When --release-ip is set, resolve the LB BEFORE deleting so we can (a) cancel by its
	// canonical slug even if the caller passed a name/id, and (b) find its IP + strategy to
	// release afterward — but only a STATIC IP the LB owns, never the network's source-NAT IP.
	cancelSlug := slug
	var releaseSlug, releaseAddr, ipNote string
	if releaseIP {
		if lb := findLoadBalancer(ctx, client, region, project, slug); lb != nil {
			cancelSlug = lb.Slug
			releaseSlug, releaseAddr, ipNote = lbReleasableIP(ctx, client, region, project, lb)
		} else {
			// Not found in the listing — a wrong ref, another page, or already gone. Don't
			// silently skip: still delete via the raw ref, and tell the user how to free the IP.
			ipNote = fmt.Sprintf("could not find load balancer %q to look up its public IP; if it had a dedicated IP, free it with 'zcp ip release <ip-slug>'", slug)
		}
	}

	// Delete the LB through the unified service-cancellation workflow (matches the Web UI).
	req := billing.CancelServiceRequest{
		ServiceName:  "Load Balancer",
		Reason:       "not_needed_anymore",
		Type:         "Immediate",
		Status:       "Pending",
		BillingCycle: normalizeCancelBillingCycle(billingCycle),
	}
	switch err := billing.NewService(client).CancelService(ctx, cancelSlug, req); {
	case err == nil:
		printer.Fprintf("Deletion requested for %q; the load balancer is being removed in the background.\n", cancelSlug)
	case apierrors.IsResourceNotFound(err):
		// Already deleted (or a stale ref). Fall through to release a resolved dedicated IP,
		// which may now be orphaned.
		fmt.Fprintf(os.Stderr, "Load balancer %q not found — already deleted.\n", cancelSlug)
	default:
		return fmt.Errorf("loadbalancer delete: %w", err)
	}

	if releaseIP {
		if releaseSlug != "" {
			// Give the release its own budget from a fresh context: the LB delete is async and
			// the IP can stay attached for a few seconds, so the retry backoff must not be
			// starved by whatever is left of the command timeout after the list+cancel calls.
			relBudget := time.Duration(getTimeout(cmd)) * time.Second
			if relBudget < 20*time.Second {
				relBudget = 20 * time.Second
			}
			relCtx, relCancel := context.WithTimeout(context.Background(), relBudget)
			defer relCancel()
			releaseLBIP(relCtx, client, printer, releaseSlug, releaseAddr)
		} else if ipNote != "" {
			fmt.Fprintf(os.Stderr, "Public IP not released: %s.\n", ipNote)
		}
	}
	return nil
}

// findLoadBalancer resolves a load balancer by slug, name, or id within the region/project.
// Returns nil if no match is in the listing (a wrong ref, a paginated page we didn't get,
// or already deleted) — the caller decides how to proceed.
func findLoadBalancer(ctx context.Context, client *httpclient.Client, region, project, ref string) *loadbalancer.LoadBalancer {
	lbs, err := loadbalancer.NewService(client).List(ctx, region, project)
	if err != nil {
		return nil
	}
	for i := range lbs {
		if lbs[i].Slug == ref || lbs[i].Name == ref || lbs[i].ID == ref {
			return &lbs[i]
		}
	}
	return nil
}

// lbReleasableIP decides whether the resolved LB's public IP is safe to release. It returns
// (ipSlug, ipAddress, "") when the IP is a releasable IP the LB owns, or ("", "", note) when
// it must be left alone: no IP, a network source-NAT IP, or an IP whose strategy could not be
// confirmed (in which case it is skipped for safety — we never risk releasing a source-NAT —
// and the note tells the user how to release it manually).
func lbReleasableIP(ctx context.Context, client *httpclient.Client, region, project string, lb *loadbalancer.LoadBalancer) (ipSlug, ipAddr, note string) {
	if lb.IPAddress == nil || lb.IPAddress.Slug == "" {
		return "", "", "the load balancer has no dedicated public IP"
	}
	ips, err := ipaddress.NewService(client).List(ctx, "", region, project)
	if err != nil {
		return "", "", fmt.Sprintf("could not look up the IP's strategy (%v) — free it manually with 'zcp ip release %s' if it is a dedicated IP", err, lb.IPAddress.Slug)
	}
	for _, ip := range ips {
		if ip.Slug == lb.IPAddress.Slug {
			if strings.EqualFold(ip.Strategy, "SOURCE-NAT") {
				return "", "", fmt.Sprintf("IP %s is the network source-NAT (shared with the network)", ip.IPAddress)
			}
			return ip.Slug, ip.IPAddress, ""
		}
	}
	// The LB's IP was not in the listing (already gone, another page, or a different scope).
	// We can't confirm it isn't a source-NAT, so skip for safety and hand the user the slug.
	return "", "", fmt.Sprintf("could not confirm IP %s is safe to release; free it manually with 'zcp ip release %s'", lb.IPAddress.IPAddress, lb.IPAddress.Slug)
}

// releaseLBIP releases the load balancer's IP, retrying briefly because the LB deletion
// is async and the IP can stay attached (and un-releasable) for a few seconds.
func releaseLBIP(ctx context.Context, client *httpclient.Client, printer *output.Printer, ipSlug, ipAddr string) {
	ipSvc := ipaddress.NewService(client)
	var err error
	for attempt := 0; attempt < 4; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				err = ctx.Err()
			case <-time.After(lbIPReleaseWait(attempt - 1)):
			}
			if ctx.Err() != nil {
				break
			}
		}
		if err = ipSvc.Release(ctx, ipSlug); err == nil {
			printer.Fprintf("Released public IP %s (%s).\n", ipAddr, ipSlug)
			return
		}
		if apierrors.IsResourceNotFound(err) {
			printer.Fprintf("Public IP %s already released.\n", ipAddr)
			return
		}
	}
	fmt.Fprintf(os.Stderr, "Load balancer deleted, but its public IP %s could not be released yet (%v).\nRelease it once deletion completes: zcp ip release %s\n", ipAddr, err, ipSlug)
}

// billingCycleUnit maps a billing-cycle string (hourly/monthly, or hour/month) to the unit
// form the service-cancellation endpoint expects (hour/month), reporting whether it matched.
func billingCycleUnit(s string) (string, bool) {
	switch low := strings.ToLower(strings.TrimSpace(s)); {
	case strings.HasPrefix(low, "hour"):
		return "hour", true
	case strings.HasPrefix(low, "month"):
		return "month", true
	default:
		return "", false
	}
}

// normalizeCancelBillingCycle maps a billing cycle to the endpoint's unit form, defaulting
// to "month" for anything unrecognized. Callers that pass user input should validate with
// billingCycleUnit first so a typo doesn't silently become "month".
func normalizeCancelBillingCycle(s string) string {
	if u, ok := billingCycleUnit(s); ok {
		return u
	}
	return "month"
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
		Example: `  zcp loadbalancer attach-vm my-lb rule-123 --vm vm-slug-1 --vm vm-slug-2 --region yul-1 --project default-9
  zcp loadbalancer attach-vm my-lb rule-123 --vm vm-slug-1 --region yul-1 --project default-9 --yes`,
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
