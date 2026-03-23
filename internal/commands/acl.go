package commands

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/zsoftly/zcp-cli/internal/api/acl"
)

// NewACLCmd returns the 'acl' cobra command.
func NewACLCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "acl",
		Short: "Manage Network ACLs",
	}
	cmd.AddCommand(newACLListCmd())
	cmd.AddCommand(newACLCreateCmd())
	cmd.AddCommand(newACLDeleteCmd())
	cmd.AddCommand(newACLReplaceCmd())
	return cmd
}

func newACLListCmd() *cobra.Command {
	var zoneUUID, vpcUUID string

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List network ACLs in a zone",
		Example: `  zcp acl list --zone <uuid>
  zcp acl list --zone <uuid> --vpc <uuid>`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if zoneUUID == "" {
				return fmt.Errorf("--zone is required")
			}
			return runACLList(cmd, zoneUUID, vpcUUID)
		},
	}
	cmd.Flags().StringVar(&zoneUUID, "zone", "", "Zone UUID (required)")
	cmd.Flags().StringVar(&vpcUUID, "vpc", "", "Filter by VPC UUID")
	return cmd
}

func runACLList(cmd *cobra.Command, zoneUUID, vpcUUID string) error {
	_, client, printer, err := buildClientAndPrinter(cmd)
	if err != nil {
		return err
	}

	svc := acl.NewService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	acls, err := svc.List(ctx, zoneUUID, "", vpcUUID)
	if err != nil {
		return fmt.Errorf("acl list: %w", err)
	}

	headers := []string{"UUID", "NAME", "DESCRIPTION", "VPC", "STATUS"}
	rows := make([][]string, 0, len(acls))
	for _, a := range acls {
		rows = append(rows, []string{
			a.UUID,
			a.Name,
			a.Description,
			a.VPCUUID,
			a.Status,
		})
	}
	return printer.PrintTable(headers, rows)
}

func newACLCreateCmd() *cobra.Command {
	var vpcUUID, name, description string

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new network ACL",
		Example: `  zcp acl create --vpc <uuid> --name my-acl
  zcp acl create --vpc <uuid> --name my-acl --description "Web tier ACL"`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if vpcUUID == "" {
				return fmt.Errorf("--vpc is required")
			}
			if name == "" {
				return fmt.Errorf("--name is required")
			}
			return runACLCreate(cmd, acl.CreateRequest{
				Name:        name,
				VPCUUID:     vpcUUID,
				Description: description,
			})
		},
	}
	cmd.Flags().StringVar(&vpcUUID, "vpc", "", "VPC UUID (required)")
	cmd.Flags().StringVar(&name, "name", "", "ACL name (required)")
	cmd.Flags().StringVar(&description, "description", "", "ACL description")
	return cmd
}

func runACLCreate(cmd *cobra.Command, req acl.CreateRequest) error {
	_, client, printer, err := buildClientAndPrinter(cmd)
	if err != nil {
		return err
	}

	svc := acl.NewService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	a, err := svc.Create(ctx, req)
	if err != nil {
		return fmt.Errorf("acl create: %w", err)
	}

	headers := []string{"FIELD", "VALUE"}
	rows := [][]string{
		{"UUID", a.UUID},
		{"Name", a.Name},
		{"Description", a.Description},
		{"VPC UUID", a.VPCUUID},
		{"Status", a.Status},
	}
	return printer.PrintTable(headers, rows)
}

func newACLDeleteCmd() *cobra.Command {
	var yes bool

	cmd := &cobra.Command{
		Use:   "delete <uuid>",
		Short: "Delete a network ACL",
		Args:  cobra.ExactArgs(1),
		Example: `  zcp acl delete <uuid>
  zcp acl delete <uuid> --yes`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runACLDelete(cmd, args[0], yes)
		},
	}
	cmd.Flags().BoolVar(&yes, "yes", false, "Skip confirmation prompt")
	return cmd
}

func runACLDelete(cmd *cobra.Command, uuid string, yes bool) error {
	if !yes {
		fmt.Fprintf(os.Stderr, "Delete network ACL %q? This action cannot be undone. [y/N]: ", uuid)
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

	svc := acl.NewService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	if err := svc.Delete(ctx, uuid); err != nil {
		return fmt.Errorf("acl delete: %w", err)
	}

	printer.Fprintf("Network ACL %q deleted.\n", uuid)
	return nil
}

func newACLReplaceCmd() *cobra.Command {
	var networkUUID, aclUUID string

	cmd := &cobra.Command{
		Use:     "replace",
		Short:   "Replace the ACL on a network",
		Example: `  zcp acl replace --network <network-uuid> --acl <acl-uuid>`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if networkUUID == "" {
				return fmt.Errorf("--network is required")
			}
			if aclUUID == "" {
				return fmt.Errorf("--acl is required")
			}
			return runACLReplace(cmd, networkUUID, aclUUID)
		},
	}
	cmd.Flags().StringVar(&networkUUID, "network", "", "Network UUID (required)")
	cmd.Flags().StringVar(&aclUUID, "acl", "", "ACL UUID (required)")
	return cmd
}

func runACLReplace(cmd *cobra.Command, networkUUID, aclUUID string) error {
	_, client, printer, err := buildClientAndPrinter(cmd)
	if err != nil {
		return err
	}

	svc := acl.NewService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	nets, err := svc.ReplaceNetworkACL(ctx, networkUUID, aclUUID)
	if err != nil {
		return fmt.Errorf("acl replace: %w", err)
	}

	headers := []string{"UUID", "NAME"}
	rows := make([][]string, 0, len(nets))
	for _, n := range nets {
		rows = append(rows, []string{n.UUID, n.Name})
	}
	return printer.PrintTable(headers, rows)
}
