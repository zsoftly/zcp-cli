package commands

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/zsoftly/zcp-cli/pkg/api/apierrors"
	"github.com/zsoftly/zcp-cli/pkg/api/billing"
	"github.com/zsoftly/zcp-cli/pkg/api/kubernetes"
)

// NewKubernetesCmd returns the 'kubernetes' cobra command.
func NewKubernetesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "kubernetes",
		Aliases: []string{"k8s"},
		Short:   "Manage Kubernetes clusters (alias: k8s)",
	}
	cmd.AddCommand(newK8sClusterListCmd())
	cmd.AddCommand(newK8sClusterGetCmd())
	cmd.AddCommand(newK8sClusterCreateCmd())
	cmd.AddCommand(newK8sClusterStartCmd())
	cmd.AddCommand(newK8sClusterStopCmd())
	cmd.AddCommand(newK8sClusterUpgradeCmd())
	cmd.AddCommand(newK8sClusterDeleteCmd())
	cmd.AddCommand(newK8sGetConfigCmd())
	cmd.AddCommand(newK8sClusterScaleCmd())
	cmd.AddCommand(newK8sClusterUpgradeVersionCmd())
	return cmd
}

func newK8sClusterListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List Kubernetes clusters",
		Example: `  zcp kubernetes list
  zcp k8s list --output json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runK8sClusterList(cmd)
		},
	}
	return cmd
}

func runK8sClusterList(cmd *cobra.Command) error {
	_, client, printer, err := buildClientAndPrinter(cmd)
	if err != nil {
		return err
	}

	svc := kubernetes.NewService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	region, project := scopedRegionProject(cmd)
	clusters, err := svc.List(ctx, region, project)
	if err != nil {
		return fmt.Errorf("kubernetes list: %w", err)
	}

	headers := []string{"SLUG", "NAME", "STATE", "VERSION", "WORKERS", "CONTROL NODES", "HA", "REGION", "CREATED"}
	rows := make([][]string, 0, len(clusters))
	for _, c := range clusters {
		regionName := ""
		if c.Region != nil {
			regionName = c.Region.Name
		}
		rows = append(rows, []string{
			c.Slug,
			c.Name,
			c.State,
			c.Version,
			strconv.Itoa(k8sWorkerNodeCount(c)),
			strconv.Itoa(c.ControlNodes),
			strconv.FormatBool(c.EnableHA),
			regionName,
			c.CreatedAt,
		})
	}
	return printer.PrintTable(headers, rows)
}

func newK8sClusterGetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "get <slug>",
		Short:   "Show details for a Kubernetes cluster",
		Args:    exactArgs(1),
		Example: `  zcp kubernetes get my-cluster`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runK8sClusterGet(cmd, args[0])
		},
	}
	return cmd
}

func runK8sClusterGet(cmd *cobra.Command, slug string) error {
	_, client, printer, err := buildClientAndPrinter(cmd)
	if err != nil {
		return err
	}

	svc := kubernetes.NewService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	c, err := svc.Get(ctx, slug)
	if err != nil {
		return fmt.Errorf("kubernetes get: %w", err)
	}

	publicIP := ""
	if c.PublicIP != nil {
		publicIP = *c.PublicIP
	}
	privateIP := ""
	if c.PrivateIP != nil {
		privateIP = *c.PrivateIP
	}
	regionName := ""
	if c.Region != nil {
		regionName = c.Region.Name
	}
	version := c.Version
	workers := strconv.Itoa(k8sWorkerNodeCount(*c))
	controlNodes := strconv.Itoa(c.ControlNodes)
	endpoint := ""

	// Prefer the platform-side meta fields — they populate after the cluster is Running.
	if m := c.Meta; m != nil {
		if m.KubernetesVersionName != "" {
			version = m.KubernetesVersionName
		}
		if m.Size != "" && m.Size != "0" {
			workers = m.Size
		}
		if m.ControlNodes != "" && m.ControlNodes != "0" {
			controlNodes = m.ControlNodes
		}
		if m.IPAddress != "" {
			publicIP = m.IPAddress
		}
		if m.Endpoint != "" {
			endpoint = m.Endpoint
		}
	}

	headers := []string{"FIELD", "VALUE"}
	rows := [][]string{
		{"Slug", c.Slug},
		{"Name", c.Name},
		{"State", c.State},
		{"Version", version},
		{"Workers", workers},
		{"Control Nodes", controlNodes},
		{"HA", strconv.FormatBool(c.EnableHA)},
		{"Public IP", publicIP},
		{"Private IP", privateIP},
		{"Endpoint", endpoint},
		{"Region", regionName},
		{"Created", c.CreatedAt},
	}
	if c.Meta != nil {
		rows = append(rows,
			[]string{"Total CPU", c.Meta.CPU},
			[]string{"Total RAM", k8sMemoryString(c.Meta.FormattedMemory, c.Meta.Memory)},
		)
		autoscaling := k8sAutoscalingStatus(c.Meta.AutoscalingEnabled, c.Autoscale != nil)
		rows = append(rows, []string{"Autoscaling", autoscaling})
		if autoscaling == "Enabled" {
			rows = append(rows,
				[]string{"Minimum workers", k8sRawValue(c.Meta.MinSize)},
				[]string{"Maximum workers", k8sRawValue(c.Meta.MaxSize)},
			)
		}
	}
	if c.Offering != nil {
		controlPlan, controlCPU, controlMemory, controlStorage := k8sPlanDetails(
			c.Offering.MasterPlan,
			c.Offering.MasterCustomCPU,
			c.Offering.MasterCustomMemory,
			c.Offering.MasterCustomStorage,
			c.Offering.FormattedMasterCustomMemory,
			c.Offering.FormattedMasterCustomStorage,
			c.Offering.CPU,
			c.Offering.Memory,
			c.Offering.Storage,
			c.Offering.FormattedMemory,
			c.Offering.FormattedStorage,
		)
		workerPlan, workerCPU, workerMemory, workerStorage := k8sPlanDetails(
			c.Offering.WorkerPlan,
			c.Offering.WorkerCustomCPU,
			c.Offering.WorkerCustomMemory,
			c.Offering.WorkerCustomStorage,
			c.Offering.FormattedWorkerCustomMemory,
			c.Offering.FormattedWorkerCustomStorage,
			c.Offering.CPU,
			c.Offering.Memory,
			c.Offering.Storage,
			c.Offering.FormattedMemory,
			c.Offering.FormattedStorage,
		)
		rows = append(rows,
			[]string{"Control-plane plan", controlPlan},
			[]string{"Control-plane CPU", controlCPU},
			[]string{"Control-plane memory", controlMemory},
			[]string{"Control-plane plan storage", controlStorage},
			[]string{"Worker plan", workerPlan},
			[]string{"Worker CPU", workerCPU},
			[]string{"Worker memory", workerMemory},
			[]string{"Worker plan storage", workerStorage},
		)
	}
	rootVolumes := 0
	for _, volume := range c.BlockStorages {
		if !volume.IsRoot {
			continue
		}
		rootVolumes++
		rows = append(rows, []string{
			"Root volume " + k8sFirstNonEmpty(volume.Name, volume.Slug, "-"),
			k8sFirstNonEmpty(k8sStorageValue("", volume.Size), "-"),
		})
	}
	if rootVolumes > 0 {
		rows = append(rows, []string{"Root volumes", strconv.Itoa(rootVolumes)})
	}
	if c.Network != nil {
		rows = append(rows, []string{"Network", k8sFirstNonEmpty(c.Network.Name, c.Network.Slug)})
	}
	return printer.PrintTable(headers, rows)
}

func k8sWorkerNodeCount(cluster kubernetes.Cluster) int {
	if cluster.WorkerNodeSize > 0 {
		return cluster.WorkerNodeSize
	}
	return cluster.NodeSize
}

func k8sObservedWorkerCount(cluster *kubernetes.Cluster) int {
	workers := k8sWorkerNodeCount(*cluster)
	if cluster.Meta != nil && cluster.Meta.Size != "" {
		if n, err := strconv.Atoi(cluster.Meta.Size); err == nil {
			workers = n
		}
	}
	return workers
}

func k8sRawValue(raw json.RawMessage) string {
	if len(raw) == 0 || string(raw) == "null" {
		return ""
	}
	var value interface{}
	if err := json.Unmarshal(raw, &value); err != nil || value == nil {
		return ""
	}
	return fmt.Sprint(value)
}

func k8sAutoscalingEnabled(raw json.RawMessage) bool {
	switch strings.ToLower(k8sRawValue(raw)) {
	case "", "0", "false", "no":
		return false
	default:
		return true
	}
}

func k8sAutoscalingStatus(raw json.RawMessage, autoscalePresent bool) string {
	if autoscalePresent {
		return "Enabled"
	}
	if len(raw) == 0 || string(raw) == "null" {
		return "Unknown"
	}
	if k8sAutoscalingEnabled(raw) {
		return "Enabled"
	}
	return "Disabled"
}

func k8sFirstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func k8sPlanDetails(plan *kubernetes.NodePlan, customCPU, customMemory, customStorage json.RawMessage, formattedCustomMemory, formattedCustomStorage string, fallbackCPU, fallbackMemory, fallbackStorage json.RawMessage, formattedFallbackMemory, formattedFallbackStorage string) (name, cpu, memory, storage string) {
	if plan != nil {
		name = k8sFirstNonEmpty(plan.Name, plan.Slug)
		if plan.Attribute != nil {
			cpu = k8sCPUValue(k8sRawValue(plan.Attribute.FormattedCPU), plan.Attribute.CPU)
			memory = k8sMemoryValue(plan.Attribute.FormattedMemory, plan.Attribute.Memory)
			storage = k8sStorageValue(plan.Attribute.FormattedStorage, plan.Attribute.Storage)
		}
	}
	cpu = k8sFirstNonEmpty(cpu, k8sCPUValue("", customCPU))
	memory = k8sFirstNonEmpty(memory, k8sMemoryValue(formattedCustomMemory, customMemory))
	storage = k8sFirstNonEmpty(storage, k8sStorageValue(formattedCustomStorage, customStorage))
	if plan == nil {
		cpu = k8sFirstNonEmpty(cpu, k8sCPUValue("", fallbackCPU))
		memory = k8sFirstNonEmpty(memory, k8sMemoryValue(formattedFallbackMemory, fallbackMemory))
		storage = k8sFirstNonEmpty(storage, k8sStorageValue(formattedFallbackStorage, fallbackStorage))
	}
	if name == "" && (cpu != "" || memory != "" || storage != "") {
		name = "Custom"
	}
	return k8sFirstNonEmpty(name, "-"), k8sFirstNonEmpty(cpu, "-"), k8sFirstNonEmpty(memory, "-"), k8sFirstNonEmpty(storage, "-")
}

func k8sCPUValue(formatted string, raw json.RawMessage) string {
	if formatted != "" {
		if _, err := strconv.ParseFloat(formatted, 64); err == nil {
			return formatted + " Cores"
		}
		return formatted
	}
	if value := k8sRawValue(raw); value != "" {
		return value + " Cores"
	}
	return ""
}

func k8sMemoryValue(formatted string, raw json.RawMessage) string {
	return k8sMemoryString(formatted, k8sRawValue(raw))
}

func k8sMemoryString(formatted, value string) string {
	if formatted != "" {
		return formatted
	}
	if value == "" {
		return ""
	}
	amount, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return value + " MB"
	}
	if amount >= 1024 {
		return fmt.Sprintf("%g GB", amount/1024)
	}
	return fmt.Sprintf("%g MB", amount)
}

func k8sStorageValue(formatted string, raw json.RawMessage) string {
	if formatted != "" {
		return formatted
	}
	if value := k8sRawValue(raw); value != "" {
		return value + " GB"
	}
	return ""
}

func newK8sClusterCreateCmd() *cobra.Command {
	var (
		name               string
		version            string
		nodeSize           int
		controlNodes       int
		cloudProvider      string
		cloudProviderSetup string
		region             string
		project            string
		billingCycle       string
		enableHA           bool
		enableCSI          bool
		controlPlanePlan   string
		workerPlan         string
		storagePlan        string
		rootDiskSize       int
		storageCategory    string
		sshKey             string
		authMethod         string
		username           string
		password           string
	)

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new Kubernetes cluster",
		Example: `  zcp kubernetes create --name my-cluster --version v1.37.0 --control-plane-plan k8s-cpi-yul --worker-plan k8s-li-yul --storage-plan b2g1 --root-disk-size 100 --storage-category pro-nvme --region yul-1 --project default-9 --billing-cycle hourly --workers 3 --ssh-key mykey
		  zcp kubernetes create --name ha-cluster --version v1.37.0 --control-plane-plan k8s-cpi-yul --worker-plan k8s-4xli-yul --storage-plan b2g1 --root-disk-size 100 --storage-category pro-nvme --region yul-1 --project default-9 --billing-cycle hourly --workers 3 --control-nodes 3 --ha --ssh-key mykey`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if name == "" {
				return fmt.Errorf("--name is required")
			}
			if version == "" {
				return fmt.Errorf("--version is required")
			}
			if controlPlanePlan == "" {
				return fmt.Errorf("--control-plane-plan is required")
			}
			if workerPlan == "" {
				return fmt.Errorf("--worker-plan is required")
			}
			if storagePlan == "" {
				return fmt.Errorf("--storage-plan is required")
			}
			if storagePlan == "custom_plan" {
				return fmt.Errorf("--storage-plan must name a storage tier; custom_plan is not supported")
			}
			if !cmd.Flags().Changed("root-disk-size") {
				return fmt.Errorf("--root-disk-size is required when --storage-plan is set")
			}
			if rootDiskSize <= 0 {
				return fmt.Errorf("--root-disk-size must be > 0")
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
			if billingCycle == "" {
				return fmt.Errorf("--billing-cycle is required")
			}

			if nodeSize < 1 {
				return fmt.Errorf("--workers must be >= 1")
			}
			if storageCategory == "" {
				return fmt.Errorf("--storage-category is required (e.g. pro-nvme, nvme, ssd)")
			}
			if sshKey == "" && authMethod == "ssh-key" {
				return fmt.Errorf("--ssh-key is required when --auth-method is ssh-key")
			}
			if enableHA && controlNodes < 2 {
				return fmt.Errorf("--control-nodes must be >= 2 when --ha is set")
			}
			fixedPlan := false
			return runK8sClusterCreate(cmd, kubernetes.CreateRequest{
				Name:                   name,
				Version:                version,
				NodeSize:               nodeSize,
				WorkerNodeSize:         nodeSize,
				ControlNodes:           controlNodes,
				CloudProvider:          cloudProvider,
				CloudProviderSetup:     cloudProviderSetup,
				Region:                 region,
				Project:                project,
				BillingCycle:           billingCycle,
				EnableHA:               enableHA,
				EnableCSI:              enableCSI,
				Networks:               []string{},
				MasterPlan:             controlPlanePlan,
				WorkerPlan:             workerPlan,
				BlockstoragePlan:       storagePlan,
				BlockstorageCustomPlan: &kubernetes.BlockstorageCustomPlan{Storage: rootDiskSize},
				IsK8sCustomPlan:        &fixedPlan,
				MasterCustomPlan:       json.RawMessage("null"),
				IsK8sMasterCustomPlan:  &fixedPlan,
				WorkerCustomPlan:       json.RawMessage("null"),
				IsK8sWorkerCustomPlan:  &fixedPlan,
				WithPoolCard:           false,
				IsCustomPlan:           false,
				CustomPlan:             nil,
				VirtualMachine:         "",
				Coupon:                 nil,
				StorageCategory:        storageCategory,
				SSHKey:                 sshKey,
				AuthMethod:             authMethod,
				Username:               username,
				Password:               password,
			})
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "Cluster name (required)")
	cmd.Flags().StringVar(&version, "version", "", "Kubernetes version, e.g. v1.37.0 (required)")
	cmd.Flags().IntVar(&nodeSize, "workers", 0, "Number of worker nodes (required, >= 1)")
	cmd.Flags().IntVar(&controlNodes, "control-nodes", 1, "Number of control plane nodes (default 1)")
	cmd.Flags().StringVar(&cloudProvider, "cloud-provider", "", "Cloud provider slug (optional; auto-detected, override only)")
	cmd.Flags().StringVar(&cloudProviderSetup, "cloud-provider-setup", "", "Cloud provider setup slug, e.g. default-setup")
	cmd.Flags().StringVar(&region, "region", "", "Region slug (required)")
	cmd.Flags().StringVar(&project, "project", "", "Project slug (required)")
	cmd.Flags().StringVar(&billingCycle, "billing-cycle", "", "Billing cycle slug, e.g. hourly, monthly (required)")
	cmd.Flags().BoolVar(&enableHA, "ha", false, "Enable high availability")
	cmd.Flags().BoolVar(&enableCSI, "enable-csi", false, "Enable Cloud Storage Integration (CSI)")
	cmd.Flags().StringVar(&controlPlanePlan, "control-plane-plan", "", "Control-plane node plan slug (required)")
	cmd.Flags().StringVar(&workerPlan, "worker-plan", "", "Worker node plan slug (required)")
	cmd.Flags().StringVar(&storagePlan, "storage-plan", "", "Root-volume plan slug applied to each control-plane and worker node (required)")
	cmd.Flags().IntVar(&rootDiskSize, "root-disk-size", 0, "Root-volume capacity in GB for the selected storage tier (required)")
	cmd.Flags().StringVar(&storageCategory, "storage-category", "", "Storage category slug, e.g. pro-nvme, nvme, ssd (required)")
	cmd.Flags().StringVar(&sshKey, "ssh-key", "", "SSH key name")
	cmd.Flags().StringVar(&authMethod, "auth-method", "ssh-key", "Authentication method: ssh-key or password")
	cmd.Flags().StringVar(&username, "username", "", "Username for password auth (optional)")
	cmd.Flags().StringVar(&password, "password", "", "Password for password auth (optional)")
	return cmd
}

func runK8sClusterCreate(cmd *cobra.Command, req kubernetes.CreateRequest) error {
	_, client, printer, err := buildClientAndPrinter(cmd)
	if err != nil {
		return err
	}

	svc := kubernetes.NewService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	cluster, err := svc.Create(ctx, req)
	if err != nil {
		return fmt.Errorf("kubernetes create: %w", err)
	}

	headers := []string{"SLUG", "NAME", "STATE", "VERSION", "WORKERS", "CONTROL NODES", "HA"}
	rows := [][]string{{
		cluster.Slug,
		cluster.Name,
		cluster.State,
		cluster.Version,
		strconv.Itoa(k8sWorkerNodeCount(*cluster)),
		strconv.Itoa(cluster.ControlNodes),
		strconv.FormatBool(cluster.EnableHA),
	}}
	return printer.PrintTable(headers, rows)
}

func newK8sClusterStartCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "start <slug>",
		Short:   "Start a stopped Kubernetes cluster",
		Args:    exactArgs(1),
		Example: `  zcp kubernetes start my-cluster`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runK8sClusterStart(cmd, args[0])
		},
	}
	return cmd
}

func runK8sClusterStart(cmd *cobra.Command, slug string) error {
	_, client, _, err := buildClientAndPrinter(cmd)
	if err != nil {
		return err
	}

	svc := kubernetes.NewService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	if err := svc.Start(ctx, slug); err != nil {
		return fmt.Errorf("kubernetes start: %w", err)
	}

	fmt.Fprintf(os.Stdout, "Kubernetes cluster %q start requested.\n", slug)
	return nil
}

func newK8sClusterStopCmd() *cobra.Command {
	var yes bool

	cmd := &cobra.Command{
		Use:   "stop <slug>",
		Short: "Stop a running Kubernetes cluster",
		Args:  exactArgs(1),
		Example: `  zcp kubernetes stop my-cluster
  zcp kubernetes stop my-cluster --yes`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runK8sClusterStop(cmd, args[0], yes)
		},
	}
	cmd.Flags().BoolVar(&yes, "yes", false, "Skip confirmation prompt")
	return cmd
}

func runK8sClusterStop(cmd *cobra.Command, slug string, yes bool) error {
	if !yes && !autoApproved(cmd) {
		fmt.Fprintf(os.Stderr, "Stop cluster %q? [y/N]: ", slug)
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

	svc := kubernetes.NewService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	if err := svc.Stop(ctx, slug); err != nil {
		return fmt.Errorf("kubernetes stop: %w", err)
	}

	fmt.Fprintf(os.Stdout, "Kubernetes cluster %q stop requested.\n", slug)
	return nil
}

func newK8sClusterScaleCmd() *cobra.Command {
	var (
		workers            int
		minWorkers         int
		maxWorkers         int
		enableAutoscaling  bool
		disableAutoscaling bool
		wait               bool
	)

	cmd := &cobra.Command{
		Use:   "scale <slug>",
		Short: "Scale workers or configure worker autoscaling on a Kubernetes cluster",
		Args:  exactArgs(1),
		Example: `  zcp kubernetes scale my-cluster --workers 5
	  zcp kubernetes scale my-cluster --enable-autoscaling --min-workers 2 --max-workers 5
	  zcp kubernetes scale my-cluster --disable-autoscaling --workers 3 --wait`,
		RunE: func(cmd *cobra.Command, args []string) error {
			slug := args[0]
			if enableAutoscaling && disableAutoscaling {
				return fmt.Errorf("--enable-autoscaling and --disable-autoscaling cannot be used together")
			}
			if enableAutoscaling {
				if workers != 0 {
					return fmt.Errorf("--workers cannot be used with --enable-autoscaling")
				}
				if minWorkers < 1 {
					return fmt.Errorf("--min-workers must be >= 1 when enabling autoscaling")
				}
				if maxWorkers < minWorkers {
					return fmt.Errorf("--max-workers must be >= --min-workers when enabling autoscaling")
				}
				if wait {
					return fmt.Errorf("--wait cannot be used with --enable-autoscaling")
				}
			} else if workers < 1 {
				return fmt.Errorf("--workers must be >= 1")
			}
			if !enableAutoscaling && (minWorkers != 0 || maxWorkers != 0) {
				return fmt.Errorf("--min-workers and --max-workers require --enable-autoscaling")
			}
			_, client, _, err := buildClientAndPrinter(cmd)
			if err != nil {
				return err
			}
			svc := kubernetes.NewService(client)
			ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
			defer cancel()

			current, err := svc.Get(ctx, slug)
			if err != nil {
				return fmt.Errorf("kubernetes scale: %w", err)
			}
			switch current.State {
			case "Running", "Scaling":
			default:
				return fmt.Errorf("cluster %q is in state %q — scale requires Running or Scaling state", slug, current.State)
			}
			if enableAutoscaling {
				if err := svc.EnableAutoscaling(ctx, slug, minWorkers, maxWorkers); err != nil {
					return fmt.Errorf("kubernetes scale: %w", err)
				}
				fmt.Fprintf(os.Stdout, "Enabling autoscaling for %q with %d–%d worker(s) requested.\n", slug, minWorkers, maxWorkers)
				return nil
			}

			currentWorkers := k8sObservedWorkerCount(current)
			if !disableAutoscaling && currentWorkers == workers {
				fmt.Fprintf(os.Stdout, "Cluster %q already has %d worker(s) — no change made.\n", slug, workers)
				return nil
			}

			if disableAutoscaling {
				err = svc.DisableAutoscaling(ctx, slug, workers)
			} else {
				err = svc.Scale(ctx, slug, workers)
			}
			if err != nil {
				return fmt.Errorf("kubernetes scale: %w", err)
			}
			if disableAutoscaling {
				fmt.Fprintf(os.Stdout, "Disabling autoscaling for %q with %d worker(s) requested.\n", slug, workers)
			} else {
				fmt.Fprintf(os.Stdout, "Scaling %q from %d → %d worker(s) requested.\n", slug, currentWorkers, workers)
			}

			if wait {
				const maxWait = 10 * time.Minute
				waitCtx, waitCancel := context.WithTimeout(cmd.Context(), maxWait)
				defer waitCancel()
				fmt.Fprintf(os.Stdout, "Waiting for scaling to complete (max %s)...\n", maxWait)
				for {
					select {
					case <-waitCtx.Done():
						return fmt.Errorf("timed out waiting for cluster %q to finish scaling", slug)
					case <-time.After(15 * time.Second):
					}
					c, err := svc.Get(waitCtx, slug)
					if err != nil {
						return fmt.Errorf("polling cluster state: %w", err)
					}
					workerCount := k8sObservedWorkerCount(c)
					switch c.State {
					case "Scaling":
						// still in progress
					case "Running":
						if k8sScaleComplete(c, workers, disableAutoscaling) {
							fmt.Fprintf(os.Stdout, "Done — state: %s, workers: %d\n", c.State, workerCount)
							return nil
						}
					default:
						return fmt.Errorf("cluster %q entered unexpected state %q during scaling", slug, c.State)
					}
				}
			}

			fmt.Fprintf(os.Stdout, "To check progress:  zcp kubernetes get %s\n", slug)
			return nil
		},
	}
	cmd.Flags().IntVar(&workers, "workers", 0, "Target worker count; required unless enabling autoscaling")
	cmd.Flags().BoolVar(&enableAutoscaling, "enable-autoscaling", false, "Enable worker autoscaling")
	cmd.Flags().BoolVar(&disableAutoscaling, "disable-autoscaling", false, "Disable worker autoscaling")
	cmd.Flags().IntVar(&minWorkers, "min-workers", 0, "Minimum workers when enabling autoscaling")
	cmd.Flags().IntVar(&maxWorkers, "max-workers", 0, "Maximum workers when enabling autoscaling")
	cmd.Flags().BoolVar(&wait, "wait", false, "Block until scaling completes")
	return cmd
}

func k8sScaleComplete(cluster *kubernetes.Cluster, workers int, autoscalingDisabled bool) bool {
	if cluster.State != "Running" || k8sObservedWorkerCount(cluster) != workers {
		return false
	}
	if autoscalingDisabled && cluster.Autoscale != nil {
		return false
	}
	if autoscalingDisabled && cluster.Meta != nil && len(cluster.Meta.AutoscalingEnabled) > 0 && string(cluster.Meta.AutoscalingEnabled) != "null" && k8sAutoscalingEnabled(cluster.Meta.AutoscalingEnabled) {
		return false
	}
	return true
}

func newK8sGetConfigCmd() *cobra.Command {
	var (
		outputPath string
		print      bool
	)

	cmd := &cobra.Command{
		Use:   "get-config <slug>",
		Short: "Download the kubeconfig for a Kubernetes cluster",
		Args:  exactArgs(1),
		Example: `  zcp kubernetes get-config my-cluster                      # prints kubeconfig to stdout
  zcp kubernetes get-config my-cluster --output ~/.kube/zcp  # saves to a file`,
		RunE: func(cmd *cobra.Command, args []string) error {
			slug := args[0]
			_, client, _, err := buildClientAndPrinter(cmd)
			if err != nil {
				return err
			}

			svc := kubernetes.NewService(client)
			ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
			defer cancel()

			cfg, err := svc.GetKubeconfig(ctx, slug)
			if err != nil {
				return fmt.Errorf("kubernetes get-config: %w", err)
			}

			if print || outputPath == "" {
				fmt.Fprint(os.Stdout, cfg)
				return nil
			}

			if dir := filepath.Dir(outputPath); dir != "." {
				if err := os.MkdirAll(dir, 0700); err != nil {
					return fmt.Errorf("creating directory: %w", err)
				}
			}
			if err := os.WriteFile(outputPath, []byte(cfg), 0600); err != nil {
				return fmt.Errorf("writing kubeconfig to %s: %w", outputPath, err)
			}
			fmt.Fprintf(os.Stdout, "Kubeconfig written to %s\n", outputPath)
			fmt.Fprintf(os.Stdout, "  export KUBECONFIG=%s\n", outputPath)
			return nil
		},
	}
	cmd.Flags().StringVarP(&outputPath, "output", "o", "", "Write kubeconfig to this file path (default: print to stdout)")
	cmd.Flags().BoolVar(&print, "print", false, "Print kubeconfig to stdout even when --output is set")
	return cmd
}

func newK8sClusterUpgradeCmd() *cobra.Command {
	var (
		plan         string
		billingCycle string
	)

	cmd := &cobra.Command{
		Use:   "upgrade <slug>",
		Short: "Upgrade (change plan of) a Kubernetes cluster",
		Args:  exactArgs(1),
		Example: `  zcp kubernetes upgrade my-cluster --plan k8s-xla-yul-1
  zcp kubernetes upgrade my-cluster --plan k8s-xla-yul-1 --billing-cycle hourly`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if plan == "" {
				return fmt.Errorf("--plan is required")
			}
			return runK8sClusterUpgrade(cmd, args[0], plan, billingCycle)
		},
	}
	cmd.Flags().StringVar(&plan, "plan", "", "New plan slug (required)")
	cmd.Flags().StringVar(&billingCycle, "billing-cycle", "", "Billing cycle slug, e.g. hourly, monthly (optional)")
	return cmd
}

func newK8sClusterDeleteCmd() *cobra.Command {
	var yes bool

	cmd := &cobra.Command{
		Use:   "delete <slug>",
		Short: "Permanently delete a Kubernetes cluster",
		Args:  exactArgs(1),
		Example: `  zcp kubernetes delete my-cluster
  zcp kubernetes delete my-cluster --yes`,
		RunE: func(cmd *cobra.Command, args []string) error {
			slug := args[0]
			if !yes && !autoApproved(cmd) {
				fmt.Fprintf(os.Stderr, "Delete Kubernetes cluster %q? This cannot be undone. [y/N]: ", slug)
				scanner := bufio.NewScanner(os.Stdin)
				scanner.Scan()
				answer := strings.TrimSpace(strings.ToLower(scanner.Text()))
				if answer != "y" && answer != "yes" {
					fmt.Fprintln(os.Stderr, "Aborted.")
					return nil
				}
			}
			return runK8sClusterDelete(cmd, slug)
		},
	}
	cmd.Flags().BoolVar(&yes, "yes", false, "Skip confirmation prompt")
	return cmd
}

func runK8sClusterDelete(cmd *cobra.Command, slug string) error {
	_, client, printer, err := buildClientAndPrinter(cmd)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	cluster, err := kubernetes.NewService(client).Get(ctx, slug)
	if err != nil {
		if apierrors.IsResourceNotFound(err) {
			fmt.Fprintf(os.Stderr, "Kubernetes cluster %q not found — already deleted.\n", slug)
			return nil
		}
		return fmt.Errorf("kubernetes delete: %w", err)
	}

	deletePublicIP := true
	req := billing.CancelServiceRequest{
		ServiceName:    "Kubernetes",
		Reason:         "not_needed_anymore",
		Type:           "Immediate",
		Status:         "Pending",
		BillingCycle:   k8sCancelBillingCycle(cluster),
		DeletePublicIP: &deletePublicIP,
	}
	if err := billing.NewService(client).CancelService(ctx, cluster.Slug, req); err != nil {
		if apierrors.IsResourceNotFound(err) {
			fmt.Fprintf(os.Stderr, "Kubernetes cluster %q not found — already deleted.\n", cluster.Slug)
			return nil
		}
		return fmt.Errorf("kubernetes delete: %w", err)
	}

	printer.Fprintf("Deletion requested for %q; the Kubernetes cluster is being removed in the background.\n", cluster.Slug)
	return nil
}

func k8sCancelBillingCycle(cluster *kubernetes.Cluster) string {
	if cluster == nil {
		return ""
	}
	cycles := make([]*kubernetes.BillingCycle, 0, 2)
	if cluster.Offering != nil {
		cycles = append(cycles, cluster.Offering.BillingCycle)
	}
	cycles = append(cycles, cluster.BillingCycle)
	for _, cycle := range cycles {
		if cycle == nil {
			continue
		}
		for _, value := range []string{cycle.Unit, cycle.Slug, cycle.Name} {
			if unit, ok := billingCycleUnit(value); ok {
				return unit
			}
		}
	}
	return ""
}

func newK8sClusterUpgradeVersionCmd() *cobra.Command {
	var version string

	cmd := &cobra.Command{
		Use:   "upgrade-version <cluster-slug>",
		Short: "Upgrade the Kubernetes version of a cluster",
		Args:  exactArgs(1),
		Example: `  zcp kubernetes upgrade-version my-cluster --version v1.36.4
	  zcp kubernetes upgrade-version my-cluster --version v1.37.0`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if version == "" {
				return fmt.Errorf("--version is required")
			}
			return runK8sClusterUpgradeVersion(cmd, args[0], version)
		},
	}
	cmd.Flags().StringVar(&version, "version", "", "Target Kubernetes version, e.g. v1.35.1 (required)")
	return cmd
}

func runK8sClusterUpgradeVersion(cmd *cobra.Command, clusterSlug, targetVersion string) error {
	_, client, _, err := buildClientAndPrinter(cmd)
	if err != nil {
		return err
	}

	svc := kubernetes.NewService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	cluster, err := svc.Get(ctx, clusterSlug)
	if err != nil {
		return fmt.Errorf("kubernetes upgrade-version: %w", err)
	}
	if cluster.Meta == nil {
		return fmt.Errorf("kubernetes upgrade-version: cluster metadata unavailable")
	}

	versions, err := svc.ListVersions(ctx)
	if err != nil {
		return fmt.Errorf("kubernetes upgrade-version: %w", err)
	}

	var regionID string
	for _, v := range versions {
		if v.KubernetesClusterVersionID == cluster.Meta.KubernetesVersionID {
			regionID = v.RegionID
			break
		}
	}
	if regionID == "" {
		return fmt.Errorf("kubernetes upgrade-version: could not determine region for cluster %q", clusterSlug)
	}

	var versionSlug string
	for _, v := range versions {
		if v.Version == targetVersion && v.RegionID == regionID {
			versionSlug = v.Slug
			break
		}
	}
	if versionSlug == "" {
		return fmt.Errorf("kubernetes version %q not available in this cluster's region", targetVersion)
	}

	req := kubernetes.UpgradeVersionRequest{Slug: versionSlug}
	if err := svc.UpgradeVersion(ctx, clusterSlug, req); err != nil {
		return fmt.Errorf("kubernetes upgrade-version: %w", err)
	}

	fmt.Fprintf(os.Stdout, "Kubernetes cluster %q version upgrade to %q requested.\n", clusterSlug, targetVersion)
	return nil
}

func runK8sClusterUpgrade(cmd *cobra.Command, slug, plan, billingCycle string) error {
	_, client, _, err := buildClientAndPrinter(cmd)
	if err != nil {
		return err
	}

	svc := kubernetes.NewService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	req := kubernetes.UpgradeRequest{
		Plan:         plan,
		Slug:         slug,
		BillingCycle: billingCycle,
		IsCustomPlan: false,
		CustomPlan:   nil,
	}
	if err := svc.Upgrade(ctx, slug, req); err != nil {
		return fmt.Errorf("kubernetes upgrade: %w", err)
	}

	fmt.Fprintf(os.Stdout, "Kubernetes cluster %q upgrade to plan %q requested.\n", slug, plan)
	return nil
}
