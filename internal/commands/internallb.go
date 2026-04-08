package commands

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/zsoftly/zcp-cli/internal/api/internallb"
)

// NewInternalLBCmd returns the 'internal-lb' cobra command.
func NewInternalLBCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "internal-lb",
		Short: "Manage internal load balancers",
	}
	cmd.AddCommand(newInternalLBListCmd())
	cmd.AddCommand(newInternalLBCreateCmd())
	cmd.AddCommand(newInternalLBDeleteCmd())
	return cmd
}

func newInternalLBListCmd() *cobra.Command {
	var zoneUUID, networkUUID string

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List internal load balancers in a zone",
		Example: `  zcp internal-lb list --zone <uuid>
  zcp internal-lb list --zone <uuid> --network <uuid>`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runInternalLBList(cmd, zoneUUID, networkUUID)
		},
	}
	cmd.Flags().StringVar(&zoneUUID, "zone", "", "Zone UUID (overrides default zone)")
	cmd.Flags().StringVar(&networkUUID, "network", "", "Filter by network UUID")
	return cmd
}

func runInternalLBList(cmd *cobra.Command, zoneUUID, networkUUID string) error {
	profile, client, printer, err := buildClientAndPrinter(cmd)
	if err != nil {
		return err
	}
	zoneUUID = resolveZone(profile, zoneUUID)
	if zoneUUID == "" {
		return errNoZone()
	}

	svc := internallb.NewService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	lbs, err := svc.List(ctx, zoneUUID, "", networkUUID)
	if err != nil {
		return fmt.Errorf("internal-lb list: %w", err)
	}

	headers := []string{"UUID", "NAME", "ALGORITHM", "SOURCE IP", "SOURCE PORT", "INSTANCE PORT", "NETWORK"}
	rows := make([][]string, 0, len(lbs))
	for _, lb := range lbs {
		rows = append(rows, []string{
			lb.UUID,
			lb.Name,
			lb.Algorithm,
			lb.SourceIPAddress,
			lb.SourcePort,
			lb.InstancePort,
			lb.NetworkUUID,
		})
	}
	return printer.PrintTable(headers, rows)
}

func newInternalLBCreateCmd() *cobra.Command {
	var networkUUID, name, sourcePort, instancePort, algorithm, sourceIP string

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new internal load balancer",
		Example: `  zcp internal-lb create --network <uuid> --name my-ilb --source-port 80 --instance-port 8080 --algorithm roundrobin
  zcp internal-lb create --network <uuid> --name my-ilb --source-port 80 --instance-port 8080 --algorithm roundrobin --source-ip 10.0.0.5`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if networkUUID == "" {
				return fmt.Errorf("--network is required")
			}
			if name == "" {
				return fmt.Errorf("--name is required")
			}
			if sourcePort == "" {
				return fmt.Errorf("--source-port is required")
			}
			if instancePort == "" {
				return fmt.Errorf("--instance-port is required")
			}
			if algorithm == "" {
				return fmt.Errorf("--algorithm is required")
			}
			if !validLBAlgorithms[algorithm] {
				return fmt.Errorf("--algorithm must be one of: roundrobin, leastconn, source")
			}
			return runInternalLBCreate(cmd, internallb.CreateRequest{
				Name:            name,
				NetworkUUID:     networkUUID,
				SourcePort:      sourcePort,
				InstancePort:    instancePort,
				Algorithm:       algorithm,
				SourceIPAddress: sourceIP,
			})
		},
	}
	cmd.Flags().StringVar(&networkUUID, "network", "", "Network UUID (required)")
	cmd.Flags().StringVar(&name, "name", "", "Internal LB name (required)")
	cmd.Flags().StringVar(&sourcePort, "source-port", "", "Source port (required)")
	cmd.Flags().StringVar(&instancePort, "instance-port", "", "Instance port (required)")
	cmd.Flags().StringVar(&algorithm, "algorithm", "", "Algorithm: roundrobin, leastconn, or source (required)")
	cmd.Flags().StringVar(&sourceIP, "source-ip", "", "Source IP address")
	return cmd
}

func runInternalLBCreate(cmd *cobra.Command, req internallb.CreateRequest) error {
	_, client, printer, err := buildClientAndPrinter(cmd)
	if err != nil {
		return err
	}

	svc := internallb.NewService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	lb, err := svc.Create(ctx, req)
	if err != nil {
		return fmt.Errorf("internal-lb create: %w", err)
	}

	headers := []string{"FIELD", "VALUE"}
	rows := [][]string{
		{"UUID", lb.UUID},
		{"Name", lb.Name},
		{"Algorithm", lb.Algorithm},
		{"Source IP", lb.SourceIPAddress},
		{"Source Port", lb.SourcePort},
		{"Instance Port", lb.InstancePort},
		{"Network UUID", lb.NetworkUUID},
		{"Status", lb.Status},
	}
	return printer.PrintTable(headers, rows)
}

func newInternalLBDeleteCmd() *cobra.Command {
	var yes bool

	cmd := &cobra.Command{
		Use:   "delete <uuid>",
		Short: "Delete an internal load balancer",
		Args:  cobra.ExactArgs(1),
		Example: `  zcp internal-lb delete <uuid>
  zcp internal-lb delete <uuid> --yes`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runInternalLBDelete(cmd, args[0], yes)
		},
	}
	cmd.Flags().BoolVar(&yes, "yes", false, "Skip confirmation prompt")
	return cmd
}

func runInternalLBDelete(cmd *cobra.Command, uuid string, yes bool) error {
	if !yes && !autoApproved(cmd) {
		fmt.Fprintf(os.Stderr, "Delete internal load balancer %q? This action cannot be undone. [y/N]: ", uuid)
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

	svc := internallb.NewService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	if err := svc.Delete(ctx, uuid); err != nil {
		return fmt.Errorf("internal-lb delete: %w", err)
	}

	printer.Fprintf("Internal load balancer %q deleted.\n", uuid)
	return nil
}
