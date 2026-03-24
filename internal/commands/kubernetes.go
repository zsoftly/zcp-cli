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
	cmd.AddCommand(newK8sVersionListCmd())
	cmd.AddCommand(newK8sClusterListCmd())
	cmd.AddCommand(newK8sClusterCreateCmd())
	cmd.AddCommand(newK8sClusterDeleteCmd())
	cmd.AddCommand(newK8sClusterStartCmd())
	cmd.AddCommand(newK8sClusterStopCmd())
	cmd.AddCommand(newK8sClusterScaleCmd())
	cmd.AddCommand(newK8sNodeListCmd())
	return cmd
}

func newK8sVersionListCmd() *cobra.Command {
	var zoneUUID string

	cmd := &cobra.Command{
		Use:   "version list",
		Short: "List supported Kubernetes versions for a zone",
		Example: `  zcp kubernetes version list --zone <uuid>
  zcp k8s version list --zone <uuid> --output json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runK8sVersionList(cmd, zoneUUID)
		},
	}
	cmd.Flags().StringVar(&zoneUUID, "zone", "", "Zone UUID (overrides default zone)")
	return cmd
}

func runK8sVersionList(cmd *cobra.Command, zoneUUID string) error {
	profile, client, printer, err := buildClientAndPrinter(cmd)
	if err != nil {
		return err
	}
	zoneUUID = resolveZone(profile, zoneUUID)
	if zoneUUID == "" {
		return errNoZone()
	}

	svc := kubernetes.NewService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	versions, err := svc.ListVersions(ctx, zoneUUID)
	if err != nil {
		return fmt.Errorf("kubernetes version list: %w", err)
	}

	headers := []string{"UUID", "NAME", "DESCRIPTION", "MIN MEMORY", "MIN CPU", "ACTIVE"}
	rows := make([][]string, 0, len(versions))
	for _, v := range versions {
		rows = append(rows, []string{
			v.UUID,
			v.Name,
			v.Description,
			v.MinMemory,
			v.MinCPUNumber,
			strconv.FormatBool(v.IsActive),
		})
	}
	return printer.PrintTable(headers, rows)
}

func newK8sClusterListCmd() *cobra.Command {
	var clusterUUID string

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List Kubernetes clusters",
		Example: `  zcp kubernetes list
  zcp kubernetes list --cluster <uuid>
  zcp k8s list --output json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runK8sClusterList(cmd, clusterUUID)
		},
	}
	cmd.Flags().StringVar(&clusterUUID, "cluster", "", "Filter by cluster UUID (optional)")
	return cmd
}

func runK8sClusterList(cmd *cobra.Command, clusterUUID string) error {
	_, client, printer, err := buildClientAndPrinter(cmd)
	if err != nil {
		return err
	}

	svc := kubernetes.NewService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	clusters, err := svc.List(ctx, clusterUUID)
	if err != nil {
		return fmt.Errorf("kubernetes list: %w", err)
	}

	headers := []string{"UUID", "NAME", "STATE", "WORKERS", "CONTROL NODES", "NETWORK"}
	rows := make([][]string, 0, len(clusters))
	for _, c := range clusters {
		rows = append(rows, []string{
			c.UUID,
			c.Name,
			c.State,
			strconv.Itoa(c.Size),
			strconv.Itoa(c.ControlNodes),
			c.TransNetworkUUID,
		})
	}
	return printer.PrintTable(headers, rows)
}

func newK8sClusterCreateCmd() *cobra.Command {
	var (
		zoneUUID            string
		name                string
		versionUUID         string
		computeOfferingUUID string
		networkUUID         string
		workers             int
		controlNodes        int
		sshKeyName          string
		haEnabled           bool
		description         string
		diskSize            int64
		externalLBIP        string
	)

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new Kubernetes cluster",
		Example: `  zcp kubernetes create --zone <uuid> --name my-cluster --version <uuid> --compute-offering <uuid> --network <uuid> --workers 3 --ssh-key mykey
  zcp kubernetes create --zone <uuid> --name ha-cluster --version <uuid> --compute-offering <uuid> --network <uuid> --workers 3 --ssh-key mykey --ha --control-nodes 3`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if name == "" {
				return fmt.Errorf("--name is required")
			}
			if versionUUID == "" {
				return fmt.Errorf("--version is required")
			}
			if computeOfferingUUID == "" {
				return fmt.Errorf("--compute-offering is required")
			}
			if networkUUID == "" {
				return fmt.Errorf("--network is required")
			}
			if workers < 1 {
				return fmt.Errorf("--workers must be >= 1")
			}
			if sshKeyName == "" {
				return fmt.Errorf("--ssh-key is required")
			}
			if haEnabled && controlNodes < 3 {
				fmt.Fprintf(os.Stderr, "WARNING: --ha is set but --control-nodes is %d; HA clusters typically require >= 3 control nodes\n", controlNodes)
			}
			return runK8sClusterCreate(cmd, kubernetes.CreateRequest{
				Name:                   name,
				ZoneUUID:               zoneUUID,
				VersionUUID:            versionUUID,
				ComputeOfferingUUID:    computeOfferingUUID,
				TransNetworkUUID:       networkUUID,
				Size:                   int64(workers),
				ControlNodes:           int64(controlNodes),
				SSHKeyName:             sshKeyName,
				HAEnabled:              haEnabled,
				Description:            description,
				NodeRootDiskSize:       diskSize,
				ExternalLoadbalancerIP: externalLBIP,
			})
		},
	}
	cmd.Flags().StringVar(&zoneUUID, "zone", "", "Zone UUID (overrides default zone)")
	cmd.Flags().StringVar(&name, "name", "", "Cluster name (required)")
	cmd.Flags().StringVar(&versionUUID, "version", "", "Kubernetes version UUID (required)")
	cmd.Flags().StringVar(&computeOfferingUUID, "compute-offering", "", "Compute offering UUID (required)")
	cmd.Flags().StringVar(&networkUUID, "network", "", "Transit network UUID (required)")
	cmd.Flags().IntVar(&workers, "workers", 0, "Number of worker nodes (required, >= 1)")
	cmd.Flags().StringVar(&sshKeyName, "ssh-key", "", "SSH key name (required)")
	cmd.Flags().IntVar(&controlNodes, "control-nodes", 1, "Number of control plane nodes (default 1)")
	cmd.Flags().BoolVar(&haEnabled, "ha", false, "Enable high availability")
	cmd.Flags().StringVar(&description, "description", "", "Cluster description (optional)")
	cmd.Flags().Int64Var(&diskSize, "disk-size", 0, "Node root disk size in GB (optional)")
	cmd.Flags().StringVar(&externalLBIP, "external-lb-ip", "", "External load balancer IP address (optional)")
	return cmd
}

func runK8sClusterCreate(cmd *cobra.Command, req kubernetes.CreateRequest) error {
	profile, client, printer, err := buildClientAndPrinter(cmd)
	if err != nil {
		return err
	}
	req.ZoneUUID = resolveZone(profile, req.ZoneUUID)
	if req.ZoneUUID == "" {
		return errNoZone()
	}

	svc := kubernetes.NewService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	cluster, err := svc.Create(ctx, req)
	if err != nil {
		return fmt.Errorf("kubernetes create: %w", err)
	}

	headers := []string{"UUID", "NAME", "STATE", "WORKERS", "CONTROL NODES", "NETWORK"}
	rows := [][]string{{
		cluster.UUID,
		cluster.Name,
		cluster.State,
		strconv.Itoa(cluster.Size),
		strconv.Itoa(cluster.ControlNodes),
		cluster.TransNetworkUUID,
	}}
	return printer.PrintTable(headers, rows)
}

func newK8sClusterDeleteCmd() *cobra.Command {
	var yes bool

	cmd := &cobra.Command{
		Use:   "delete <uuid>",
		Short: "Delete a Kubernetes cluster",
		Args:  cobra.ExactArgs(1),
		Example: `  zcp kubernetes delete <uuid>
  zcp kubernetes delete <uuid> --yes`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runK8sClusterDelete(cmd, args[0], yes)
		},
	}
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "Skip confirmation prompt")
	return cmd
}

func runK8sClusterDelete(cmd *cobra.Command, uuid string, yes bool) error {
	if !yes {
		fmt.Fprintf(os.Stdout, "Delete Kubernetes cluster %q? [y/N]: ", uuid)
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

	if err := svc.Delete(ctx, uuid); err != nil {
		return fmt.Errorf("kubernetes delete: %w", err)
	}

	fmt.Fprintf(os.Stdout, "Kubernetes cluster %q deleted.\n", uuid)
	return nil
}

func newK8sClusterStartCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "start <uuid>",
		Short:   "Start a stopped Kubernetes cluster",
		Args:    cobra.ExactArgs(1),
		Example: `  zcp kubernetes start <uuid>`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runK8sClusterStart(cmd, args[0])
		},
	}
	return cmd
}

func runK8sClusterStart(cmd *cobra.Command, uuid string) error {
	_, client, printer, err := buildClientAndPrinter(cmd)
	if err != nil {
		return err
	}

	svc := kubernetes.NewService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	cluster, err := svc.Start(ctx, uuid)
	if err != nil {
		return fmt.Errorf("kubernetes start: %w", err)
	}

	headers := []string{"UUID", "NAME", "STATE"}
	rows := [][]string{{cluster.UUID, cluster.Name, cluster.State}}
	return printer.PrintTable(headers, rows)
}

func newK8sClusterStopCmd() *cobra.Command {
	var yes bool

	cmd := &cobra.Command{
		Use:   "stop <uuid>",
		Short: "Stop a running Kubernetes cluster",
		Args:  cobra.ExactArgs(1),
		Example: `  zcp kubernetes stop <uuid>
  zcp kubernetes stop <uuid> --yes`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runK8sClusterStop(cmd, args[0], yes)
		},
	}
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "Skip confirmation prompt")
	return cmd
}

func runK8sClusterStop(cmd *cobra.Command, uuid string, yes bool) error {
	if !yes {
		fmt.Fprintf(os.Stdout, "Stop cluster %q? [y/N]: ", uuid)
		var answer string
		fmt.Scanln(&answer)
		if strings.ToLower(strings.TrimSpace(answer)) != "y" {
			fmt.Fprintln(os.Stdout, "Aborted.")
			return nil
		}
	}

	_, client, printer, err := buildClientAndPrinter(cmd)
	if err != nil {
		return err
	}

	svc := kubernetes.NewService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	cluster, err := svc.Stop(ctx, uuid)
	if err != nil {
		return fmt.Errorf("kubernetes stop: %w", err)
	}

	headers := []string{"UUID", "NAME", "STATE"}
	rows := [][]string{{cluster.UUID, cluster.Name, cluster.State}}
	return printer.PrintTable(headers, rows)
}

func newK8sClusterScaleCmd() *cobra.Command {
	var (
		workers     int
		autoscaling bool
	)

	cmd := &cobra.Command{
		Use:   "scale <uuid>",
		Short: "Scale the worker node count of a Kubernetes cluster",
		Args:  cobra.ExactArgs(1),
		Example: `  zcp kubernetes scale <uuid> --workers 5
  zcp kubernetes scale <uuid> --workers 5 --autoscaling`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if !cmd.Flags().Changed("workers") {
				return fmt.Errorf("--workers is required")
			}
			return runK8sClusterScale(cmd, args[0], workers, autoscaling)
		},
	}
	cmd.Flags().IntVar(&workers, "workers", 0, "New worker node count (required)")
	cmd.Flags().BoolVar(&autoscaling, "autoscaling", false, "Enable autoscaling")
	return cmd
}

func runK8sClusterScale(cmd *cobra.Command, uuid string, workers int, autoscaling bool) error {
	_, client, printer, err := buildClientAndPrinter(cmd)
	if err != nil {
		return err
	}

	svc := kubernetes.NewService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	cluster, err := svc.Scale(ctx, uuid, workers, autoscaling)
	if err != nil {
		return fmt.Errorf("kubernetes scale: %w", err)
	}

	headers := []string{"UUID", "NAME", "STATE", "WORKERS"}
	rows := [][]string{{cluster.UUID, cluster.Name, cluster.State, strconv.Itoa(cluster.Size)}}
	return printer.PrintTable(headers, rows)
}

func newK8sNodeListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "nodes <cluster-uuid>",
		Short: "List nodes in a Kubernetes cluster",
		Args:  cobra.ExactArgs(1),
		Example: `  zcp kubernetes nodes <cluster-uuid>
  zcp k8s nodes <cluster-uuid> --output json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runK8sNodeList(cmd, args[0])
		},
	}
	return cmd
}

func runK8sNodeList(cmd *cobra.Command, clusterUUID string) error {
	_, client, printer, err := buildClientAndPrinter(cmd)
	if err != nil {
		return err
	}

	svc := kubernetes.NewService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	nodes, err := svc.ListNodes(ctx, clusterUUID)
	if err != nil {
		return fmt.Errorf("kubernetes nodes: %w", err)
	}

	headers := []string{"UUID", "NAME", "STATE", "MEMORY", "PRIVATE IP", "ZONE"}
	rows := make([][]string, 0, len(nodes))
	for _, n := range nodes {
		rows = append(rows, []string{
			n.UUID,
			n.Name,
			n.State,
			n.Memory,
			n.PrivateIP,
			n.ZoneUUID,
		})
	}
	return printer.PrintTable(headers, rows)
}
