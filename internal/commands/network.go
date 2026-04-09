package commands

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"github.com/zsoftly/zcp-cli/internal/api/network"
)

// NewNetworkCmd returns the 'network' cobra command.
func NewNetworkCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "network",
		Short: "Manage isolated networks",
	}
	cmd.AddCommand(newNetworkListCmd())
	cmd.AddCommand(newNetworkCreateCmd())
	cmd.AddCommand(newNetworkUpdateCmd())
	cmd.AddCommand(newNetworkCategoriesCmd())
	return cmd
}

func newNetworkListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List isolated networks",
		Example: `  zcp network list
  zcp network list --output json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runNetworkList(cmd)
		},
	}
	return cmd
}

func runNetworkList(cmd *cobra.Command) error {
	_, client, printer, err := buildClientAndPrinter(cmd)
	if err != nil {
		return err
	}

	svc := network.NewService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	nets, err := svc.List(ctx)
	if err != nil {
		return fmt.Errorf("network list: %w", err)
	}

	headers := []string{"SLUG", "NAME", "TYPE", "CIDR", "GATEWAY", "STATUS", "ZONE"}
	rows := make([][]string, 0, len(nets))
	for _, n := range nets {
		rows = append(rows, []string{
			n.Slug,
			n.Name,
			n.NetworkType,
			n.CIDR,
			n.Gateway,
			n.Status,
			n.ZoneSlug,
		})
	}
	return printer.PrintTable(headers, rows)
}

func newNetworkCreateCmd() *cobra.Command {
	var name, categorySlug, zoneSlug, gateway, netmask, description string
	var cloudProvider, region, project string

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new isolated network",
		Example: `  zcp network create --name my-net --category <slug> --cloud-provider <slug> --region <slug> --project <slug>
  zcp network create --name my-net --category <slug> --gateway 10.1.1.1 --netmask 255.255.255.0 --cloud-provider <slug> --region <slug> --project <slug>`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if name == "" {
				return fmt.Errorf("--name is required")
			}
			if categorySlug == "" {
				return fmt.Errorf("--category is required")
			}
			if cloudProvider == "" {
				return fmt.Errorf("--cloud-provider is required")
			}
			if region == "" {
				return fmt.Errorf("--region is required")
			}
			if project == "" {
				return fmt.Errorf("--project is required")
			}
			return runNetworkCreate(cmd, network.CreateRequest{
				Name:          name,
				CategorySlug:  categorySlug,
				ZoneSlug:      zoneSlug,
				Gateway:       gateway,
				Netmask:       netmask,
				Description:   description,
				CloudProvider: cloudProvider,
				Region:        region,
				Project:       project,
			})
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "Network name (required)")
	cmd.Flags().StringVar(&categorySlug, "category", "", "Network category slug (required)")
	cmd.Flags().StringVar(&zoneSlug, "zone", "", "Zone slug")
	cmd.Flags().StringVar(&gateway, "gateway", "", "Gateway IP")
	cmd.Flags().StringVar(&netmask, "netmask", "", "Netmask (e.g. 255.255.255.0)")
	cmd.Flags().StringVar(&description, "description", "", "Network description")
	cmd.Flags().StringVar(&cloudProvider, "cloud-provider", "", "Cloud provider slug (required)")
	cmd.Flags().StringVar(&region, "region", "", "Region slug (required)")
	cmd.Flags().StringVar(&project, "project", "", "Project slug (required)")
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
		{"Slug", n.Slug},
		{"Name", n.Name},
		{"Type", n.NetworkType},
		{"CIDR", n.CIDR},
		{"Gateway", n.Gateway},
		{"Status", n.Status},
		{"Zone", n.ZoneSlug},
	}
	return printer.PrintTable(headers, rows)
}

func newNetworkUpdateCmd() *cobra.Command {
	var name, description string

	cmd := &cobra.Command{
		Use:   "update <slug>",
		Short: "Update a network",
		Args:  cobra.ExactArgs(1),
		Example: `  zcp network update <slug> --name new-name
  zcp network update <slug> --description "Updated description"`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if name == "" && description == "" {
				return fmt.Errorf("at least one of --name or --description is required")
			}
			return runNetworkUpdate(cmd, args[0], network.UpdateRequest{
				Name:        name,
				Description: description,
			})
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "New network name")
	cmd.Flags().StringVar(&description, "description", "", "New description")
	return cmd
}

func runNetworkUpdate(cmd *cobra.Command, slug string, req network.UpdateRequest) error {
	_, client, printer, err := buildClientAndPrinter(cmd)
	if err != nil {
		return err
	}

	svc := network.NewService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	n, err := svc.Update(ctx, slug, req)
	if err != nil {
		return fmt.Errorf("network update: %w", err)
	}

	headers := []string{"FIELD", "VALUE"}
	rows := [][]string{
		{"Slug", n.Slug},
		{"Name", n.Name},
		{"Type", n.NetworkType},
		{"CIDR", n.CIDR},
		{"Gateway", n.Gateway},
		{"Status", n.Status},
	}
	return printer.PrintTable(headers, rows)
}

func newNetworkCategoriesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "categories",
		Short: "List network categories",
		Example: `  zcp network categories
  zcp network categories --output json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runNetworkCategories(cmd)
		},
	}
	return cmd
}

func runNetworkCategories(cmd *cobra.Command) error {
	_, client, printer, err := buildClientAndPrinter(cmd)
	if err != nil {
		return err
	}

	svc := network.NewService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	cats, err := svc.ListCategories(ctx)
	if err != nil {
		return fmt.Errorf("network categories: %w", err)
	}

	headers := []string{"SLUG", "NAME", "DESCRIPTION"}
	rows := make([][]string, 0, len(cats))
	for _, c := range cats {
		rows = append(rows, []string{
			c.Slug,
			c.Name,
			c.Description,
		})
	}
	return printer.PrintTable(headers, rows)
}
