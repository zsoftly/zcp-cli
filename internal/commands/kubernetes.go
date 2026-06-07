package commands

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/zsoftly/zcp-cli/internal/api/kubernetes"
)

// resolveKubeconfigPath returns the path to write the kubeconfig to.
// If path is empty, defaults to ~/.kube/config.
func resolveKubeconfigPath(path string) (string, error) {
	if path != "" {
		return path, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("cannot determine home directory: %w", err)
	}
	return home + "/.kube/config", nil
}

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

	clusters, err := svc.List(ctx)
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
			strconv.Itoa(c.NodeSize),
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
		Args:    cobra.ExactArgs(1),
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
	workers := strconv.Itoa(c.NodeSize)
	controlNodes := strconv.Itoa(c.ControlNodes)
	endpoint := ""

	// Prefer the CloudStack-side meta fields — they populate after the cluster is Running.
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
	return printer.PrintTable(headers, rows)
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
		plan               string
		storageCategory    string
		sshKey             string
		authMethod         string
		username           string
		password           string
	)

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new Kubernetes cluster",
		Example: `  zcp kubernetes create --name my-cluster --version v1.35.0 --plan k8s-li-yow-1 --region yow-1 --project default --cloud-provider nimbo --billing-cycle hourly --workers 3 --storage-category pro-nvme --ssh-key mykey
  zcp kubernetes create --name ha-cluster --version v1.35.0 --plan k8s-li-yow-1 --region yow-1 --project default --cloud-provider nimbo --billing-cycle hourly --workers 3 --control-nodes 3 --ha --storage-category pro-nvme --ssh-key mykey`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if name == "" {
				return fmt.Errorf("--name is required")
			}
			if version == "" {
				return fmt.Errorf("--version is required")
			}
			if plan == "" {
				return fmt.Errorf("--plan is required")
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
			if enableHA && controlNodes < 3 {
				fmt.Fprintf(os.Stderr, "WARNING: --ha is set but --control-nodes is %d; HA clusters typically require >= 3 control nodes\n", controlNodes)
			}
			return runK8sClusterCreate(cmd, kubernetes.CreateRequest{
				Name:               name,
				Version:            version,
				NodeSize:           nodeSize,
				WorkerNodeSize:     nodeSize,
				ControlNodes:       controlNodes,
				CloudProvider:      cloudProvider,
				CloudProviderSetup: cloudProviderSetup,
				Region:             region,
				Project:            project,
				BillingCycle:       billingCycle,
				EnableHA:           enableHA,
				Networks:           []string{},
				Plan:               plan,
				WithPoolCard:       false,
				IsCustomPlan:       false,
				CustomPlan:         nil,
				VirtualMachine:     "",
				Coupon:             nil,
				StorageCategory:    storageCategory,
				SSHKey:             sshKey,
				AuthMethod:         authMethod,
				Username:           username,
				Password:           password,
			})
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "Cluster name (required)")
	cmd.Flags().StringVar(&version, "version", "", "Kubernetes version, e.g. v1.28.4 (required)")
	cmd.Flags().IntVar(&nodeSize, "workers", 0, "Number of worker nodes (required, >= 1)")
	cmd.Flags().IntVar(&controlNodes, "control-nodes", 1, "Number of control plane nodes (default 1)")
	cmd.Flags().StringVar(&cloudProvider, "cloud-provider", "", "Cloud provider slug (required)")
	cmd.Flags().StringVar(&cloudProviderSetup, "cloud-provider-setup", "", "Cloud provider setup slug, e.g. default-setup")
	cmd.Flags().StringVar(&region, "region", "", "Region slug (required)")
	cmd.Flags().StringVar(&project, "project", "", "Project slug (required)")
	cmd.Flags().StringVar(&billingCycle, "billing-cycle", "", "Billing cycle slug, e.g. hourly, monthly (required)")
	cmd.Flags().BoolVar(&enableHA, "ha", false, "Enable high availability")
	cmd.Flags().StringVar(&plan, "plan", "", "Plan slug (required)")
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
		strconv.Itoa(cluster.NodeSize),
		strconv.Itoa(cluster.ControlNodes),
		strconv.FormatBool(cluster.EnableHA),
	}}
	return printer.PrintTable(headers, rows)
}

func newK8sClusterStartCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "start <slug>",
		Short:   "Start a stopped Kubernetes cluster",
		Args:    cobra.ExactArgs(1),
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
		Args:  cobra.ExactArgs(1),
		Example: `  zcp kubernetes stop my-cluster
  zcp kubernetes stop my-cluster --yes`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runK8sClusterStop(cmd, args[0], yes)
		},
	}
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "Skip confirmation prompt")
	return cmd
}

func runK8sClusterStop(cmd *cobra.Command, slug string, yes bool) error {
	if !yes && !autoApproved(cmd) {
		fmt.Fprintf(os.Stdout, "Stop cluster %q? [y/N]: ", slug)
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
	var workers int
	var wait bool

	cmd := &cobra.Command{
		Use:   "scale <slug>",
		Short: "Scale the number of worker nodes on a Kubernetes cluster",
		Args:  cobra.ExactArgs(1),
		Example: `  zcp kubernetes scale my-cluster --workers 5
  zcp k8s scale my-cluster --workers 3 --wait`,
		RunE: func(cmd *cobra.Command, args []string) error {
			slug := args[0]
			if workers < 1 {
				return fmt.Errorf("--workers must be >= 1")
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
			currentWorkers := current.NodeSize
			if current.Meta != nil && current.Meta.Size != "" {
				currentWorkers, _ = strconv.Atoi(current.Meta.Size)
			}
			if currentWorkers == workers {
				fmt.Fprintf(os.Stdout, "Cluster %q already has %d worker(s) — no change made.\n", slug, workers)
				return nil
			}

			if err := svc.Scale(ctx, slug, workers); err != nil {
				return fmt.Errorf("kubernetes scale: %w", err)
			}
			fmt.Fprintf(os.Stdout, "Scaling %q from %d → %d worker(s) requested.\n", slug, currentWorkers, workers)

			if wait {
				fmt.Fprintf(os.Stdout, "Waiting for cluster to return to Running...\n")
				for {
					time.Sleep(15 * time.Second)
					c, err := svc.Get(context.Background(), slug)
					if err != nil {
						return fmt.Errorf("polling cluster state: %w", err)
					}
					workers := c.NodeSize
					if c.Meta != nil && c.Meta.Size != "" {
						workers, _ = strconv.Atoi(c.Meta.Size)
					}
					if c.State != "Scaling" {
						fmt.Fprintf(os.Stdout, "Done — state: %s, workers: %d\n", c.State, workers)
						return nil
					}
				}
			}

			fmt.Fprintf(os.Stdout, "To check progress:  zcp kubernetes get %s\n", slug)
			return nil
		},
	}
	cmd.Flags().IntVar(&workers, "workers", 0, "Target number of worker nodes (required)")
	cmd.Flags().BoolVar(&wait, "wait", false, "Block until scaling completes")
	_ = cmd.MarkFlagRequired("workers")
	return cmd
}

func newK8sGetConfigCmd() *cobra.Command {
	var (
		outputPath string
		print      bool
	)

	cmd := &cobra.Command{
		Use:   "get-config <slug>",
		Short: "Download the kubeconfig for a Kubernetes cluster",
		Args:  cobra.ExactArgs(1),
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

			if err := os.MkdirAll(strings.TrimRight(outputPath[:strings.LastIndex(outputPath, "/")], "/"), 0700); err != nil {
				return fmt.Errorf("creating directory: %w", err)
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
		Args:  cobra.ExactArgs(1),
		Example: `  zcp kubernetes upgrade my-cluster --plan k8s-plan-2
  zcp kubernetes upgrade my-cluster --plan k8s-plan-2 --billing-cycle hourly`,
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
		Args:  cobra.ExactArgs(1),
		Example: `  zcp kubernetes delete my-cluster
  zcp kubernetes delete my-cluster --yes`,
		RunE: func(cmd *cobra.Command, args []string) error {
			slug := args[0]
			if !yes && !autoApproved(cmd) {
				fmt.Fprintf(os.Stdout, "Delete Kubernetes cluster %q? This cannot be undone. [y/N]: ", slug)
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
			svc := kubernetes.NewService(client)
			ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
			defer cancel()
			if err := svc.Delete(ctx, slug); err != nil {
				return fmt.Errorf("kubernetes delete: %w", err)
			}
			fmt.Fprintf(os.Stdout, "Kubernetes cluster %q deletion requested.\n", slug)
			return nil
		},
	}
	cmd.Flags().BoolVar(&yes, "yes", false, "Skip confirmation prompt")
	return cmd
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
