package commands

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/zsoftly/zcp-cli/internal/output"
	"github.com/zsoftly/zcp-cli/pkg/api/apierrors"
	"github.com/zsoftly/zcp-cli/pkg/api/billing"
	"github.com/zsoftly/zcp-cli/pkg/api/instance"
)

// instanceGetRetryWait controls the backoff between transient-routing-error retries.
// Overridden in tests to avoid real sleeps.
var instanceGetRetryWait = func(attempt int) time.Duration {
	return time.Duration(2<<uint(attempt)) * time.Second
}

// dashIfEmpty renders an empty value as "-" for table output readability. The SDK
// accessors return "" for "no value" so callers like `instance ssh` can detect it;
// the placeholder is applied only where we display.
func dashIfEmpty(s string) string {
	if s == "" {
		return "-"
	}
	return s
}

// NewInstanceCmd returns the 'instance' cobra command.
func NewInstanceCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "instance",
		Short: "Manage virtual machine instances",
	}
	cmd.AddCommand(newInstanceListCmd())
	cmd.AddCommand(newInstanceGetCmd())
	cmd.AddCommand(newInstanceCreateCmd())
	cmd.AddCommand(newInstanceStartCmd())
	cmd.AddCommand(newInstanceStopCmd())
	cmd.AddCommand(newInstanceRebootCmd())
	cmd.AddCommand(newInstanceResetCmd())
	cmd.AddCommand(newInstanceLogsCmd())
	cmd.AddCommand(newInstanceTagCreateCmd())
	cmd.AddCommand(newInstanceTagDeleteCmd())
	cmd.AddCommand(newInstanceChangeHostnameCmd())
	cmd.AddCommand(newInstanceChangePasswordCmd())
	cmd.AddCommand(newInstanceChangePlanCmd())
	cmd.AddCommand(newInstanceChangeOSCmd())
	cmd.AddCommand(newInstanceChangeScriptCmd())
	cmd.AddCommand(newInstanceAddNetworkCmd())
	cmd.AddCommand(newInstanceAddonsCmd())
	cmd.AddCommand(newInstancePurchaseAddonCmd())
	cmd.AddCommand(newInstanceSSHCmd())
	cmd.AddCommand(newInstanceDeleteCmd())
	return cmd
}

// ---------- list ----------

func newInstanceListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List virtual machines",
		Example: `  zcp instance list
  zcp instance list --output json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runInstanceList(cmd)
		},
	}
	return cmd
}

func runInstanceList(cmd *cobra.Command) error {
	_, client, printer, err := buildClientAndPrinter(cmd)
	if err != nil {
		return err
	}

	svc := instance.NewService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	region, project := scopedRegionProject(cmd)
	vms, err := svc.List(ctx, region, project)
	if err != nil {
		return fmt.Errorf("instance list: %w", err)
	}

	// Structured output gets the full VM objects rather than a flattened,
	// all-string subset of the table columns.
	if printer.Format() == output.FormatJSON || printer.Format() == output.FormatYAML {
		return printer.Print(vms)
	}

	expandedOutput := debugEnabled(cmd)
	headers := []string{"ID", "NAME", "STATE", "PRIVATE IP", "PUBLIC IP", "REGION"}
	if expandedOutput {
		headers = []string{"ID", "SLUG", "NAME", "STATE", "PRIVATE IP", "PUBLIC IP", "REGION", "TEMPLATE", "CREATED"}
	}
	rows := make([][]string, 0, len(vms))
	for _, vm := range vms {
		templateName := ""
		if vm.Template != nil {
			templateName = vm.Template.Name
		}
		regionName := ""
		if vm.Region != nil {
			regionName = vm.Region.Name
		}
		privateIP := instance.StringVal(vm.PrivateIP)
		if privateIP == "" {
			privateIP = vm.NetworkPrivateIP()
		}
		privateIP = dashIfEmpty(privateIP)
		publicIP := dashIfEmpty(vm.GetPublicIPAddress())
		row := []string{
			instanceDisplayID(vm),
			vm.Name,
			vm.State,
			privateIP,
			publicIP,
			regionName,
		}
		if expandedOutput {
			row = []string{
				instanceDisplayID(vm),
				vm.Slug,
				vm.Name,
				vm.State,
				privateIP,
				publicIP,
				regionName,
				templateName,
				vm.CreatedAt,
			}
		}
		rows = append(rows, row)
	}
	return printer.PrintTable(headers, rows)
}

// ---------- get ----------

func newInstanceGetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get <slug>",
		Short: "Get details of a specific virtual machine",
		Args:  exactArgs(1),
		Example: `  zcp instance get my-vm
  zcp instance get my-vm --output json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runInstanceGet(cmd, args[0])
		},
	}
	return cmd
}

func runInstanceGet(cmd *cobra.Command, slug string) error {
	_, client, printer, err := buildClientAndPrinter(cmd)
	if err != nil {
		return err
	}

	svc := instance.NewService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	resolved, err := resolveInstanceRef(ctx, cmd, svc, slug)
	if err != nil {
		return err
	}

	var vm *instance.VirtualMachine
	for attempt := 0; attempt < 5; attempt++ {
		vm, err = svc.Get(ctx, resolved.Slug)
		if err == nil {
			break
		}
		if apierrors.IsTransientRoutingError(err) && attempt < 4 {
			wait := instanceGetRetryWait(attempt)
			fmt.Fprintf(cmd.ErrOrStderr(), "VM routing not ready yet, retrying in %v...\n", wait)
			timer := time.NewTimer(wait)
			select {
			case <-ctx.Done():
				timer.Stop()
				return fmt.Errorf("instance get: %w", ctx.Err())
			case <-timer.C:
			}
			continue
		}
		return fmt.Errorf("instance get: %w", err)
	}
	if err != nil {
		return fmt.Errorf("instance get: %w", err)
	}

	templateName := ""
	osFamily := ""
	if vm.Template != nil {
		templateName = vm.Template.Name
		if vm.Template.OperatingSystem != nil {
			osFamily = vm.Template.OperatingSystem.Family
		}
	}
	regionName := ""
	if vm.Region != nil {
		regionName = vm.Region.Name
	}
	billingCycle := ""
	if vm.Offering != nil && vm.Offering.BillingCycle != nil {
		billingCycle = vm.Offering.BillingCycle.Name
	}
	storageName := ""
	if vm.StorageSetting != nil {
		storageName = vm.StorageSetting.Name
	}

	privateIP := instance.StringVal(vm.PrivateIP)
	if privateIP == "" {
		privateIP = vm.NetworkPrivateIP()
	}
	privateIP = dashIfEmpty(privateIP)

	publicIP := dashIfEmpty(vm.GetPublicIPAddress())

	headers := []string{"FIELD", "VALUE"}
	rows := [][]string{
		{"ID", instanceDisplayID(*vm)},
		{"Slug", vm.Slug},
		{"Name", vm.Name},
		{"Hostname", vm.Hostname},
		{"State", vm.State},
		{"Username", vm.Username},
		{"Private IP", privateIP},
		{"Public IP", publicIP},
		{"Region", regionName},
		{"Template", templateName},
		{"OS Family", osFamily},
		{"Billing Cycle", billingCycle},
		{"Storage", storageName},
		{"Total Consumption", fmt.Sprintf("%.2f", vm.AllTimeConsumption)},
		{"Has Contract", strconv.FormatBool(vm.HasContract)},
		{"Created", vm.CreatedAt},
		{"Updated", vm.UpdatedAt},
	}
	if vm.Description != nil && *vm.Description != "" {
		rows = append(rows, []string{"Description", *vm.Description})
	}
	if vm.ID != "" && vm.ID != instanceDisplayID(*vm) {
		rows = append(rows, []string{"Record ID", vm.ID})
	}
	return printer.PrintTable(headers, rows)
}

func instanceDisplayID(vm instance.VirtualMachine) string {
	if vm.VMID != "" {
		return vm.VMID
	}
	return vm.ID
}

// errInstanceNotFound marks the "no match in this listing" case so the caller
// can decide whether to widen the search; ambiguous-match errors do not wrap it.
var errInstanceNotFound = errors.New("instance not found")

// resolveInstanceRef maps a user-supplied reference (instance ID, vm_id, name,
// or slug) to a concrete VM. It first searches the active region/project scope,
// then — only when the reference simply wasn't found there — retries unscoped so
// that operating on a globally-unique slug or ID keeps working even when the VM
// lives outside the active scope (the pre-resolution behavior, where the slug
// hit the API directly). Ambiguous-name errors are returned as-is, since
// widening the search cannot disambiguate them.
func resolveInstanceRef(ctx context.Context, cmd *cobra.Command, svc *instance.Service, ref string) (*instance.VirtualMachine, error) {
	region, project := scopedRegionProject(cmd)
	vm, err := resolveInstanceInScope(ctx, svc, region, project, ref)
	if err == nil {
		return vm, nil
	}
	if (region != "" || project != "") && errors.Is(err, errInstanceNotFound) {
		vm, uerr := resolveInstanceInScope(ctx, svc, "", "", ref)
		if uerr == nil {
			return vm, nil
		}
		// A genuine failure (API error, or an ambiguous-name match) from the
		// widened search is meaningful — surface it instead of masking it with
		// the original scoped not-found error.
		if !errors.Is(uerr, errInstanceNotFound) {
			return nil, uerr
		}
	}
	return nil, err
}

// resolveInstanceInScope lists VMs for the given region/project and resolves ref
// against that set. Match precedence: exact instance ID or vm_id, then name
// (reported as ambiguous if more than one matches), then slug.
func resolveInstanceInScope(ctx context.Context, svc *instance.Service, region, project, ref string) (*instance.VirtualMachine, error) {
	vms, err := svc.List(ctx, region, project)
	if err != nil {
		return nil, fmt.Errorf("resolving instance %q: %w", ref, err)
	}

	for i := range vms {
		if vms[i].ID == ref || vms[i].VMID == ref {
			return &vms[i], nil
		}
	}

	matches := make([]instance.VirtualMachine, 0)
	for i := range vms {
		if vms[i].Name == ref {
			matches = append(matches, vms[i])
		}
	}
	switch len(matches) {
	case 1:
		return &matches[0], nil
	case 0:
		for i := range vms {
			if vms[i].Slug == ref {
				return &vms[i], nil
			}
		}
		scope := fmt.Sprintf("region %q project %q", region, project)
		if region == "" && project == "" {
			scope = "any region or project"
		}
		return nil, fmt.Errorf("%w: %q was not found in %s; run `zcp instance list` and use the instance ID", errInstanceNotFound, ref, scope)
	default:
		choices := make([]string, 0, len(matches))
		for _, vm := range matches {
			choices = append(choices, fmt.Sprintf("%s (%s)", instanceDisplayID(vm), vm.Slug))
		}
		return nil, fmt.Errorf("instance name %q matches %d instances; use the correct instance ID instead: %s", ref, len(matches), strings.Join(choices, ", "))
	}
}

// ---------- create ----------

func newInstanceCreateCmd() *cobra.Command {
	var (
		name             string
		cloudProvider    string
		project          string
		region           string
		template         string
		plan             string
		billingCycle     string
		networkType      string
		sshKey           string
		hostname         string
		storageCategory  string
		computeCategory  string
		blockstoragePlan string
		networkPlan      string
		userData         string
		userDataFile     string
		cpu              int
		memory           int
		disk             int
		wait             bool
		isPublic         bool
	)

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new virtual machine",
		Example: `  zcp instance create --name my-vm --project default-9 --region yul-1 --template ubuntu-2604-lts-1 --plan ca2sl --billing-cycle hourly --network-plan pnet-yul --storage-category premium-ssd
  zcp instance create --name my-vm --project default-9 --region yul-1 --template ubuntu-2604-lts-1 --plan ca2sl --billing-cycle hourly --network-plan pnet-yul --storage-category premium-ssd --wait
  zcp instance create --name my-vm --project default-9 --region yul-1 --template ubuntu-2604-lts-1 --plan ca2sl --billing-cycle hourly --network-plan pnet-yul --storage-category premium-ssd --ssh-key mykey   # import the key first with 'zcp ssh-key import'`,
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
			if template == "" {
				return fmt.Errorf("--template is required")
			}
			if plan == "" {
				return fmt.Errorf("--plan is required")
			}
			if billingCycle == "" {
				return fmt.Errorf("--billing-cycle is required")
			}
			if storageCategory == "" {
				return fmt.Errorf("--storage-category is required")
			}
			if networkPlan == "" {
				return fmt.Errorf("--network-plan is required")
			}
			if userData != "" && userDataFile != "" {
				return fmt.Errorf("--user-data and --user-data-file are mutually exclusive")
			}
			if userDataFile != "" {
				data, err := os.ReadFile(userDataFile)
				if err != nil {
					return fmt.Errorf("reading user-data file %q: %w", userDataFile, err)
				}
				userData = string(data)
			}

			if networkType == "L2" && isPublic {
				return fmt.Errorf("--is-public cannot be true for L2 networks; pass --is-public=false")
			}

			h := hostname
			if h == "" {
				h = name
			}

			var sshKeyPtr *string
			var passwordPtr *string
			authMethod := ""
			if sshKey != "" {
				sshKeyPtr = &sshKey
				authMethod = "ssh-key"
				empty := ""
				passwordPtr = &empty
			}

			var userDataPtr *string
			if userData != "" {
				userDataPtr = &userData
			}

			if cmd.Flags().Changed("cpu") && cpu <= 0 {
				return fmt.Errorf("invalid value for --cpu: must be > 0")
			}
			if cmd.Flags().Changed("memory") && memory <= 0 {
				return fmt.Errorf("invalid value for --memory: must be > 0")
			}
			if cmd.Flags().Changed("disk") && disk <= 0 {
				return fmt.Errorf("invalid value for --disk: must be > 0")
			}

			var customPlan *instance.CustomPlan
			if cpu > 0 || memory > 0 || disk > 0 {
				customPlan = &instance.CustomPlan{}
				if cpu > 0 {
					customPlan.CPU = strconv.Itoa(cpu)
				}
				if memory > 0 {
					customPlan.Memory = strconv.Itoa(memory)
				}
				if disk > 0 {
					customPlan.Storage = strconv.Itoa(disk)
				}
			}

			req := instance.CreateRequest{
				Name:             name,
				CloudProvider:    cloudProvider,
				Project:          project,
				Region:           region,
				BootSource:       "image",
				Server:           "cloud-compute",
				Template:         template,
				IsPublic:         isPublic,
				NetworkType:      networkType,
				Networks:         []string{},
				BillingCycle:     billingCycle,
				SSHKey:           sshKeyPtr,
				AuthMethod:       authMethod,
				Password:         passwordPtr,
				Plan:             plan,
				CustomPlan:       customPlan,
				OSFamily:         "Linux",
				TemplateType:     "Operating System",
				Hostname:         h,
				Addons:           []string{},
				StorageCategory:  storageCategory,
				ComputeCategory:  computeCategory,
				BlockstoragePlan: blockstoragePlan,
				NetworkPlan:      networkPlan,
				UserData:         userDataPtr,
			}
			return runInstanceCreate(cmd, req, wait)
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "VM name (required)")
	cmd.Flags().StringVar(&cloudProvider, "cloud-provider", "", "Cloud provider slug (optional; auto-detected, override only)")
	cmd.Flags().StringVar(&project, "project", "", "Project slug (required)")
	cmd.Flags().StringVar(&region, "region", "", "Region slug (required)")
	cmd.Flags().StringVar(&template, "template", "", "Template slug (required)")
	cmd.Flags().StringVar(&plan, "plan", "", "Plan slug (required)")
	cmd.Flags().StringVar(&billingCycle, "billing-cycle", "", "Billing cycle slug: hourly, monthly, etc. (required)")
	cmd.Flags().StringVar(&networkType, "network-type", "Isolated", "Network type (default: Isolated)")
	cmd.Flags().StringVar(&sshKey, "ssh-key", "", "Name of an existing SSH key to attach for login (optional; see 'zcp ssh-key list')")
	cmd.Flags().StringVar(&hostname, "hostname", "", "Hostname (defaults to --name)")
	cmd.Flags().StringVar(&storageCategory, "storage-category", "", "Storage category (required, e.g. premium-ssd - see: zcp plan storage)")
	cmd.Flags().StringVar(&computeCategory, "compute-category", "", "Compute category slug (optional)")
	cmd.Flags().StringVar(&blockstoragePlan, "blockstorage-plan", "", "Block storage plan slug (optional, e.g. b2g1 — see: zcp plan storage)")
	cmd.Flags().StringVar(&networkPlan, "network-plan", "", "Network plan slug (required, e.g. pnet-yow, pnet-yul — see: zcp plan network)")
	cmd.Flags().StringVar(&userData, "user-data", "", "Startup script content (cloud-init / bash)")
	cmd.Flags().StringVar(&userDataFile, "user-data-file", "", "Path to a file containing the startup script")
	cmd.Flags().IntVar(&cpu, "cpu", 0, "Number of vCPUs for a custom plan (e.g. 2)")
	cmd.Flags().IntVar(&memory, "memory", 0, "RAM in MB for a custom plan (e.g. 2048)")
	cmd.Flags().IntVar(&disk, "disk", 0, "Root disk size in GB for a custom plan (e.g. 50)")
	cmd.Flags().BoolVar(&wait, "wait", false, "Wait for the instance to reach Running state")
	cmd.Flags().BoolVar(&isPublic, "is-public", true, "Assign a public IP address")
	return cmd
}

func runInstanceCreate(cmd *cobra.Command, req instance.CreateRequest, wait bool) error {
	_, client, printer, err := buildClientAndPrinter(cmd)
	if err != nil {
		return err
	}

	svc := instance.NewService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	vm, err := svc.Create(ctx, req)
	if err != nil {
		return fmt.Errorf("instance create: %w", err)
	}

	if wait {
		fmt.Fprintf(os.Stderr, "Waiting for instance %s to be Running...\n", vm.Slug)
		waitCtx, waitCancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd)+300)*time.Second)
		defer waitCancel()
		vm, err = svc.WaitForState(waitCtx, vm.Slug, []string{"Running"}, 0)
		if err != nil {
			return fmt.Errorf("waiting for instance create: %w", err)
		}
		fmt.Fprintf(os.Stderr, "Instance %s is now %s\n", vm.Slug, vm.State)
	}

	templateName := ""
	if vm.Template != nil {
		templateName = vm.Template.Name
	}
	headers := []string{"ID", "SLUG", "NAME", "STATE", "TEMPLATE", "CREATED"}
	rows := [][]string{
		{instanceDisplayID(*vm), vm.Slug, vm.Name, vm.State, templateName, vm.CreatedAt},
	}
	return printer.PrintTable(headers, rows)
}

// ---------- start ----------

func newInstanceStartCmd() *cobra.Command {
	var wait bool

	cmd := &cobra.Command{
		Use:     "start <slug>",
		Short:   "Start a stopped virtual machine",
		Args:    exactArgs(1),
		Example: `  zcp instance start my-vm`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runInstanceStart(cmd, args[0], wait)
		},
	}
	cmd.Flags().BoolVar(&wait, "wait", false, "Wait for the instance to reach Running state")
	return cmd
}

func runInstanceStart(cmd *cobra.Command, slug string, wait bool) error {
	_, client, printer, err := buildClientAndPrinter(cmd)
	if err != nil {
		return err
	}

	svc := instance.NewService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	vm, err := resolveInstanceRef(ctx, cmd, svc, slug)
	if err != nil {
		return err
	}

	resp, err := svc.Start(ctx, vm.Slug)
	if err != nil {
		return fmt.Errorf("instance start: %w", err)
	}

	if wait {
		fmt.Fprintf(os.Stderr, "Waiting for instance %s to be Running...\n", vm.Slug)
		waitCtx, waitCancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd)+300)*time.Second)
		defer waitCancel()
		vm, err := svc.WaitForState(waitCtx, vm.Slug, []string{"Running"}, 0)
		if err != nil {
			return fmt.Errorf("waiting for instance start: %w", err)
		}
		fmt.Fprintf(os.Stderr, "Instance %s is now %s\n", vm.Slug, vm.State)
	}

	headers := []string{"STATUS", "MESSAGE"}
	rows := [][]string{{resp.Status, resp.Message}}
	return printer.PrintTable(headers, rows)
}

// ---------- stop ----------

func newInstanceStopCmd() *cobra.Command {
	var wait bool

	cmd := &cobra.Command{
		Use:     "stop <slug>",
		Short:   "Stop a running virtual machine",
		Args:    exactArgs(1),
		Example: `  zcp instance stop my-vm`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runInstanceStop(cmd, args[0], wait)
		},
	}
	cmd.Flags().BoolVar(&wait, "wait", false, "Wait for the instance to reach Stopped state")
	return cmd
}

func runInstanceStop(cmd *cobra.Command, slug string, wait bool) error {
	_, client, printer, err := buildClientAndPrinter(cmd)
	if err != nil {
		return err
	}

	svc := instance.NewService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	vm, err := resolveInstanceRef(ctx, cmd, svc, slug)
	if err != nil {
		return err
	}

	resp, err := svc.Stop(ctx, vm.Slug)
	if err != nil {
		return fmt.Errorf("instance stop: %w", err)
	}

	if wait {
		fmt.Fprintf(os.Stderr, "Waiting for instance %s to be Stopped...\n", vm.Slug)
		waitCtx, waitCancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd)+300)*time.Second)
		defer waitCancel()
		vm, err := svc.WaitForState(waitCtx, vm.Slug, []string{"Stopped"}, 0)
		if err != nil {
			return fmt.Errorf("waiting for instance stop: %w", err)
		}
		fmt.Fprintf(os.Stderr, "Instance %s is now %s\n", vm.Slug, vm.State)
	}

	headers := []string{"STATUS", "MESSAGE"}
	rows := [][]string{{resp.Status, resp.Message}}
	return printer.PrintTable(headers, rows)
}

// ---------- reboot ----------

func newInstanceRebootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "reboot <slug>",
		Short:   "Reboot a virtual machine",
		Args:    exactArgs(1),
		Example: `  zcp instance reboot my-vm`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runInstanceReboot(cmd, args[0])
		},
	}
	return cmd
}

func runInstanceReboot(cmd *cobra.Command, slug string) error {
	_, client, printer, err := buildClientAndPrinter(cmd)
	if err != nil {
		return err
	}

	svc := instance.NewService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	vm, err := resolveInstanceRef(ctx, cmd, svc, slug)
	if err != nil {
		return err
	}
	// resolveInstanceRef just listed the VM, so its state is fresh — no need for
	// a second GET to gate the reboot.
	if !strings.EqualFold(vm.State, "Running") {
		return fmt.Errorf("instance %q is %s; it must be Running before it can be rebooted", vm.Slug, vm.State)
	}

	resp, err := svc.Reboot(ctx, vm.Slug)
	if err != nil {
		return fmt.Errorf("instance reboot: %w", err)
	}

	headers := []string{"STATUS", "MESSAGE"}
	rows := [][]string{{resp.Status, resp.Message}}
	return printer.PrintTable(headers, rows)
}

// ---------- reset ----------

func newInstanceResetCmd() *cobra.Command {
	var yes bool

	cmd := &cobra.Command{
		Use:   "reset <slug>",
		Short: "Reset a virtual machine (hard reset, may lose unsaved data)",
		Args:  exactArgs(1),
		Example: `  zcp instance reset my-vm
  zcp instance reset my-vm --yes`,
		RunE: func(cmd *cobra.Command, args []string) error {
			slug := args[0]
			if !yes && !autoApproved(cmd) {
				fmt.Fprintf(os.Stderr, "WARNING: Reset %q will forcefully restart the VM. Unsaved data may be lost. [y/N]: ", slug)
				scanner := bufio.NewScanner(os.Stdin)
				scanner.Scan()
				answer := strings.TrimSpace(strings.ToLower(scanner.Text()))
				if answer != "y" && answer != "yes" {
					fmt.Fprintln(os.Stderr, "Aborted.")
					return nil
				}
			}
			return runInstanceReset(cmd, slug)
		},
	}
	cmd.Flags().BoolVar(&yes, "yes", false, "Skip confirmation prompt")
	return cmd
}

func runInstanceReset(cmd *cobra.Command, slug string) error {
	_, client, printer, err := buildClientAndPrinter(cmd)
	if err != nil {
		return err
	}

	svc := instance.NewService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	vm, err := resolveInstanceRef(ctx, cmd, svc, slug)
	if err != nil {
		return err
	}

	resp, err := svc.Reset(ctx, vm.Slug)
	if err != nil {
		return fmt.Errorf("instance reset: %w", err)
	}

	headers := []string{"STATUS", "MESSAGE"}
	rows := [][]string{{resp.Status, resp.Message}}
	return printer.PrintTable(headers, rows)
}

// ---------- logs ----------

func newInstanceLogsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "logs <slug>",
		Short:   "Show activity logs for a virtual machine",
		Args:    exactArgs(1),
		Example: `  zcp instance logs my-vm`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runInstanceLogs(cmd, args[0])
		},
	}
	return cmd
}

func runInstanceLogs(cmd *cobra.Command, slug string) error {
	_, client, printer, err := buildClientAndPrinter(cmd)
	if err != nil {
		return err
	}

	svc := instance.NewService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	vm, err := resolveInstanceRef(ctx, cmd, svc, slug)
	if err != nil {
		return err
	}

	logs, err := svc.ActivityLogs(ctx, vm.Slug)
	if err != nil {
		return fmt.Errorf("instance logs: %w", err)
	}

	headers := []string{"ID", "ACTION", "STATUS", "DESCRIPTION", "CREATED"}
	rows := make([][]string, 0, len(logs))
	for _, l := range logs {
		rows = append(rows, []string{
			l.ID.String(),
			l.Action,
			l.Status,
			l.Description,
			l.CreatedAt,
		})
	}
	return printer.PrintTable(headers, rows)
}

// ---------- tag create ----------

func newInstanceTagCreateCmd() *cobra.Command {
	var key, value string

	cmd := &cobra.Command{
		Use:     "tag-create <slug>",
		Short:   "Create a tag on a virtual machine",
		Args:    exactArgs(1),
		Example: `  zcp instance tag-create my-vm --key Environment --value Production`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if key == "" {
				return fmt.Errorf("--key is required")
			}
			if value == "" {
				return fmt.Errorf("--value is required")
			}
			return runInstanceTagCreate(cmd, args[0], key, value)
		},
	}
	cmd.Flags().StringVar(&key, "key", "", "Tag key (required)")
	cmd.Flags().StringVar(&value, "value", "", "Tag value (required)")
	return cmd
}

func runInstanceTagCreate(cmd *cobra.Command, slug, key, value string) error {
	_, client, printer, err := buildClientAndPrinter(cmd)
	if err != nil {
		return err
	}

	svc := instance.NewService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	vm, err := resolveInstanceRef(ctx, cmd, svc, slug)
	if err != nil {
		return err
	}

	resp, err := svc.CreateTag(ctx, vm.Slug, instance.TagRequest{Key: key, Value: value})
	if err != nil {
		return fmt.Errorf("instance tag-create: %w", err)
	}

	headers := []string{"STATUS", "MESSAGE"}
	rows := [][]string{{resp.Status, resp.Message}}
	return printer.PrintTable(headers, rows)
}

// ---------- tag delete ----------

func newInstanceTagDeleteCmd() *cobra.Command {
	var key string

	cmd := &cobra.Command{
		Use:     "tag-delete <slug>",
		Short:   "Delete a tag from a virtual machine",
		Args:    exactArgs(1),
		Example: `  zcp instance tag-delete my-vm --key Environment`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if key == "" {
				return fmt.Errorf("--key is required")
			}
			return runInstanceTagDelete(cmd, args[0], key)
		},
	}
	cmd.Flags().StringVar(&key, "key", "", "Tag key to delete (required)")
	return cmd
}

func runInstanceTagDelete(cmd *cobra.Command, slug, key string) error {
	_, client, _, err := buildClientAndPrinter(cmd)
	if err != nil {
		return err
	}

	svc := instance.NewService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	vm, err := resolveInstanceRef(ctx, cmd, svc, slug)
	if err != nil {
		return err
	}

	if err := svc.DeleteTag(ctx, vm.Slug, key); err != nil {
		if apierrors.IsResourceNotFound(err) {
			fmt.Fprintf(os.Stderr, "Instance tag %q not found — already deleted.\n", key)
			return nil
		}
		return fmt.Errorf("instance tag-delete: %w", err)
	}

	fmt.Fprintf(os.Stdout, "Tag %q deleted from instance %q.\n", key, vm.Slug)
	return nil
}

// ---------- change-hostname ----------

func newInstanceChangeHostnameCmd() *cobra.Command {
	var hostname string

	cmd := &cobra.Command{
		Use:     "change-hostname <slug>",
		Short:   "Change the hostname of a virtual machine",
		Args:    exactArgs(1),
		Example: `  zcp instance change-hostname my-vm --hostname new-hostname`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if hostname == "" {
				return fmt.Errorf("--hostname is required")
			}
			return runInstanceChangeHostname(cmd, args[0], hostname)
		},
	}
	cmd.Flags().StringVar(&hostname, "hostname", "", "New hostname (required)")
	return cmd
}

func runInstanceChangeHostname(cmd *cobra.Command, slug, hostname string) error {
	_, client, printer, err := buildClientAndPrinter(cmd)
	if err != nil {
		return err
	}

	svc := instance.NewService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	vm, err := resolveInstanceRef(ctx, cmd, svc, slug)
	if err != nil {
		return err
	}

	resp, err := svc.ChangeHostname(ctx, vm.Slug, instance.ChangeLabelRequest{
		Name:     hostname,
		Hostname: hostname,
	})
	if err != nil {
		return fmt.Errorf("instance change-hostname: %w", err)
	}

	headers := []string{"STATUS", "MESSAGE"}
	rows := [][]string{{resp.Status, resp.Message}}
	return printer.PrintTable(headers, rows)
}

// ---------- change-password ----------

func newInstanceChangePasswordCmd() *cobra.Command {
	var password string

	cmd := &cobra.Command{
		Use:     "change-password <slug>",
		Short:   "Reset the password of a virtual machine",
		Args:    exactArgs(1),
		Example: `  zcp instance change-password my-vm --password "newSecureP@ss"`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if password == "" {
				return fmt.Errorf("--password is required")
			}
			return runInstanceChangePassword(cmd, args[0], password)
		},
	}
	cmd.Flags().StringVar(&password, "password", "", "New password (required)")
	return cmd
}

func runInstanceChangePassword(cmd *cobra.Command, slug, password string) error {
	_, client, printer, err := buildClientAndPrinter(cmd)
	if err != nil {
		return err
	}

	svc := instance.NewService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	vm, err := resolveInstanceRef(ctx, cmd, svc, slug)
	if err != nil {
		return err
	}

	resp, err := svc.ChangePassword(ctx, vm.Slug, instance.ChangePasswordRequest{
		Password: password,
		VM:       vm.Slug,
	})
	if err != nil {
		return fmt.Errorf("instance change-password: %w", err)
	}

	headers := []string{"STATUS", "MESSAGE"}
	rows := [][]string{{resp.Status, resp.Message}}
	return printer.PrintTable(headers, rows)
}

// ---------- change-plan ----------

func newInstanceChangePlanCmd() *cobra.Command {
	var plan, billingCycle string

	cmd := &cobra.Command{
		Use:     "change-plan <slug>",
		Short:   "Change the plan of a virtual machine",
		Args:    exactArgs(1),
		Example: `  zcp instance change-plan my-vm --plan ca2sm --billing-cycle hourly`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if plan == "" {
				return fmt.Errorf("--plan is required")
			}
			if billingCycle == "" {
				return fmt.Errorf("--billing-cycle is required")
			}
			return runInstanceChangePlan(cmd, args[0], plan, billingCycle)
		},
	}
	cmd.Flags().StringVar(&plan, "plan", "", "New plan slug (required)")
	cmd.Flags().StringVar(&billingCycle, "billing-cycle", "", "Billing cycle slug (required)")
	return cmd
}

func runInstanceChangePlan(cmd *cobra.Command, slug, plan, billingCycle string) error {
	_, client, printer, err := buildClientAndPrinter(cmd)
	if err != nil {
		return err
	}

	svc := instance.NewService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	vm, err := resolveInstanceRef(ctx, cmd, svc, slug)
	if err != nil {
		return err
	}

	resp, err := svc.ChangePlan(ctx, vm.Slug, instance.ChangePlanRequest{
		Plan:         plan,
		Slug:         vm.Slug,
		VM:           vm.Slug,
		BillingCycle: billingCycle,
	})
	if err != nil {
		return fmt.Errorf("instance change-plan: %w", err)
	}

	headers := []string{"STATUS", "MESSAGE"}
	rows := [][]string{{resp.Status, resp.Message}}
	return printer.PrintTable(headers, rows)
}

// ---------- change-os ----------

func newInstanceChangeOSCmd() *cobra.Command {
	var template string
	var yes bool

	cmd := &cobra.Command{
		Use:   "change-os <slug>",
		Short: "Change the OS template of a virtual machine (DESTRUCTIVE)",
		Args:  exactArgs(1),
		Example: `  zcp instance change-os my-vm --template ubuntu-2604-lts-1
  zcp instance change-os my-vm --template ubuntu-2604-lts-1 --yes`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if template == "" {
				return fmt.Errorf("--template is required")
			}
			slug := args[0]
			if !yes && !autoApproved(cmd) {
				fmt.Fprintf(os.Stderr, "WARNING: Changing OS on %q will reinstall the VM and erase all data. [y/N]: ", slug)
				scanner := bufio.NewScanner(os.Stdin)
				scanner.Scan()
				answer := strings.TrimSpace(strings.ToLower(scanner.Text()))
				if answer != "y" && answer != "yes" {
					fmt.Fprintln(os.Stderr, "Aborted.")
					return nil
				}
			}
			return runInstanceChangeOS(cmd, slug, template)
		},
	}
	cmd.Flags().StringVar(&template, "template", "", "New template slug (required)")
	cmd.Flags().BoolVar(&yes, "yes", false, "Skip confirmation prompt")
	return cmd
}

func runInstanceChangeOS(cmd *cobra.Command, slug, template string) error {
	_, client, printer, err := buildClientAndPrinter(cmd)
	if err != nil {
		return err
	}

	svc := instance.NewService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	vm, err := resolveInstanceRef(ctx, cmd, svc, slug)
	if err != nil {
		return err
	}

	resp, err := svc.ChangeOS(ctx, vm.Slug, instance.ChangeTemplateRequest{Template: template})
	if err != nil {
		return fmt.Errorf("instance change-os: %w", err)
	}

	headers := []string{"STATUS", "MESSAGE"}
	rows := [][]string{{resp.Status, resp.Message}}
	return printer.PrintTable(headers, rows)
}

// ---------- change-script ----------

func newInstanceChangeScriptCmd() *cobra.Command {
	var userData string

	cmd := &cobra.Command{
		Use:     "change-script <slug>",
		Short:   "Change the startup script of a virtual machine",
		Args:    exactArgs(1),
		Example: `  zcp instance change-script my-vm --user-data "#!/bin/bash\napt update"`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if userData == "" {
				return fmt.Errorf("--user-data is required")
			}
			return runInstanceChangeScript(cmd, args[0], userData)
		},
	}
	cmd.Flags().StringVar(&userData, "user-data", "", "Startup script content (required)")
	return cmd
}

func runInstanceChangeScript(cmd *cobra.Command, slug, userData string) error {
	_, client, printer, err := buildClientAndPrinter(cmd)
	if err != nil {
		return err
	}

	svc := instance.NewService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	vm, err := resolveInstanceRef(ctx, cmd, svc, slug)
	if err != nil {
		return err
	}

	resp, err := svc.ChangeStartupScript(ctx, vm.Slug, instance.ChangeStartupScriptRequest{UserData: userData})
	if err != nil {
		return fmt.Errorf("instance change-script: %w", err)
	}

	headers := []string{"STATUS", "MESSAGE"}
	rows := [][]string{{resp.Status, resp.Message}}
	return printer.PrintTable(headers, rows)
}

// ---------- add-network ----------

func newInstanceAddNetworkCmd() *cobra.Command {
	var network string

	cmd := &cobra.Command{
		Use:     "add-network <slug>",
		Short:   "Add a network to a virtual machine",
		Args:    exactArgs(1),
		Example: `  zcp instance add-network my-vm --network my-network-slug`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if network == "" {
				return fmt.Errorf("--network is required")
			}
			return runInstanceAddNetwork(cmd, args[0], network)
		},
	}
	cmd.Flags().StringVar(&network, "network", "", "Network slug to add (required)")
	return cmd
}

func runInstanceAddNetwork(cmd *cobra.Command, slug, network string) error {
	_, client, printer, err := buildClientAndPrinter(cmd)
	if err != nil {
		return err
	}

	svc := instance.NewService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	vm, err := resolveInstanceRef(ctx, cmd, svc, slug)
	if err != nil {
		return err
	}

	resp, err := svc.AddNetwork(ctx, vm.Slug, instance.AddNetworkRequest{Network: network})
	if err != nil {
		return fmt.Errorf("instance add-network: %w", err)
	}

	headers := []string{"STATUS", "MESSAGE"}
	rows := [][]string{{resp.Status, resp.Message}}
	return printer.PrintTable(headers, rows)
}

// ---------- addons ----------

func newInstanceAddonsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "addons <slug>",
		Short:   "List addons for a virtual machine",
		Args:    exactArgs(1),
		Example: `  zcp instance addons my-vm`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runInstanceAddons(cmd, args[0])
		},
	}
	return cmd
}

func runInstanceAddons(cmd *cobra.Command, slug string) error {
	_, client, printer, err := buildClientAndPrinter(cmd)
	if err != nil {
		return err
	}

	svc := instance.NewService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	vm, err := resolveInstanceRef(ctx, cmd, svc, slug)
	if err != nil {
		return err
	}

	addons, err := svc.ListAddons(ctx, vm.Slug)
	if err != nil {
		return fmt.Errorf("instance addons: %w", err)
	}

	if len(addons) == 0 {
		fmt.Fprintln(os.Stdout, "No addons found.")
		return nil
	}

	headers := []string{"ID", "NAME", "SLUG", "STATUS", "CREATED"}
	rows := make([][]string, 0, len(addons))
	for _, a := range addons {
		rows = append(rows, []string{
			a.ID,
			a.Name,
			a.Slug,
			strconv.FormatBool(a.Status),
			a.CreatedAt,
		})
	}
	return printer.PrintTable(headers, rows)
}

// ---------- purchase-addon ----------

func newInstancePurchaseAddonCmd() *cobra.Command {
	var (
		vmSlug        string
		project       string
		region        string
		cloudProvider string
		addonSlug     string
		addonCategory string
		addonID       string
		billingCycle  string
		quantity      int
	)

	cmd := &cobra.Command{
		Use:     "purchase-addon",
		Short:   "Purchase an addon for a virtual machine",
		Example: `  zcp instance purchase-addon --vm my-vm --project default-9 --region yul-1 --addon-slug remote-desktop-license --addon-category microsoft-spla-licenses --addon-id a1b2c3d4-e5f6-7890-abcd-ef1234567890 --billing-cycle hourly`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if vmSlug == "" {
				return fmt.Errorf("--vm is required")
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
			if addonSlug == "" {
				return fmt.Errorf("--addon-slug is required")
			}
			if addonID == "" {
				return fmt.Errorf("--addon-id is required")
			}
			if billingCycle == "" {
				return fmt.Errorf("--billing-cycle is required")
			}
			return runInstancePurchaseAddon(cmd, vmSlug, project, region, cloudProvider, addonSlug, addonCategory, addonID, billingCycle, quantity)
		},
	}
	cmd.Flags().StringVar(&vmSlug, "vm", "", "VM slug (required)")
	cmd.Flags().StringVar(&project, "project", "", "Project slug (required)")
	cmd.Flags().StringVar(&region, "region", "", "Region slug (required)")
	cmd.Flags().StringVar(&cloudProvider, "cloud-provider", "", "Cloud provider slug (optional; auto-detected, override only)")
	cmd.Flags().StringVar(&addonSlug, "addon-slug", "", "Addon slug (required)")
	cmd.Flags().StringVar(&addonCategory, "addon-category", "", "Addon category slug (optional)")
	cmd.Flags().StringVar(&addonID, "addon-id", "", "Addon ID (required)")
	cmd.Flags().StringVar(&billingCycle, "billing-cycle", "", "Billing cycle slug (required)")
	cmd.Flags().IntVar(&quantity, "quantity", 1, "Quantity (default: 1)")
	return cmd
}

func runInstancePurchaseAddon(cmd *cobra.Command, vmSlug, project, region, cloudProvider, addonSlug, addonCategory, addonID, billingCycle string, quantity int) error {
	_, client, printer, err := buildClientAndPrinter(cmd)
	if err != nil {
		return err
	}

	svc := instance.NewService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	vm, err := resolveInstanceRef(ctx, cmd, svc, vmSlug)
	if err != nil {
		return err
	}

	req := instance.PurchaseAddonRequest{
		VirtualMachine: vm.Slug,
		Project:        project,
		Region:         region,
		CloudProvider:  cloudProvider,
		Addons: []instance.AddonInput{
			{
				Category: addonCategory,
				ID:       addonID,
				Slug:     addonSlug,
				Quantity: quantity,
			},
		},
		Service:      "Store",
		BillingCycle: billingCycle,
	}

	resp, err := svc.PurchaseAddon(ctx, req)
	if err != nil {
		return fmt.Errorf("instance purchase-addon: %w", err)
	}

	headers := []string{"STATUS", "MESSAGE"}
	rows := [][]string{{resp.Status, resp.Message}}
	return printer.PrintTable(headers, rows)
}

// ---------- delete ----------

func newInstanceDeleteCmd() *cobra.Command {
	var yes, force, deletePublicIP bool

	cmd := &cobra.Command{
		Use:   "delete <slug>",
		Short: "Permanently delete a virtual machine",
		Long: `Permanently delete a virtual machine.

This submits an immediate service-cancellation request via
POST /billing/service-cancel-requests/{slug} — the same workflow the CMP Web UI
uses — so the VM's auto-assigned public IP is released along with the VM (unless
--delete-public-ip=false). Deletion runs asynchronously in the background: a
successful response means the request was accepted, not that the VM is already
gone. Poll with 'zcp instance get <slug>' to confirm removal.

The VM must be in a destroyable state; a VM that is mid-transition (e.g.
Starting/Stopping) is rejected until it settles. Manually-acquired and source-NAT
IPs are never touched by this command — release those with 'zcp ip release'.`,
		Args: exactArgs(1),
		Example: `  zcp instance delete my-vm
  zcp instance delete my-vm --yes
  # keep the auto-assigned public IP allocated:
  zcp instance delete my-vm --yes --delete-public-ip=false`,
		RunE: func(cmd *cobra.Command, args []string) error {
			slug := args[0]
			if !yes && !autoApproved(cmd) {
				ipNote := " Its auto-assigned public IP will be retained (--delete-public-ip=false)."
				if deletePublicIP {
					ipNote = " Its auto-assigned public IP will also be released."
				}
				fmt.Fprintf(os.Stderr, "WARNING: Delete %q is permanent and cannot be undone.%s [y/N]: ", slug, ipNote)
				scanner := bufio.NewScanner(os.Stdin)
				scanner.Scan()
				answer := strings.TrimSpace(strings.ToLower(scanner.Text()))
				if answer != "y" && answer != "yes" {
					fmt.Fprintln(os.Stderr, "Aborted.")
					return nil
				}
			}
			return runInstanceDelete(cmd, slug, deletePublicIP)
		},
	}
	cmd.Flags().BoolVar(&yes, "yes", false, "Skip confirmation prompt")
	cmd.Flags().BoolVar(&deletePublicIP, "delete-public-ip", true, "Release the VM's auto-assigned public IP as part of the deletion. Set to false to keep it Allocated. Manually-acquired and source-NAT IPs are unaffected either way")
	// --force previously forced a hypervisor expunge on the direct DELETE endpoint.
	// The service-cancellation workflow deletes immediately (type=Immediate), so the
	// flag is a no-op; kept hidden to avoid breaking existing scripts.
	cmd.Flags().BoolVar(&force, "force", false, "Deprecated: no-op (deletion is already immediate)")
	_ = cmd.Flags().MarkHidden("force")
	_ = cmd.Flags().MarkDeprecated("force", "deletion is already immediate; this flag has no effect")
	return cmd
}

func runInstanceDelete(cmd *cobra.Command, slug string, deletePublicIP bool) error {
	_, client, _, err := buildClientAndPrinter(cmd)
	if err != nil {
		return err
	}

	svc := instance.NewService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	vm, err := resolveInstanceRef(ctx, cmd, svc, slug)
	if err != nil {
		// Deleting an already-deleted instance is a no-op success; only a
		// genuine resolution failure (ambiguous match, API error) is fatal.
		if errors.Is(err, errInstanceNotFound) {
			fmt.Fprintf(os.Stderr, "Instance %q not found — already deleted.\n", slug)
			return nil
		}
		return err
	}

	// The CMP Web UI deletes a VM (and releases its auto-assigned public IP) through the
	// unified service-cancellation workflow, NOT the direct DELETE endpoint — the latter
	// ignores delete_public_ip and leaves the IP Allocated/billable. Match the UI.
	dip := deletePublicIP
	req := billing.CancelServiceRequest{
		ServiceName:    "Virtual Machine",
		Reason:         "not_needed_anymore",
		Type:           "Immediate",
		Status:         "Pending",
		BillingCycle:   cancelBillingCycle(vm),
		DeletePublicIP: &dip,
	}
	if err := billing.NewService(client).CancelService(ctx, vm.Slug, req); err != nil {
		if apierrors.IsResourceNotFound(err) {
			fmt.Fprintf(os.Stderr, "Instance %q not found — already deleted.\n", vm.Slug)
			return nil
		}
		return fmt.Errorf("instance delete: %w", err)
	}

	if deletePublicIP {
		fmt.Fprintf(os.Stdout, "Deletion requested for %q; the VM and its auto-assigned public IP are being released in the background.\n", vm.Slug)
	} else {
		fmt.Fprintf(os.Stdout, "Deletion requested for %q (public IP retained); the VM is being removed in the background.\n", vm.Slug)
	}
	return nil
}

// cancelBillingCycle maps a VM's billing cycle to the "hour"/"month" form the
// service-cancellation endpoint documents (create/order requests use "hourly"/"monthly",
// but the cancel body uses the unit form). It tries the VM's unit, slug, then name, and
// defaults to "month" when the VM carries no recognizable billing cycle.
func cancelBillingCycle(vm *instance.VirtualMachine) string {
	if vm != nil && vm.BillingCycle != nil {
		for _, v := range []string{vm.BillingCycle.Unit, vm.BillingCycle.Slug, vm.BillingCycle.Name} {
			if u, ok := billingCycleUnit(v); ok {
				return u
			}
		}
	}
	return "month"
}

// ---------- ssh ----------

func newInstanceSSHCmd() *cobra.Command {
	var user, identityFile string
	var port int

	cmd := &cobra.Command{
		Use:   "ssh <slug>",
		Short: "Open an SSH session to a virtual machine",
		Long: `Open an SSH session to a virtual machine by resolving its IP address via the API.

The CLI looks up the VM's private or public IP and connects. The default SSH user
is "root"; use --user to override.

Requirements:
  - ssh must be installed and available in your PATH
  - The VM must be reachable from your local machine (VPN or public IP)`,
		Args: exactArgs(1),
		Example: `  zcp instance ssh my-vm
  zcp instance ssh my-vm --user ubuntu
  zcp instance ssh my-vm --user root --identity-file ~/.ssh/my-key.pem`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runInstanceSSH(cmd, args[0], user, identityFile, port)
		},
	}
	cmd.Flags().StringVar(&user, "user", "root", "SSH username")
	cmd.Flags().StringVarP(&identityFile, "identity-file", "i", "", "Path to SSH private key file")
	cmd.Flags().IntVar(&port, "port", 22, "SSH port")
	return cmd
}

func runInstanceSSH(cmd *cobra.Command, slug, user, identityFile string, port int) error {
	_, client, _, err := buildClientAndPrinter(cmd)
	if err != nil {
		return err
	}

	svc := instance.NewService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	resolved, err := resolveInstanceRef(ctx, cmd, svc, slug)
	if err != nil {
		return err
	}

	vm, err := svc.Get(ctx, resolved.Slug)
	if err != nil {
		return fmt.Errorf("resolving instance IP: %w", err)
	}

	// Prefer private IP; fall back to network pivot IP, then public IP
	ip := instance.StringVal(vm.PrivateIP)
	if ip == "" {
		ip = vm.NetworkPrivateIP()
	}
	if ip == "" {
		ip = instance.StringVal(vm.PublicIP)
	}
	if ip == "" {
		return fmt.Errorf("instance %s has no usable IP address", slug)
	}

	// Use VM username if no --user override and username is available
	if user == "root" && vm.Username != "" {
		user = vm.Username
	}

	// Build SSH command
	sshArgs := []string{}
	if identityFile != "" {
		sshArgs = append(sshArgs, "-i", identityFile)
	}
	if port != 22 {
		sshArgs = append(sshArgs, "-p", strconv.Itoa(port))
	}
	sshArgs = append(sshArgs, fmt.Sprintf("%s@%s", user, ip))

	sshPath, err := exec.LookPath("ssh")
	if err != nil {
		return fmt.Errorf("ssh not found in PATH: %w", err)
	}

	fmt.Fprintf(os.Stderr, "Connecting to %s@%s...\n", user, ip)

	sshCmd := exec.CommandContext(context.Background(), sshPath, sshArgs...)
	sshCmd.Stdin = os.Stdin
	sshCmd.Stdout = os.Stdout
	sshCmd.Stderr = os.Stderr
	return sshCmd.Run()
}
