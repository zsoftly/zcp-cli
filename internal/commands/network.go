package commands

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/zsoftly/zcp-cli/internal/api/network"
)

// NewNetworkCmd returns the 'network' cobra command.
func NewNetworkCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "network",
		Short: "Manage virtual networks",
	}
	cmd.AddCommand(newNetworkListCmd())
	cmd.AddCommand(newNetworkGetCmd())
	cmd.AddCommand(newNetworkCreateCmd())
	cmd.AddCommand(newNetworkDeleteCmd())
	cmd.AddCommand(newNetworkRestartCmd())
	return cmd
}

func newNetworkListCmd() *cobra.Command {
	var zoneUUID string

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List networks in a zone",
		Example: `  zcp network list --zone <uuid>
  zcp network list --zone <uuid> --output json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if zoneUUID == "" {
				return fmt.Errorf("--zone is required")
			}
			return runNetworkList(cmd, zoneUUID)
		},
	}
	cmd.Flags().StringVar(&zoneUUID, "zone", "", "Zone UUID (required)")
	return cmd
}

func runNetworkList(cmd *cobra.Command, zoneUUID string) error {
	_, client, printer, err := buildClientAndPrinter(cmd)
	if err != nil {
		return err
	}

	svc := network.NewService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	nets, err := svc.List(ctx, zoneUUID, "")
	if err != nil {
		return fmt.Errorf("network list: %w", err)
	}

	headers := []string{"UUID", "NAME", "TYPE", "CIDR", "GATEWAY", "STATUS"}
	rows := make([][]string, 0, len(nets))
	for _, n := range nets {
		rows = append(rows, []string{
			n.UUID,
			n.Name,
			n.NetworkType,
			n.CIDR,
			n.Gateway,
			n.Status,
		})
	}
	return printer.PrintTable(headers, rows)
}

func newNetworkGetCmd() *cobra.Command {
	var zoneUUID string

	cmd := &cobra.Command{
		Use:     "get <uuid>",
		Short:   "Get details of a network",
		Args:    cobra.ExactArgs(1),
		Example: `  zcp network get <uuid> --zone <uuid>`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if zoneUUID == "" {
				return fmt.Errorf("--zone is required")
			}
			return runNetworkGet(cmd, args[0], zoneUUID)
		},
	}
	cmd.Flags().StringVar(&zoneUUID, "zone", "", "Zone UUID (required)")
	return cmd
}

func runNetworkGet(cmd *cobra.Command, uuid, zoneUUID string) error {
	_, client, printer, err := buildClientAndPrinter(cmd)
	if err != nil {
		return err
	}

	svc := network.NewService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	n, err := svc.Get(ctx, zoneUUID, uuid)
	if err != nil {
		return fmt.Errorf("network get: %w", err)
	}

	headers := []string{"FIELD", "VALUE"}
	rows := [][]string{
		{"UUID", n.UUID},
		{"Name", n.Name},
		{"Type", n.NetworkType},
		{"CIDR", n.CIDR},
		{"Gateway", n.Gateway},
		{"Status", n.Status},
		{"Zone UUID", n.ZoneUUID},
		{"Domain Name", n.DomainName},
		{"Network Domain", n.NetworkDomain},
		{"Offering UUID", n.NetworkOfferingUUID},
	}
	return printer.PrintTable(headers, rows)
}

func newNetworkCreateCmd() *cobra.Command {
	var zoneUUID, name, offeringUUID, vmUUID string
	var isPublic bool

	cmd := &cobra.Command{
		Use:     "create",
		Short:   "Create a new network",
		Example: `  zcp network create --zone <uuid> --name my-net --offering <uuid>`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if zoneUUID == "" {
				return fmt.Errorf("--zone is required")
			}
			if name == "" {
				return fmt.Errorf("--name is required")
			}
			if offeringUUID == "" {
				return fmt.Errorf("--offering is required")
			}
			return runNetworkCreate(cmd, network.CreateRequest{
				Name:                name,
				ZoneUUID:            zoneUUID,
				NetworkOfferingUUID: offeringUUID,
				VirtualMachineUUID:  vmUUID,
				IsPublic:            isPublic,
			})
		},
	}
	cmd.Flags().StringVar(&zoneUUID, "zone", "", "Zone UUID (required)")
	cmd.Flags().StringVar(&name, "name", "", "Network name (required)")
	cmd.Flags().StringVar(&offeringUUID, "offering", "", "Network offering UUID (required)")
	cmd.Flags().StringVar(&vmUUID, "instance", "", "Virtual machine UUID to attach on creation")
	cmd.Flags().BoolVar(&isPublic, "public", false, "Mark network as public")
	return cmd
}

func runNetworkCreate(cmd *cobra.Command, req network.CreateRequest) error {
	_, client, printer, err := buildClientAndPrinter(cmd)
	if err != nil {
		return err
	}

	svc := network.NewService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	n, err := svc.Create(ctx, req)
	if err != nil {
		return fmt.Errorf("network create: %w", err)
	}

	headers := []string{"FIELD", "VALUE"}
	rows := [][]string{
		{"UUID", n.UUID},
		{"Name", n.Name},
		{"Type", n.NetworkType},
		{"CIDR", n.CIDR},
		{"Gateway", n.Gateway},
		{"Status", n.Status},
		{"Zone UUID", n.ZoneUUID},
	}
	return printer.PrintTable(headers, rows)
}

func newNetworkDeleteCmd() *cobra.Command {
	var yes bool

	cmd := &cobra.Command{
		Use:   "delete <uuid>",
		Short: "Delete a network",
		Args:  cobra.ExactArgs(1),
		Example: `  zcp network delete <uuid>
  zcp network delete <uuid> --yes`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runNetworkDelete(cmd, args[0], yes)
		},
	}
	cmd.Flags().BoolVar(&yes, "yes", false, "Skip confirmation prompt")
	return cmd
}

func runNetworkDelete(cmd *cobra.Command, uuid string, yes bool) error {
	if !yes {
		fmt.Fprintf(os.Stderr, "Delete network %q? This action cannot be undone. [y/N]: ", uuid)
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

	svc := network.NewService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	if err := svc.Delete(ctx, uuid); err != nil {
		return fmt.Errorf("network delete: %w", err)
	}

	printer.Fprintf("Network %q deleted.\n", uuid)
	return nil
}

func newNetworkRestartCmd() *cobra.Command {
	var cleanUp bool

	cmd := &cobra.Command{
		Use:   "restart <uuid>",
		Short: "Restart a network",
		Args:  cobra.ExactArgs(1),
		Example: `  zcp network restart <uuid>
  zcp network restart <uuid> --cleanup`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runNetworkRestart(cmd, args[0], cleanUp)
		},
	}
	cmd.Flags().BoolVar(&cleanUp, "cleanup", false, "Clean up stale resources during restart")
	return cmd
}

func runNetworkRestart(cmd *cobra.Command, uuid string, cleanUp bool) error {
	_, client, printer, err := buildClientAndPrinter(cmd)
	if err != nil {
		return err
	}

	svc := network.NewService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	n, err := svc.Restart(ctx, uuid, cleanUp)
	if err != nil {
		return fmt.Errorf("network restart: %w", err)
	}

	headers := []string{"FIELD", "VALUE"}
	rows := [][]string{
		{"UUID", n.UUID},
		{"Name", n.Name},
		{"Status", n.Status},
	}
	return printer.PrintTable(headers, rows)
}
