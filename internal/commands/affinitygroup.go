package commands

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/zsoftly/zcp-cli/internal/api/affinitygroup"
)

// NewAffinityGroupCmd returns the 'affinity-group' cobra command.
func NewAffinityGroupCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "affinity-group",
		Short: "Manage affinity groups",
	}
	cmd.AddCommand(newAffinityGroupListCmd())
	cmd.AddCommand(newAffinityGroupCreateCmd())
	cmd.AddCommand(newAffinityGroupDeleteCmd())
	return cmd
}

func newAffinityGroupListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List affinity groups",
		Example: `  zcp affinity-group list
  zcp affinity-group list --output json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runAffinityGroupList(cmd)
		},
	}
	return cmd
}

func runAffinityGroupList(cmd *cobra.Command) error {
	_, client, printer, err := buildClientAndPrinter(cmd)
	if err != nil {
		return err
	}

	svc := affinitygroup.NewService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	groups, err := svc.List(ctx)
	if err != nil {
		return fmt.Errorf("affinity-group list: %w", err)
	}

	headers := []string{"SLUG", "NAME", "TYPE", "DESCRIPTION", "CREATED"}
	rows := make([][]string, 0, len(groups))
	for _, g := range groups {
		rows = append(rows, []string{
			g.Slug,
			g.Name,
			g.Type,
			g.Description,
			g.CreatedAt,
		})
	}
	return printer.PrintTable(headers, rows)
}

func newAffinityGroupCreateCmd() *cobra.Command {
	var (
		name          string
		groupType     string
		description   string
		project       string
		region        string
		cloudProvider string
	)

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create an affinity group",
		Example: `  zcp affinity-group create --name my-group --type "host affinity" \
    --cloud-provider zcp --project my-project --region yow-1`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if name == "" {
				return fmt.Errorf("--name is required")
			}
			if groupType == "" {
				return fmt.Errorf("--type is required")
			}
			cloudProvider = resolveCloudProvider(cloudProvider)
			if cloudProvider == "" {
				return fmt.Errorf("--cloud-provider is required")
			}
			project = resolveProject(project)
			if project == "" {
				return fmt.Errorf("--project is required")
			}
			region = resolveRegion(region)
			if region == "" {
				return fmt.Errorf("--region is required")
			}
			req := affinitygroup.CreateRequest{
				Name:          name,
				Type:          groupType,
				Description:   description,
				Project:       project,
				Region:        region,
				CloudProvider: cloudProvider,
			}
			return runAffinityGroupCreate(cmd, req)
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "Group name (required)")
	cmd.Flags().StringVar(&groupType, "type", "", "Affinity type: 'host affinity' or 'host anti-affinity' (required)")
	cmd.Flags().StringVar(&description, "description", "", "Group description")
	cmd.Flags().StringVar(&project, "project", "", "Project slug (required)")
	cmd.Flags().StringVar(&region, "region", "", "Region slug (required)")
	cmd.Flags().StringVar(&cloudProvider, "cloud-provider", "", "Cloud provider slug (required)")
	return cmd
}

func runAffinityGroupCreate(cmd *cobra.Command, req affinitygroup.CreateRequest) error {
	_, client, printer, err := buildClientAndPrinter(cmd)
	if err != nil {
		return err
	}

	svc := affinitygroup.NewService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	g, err := svc.Create(ctx, req)
	if err != nil {
		return fmt.Errorf("affinity-group create: %w", err)
	}

	headers := []string{"FIELD", "VALUE"}
	rows := [][]string{
		{"Slug", g.Slug},
		{"Name", g.Name},
		{"Type", g.Type},
		{"Description", g.Description},
		{"Created", g.CreatedAt},
	}
	return printer.PrintTable(headers, rows)
}

func newAffinityGroupDeleteCmd() *cobra.Command {
	var yes bool

	cmd := &cobra.Command{
		Use:   "delete <slug>",
		Short: "Delete an affinity group",
		Args:  cobra.ExactArgs(1),
		Example: `  zcp affinity-group delete my-group
  zcp affinity-group delete my-group --yes`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runAffinityGroupDelete(cmd, args[0], yes)
		},
	}
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "Skip confirmation prompt")
	return cmd
}

func runAffinityGroupDelete(cmd *cobra.Command, slug string, yes bool) error {
	if !yes && !autoApproved(cmd) {
		fmt.Fprintf(os.Stderr, "Delete affinity group %q? [y/N]: ", slug)
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

	svc := affinitygroup.NewService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	if err := svc.Delete(ctx, slug); err != nil {
		return fmt.Errorf("affinity-group delete: %w", err)
	}

	printer.Fprintf("Affinity group %q deleted.\n", slug)
	return nil
}
