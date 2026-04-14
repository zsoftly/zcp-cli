package commands

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/zsoftly/zcp-cli/internal/api/virtualrouter"
)

// NewVirtualRouterCmd returns the 'virtual-router' cobra command.
func NewVirtualRouterCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "virtual-router",
		Aliases: []string{"vr"},
		Short:   "Manage virtual routers",
	}
	cmd.AddCommand(newVirtualRouterListCmd())
	cmd.AddCommand(newVirtualRouterCreateCmd())
	cmd.AddCommand(newVirtualRouterRebootCmd())
	return cmd
}

func newVirtualRouterListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List virtual routers",
		Example: `  zcp virtual-router list
  zcp vr list --output json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runVirtualRouterList(cmd)
		},
	}
	return cmd
}

func runVirtualRouterList(cmd *cobra.Command) error {
	_, client, printer, err := buildClientAndPrinter(cmd)
	if err != nil {
		return err
	}

	svc := virtualrouter.NewService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	routers, err := svc.List(ctx)
	if err != nil {
		return fmt.Errorf("virtual-router list: %w", err)
	}

	headers := []string{"SLUG", "NAME", "STATE", "PUBLIC IP", "GUEST IP", "ZONE", "ROLE"}
	rows := make([][]string, 0, len(routers))
	for _, r := range routers {
		rows = append(rows, []string{
			r.Slug,
			r.Name,
			r.State,
			r.PublicIP,
			r.GuestIP,
			r.ZoneSlug,
			r.Role,
		})
	}
	return printer.PrintTable(headers, rows)
}

func newVirtualRouterCreateCmd() *cobra.Command {
	var name, networkSlug, planSlug string
	var cloudProvider, region, project string

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a virtual router",
		Example: `  zcp virtual-router create --name my-router --network <slug> --cloud-provider zcp --region yow-1 --project my-project
  zcp virtual-router create --name my-router --network <slug> --plan <slug> --cloud-provider zcp --region yow-1 --project my-project`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if name == "" {
				return fmt.Errorf("--name is required")
			}
			if networkSlug == "" {
				return fmt.Errorf("--network is required")
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
			return runVirtualRouterCreate(cmd, virtualrouter.CreateRequest{
				Name:          name,
				NetworkSlug:   networkSlug,
				PlanSlug:      planSlug,
				CloudProvider: cloudProvider,
				Region:        region,
				Project:       project,
			})
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "Virtual router name (required)")
	cmd.Flags().StringVar(&networkSlug, "network", "", "Network slug (required)")
	cmd.Flags().StringVar(&planSlug, "plan", "", "Virtual router plan slug")
	cmd.Flags().StringVar(&cloudProvider, "cloud-provider", "", "Cloud provider slug (required)")
	cmd.Flags().StringVar(&region, "region", "", "Region slug (required)")
	cmd.Flags().StringVar(&project, "project", "", "Project slug (required)")
	return cmd
}

func runVirtualRouterCreate(cmd *cobra.Command, req virtualrouter.CreateRequest) error {
	_, client, printer, err := buildClientAndPrinter(cmd)
	if err != nil {
		return err
	}

	svc := virtualrouter.NewService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	vr, err := svc.Create(ctx, req)
	if err != nil {
		return fmt.Errorf("virtual-router create: %w", err)
	}

	headers := []string{"FIELD", "VALUE"}
	rows := [][]string{
		{"Slug", vr.Slug},
		{"Name", vr.Name},
		{"State", vr.State},
		{"Public IP", vr.PublicIP},
		{"Guest IP", vr.GuestIP},
		{"Zone", vr.ZoneSlug},
		{"Role", vr.Role},
	}
	return printer.PrintTable(headers, rows)
}

func newVirtualRouterRebootCmd() *cobra.Command {
	var yes bool

	cmd := &cobra.Command{
		Use:   "reboot <slug>",
		Short: "Reboot a virtual router",
		Args:  cobra.ExactArgs(1),
		Example: `  zcp virtual-router reboot <slug>
  zcp vr reboot <slug> --yes`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runVirtualRouterReboot(cmd, args[0], yes)
		},
	}
	cmd.Flags().BoolVar(&yes, "yes", false, "Skip confirmation prompt")
	return cmd
}

func runVirtualRouterReboot(cmd *cobra.Command, slug string, yes bool) error {
	if !yes && !autoApproved(cmd) {
		fmt.Fprintf(os.Stderr, "Reboot virtual router %q? [y/N]: ", slug)
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

	svc := virtualrouter.NewService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	vr, err := svc.Reboot(ctx, slug)
	if err != nil {
		return fmt.Errorf("virtual-router reboot: %w", err)
	}

	headers := []string{"FIELD", "VALUE"}
	rows := [][]string{
		{"Slug", vr.Slug},
		{"Name", vr.Name},
		{"State", vr.State},
	}
	return printer.PrintTable(headers, rows)
}
