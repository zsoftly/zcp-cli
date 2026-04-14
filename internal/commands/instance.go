package commands

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/zsoftly/zcp-cli/internal/api/instance"
)

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

	vms, err := svc.List(ctx)
	if err != nil {
		return fmt.Errorf("instance list: %w", err)
	}

	headers := []string{"SLUG", "NAME", "STATE", "PRIVATE IP", "PUBLIC IP", "REGION", "TEMPLATE", "CREATED"}
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
		rows = append(rows, []string{
			vm.Slug,
			vm.Name,
			vm.State,
			instance.StringVal(vm.PrivateIP),
			instance.StringVal(vm.PublicIP),
			regionName,
			templateName,
			vm.CreatedAt,
		})
	}
	return printer.PrintTable(headers, rows)
}

// ---------- get ----------

func newInstanceGetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get <slug>",
		Short: "Get details of a specific virtual machine",
		Args:  cobra.ExactArgs(1),
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

	vm, err := svc.Get(ctx, slug)
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
	if vm.BillingCycle != nil {
		billingCycle = vm.BillingCycle.Name
	}
	storageName := ""
	if vm.StorageSetting != nil {
		storageName = vm.StorageSetting.Name
	}

	headers := []string{"FIELD", "VALUE"}
	rows := [][]string{
		{"Slug", vm.Slug},
		{"Name", vm.Name},
		{"Hostname", vm.Hostname},
		{"State", vm.State},
		{"Username", vm.Username},
		{"Private IP", instance.StringVal(vm.PrivateIP)},
		{"Public IP", instance.StringVal(vm.PublicIP)},
		{"Region", regionName},
		{"Template", templateName},
		{"OS Family", osFamily},
		{"Billing Cycle", billingCycle},
		{"Storage", storageName},
		{"Service", vm.ServiceName},
		{"Total Consumption", fmt.Sprintf("%.2f", vm.AllTimeConsumption)},
		{"Has Contract", strconv.FormatBool(vm.HasContract)},
		{"Created", vm.CreatedAt},
		{"Updated", vm.UpdatedAt},
	}
	if vm.Description != nil && *vm.Description != "" {
		rows = append(rows, []string{"Description", *vm.Description})
	}
	return printer.PrintTable(headers, rows)
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
		wait             bool
	)

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new virtual machine",
		Example: `  zcp instance create --name my-vm --cloud-provider zcp --project default --region yow-1 --template ubuntu-22f --plan compute-4vcpu-8gb --billing-cycle hourly
  zcp instance create --name my-vm --cloud-provider zcp --project default --region yow-1 --template ubuntu-22f --plan compute-4vcpu-8gb --billing-cycle hourly --wait`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if name == "" {
				return fmt.Errorf("--name is required")
			}
			cloudProvider = resolveCloudProvider(cloudProvider)
			if cloudProvider == "" {
				return fmt.Errorf("--cloud-provider is required")
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
			if blockstoragePlan == "" {
				return fmt.Errorf("--blockstorage-plan is required (e.g. 50-gb-2, 100gb — see: zcp plan storage)")
			}

			h := hostname
			if h == "" {
				h = name
			}

			var sshKeyPtr *string
			if sshKey != "" {
				sshKeyPtr = &sshKey
			}

			req := instance.CreateRequest{
				Name:             name,
				CloudProvider:    cloudProvider,
				Project:          project,
				Region:           region,
				BootSource:       "image",
				Server:           "cloud-compute",
				Template:         template,
				IsPublic:         true,
				NetworkType:      networkType,
				Networks:         []string{},
				BillingCycle:     billingCycle,
				SSHKey:           sshKeyPtr,
				Plan:             plan,
				OSFamily:         "Linux",
				TemplateType:     "Operating System",
				Hostname:         h,
				Addons:           []string{},
				StorageCategory:  storageCategory,
				ComputeCategory:  computeCategory,
				BlockstoragePlan: blockstoragePlan,
			}
			return runInstanceCreate(cmd, req, wait)
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "VM name (required)")
	cmd.Flags().StringVar(&cloudProvider, "cloud-provider", "", "Cloud provider slug (required)")
	cmd.Flags().StringVar(&project, "project", "", "Project slug (required)")
	cmd.Flags().StringVar(&region, "region", "", "Region slug (required)")
	cmd.Flags().StringVar(&template, "template", "", "Template slug (required)")
	cmd.Flags().StringVar(&plan, "plan", "", "Plan slug (required)")
	cmd.Flags().StringVar(&billingCycle, "billing-cycle", "", "Billing cycle slug: hourly, monthly, etc. (required)")
	cmd.Flags().StringVar(&networkType, "network-type", "Isolated", "Network type (default: Isolated)")
	cmd.Flags().StringVar(&sshKey, "ssh-key", "", "SSH key name (optional)")
	cmd.Flags().StringVar(&hostname, "hostname", "", "Hostname (defaults to --name)")
	cmd.Flags().StringVar(&storageCategory, "storage-category", "", "Storage category slug (optional)")
	cmd.Flags().StringVar(&computeCategory, "compute-category", "", "Compute category slug (optional)")
	cmd.Flags().StringVar(&blockstoragePlan, "blockstorage-plan", "", "Block storage plan slug, e.g. 50-gb-2 (required)")
	cmd.Flags().BoolVar(&wait, "wait", false, "Wait for the instance to reach Running state")
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
	headers := []string{"SLUG", "NAME", "STATE", "TEMPLATE", "CREATED"}
	rows := [][]string{
		{vm.Slug, vm.Name, vm.State, templateName, vm.CreatedAt},
	}
	return printer.PrintTable(headers, rows)
}

// ---------- start ----------

func newInstanceStartCmd() *cobra.Command {
	var wait bool

	cmd := &cobra.Command{
		Use:     "start <slug>",
		Short:   "Start a stopped virtual machine",
		Args:    cobra.ExactArgs(1),
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

	resp, err := svc.Start(ctx, slug)
	if err != nil {
		return fmt.Errorf("instance start: %w", err)
	}

	if wait {
		fmt.Fprintf(os.Stderr, "Waiting for instance %s to be Running...\n", slug)
		waitCtx, waitCancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd)+300)*time.Second)
		defer waitCancel()
		vm, err := svc.WaitForState(waitCtx, slug, []string{"Running"}, 0)
		if err != nil {
			return fmt.Errorf("waiting for instance start: %w", err)
		}
		fmt.Fprintf(os.Stderr, "Instance %s is now %s\n", slug, vm.State)
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
		Args:    cobra.ExactArgs(1),
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

	resp, err := svc.Stop(ctx, slug)
	if err != nil {
		return fmt.Errorf("instance stop: %w", err)
	}

	if wait {
		fmt.Fprintf(os.Stderr, "Waiting for instance %s to be Stopped...\n", slug)
		waitCtx, waitCancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd)+300)*time.Second)
		defer waitCancel()
		vm, err := svc.WaitForState(waitCtx, slug, []string{"Stopped"}, 0)
		if err != nil {
			return fmt.Errorf("waiting for instance stop: %w", err)
		}
		fmt.Fprintf(os.Stderr, "Instance %s is now %s\n", slug, vm.State)
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
		Args:    cobra.ExactArgs(1),
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

	resp, err := svc.Reboot(ctx, slug)
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
		Args:  cobra.ExactArgs(1),
		Example: `  zcp instance reset my-vm
  zcp instance reset my-vm --yes`,
		RunE: func(cmd *cobra.Command, args []string) error {
			slug := args[0]
			if !yes && !autoApproved(cmd) {
				fmt.Fprintf(os.Stdout, "WARNING: Reset %q will forcefully restart the VM. Unsaved data may be lost. [y/N]: ", slug)
				var answer string
				fmt.Scanln(&answer)
				if strings.ToLower(strings.TrimSpace(answer)) != "y" {
					fmt.Fprintln(os.Stdout, "Aborted.")
					return nil
				}
			}
			return runInstanceReset(cmd, slug)
		},
	}
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "Skip confirmation prompt")
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

	resp, err := svc.Reset(ctx, slug)
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
		Args:    cobra.ExactArgs(1),
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

	logs, err := svc.ActivityLogs(ctx, slug)
	if err != nil {
		return fmt.Errorf("instance logs: %w", err)
	}

	headers := []string{"ID", "ACTION", "STATUS", "DESCRIPTION", "CREATED"}
	rows := make([][]string, 0, len(logs))
	for _, l := range logs {
		rows = append(rows, []string{
			l.ID,
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
		Args:    cobra.ExactArgs(1),
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

	resp, err := svc.CreateTag(ctx, slug, instance.TagRequest{Key: key, Value: value})
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
		Args:    cobra.ExactArgs(1),
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

	if err := svc.DeleteTag(ctx, slug, key); err != nil {
		return fmt.Errorf("instance tag-delete: %w", err)
	}

	fmt.Fprintf(os.Stdout, "Tag %q deleted from instance %q.\n", key, slug)
	return nil
}

// ---------- change-hostname ----------

func newInstanceChangeHostnameCmd() *cobra.Command {
	var hostname string

	cmd := &cobra.Command{
		Use:     "change-hostname <slug>",
		Short:   "Change the hostname of a virtual machine",
		Args:    cobra.ExactArgs(1),
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

	resp, err := svc.ChangeHostname(ctx, slug, instance.ChangeLabelRequest{
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
		Args:    cobra.ExactArgs(1),
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

	resp, err := svc.ChangePassword(ctx, slug, instance.ChangePasswordRequest{
		Password: password,
		VM:       slug,
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
		Args:    cobra.ExactArgs(1),
		Example: `  zcp instance change-plan my-vm --plan box2cm4 --billing-cycle hourly`,
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

	resp, err := svc.ChangePlan(ctx, slug, instance.ChangePlanRequest{
		Plan:         plan,
		Slug:         slug,
		VM:           slug,
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
		Args:  cobra.ExactArgs(1),
		Example: `  zcp instance change-os my-vm --template ubuntu-22f
  zcp instance change-os my-vm --template ubuntu-22f --yes`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if template == "" {
				return fmt.Errorf("--template is required")
			}
			slug := args[0]
			if !yes && !autoApproved(cmd) {
				fmt.Fprintf(os.Stdout, "WARNING: Changing OS on %q will reinstall the VM and erase all data. [y/N]: ", slug)
				var answer string
				fmt.Scanln(&answer)
				if strings.ToLower(strings.TrimSpace(answer)) != "y" {
					fmt.Fprintln(os.Stdout, "Aborted.")
					return nil
				}
			}
			return runInstanceChangeOS(cmd, slug, template)
		},
	}
	cmd.Flags().StringVar(&template, "template", "", "New template slug (required)")
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "Skip confirmation prompt")
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

	resp, err := svc.ChangeOS(ctx, slug, instance.ChangeTemplateRequest{Template: template})
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
		Args:    cobra.ExactArgs(1),
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

	resp, err := svc.ChangeStartupScript(ctx, slug, instance.ChangeStartupScriptRequest{UserData: userData})
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
		Args:    cobra.ExactArgs(1),
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

	resp, err := svc.AddNetwork(ctx, slug, instance.AddNetworkRequest{Network: network})
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
		Args:    cobra.ExactArgs(1),
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

	addons, err := svc.ListAddons(ctx, slug)
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
		Example: `  zcp instance purchase-addon --vm my-vm --project default --region yow-1 --cloud-provider zcp --addon-slug remote-desktop-license --addon-category microsoft-spla-licenses --addon-id <id> --billing-cycle hourly`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if vmSlug == "" {
				return fmt.Errorf("--vm is required")
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
	cmd.Flags().StringVar(&cloudProvider, "cloud-provider", "", "Cloud provider slug (required)")
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

	req := instance.PurchaseAddonRequest{
		VirtualMachine: vmSlug,
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
		Args: cobra.ExactArgs(1),
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

	vm, err := svc.Get(ctx, slug)
	if err != nil {
		return fmt.Errorf("resolving instance IP: %w", err)
	}

	// Prefer private IP; fall back to public IP
	ip := instance.StringVal(vm.PrivateIP)
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
