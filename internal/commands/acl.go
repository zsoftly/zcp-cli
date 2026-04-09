package commands

import (
	"context"
	"fmt"
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
	cmd.AddCommand(newACLReplaceCmd())
	return cmd
}

func newACLListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "list <vpc-slug>",
		Short:   "List network ACLs for a VPC",
		Args:    cobra.ExactArgs(1),
		Example: `  zcp acl list <vpc-slug>`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runACLList(cmd, args[0])
		},
	}
	return cmd
}

func runACLList(cmd *cobra.Command, vpcSlug string) error {
	_, client, printer, err := buildClientAndPrinter(cmd)
	if err != nil {
		return err
	}

	svc := acl.NewService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	acls, err := svc.List(ctx, vpcSlug)
	if err != nil {
		return fmt.Errorf("acl list: %w", err)
	}

	headers := []string{"SLUG", "NAME", "DESCRIPTION", "VPC", "STATUS"}
	rows := make([][]string, 0, len(acls))
	for _, a := range acls {
		rows = append(rows, []string{
			a.Slug,
			a.Name,
			a.Description,
			a.VPCSlug,
			a.Status,
		})
	}
	return printer.PrintTable(headers, rows)
}

func newACLCreateCmd() *cobra.Command {
	var name, description string

	cmd := &cobra.Command{
		Use:     "create <vpc-slug>",
		Short:   "Create a network ACL in a VPC",
		Args:    cobra.ExactArgs(1),
		Example: `  zcp acl create my-vpc --name allow-web --description "Allow HTTP traffic"`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if name == "" {
				return fmt.Errorf("--name is required")
			}
			return runACLCreate(cmd, args[0], acl.ACLCreateRequest{
				Name:        name,
				Description: description,
				VPC:         args[0],
			})
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "ACL name (required)")
	cmd.Flags().StringVar(&description, "description", "", "ACL description")
	return cmd
}

func runACLCreate(cmd *cobra.Command, vpcSlug string, req acl.ACLCreateRequest) error {
	_, client, _, err := buildClientAndPrinter(cmd)
	if err != nil {
		return err
	}

	svc := acl.NewService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	if err := svc.Create(ctx, vpcSlug, req); err != nil {
		return fmt.Errorf("acl create: %w", err)
	}

	fmt.Fprintf(cmd.ErrOrStderr(), "ACL %q created in VPC %q.\n", req.Name, vpcSlug)
	return nil
}

func newACLReplaceCmd() *cobra.Command {
	var networkSlug, aclSlug string

	cmd := &cobra.Command{
		Use:     "replace",
		Short:   "Replace the ACL on a network",
		Example: `  zcp acl replace --network <network-slug> --acl <acl-slug>`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if networkSlug == "" {
				return fmt.Errorf("--network is required")
			}
			if aclSlug == "" {
				return fmt.Errorf("--acl is required")
			}
			return runACLReplace(cmd, networkSlug, aclSlug)
		},
	}
	cmd.Flags().StringVar(&networkSlug, "network", "", "Network slug (required)")
	cmd.Flags().StringVar(&aclSlug, "acl", "", "ACL slug (required)")
	return cmd
}

func runACLReplace(cmd *cobra.Command, networkSlug, aclSlug string) error {
	_, client, printer, err := buildClientAndPrinter(cmd)
	if err != nil {
		return err
	}

	svc := acl.NewService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	if err := svc.ReplaceNetworkACL(ctx, networkSlug, aclSlug); err != nil {
		return fmt.Errorf("acl replace: %w", err)
	}

	printer.Fprintf("ACL replaced on network %q.\n", networkSlug)
	return nil
}
