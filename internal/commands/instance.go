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
	cmd.AddCommand(newInstanceDeleteCmd())
	cmd.AddCommand(newInstanceRebootCmd())
	cmd.AddCommand(newInstanceResizeCmd())
	cmd.AddCommand(newInstanceNetworkListCmd())
	cmd.AddCommand(newInstancePasswordListCmd())
	cmd.AddCommand(newInstanceStatusCmd())
	cmd.AddCommand(newInstanceRecoverCmd())
	cmd.AddCommand(newInstanceRenameCmd())
	cmd.AddCommand(newInstanceSSHCmd())
	return cmd
}

func newInstanceListCmd() *cobra.Command {
	var zoneUUID string

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List instances in a zone",
		Example: `  zcp instance list --zone <uuid>
  zcp instance list --zone <uuid> --output json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if zoneUUID == "" {
				return fmt.Errorf("--zone is required")
			}
			return runInstanceList(cmd, zoneUUID)
		},
	}
	cmd.Flags().StringVar(&zoneUUID, "zone", "", "Zone UUID (required)")
	return cmd
}

func runInstanceList(cmd *cobra.Command, zoneUUID string) error {
	_, client, printer, err := buildClientAndPrinter(cmd)
	if err != nil {
		return err
	}

	svc := instance.NewService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	instances, err := svc.List(ctx, zoneUUID, "")
	if err != nil {
		return fmt.Errorf("instance list: %w", err)
	}

	headers := []string{"UUID", "NAME", "STATE", "PRIVATE IP", "MEMORY", "TEMPLATE", "ZONE"}
	rows := make([][]string, 0, len(instances))
	for _, inst := range instances {
		rows = append(rows, []string{
			inst.UUID,
			inst.Name,
			inst.State,
			inst.PrivateIP,
			inst.Memory,
			inst.TemplateName,
			inst.ZoneUUID,
		})
	}
	return printer.PrintTable(headers, rows)
}

func newInstanceGetCmd() *cobra.Command {
	var zoneUUID string

	cmd := &cobra.Command{
		Use:   "get <uuid>",
		Short: "Get details of a specific instance",
		Args:  cobra.ExactArgs(1),
		Example: `  zcp instance get <uuid> --zone <zone-uuid>
  zcp instance get <uuid> --zone <zone-uuid> --output json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if zoneUUID == "" {
				return fmt.Errorf("--zone is required")
			}
			return runInstanceGet(cmd, args[0], zoneUUID)
		},
	}
	cmd.Flags().StringVar(&zoneUUID, "zone", "", "Zone UUID (required)")
	return cmd
}

func runInstanceGet(cmd *cobra.Command, vmUUID, zoneUUID string) error {
	_, client, printer, err := buildClientAndPrinter(cmd)
	if err != nil {
		return err
	}

	svc := instance.NewService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	inst, err := svc.Get(ctx, zoneUUID, vmUUID)
	if err != nil {
		return fmt.Errorf("instance get: %w", err)
	}

	// Render as two-column key-value table
	headers := []string{"FIELD", "VALUE"}
	rows := [][]string{
		{"UUID", inst.UUID},
		{"Name", inst.Name},
		{"Display Name", inst.DisplayName},
		{"Description", inst.Description},
		{"State", inst.State},
		{"Active", strconv.FormatBool(inst.IsActive)},
		{"Memory", inst.Memory},
		{"Template Name", inst.TemplateName},
		{"Template UUID", inst.TemplateUUID},
		{"Compute Offering UUID", inst.ComputeOfferingUUID},
		{"Storage Offering UUID", inst.StorageOfferingUUID},
		{"Network Name", inst.NetworkName},
		{"Network UUID", inst.NetworkUUID},
		{"Private IP", inst.PrivateIP},
		{"Zone UUID", inst.ZoneUUID},
		{"SSH Key UUID", inst.SSHKeyUUID},
		{"Owner", inst.OwnerName},
		{"Root Disk Size", strconv.FormatInt(inst.RootDiskSize, 10)},
		{"Volume Size", strconv.FormatInt(inst.VolumeSize, 10)},
		{"Disk Size", strconv.FormatInt(inst.DiskSize, 10)},
	}
	return printer.PrintTable(headers, rows)
}

func newInstanceCreateCmd() *cobra.Command {
	var (
		zoneUUID            string
		name                string
		templateUUID        string
		computeOfferingUUID string
		networkUUID         string
		storageOfferingUUID string
		diskSize            int
		rootDiskSize        int
		sshKeyName          string
		securityGroup       string
		wait                bool
	)

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new instance",
		Example: `  zcp instance create --zone <uuid> --name my-vm --template <uuid> --compute-offering <uuid> --network <uuid>
  zcp instance create --zone <uuid> --name my-vm --template <uuid> --compute-offering <uuid> --network <uuid> --ssh-key mykey`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if zoneUUID == "" {
				return fmt.Errorf("--zone is required")
			}
			if name == "" {
				return fmt.Errorf("--name is required")
			}
			if templateUUID == "" {
				return fmt.Errorf("--template is required")
			}
			if computeOfferingUUID == "" {
				return fmt.Errorf("--compute-offering is required")
			}
			if networkUUID == "" {
				return fmt.Errorf("--network is required")
			}
			return runInstanceCreate(cmd, instance.CreateRequest{
				Name:                name,
				ZoneUUID:            zoneUUID,
				TemplateUUID:        templateUUID,
				ComputeOfferingUUID: computeOfferingUUID,
				NetworkUUID:         networkUUID,
				StorageOfferingUUID: storageOfferingUUID,
				DiskSize:            diskSize,
				RootDiskSize:        rootDiskSize,
				SSHKeyName:          sshKeyName,
				SecurityGroupName:   securityGroup,
			}, wait)
		},
	}
	cmd.Flags().StringVar(&zoneUUID, "zone", "", "Zone UUID (required)")
	cmd.Flags().StringVar(&name, "name", "", "Instance name (required)")
	cmd.Flags().StringVar(&templateUUID, "template", "", "Template UUID (required)")
	cmd.Flags().StringVar(&computeOfferingUUID, "compute-offering", "", "Compute offering UUID (required)")
	cmd.Flags().StringVar(&networkUUID, "network", "", "Network UUID (required)")
	cmd.Flags().StringVar(&storageOfferingUUID, "storage-offering", "", "Storage offering UUID (optional)")
	cmd.Flags().IntVar(&diskSize, "disk-size", 0, "Data disk size in GB (optional)")
	cmd.Flags().IntVar(&rootDiskSize, "root-disk-size", 0, "Root disk size in GB (optional)")
	cmd.Flags().StringVar(&sshKeyName, "ssh-key", "", "SSH key name (optional)")
	cmd.Flags().StringVar(&securityGroup, "security-group", "", "Security group name (optional)")
	cmd.Flags().BoolVar(&wait, "wait", false, "Wait for the instance to reach Running state (polls until Running or timeout)")
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

	inst, err := svc.Create(ctx, req)
	if err != nil {
		return fmt.Errorf("instance create: %w", err)
	}

	if wait {
		fmt.Fprintf(os.Stderr, "Waiting for instance %s to be Running...\n", inst.UUID)
		waitCtx, waitCancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd)+300)*time.Second)
		defer waitCancel()
		status, err := svc.WaitForState(waitCtx, inst.UUID, []string{"Running"}, 0)
		if err != nil {
			return fmt.Errorf("waiting for instance create: %w", err)
		}
		fmt.Fprintf(os.Stderr, "Instance %s is now %s\n", inst.UUID, status.Status)
	}

	headers := []string{"UUID", "NAME", "STATE", "PRIVATE IP", "MEMORY", "TEMPLATE", "ZONE"}
	rows := [][]string{
		{inst.UUID, inst.Name, inst.State, inst.PrivateIP, inst.Memory, inst.TemplateName, inst.ZoneUUID},
	}
	return printer.PrintTable(headers, rows)
}

func newInstanceStartCmd() *cobra.Command {
	var wait bool

	cmd := &cobra.Command{
		Use:     "start <uuid>",
		Short:   "Start a stopped instance",
		Args:    cobra.ExactArgs(1),
		Example: `  zcp instance start <uuid>`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runInstanceStart(cmd, args[0], wait)
		},
	}
	cmd.Flags().BoolVar(&wait, "wait", false, "Wait for the instance to reach Running state (polls until Running or timeout)")
	return cmd
}

func runInstanceStart(cmd *cobra.Command, uuid string, wait bool) error {
	_, client, printer, err := buildClientAndPrinter(cmd)
	if err != nil {
		return err
	}

	svc := instance.NewService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	inst, err := svc.Start(ctx, uuid)
	if err != nil {
		return fmt.Errorf("instance start: %w", err)
	}

	if wait {
		fmt.Fprintf(os.Stderr, "Waiting for instance %s to be Running...\n", uuid)
		waitCtx, waitCancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd)+300)*time.Second)
		defer waitCancel()
		status, err := svc.WaitForState(waitCtx, uuid, []string{"Running"}, 0)
		if err != nil {
			return fmt.Errorf("waiting for instance start: %w", err)
		}
		fmt.Fprintf(os.Stderr, "Instance %s is now %s\n", uuid, status.Status)
	}

	headers := []string{"UUID", "NAME", "STATE"}
	rows := [][]string{{inst.UUID, inst.Name, inst.State}}
	return printer.PrintTable(headers, rows)
}

func newInstanceStopCmd() *cobra.Command {
	var force, wait bool

	cmd := &cobra.Command{
		Use:   "stop <uuid>",
		Short: "Stop a running instance",
		Args:  cobra.ExactArgs(1),
		Example: `  zcp instance stop <uuid>
  zcp instance stop <uuid> --force`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runInstanceStop(cmd, args[0], force, wait)
		},
	}
	cmd.Flags().BoolVar(&force, "force", false, "Force stop (bypass graceful shutdown)")
	cmd.Flags().BoolVar(&wait, "wait", false, "Wait for the instance to reach Stopped state (polls until Stopped or timeout)")
	return cmd
}

func runInstanceStop(cmd *cobra.Command, uuid string, force bool, wait bool) error {
	_, client, printer, err := buildClientAndPrinter(cmd)
	if err != nil {
		return err
	}

	svc := instance.NewService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	inst, err := svc.Stop(ctx, uuid, force)
	if err != nil {
		return fmt.Errorf("instance stop: %w", err)
	}

	if wait {
		fmt.Fprintf(os.Stderr, "Waiting for instance %s to be Stopped...\n", uuid)
		waitCtx, waitCancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd)+300)*time.Second)
		defer waitCancel()
		status, err := svc.WaitForState(waitCtx, uuid, []string{"Stopped"}, 0)
		if err != nil {
			return fmt.Errorf("waiting for instance stop: %w", err)
		}
		fmt.Fprintf(os.Stderr, "Instance %s is now %s\n", uuid, status.Status)
	}

	headers := []string{"UUID", "NAME", "STATE"}
	rows := [][]string{{inst.UUID, inst.Name, inst.State}}
	return printer.PrintTable(headers, rows)
}

func newInstanceDeleteCmd() *cobra.Command {
	var yes, expunge bool

	cmd := &cobra.Command{
		Use:   "delete <uuid>",
		Short: "Delete an instance",
		Args:  cobra.ExactArgs(1),
		Example: `  zcp instance delete <uuid>
  zcp instance delete <uuid> --yes
  zcp instance delete <uuid> --expunge --yes`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runInstanceDelete(cmd, args[0], yes, expunge)
		},
	}
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "Skip confirmation prompt")
	cmd.Flags().BoolVar(&expunge, "expunge", false, "Permanently expunge the instance (irreversible)")
	return cmd
}

func runInstanceDelete(cmd *cobra.Command, uuid string, yes, expunge bool) error {
	if !yes {
		if expunge {
			fmt.Fprintf(os.Stdout, "WARNING: --expunge will permanently delete instance %q and cannot be undone.\n", uuid)
		}
		fmt.Fprintf(os.Stdout, "Delete instance %q? [y/N]: ", uuid)
		var answer string
		fmt.Scanln(&answer)
		if strings.ToLower(strings.TrimSpace(answer)) != "y" {
			fmt.Fprintln(os.Stdout, "Aborted.")
			return nil
		}
	}

	_, client, _, err := buildClientAndPrinter(cmd)
	if err != nil {
		return err
	}

	svc := instance.NewService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	if err := svc.Destroy(ctx, uuid, expunge); err != nil {
		return fmt.Errorf("instance delete: %w", err)
	}

	fmt.Fprintf(os.Stdout, "Instance %q deleted.\n", uuid)
	return nil
}

func newInstanceRebootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "reboot <uuid>",
		Short:   "Reboot an instance (stop then start)",
		Args:    cobra.ExactArgs(1),
		Example: `  zcp instance reboot <uuid>`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runInstanceReboot(cmd, args[0])
		},
	}
	return cmd
}

func runInstanceReboot(cmd *cobra.Command, uuid string) error {
	_, client, printer, err := buildClientAndPrinter(cmd)
	if err != nil {
		return err
	}

	svc := instance.NewService(client)
	timeout := time.Duration(getTimeout(cmd)) * time.Second

	fmt.Fprintf(os.Stdout, "Stopping instance %q...\n", uuid)
	stopCtx, stopCancel := context.WithTimeout(context.Background(), timeout)
	defer stopCancel()

	if _, err := svc.Stop(stopCtx, uuid, false); err != nil {
		return fmt.Errorf("instance reboot (stop phase): %w", err)
	}

	fmt.Fprintf(os.Stdout, "Starting instance %q...\n", uuid)
	startCtx, startCancel := context.WithTimeout(context.Background(), timeout)
	defer startCancel()

	inst, err := svc.Start(startCtx, uuid)
	if err != nil {
		return fmt.Errorf("instance reboot (start phase): %w", err)
	}

	headers := []string{"UUID", "NAME", "STATE"}
	rows := [][]string{{inst.UUID, inst.Name, inst.State}}
	return printer.PrintTable(headers, rows)
}

func newInstanceResizeCmd() *cobra.Command {
	var offeringUUID, cpuCores, memory string

	cmd := &cobra.Command{
		Use:   "resize <uuid>",
		Short: "Resize instance compute offering",
		Args:  cobra.ExactArgs(1),
		Example: `  zcp instance resize <uuid> --offering <uuid>
  zcp instance resize <uuid> --offering <uuid> --cpu 4 --memory 8192`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if offeringUUID == "" {
				return fmt.Errorf("--offering is required")
			}
			return runInstanceResize(cmd, args[0], offeringUUID, cpuCores, memory)
		},
	}
	cmd.Flags().StringVar(&offeringUUID, "offering", "", "Compute offering UUID (required)")
	cmd.Flags().StringVar(&cpuCores, "cpu", "", "Number of CPU cores (optional, for custom offerings)")
	cmd.Flags().StringVar(&memory, "memory", "", "Memory in MB (optional, for custom offerings)")
	return cmd
}

func runInstanceResize(cmd *cobra.Command, uuid, offeringUUID, cpuCores, memory string) error {
	_, client, printer, err := buildClientAndPrinter(cmd)
	if err != nil {
		return err
	}

	svc := instance.NewService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	inst, err := svc.Resize(ctx, uuid, offeringUUID, cpuCores, memory)
	if err != nil {
		return fmt.Errorf("instance resize: %w", err)
	}

	headers := []string{"UUID", "NAME", "STATE", "COMPUTE OFFERING", "MEMORY"}
	rows := [][]string{{inst.UUID, inst.Name, inst.State, inst.ComputeOfferingUUID, inst.Memory}}
	return printer.PrintTable(headers, rows)
}

func newInstanceNetworkListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "network-list <uuid>",
		Short:   "List networks attached to an instance",
		Args:    cobra.ExactArgs(1),
		Example: `  zcp instance network-list <uuid>`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runInstanceNetworkList(cmd, args[0])
		},
	}
	return cmd
}

func runInstanceNetworkList(cmd *cobra.Command, uuid string) error {
	_, client, printer, err := buildClientAndPrinter(cmd)
	if err != nil {
		return err
	}

	svc := instance.NewService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	networks, err := svc.ListNetworks(ctx, uuid)
	if err != nil {
		return fmt.Errorf("instance network-list: %w", err)
	}

	headers := []string{"UUID", "NAME", "TYPE", "PRIVATE IP", "PUBLIC IP", "DEFAULT"}
	rows := make([][]string, 0, len(networks))
	for _, n := range networks {
		rows = append(rows, []string{
			n.UUID,
			n.Name,
			n.Type,
			n.PrivateIP,
			n.PublicIP,
			strconv.FormatBool(n.DefaultNetwork),
		})
	}
	return printer.PrintTable(headers, rows)
}

func newInstancePasswordListCmd() *cobra.Command {
	var zoneUUID, instanceUUID string

	cmd := &cobra.Command{
		Use:   "password-list",
		Short: "List instance passwords",
		Example: `  zcp instance password-list --zone <uuid>
  zcp instance password-list --zone <uuid> --instance <uuid>`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if zoneUUID == "" {
				return fmt.Errorf("--zone is required")
			}
			return runInstancePasswordList(cmd, zoneUUID, instanceUUID)
		},
	}
	cmd.Flags().StringVar(&zoneUUID, "zone", "", "Zone UUID (required)")
	cmd.Flags().StringVar(&instanceUUID, "instance", "", "Filter by instance UUID (optional)")
	return cmd
}

func runInstancePasswordList(cmd *cobra.Command, zoneUUID, instanceUUID string) error {
	_, client, printer, err := buildClientAndPrinter(cmd)
	if err != nil {
		return err
	}

	svc := instance.NewService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	passwords, err := svc.ListPasswords(ctx, zoneUUID, instanceUUID)
	if err != nil {
		return fmt.Errorf("instance password-list: %w", err)
	}

	if len(passwords) == 0 {
		fmt.Fprintln(os.Stdout, "No passwords found. Passwords may be empty until the instance is first started.")
		return nil
	}

	headers := []string{"UUID", "PASSWORD"}
	rows := make([][]string, 0, len(passwords))
	for _, p := range passwords {
		rows = append(rows, []string{p.UUID, p.Password})
	}
	return printer.PrintTable(headers, rows)
}

func newInstanceStatusCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "status <uuid>",
		Short:   "Get the current status of an instance",
		Args:    cobra.ExactArgs(1),
		Example: `  zcp instance status <uuid>`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runInstanceStatus(cmd, args[0])
		},
	}
	return cmd
}

func runInstanceStatus(cmd *cobra.Command, uuid string) error {
	_, client, printer, err := buildClientAndPrinter(cmd)
	if err != nil {
		return err
	}

	svc := instance.NewService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	status, err := svc.GetStatus(ctx, uuid)
	if err != nil {
		return fmt.Errorf("instance status: %w", err)
	}

	headers := []string{"UUID", "STATUS"}
	rows := [][]string{{status.UUID, status.Status}}
	return printer.PrintTable(headers, rows)
}

func newInstanceRecoverCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "recover <uuid>",
		Short:   "Recover an instance from an error state",
		Args:    cobra.ExactArgs(1),
		Example: `  zcp instance recover <uuid>`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runInstanceRecover(cmd, args[0])
		},
	}
	return cmd
}

func runInstanceRecover(cmd *cobra.Command, uuid string) error {
	_, client, printer, err := buildClientAndPrinter(cmd)
	if err != nil {
		return err
	}

	svc := instance.NewService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	inst, err := svc.Recover(ctx, uuid)
	if err != nil {
		return fmt.Errorf("instance recover: %w", err)
	}

	headers := []string{"UUID", "NAME", "STATE"}
	rows := [][]string{{inst.UUID, inst.Name, inst.State}}
	return printer.PrintTable(headers, rows)
}

func newInstanceRenameCmd() *cobra.Command {
	var displayName string

	cmd := &cobra.Command{
		Use:     "rename <uuid>",
		Short:   "Update an instance's display name",
		Args:    cobra.ExactArgs(1),
		Example: `  zcp instance rename <uuid> --display-name "My Web Server"`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if displayName == "" {
				return fmt.Errorf("--display-name is required")
			}
			return runInstanceRename(cmd, args[0], displayName)
		},
	}
	cmd.Flags().StringVar(&displayName, "display-name", "", "New display name for the instance (required)")
	return cmd
}

func runInstanceRename(cmd *cobra.Command, uuid, displayName string) error {
	_, client, printer, err := buildClientAndPrinter(cmd)
	if err != nil {
		return err
	}

	svc := instance.NewService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	inst, err := svc.Rename(ctx, uuid, displayName)
	if err != nil {
		return fmt.Errorf("instance rename: %w", err)
	}

	headers := []string{"UUID", "NAME", "DISPLAY NAME", "STATE"}
	rows := [][]string{{inst.UUID, inst.Name, inst.DisplayName, inst.State}}
	return printer.PrintTable(headers, rows)
}

func newInstanceSSHCmd() *cobra.Command {
	var user, identityFile string
	var port int

	cmd := &cobra.Command{
		Use:   "ssh <uuid>",
		Short: "Open an SSH session to an instance",
		Long: `Open an SSH session to an instance by resolving its IP address via the API.

The CLI looks up the instance's attached networks and connects to the first
available private IP address. If no private IP is found, it falls back to a
public IP. The default SSH user is "root"; use --user to override.

Requirements:
  - ssh must be installed and available in your PATH
  - The instance must be reachable from your local machine (VPN or public IP)`,
		Args: cobra.ExactArgs(1),
		Example: `  zcp instance ssh <uuid>
  zcp instance ssh <uuid> --user ubuntu
  zcp instance ssh <uuid> --user root --identity-file ~/.ssh/my-key.pem`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runInstanceSSH(cmd, args[0], user, identityFile, port)
		},
	}
	cmd.Flags().StringVar(&user, "user", "root", "SSH username")
	cmd.Flags().StringVarP(&identityFile, "identity-file", "i", "", "Path to SSH private key file")
	cmd.Flags().IntVar(&port, "port", 22, "SSH port")
	return cmd
}

func runInstanceSSH(cmd *cobra.Command, uuid, user, identityFile string, port int) error {
	_, client, _, err := buildClientAndPrinter(cmd)
	if err != nil {
		return err
	}

	svc := instance.NewService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	networks, err := svc.ListNetworks(ctx, uuid)
	if err != nil {
		return fmt.Errorf("resolving instance network: %w", err)
	}
	if len(networks) == 0 {
		return fmt.Errorf("instance %s has no attached networks", uuid)
	}

	// Prefer private IP; fall back to public IP
	ip := ""
	for _, n := range networks {
		if n.PrivateIP != "" {
			ip = n.PrivateIP
			break
		}
	}
	if ip == "" {
		for _, n := range networks {
			if n.PublicIP != "" {
				ip = n.PublicIP
				break
			}
		}
	}
	if ip == "" {
		return fmt.Errorf("instance %s has no usable IP address", uuid)
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
