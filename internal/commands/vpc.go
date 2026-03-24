package commands

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/zsoftly/zcp-cli/internal/api/vpc"
)

// NewVPCCmd returns the 'vpc' cobra command.
func NewVPCCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "vpc",
		Short: "Manage Virtual Private Clouds",
	}
	cmd.AddCommand(newVPCListCmd())
	cmd.AddCommand(newVPCGetCmd())
	cmd.AddCommand(newVPCCreateCmd())
	cmd.AddCommand(newVPCUpdateCmd())
	cmd.AddCommand(newVPCDeleteCmd())
	cmd.AddCommand(newVPCRestartCmd())
	return cmd
}

func newVPCListCmd() *cobra.Command {
	var zoneUUID string

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List VPCs in a zone",
		Example: `  zcp vpc list --zone <uuid>
  zcp vpc list --zone <uuid> --output json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runVPCList(cmd, zoneUUID)
		},
	}
	cmd.Flags().StringVar(&zoneUUID, "zone", "", "Zone UUID (overrides default zone)")
	return cmd
}

func runVPCList(cmd *cobra.Command, zoneUUID string) error {
	profile, client, printer, err := buildClientAndPrinter(cmd)
	if err != nil {
		return err
	}
	zoneUUID = resolveZone(profile, zoneUUID)
	if zoneUUID == "" {
		return errNoZone()
	}

	svc := vpc.NewService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	vpcs, err := svc.List(ctx, zoneUUID, "")
	if err != nil {
		return fmt.Errorf("vpc list: %w", err)
	}

	headers := []string{"UUID", "NAME", "CIDR", "STATUS", "ZONE"}
	rows := make([][]string, 0, len(vpcs))
	for _, v := range vpcs {
		rows = append(rows, []string{
			v.UUID,
			v.Name,
			v.CIDR,
			v.Status,
			v.ZoneUUID,
		})
	}
	return printer.PrintTable(headers, rows)
}

func newVPCGetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "get <uuid>",
		Short:   "Get details of a VPC",
		Args:    cobra.ExactArgs(1),
		Example: `  zcp vpc get <uuid>`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runVPCGet(cmd, args[0])
		},
	}
	return cmd
}

func runVPCGet(cmd *cobra.Command, uuid string) error {
	_, client, printer, err := buildClientAndPrinter(cmd)
	if err != nil {
		return err
	}

	svc := vpc.NewService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	v, err := svc.Get(ctx, uuid)
	if err != nil {
		return fmt.Errorf("vpc get: %w", err)
	}

	headers := []string{"FIELD", "VALUE"}
	rows := [][]string{
		{"UUID", v.UUID},
		{"Name", v.Name},
		{"Description", v.Description},
		{"CIDR", v.CIDR},
		{"Status", v.Status},
		{"Zone UUID", v.ZoneUUID},
		{"Zone Name", v.ZoneName},
		{"Domain Name", v.DomainName},
	}
	return printer.PrintTable(headers, rows)
}

func newVPCCreateCmd() *cobra.Command {
	var zoneUUID, name, offeringUUID, cidr, description, networkDomain, lbProvider string

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new VPC",
		Example: `  zcp vpc create --zone <uuid> --name my-vpc --offering <uuid> --cidr 10.0.0.0/8
  zcp vpc create --zone <uuid> --name my-vpc --offering <uuid> --cidr 10.0.0.0/8 --description "Production VPC"`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if name == "" {
				return fmt.Errorf("--name is required")
			}
			if offeringUUID == "" {
				return fmt.Errorf("--offering is required")
			}
			if cidr == "" {
				return fmt.Errorf("--cidr is required")
			}
			if !strings.Contains(cidr, "/") {
				return fmt.Errorf("--cidr must be a valid CIDR (e.g. 10.0.0.0/8)")
			}
			return runVPCCreate(cmd, vpc.CreateRequest{
				Name:                       name,
				ZoneUUID:                   zoneUUID,
				VPCOfferingUUID:            offeringUUID,
				CIDR:                       cidr,
				Description:                description,
				NetworkDomain:              networkDomain,
				PublicLoadBalancerProvider: lbProvider,
			})
		},
	}
	cmd.Flags().StringVar(&zoneUUID, "zone", "", "Zone UUID (overrides default zone)")
	cmd.Flags().StringVar(&name, "name", "", "VPC name (required)")
	cmd.Flags().StringVar(&offeringUUID, "offering", "", "VPC offering UUID (required)")
	cmd.Flags().StringVar(&cidr, "cidr", "", "CIDR block (required, e.g. 10.0.0.0/8)")
	cmd.Flags().StringVar(&description, "description", "", "VPC description")
	cmd.Flags().StringVar(&networkDomain, "network-domain", "", "Network domain")
	cmd.Flags().StringVar(&lbProvider, "lb-provider", "", "Public load balancer provider")
	return cmd
}

func runVPCCreate(cmd *cobra.Command, req vpc.CreateRequest) error {
	profile, client, printer, err := buildClientAndPrinter(cmd)
	if err != nil {
		return err
	}
	req.ZoneUUID = resolveZone(profile, req.ZoneUUID)
	if req.ZoneUUID == "" {
		return errNoZone()
	}

	svc := vpc.NewService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	v, err := svc.Create(ctx, req)
	if err != nil {
		return fmt.Errorf("vpc create: %w", err)
	}

	headers := []string{"FIELD", "VALUE"}
	rows := [][]string{
		{"UUID", v.UUID},
		{"Name", v.Name},
		{"CIDR", v.CIDR},
		{"Status", v.Status},
		{"Zone UUID", v.ZoneUUID},
	}
	return printer.PrintTable(headers, rows)
}

func newVPCUpdateCmd() *cobra.Command {
	var name, description string

	cmd := &cobra.Command{
		Use:     "update <uuid>",
		Short:   "Update a VPC",
		Args:    cobra.ExactArgs(1),
		Example: `  zcp vpc update <uuid> --name new-name --description "Updated description"`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if name == "" {
				return fmt.Errorf("--name is required")
			}
			return runVPCUpdate(cmd, vpc.UpdateRequest{
				UUID:        args[0],
				Name:        name,
				Description: description,
			})
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "New VPC name (required)")
	cmd.Flags().StringVar(&description, "description", "", "New VPC description")
	return cmd
}

func runVPCUpdate(cmd *cobra.Command, req vpc.UpdateRequest) error {
	_, client, printer, err := buildClientAndPrinter(cmd)
	if err != nil {
		return err
	}

	svc := vpc.NewService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	v, err := svc.Update(ctx, req)
	if err != nil {
		return fmt.Errorf("vpc update: %w", err)
	}

	headers := []string{"FIELD", "VALUE"}
	rows := [][]string{
		{"UUID", v.UUID},
		{"Name", v.Name},
		{"Description", v.Description},
		{"Status", v.Status},
	}
	return printer.PrintTable(headers, rows)
}

func newVPCDeleteCmd() *cobra.Command {
	var yes bool

	cmd := &cobra.Command{
		Use:   "delete <uuid>",
		Short: "Delete a VPC",
		Args:  cobra.ExactArgs(1),
		Example: `  zcp vpc delete <uuid>
  zcp vpc delete <uuid> --yes`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runVPCDelete(cmd, args[0], yes)
		},
	}
	cmd.Flags().BoolVar(&yes, "yes", false, "Skip confirmation prompt")
	return cmd
}

func runVPCDelete(cmd *cobra.Command, uuid string, yes bool) error {
	if !yes {
		fmt.Fprintf(os.Stderr, "Delete VPC %q? This action cannot be undone. [y/N]: ", uuid)
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

	svc := vpc.NewService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	if err := svc.Delete(ctx, uuid); err != nil {
		return fmt.Errorf("vpc delete: %w", err)
	}

	printer.Fprintf("VPC %q deleted.\n", uuid)
	return nil
}

func newVPCRestartCmd() *cobra.Command {
	var cleanUp, redundant bool

	cmd := &cobra.Command{
		Use:   "restart <uuid>",
		Short: "Restart a VPC",
		Args:  cobra.ExactArgs(1),
		Example: `  zcp vpc restart <uuid>
  zcp vpc restart <uuid> --cleanup --redundant`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runVPCRestart(cmd, args[0], cleanUp, redundant)
		},
	}
	cmd.Flags().BoolVar(&cleanUp, "cleanup", false, "Clean up stale resources during restart")
	cmd.Flags().BoolVar(&redundant, "redundant", false, "Enable redundant VPC router")
	return cmd
}

func runVPCRestart(cmd *cobra.Command, uuid string, cleanUp, redundant bool) error {
	_, client, printer, err := buildClientAndPrinter(cmd)
	if err != nil {
		return err
	}

	svc := vpc.NewService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	v, err := svc.Restart(ctx, uuid, cleanUp, redundant)
	if err != nil {
		return fmt.Errorf("vpc restart: %w", err)
	}

	headers := []string{"FIELD", "VALUE"}
	rows := [][]string{
		{"UUID", v.UUID},
		{"Name", v.Name},
		{"Status", v.Status},
	}
	return printer.PrintTable(headers, rows)
}
