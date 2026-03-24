package commands

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/zsoftly/zcp-cli/internal/api/loadbalancer"
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
		Short: "Manage load balancer rules",
	}
	cmd.AddCommand(newLBListCmd())
	cmd.AddCommand(newLBCreateCmd())
	cmd.AddCommand(newLBUpdateCmd())
	cmd.AddCommand(newLBDeleteCmd())
	return cmd
}

func newLBListCmd() *cobra.Command {
	var zoneUUID, ipUUID string

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List load balancer rules in a zone",
		Example: `  zcp loadbalancer list --zone <uuid>
  zcp loadbalancer list --zone <uuid> --ip <ip-uuid>`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runLBList(cmd, zoneUUID, ipUUID)
		},
	}
	cmd.Flags().StringVar(&zoneUUID, "zone", "", "Zone UUID (overrides default zone)")
	cmd.Flags().StringVar(&ipUUID, "ip", "", "Filter by IP address UUID")
	return cmd
}

func runLBList(cmd *cobra.Command, zoneUUID, ipUUID string) error {
	profile, client, printer, err := buildClientAndPrinter(cmd)
	if err != nil {
		return err
	}
	zoneUUID = resolveZone(profile, zoneUUID)
	if zoneUUID == "" {
		return errNoZone()
	}

	svc := loadbalancer.NewService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	rules, err := svc.List(ctx, zoneUUID, "", ipUUID)
	if err != nil {
		return fmt.Errorf("loadbalancer list: %w", err)
	}

	headers := []string{"UUID", "NAME", "ALGORITHM", "PUBLIC PORT", "PRIVATE PORT", "IP", "STATUS"}
	rows := make([][]string, 0, len(rules))
	for _, r := range rules {
		rows = append(rows, []string{
			r.UUID,
			r.Name,
			r.Algorithm,
			r.PublicPort,
			r.PrivatePort,
			r.IPAddressUUID,
			r.Status,
		})
	}
	return printer.PrintTable(headers, rows)
}

func newLBCreateCmd() *cobra.Command {
	var ipUUID, name, publicPort, privatePort, algorithm, networkUUID string

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new load balancer rule",
		Example: `  zcp loadbalancer create --ip <uuid> --name my-lb --public-port 80 --private-port 8080 --algorithm roundrobin
  zcp loadbalancer create --ip <uuid> --name my-lb --public-port 443 --private-port 8443 --algorithm leastconn --network <uuid>`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if ipUUID == "" {
				return fmt.Errorf("--ip is required")
			}
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
			return runLBCreate(cmd, loadbalancer.CreateRequest{
				Name:         name,
				PublicIPUUID: ipUUID,
				PublicPort:   publicPort,
				PrivatePort:  privatePort,
				Algorithm:    algorithm,
				NetworkUUID:  networkUUID,
			})
		},
	}
	cmd.Flags().StringVar(&ipUUID, "ip", "", "Public IP address UUID (required)")
	cmd.Flags().StringVar(&name, "name", "", "Load balancer rule name (required)")
	cmd.Flags().StringVar(&publicPort, "public-port", "", "Public port (required)")
	cmd.Flags().StringVar(&privatePort, "private-port", "", "Private port (required)")
	cmd.Flags().StringVar(&algorithm, "algorithm", "", "Algorithm: roundrobin, leastconn, or source (required)")
	cmd.Flags().StringVar(&networkUUID, "network", "", "Network UUID")
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

	r, err := svc.Create(ctx, req)
	if err != nil {
		return fmt.Errorf("loadbalancer create: %w", err)
	}

	headers := []string{"FIELD", "VALUE"}
	rows := [][]string{
		{"UUID", r.UUID},
		{"Name", r.Name},
		{"Algorithm", r.Algorithm},
		{"Public Port", r.PublicPort},
		{"Private Port", r.PrivatePort},
		{"IP UUID", r.IPAddressUUID},
		{"Status", r.Status},
	}
	return printer.PrintTable(headers, rows)
}

func newLBUpdateCmd() *cobra.Command {
	var name, algorithm string

	cmd := &cobra.Command{
		Use:     "update <uuid>",
		Short:   "Update a load balancer rule",
		Args:    cobra.ExactArgs(1),
		Example: `  zcp loadbalancer update <uuid> --name new-name --algorithm leastconn`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if name == "" {
				return fmt.Errorf("--name is required")
			}
			if algorithm == "" {
				return fmt.Errorf("--algorithm is required")
			}
			if !validLBAlgorithms[algorithm] {
				return fmt.Errorf("--algorithm must be one of: roundrobin, leastconn, source")
			}
			return runLBUpdate(cmd, loadbalancer.UpdateRequest{
				UUID:      args[0],
				Name:      name,
				Algorithm: algorithm,
			})
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "New load balancer rule name (required)")
	cmd.Flags().StringVar(&algorithm, "algorithm", "", "Algorithm: roundrobin, leastconn, or source (required)")
	return cmd
}

func runLBUpdate(cmd *cobra.Command, req loadbalancer.UpdateRequest) error {
	_, client, printer, err := buildClientAndPrinter(cmd)
	if err != nil {
		return err
	}

	svc := loadbalancer.NewService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	r, err := svc.Update(ctx, req)
	if err != nil {
		return fmt.Errorf("loadbalancer update: %w", err)
	}

	headers := []string{"FIELD", "VALUE"}
	rows := [][]string{
		{"UUID", r.UUID},
		{"Name", r.Name},
		{"Algorithm", r.Algorithm},
		{"Status", r.Status},
	}
	return printer.PrintTable(headers, rows)
}

func newLBDeleteCmd() *cobra.Command {
	var yes bool

	cmd := &cobra.Command{
		Use:   "delete <uuid>",
		Short: "Delete a load balancer rule",
		Args:  cobra.ExactArgs(1),
		Example: `  zcp loadbalancer delete <uuid>
  zcp loadbalancer delete <uuid> --yes`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runLBDelete(cmd, args[0], yes)
		},
	}
	cmd.Flags().BoolVar(&yes, "yes", false, "Skip confirmation prompt")
	return cmd
}

func runLBDelete(cmd *cobra.Command, uuid string, yes bool) error {
	if !yes {
		fmt.Fprintf(os.Stderr, "Delete load balancer rule %q? This action cannot be undone. [y/N]: ", uuid)
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

	if err := svc.Delete(ctx, uuid); err != nil {
		return fmt.Errorf("loadbalancer delete: %w", err)
	}

	printer.Fprintf("Load balancer rule %q deleted.\n", uuid)
	return nil
}
