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

// NewKubernetesCmd returns the 'kubernetes' cobra command.
func NewKubernetesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "kubernetes",
		Aliases: []string{"k8s"},
		Short:   "Manage Kubernetes clusters (alias: k8s)",
	}
	cmd.AddCommand(newK8sClusterListCmd())
	cmd.AddCommand(newK8sClusterCreateCmd())
	cmd.AddCommand(newK8sClusterStartCmd())
	cmd.AddCommand(newK8sClusterStopCmd())
	cmd.AddCommand(newK8sClusterUpgradeCmd())
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

func newK8sClusterCreateCmd() *cobra.Command {
	var (
		name            string
		version         string
		nodeSize        int
		controlNodes    int
		cloudProvider   string
		region          string
		project         string
		billingCycle    string
		enableHA        bool
		plan            string
		storageCategory string
		sshKey          string
		authMethod      string
		username        string
		password        string
	)

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new Kubernetes cluster",
		Example: `  zcp kubernetes create --name my-cluster --version v1.28.4 --plan k8s-plan-1 --region yow-1 --project my-project --cloud-provider zcp --billing-cycle monthly --workers 3 --ssh-key mykey
  zcp kubernetes create --name ha-cluster --version v1.28.4 --plan k8s-plan-1 --region yow-1 --project my-project --cloud-provider zcp --billing-cycle monthly --workers 3 --control-nodes 3 --ha --ssh-key mykey`,
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
			if sshKey == "" && authMethod == "ssh-key" {
				return fmt.Errorf("--ssh-key is required when --auth-method is ssh-key")
			}
			if enableHA && controlNodes < 3 {
				fmt.Fprintf(os.Stderr, "WARNING: --ha is set but --control-nodes is %d; HA clusters typically require >= 3 control nodes\n", controlNodes)
			}
			return runK8sClusterCreate(cmd, kubernetes.CreateRequest{
				Name:            name,
				Version:         version,
				NodeSize:        nodeSize,
				ControlNodes:    controlNodes,
				CloudProvider:   cloudProvider,
				Region:          region,
				Project:         project,
				BillingCycle:    billingCycle,
				EnableHA:        enableHA,
				Networks:        []string{},
				Plan:            plan,
				WithPoolCard:    false,
				IsCustomPlan:    false,
				CustomPlan:      nil,
				VirtualMachine:  "",
				Coupon:          nil,
				StorageCategory: storageCategory,
				SSHKey:          sshKey,
				AuthMethod:      authMethod,
				Username:        username,
				Password:        password,
			})
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "Cluster name (required)")
	cmd.Flags().StringVar(&version, "version", "", "Kubernetes version, e.g. v1.28.4 (required)")
	cmd.Flags().IntVar(&nodeSize, "workers", 0, "Number of worker nodes (required, >= 1)")
	cmd.Flags().IntVar(&controlNodes, "control-nodes", 1, "Number of control plane nodes (default 1)")
	cmd.Flags().StringVar(&cloudProvider, "cloud-provider", "", "Cloud provider slug (required)")
	cmd.Flags().StringVar(&region, "region", "", "Region slug (required)")
	cmd.Flags().StringVar(&project, "project", "", "Project slug (required)")
	cmd.Flags().StringVar(&billingCycle, "billing-cycle", "", "Billing cycle slug, e.g. hourly, monthly (required)")
	cmd.Flags().BoolVar(&enableHA, "ha", false, "Enable high availability")
	cmd.Flags().StringVar(&plan, "plan", "", "Plan slug (required)")
	cmd.Flags().StringVar(&storageCategory, "storage-category", "", "Storage category slug (optional)")
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
