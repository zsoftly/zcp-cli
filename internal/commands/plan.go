package commands

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/spf13/cobra"
	"github.com/zsoftly/zcp-cli/pkg/api/plan"
	"github.com/zsoftly/zcp-cli/pkg/api/storagecategory"
)

// NewPlanCmd returns the 'plan' cobra command with subcommands for each
// STKCNSL service type.
func NewPlanCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "plan",
		Short: "List service plans and pricing",
		Long: `List available service plans and pricing from the ZCP catalog.

Each subcommand queries a specific service type and displays the plans
with their resource attributes and pricing. A region is required; pass
--region, set ZCP_REGION, or configure a profile default.`,
	}
	cmd.AddCommand(newPlanVMCmd())
	cmd.AddCommand(newPlanRouterCmd())
	cmd.AddCommand(newPlanStorageCmd())
	cmd.AddCommand(newPlanLBCmd())
	cmd.AddCommand(newPlanK8sCmd())
	cmd.AddCommand(newPlanIPCmd())
	cmd.AddCommand(newPlanVMSnapshotCmd())
	cmd.AddCommand(newPlanTemplateCmd())
	cmd.AddCommand(newPlanISOCmd())
	cmd.AddCommand(newPlanBackupCmd())
	cmd.AddCommand(newPlanNetworkCmd())
	cmd.AddCommand(newPlanObjectStorageCmd())
	return cmd
}

// planRegion resolves the region for a plan listing from the --region flag or
// ZCP_REGION and requires it. Plans are region-specific, so an unscoped listing
// would mix regions and surface plans that fail to deploy in the target region.
func planRegion(cmd *cobra.Command) (string, error) {
	flagRegion, _ := cmd.Flags().GetString("region")
	return requireRegion(cmd, flagRegion)
}

// ---------------------------------------------------------------------------
// Network
// ---------------------------------------------------------------------------

func newPlanNetworkCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "network",
		Short: "List Network plans",
		Long: `List Network plans (isolated and L2 network offerings).

The plan slug is the value for "zcp network create --network-plan".`,
		Example: `  zcp plan network --region yow-1
  zcp plan network --region yow-1 --output json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			_, client, printer, err := buildClientAndPrinter(cmd)
			if err != nil {
				return err
			}
			svc := plan.NewService(client)
			ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
			defer cancel()

			region, err := planRegion(cmd)
			if err != nil {
				return err
			}
			plans, err := svc.List(ctx, plan.ServiceNetwork, region)
			if err != nil {
				return fmt.Errorf("plan network: %w", err)
			}

			headers := []string{"ID", "SLUG", "NAME", "NETWORK TYPE", "HOURLY", "MONTHLY", "ACTIVE"}
			rows := make([][]string, 0, len(plans))
			for _, p := range plans {
				rows = append(rows, []string{
					p.ID,
					p.Slug,
					p.Name,
					p.NetworkType,
					formatPrice(p.HourlyPrice),
					formatPrice(p.MonthlyPrice),
					strconv.FormatBool(p.Status),
				})
			}
			return printer.PrintTable(headers, rows)
		},
	}
}

// ---------------------------------------------------------------------------
// Virtual Machine
// ---------------------------------------------------------------------------

func newPlanVMCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "vm",
		Short: "List Virtual Machine plans",
		Example: `  zcp plan vm --region yow-1
  zcp plan vm --region yow-1 --output json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			_, client, printer, err := buildClientAndPrinter(cmd)
			if err != nil {
				return err
			}
			svc := plan.NewService(client)
			ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
			defer cancel()

			region, err := planRegion(cmd)
			if err != nil {
				return err
			}
			plans, err := svc.List(ctx, plan.ServiceVM, region)
			if err != nil {
				return fmt.Errorf("plan vm: %w", err)
			}

			headers := []string{"ID", "SLUG", "NAME", "CPU", "MEMORY", "STORAGE", "HOURLY", "MONTHLY", "ACTIVE"}
			rows := make([][]string, 0, len(plans))
			for _, p := range plans {
				rows = append(rows, []string{
					p.ID,
					p.Slug,
					p.Name,
					p.Attribute.FormattedCPU.String(),
					p.Attribute.FormattedMemory,
					p.Attribute.FormattedStorage,
					formatPrice(p.HourlyPrice),
					formatPrice(p.MonthlyPrice),
					strconv.FormatBool(p.Status),
				})
			}
			return printer.PrintTable(headers, rows)
		},
	}
}

// ---------------------------------------------------------------------------
// Virtual Router
// ---------------------------------------------------------------------------

func newPlanRouterCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "router",
		Short: "List Virtual Router plans",
		Example: `  zcp plan router --region yow-1
  zcp plan router --region yow-1 --output json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			_, client, printer, err := buildClientAndPrinter(cmd)
			if err != nil {
				return err
			}
			svc := plan.NewService(client)
			ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
			defer cancel()

			region, err := planRegion(cmd)
			if err != nil {
				return err
			}
			plans, err := svc.List(ctx, plan.ServiceVirtualRouter, region)
			if err != nil {
				return fmt.Errorf("plan router: %w", err)
			}

			headers := []string{"ID", "SLUG", "NAME", "CPU", "MEMORY", "NETWORK RATE", "HOURLY", "MONTHLY", "ACTIVE"}
			rows := make([][]string, 0, len(plans))
			for _, p := range plans {
				rows = append(rows, []string{
					p.ID,
					p.Slug,
					p.Name,
					p.Attribute.CPU.String(),
					p.Attribute.FormattedMemory,
					p.Attribute.NetworkRate.String(),
					formatPrice(p.HourlyPrice),
					formatPrice(p.MonthlyPrice),
					strconv.FormatBool(p.Status),
				})
			}
			return printer.PrintTable(headers, rows)
		},
	}
}

// ---------------------------------------------------------------------------
// Block Storage
// ---------------------------------------------------------------------------

func newPlanStorageCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "storage",
		Short: "List Block Storage plans",
		Example: `  zcp plan storage --region yow-1
  zcp plan storage --region yow-1 --output json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			_, client, printer, err := buildClientAndPrinter(cmd)
			if err != nil {
				return err
			}
			ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
			defer cancel()

			region, err := planRegion(cmd)
			if err != nil {
				return err
			}
			plans, err := plan.NewService(client).List(ctx, plan.ServiceBlockStorage, region)
			if err != nil {
				return fmt.Errorf("plan storage: %w", err)
			}

			// Build id→slug map so the table shows the usable slug, not a UUID.
			catSlug := map[string]string{}
			cats, err := storagecategory.NewService(client).List(ctx, "")
			if err == nil {
				for _, c := range cats {
					catSlug[c.ID] = c.Slug
				}
			}

			headers := []string{"SLUG", "NAME", "STORAGE CATEGORY", "CEPH POOL", "HOURLY", "MONTHLY", "ACTIVE"}
			rows := make([][]string, 0, len(plans))
			for _, p := range plans {
				slug := catSlug[p.StorageCategoryID]
				if slug == "" {
					slug = p.StorageCategoryID
				}
				rows = append(rows, []string{
					p.Slug,
					p.Name,
					slug,
					p.Attribute.StorageTags,
					formatPrice(p.HourlyPrice),
					formatPrice(p.MonthlyPrice),
					strconv.FormatBool(p.Status),
				})
			}
			return printer.PrintTable(headers, rows)
		},
	}
}

// ---------------------------------------------------------------------------
// Load Balancer
// ---------------------------------------------------------------------------

func newPlanLBCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "lb",
		Short: "List Load Balancer plans",
		Example: `  zcp plan lb --region yow-1
  zcp plan lb --region yow-1 --output json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			_, client, printer, err := buildClientAndPrinter(cmd)
			if err != nil {
				return err
			}
			svc := plan.NewService(client)
			ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
			defer cancel()

			region, err := planRegion(cmd)
			if err != nil {
				return err
			}
			plans, err := svc.List(ctx, plan.ServiceLoadBalancer, region)
			if err != nil {
				return fmt.Errorf("plan lb: %w", err)
			}

			headers := []string{"ID", "SLUG", "NAME", "TAG", "HOURLY", "MONTHLY", "ACTIVE"}
			rows := make([][]string, 0, len(plans))
			for _, p := range plans {
				rows = append(rows, []string{
					p.ID,
					p.Slug,
					p.Name,
					p.ParsedTag(),
					formatPrice(p.HourlyPrice),
					formatPrice(p.MonthlyPrice),
					strconv.FormatBool(p.Status),
				})
			}
			return printer.PrintTable(headers, rows)
		},
	}
}

// ---------------------------------------------------------------------------
// Kubernetes
// ---------------------------------------------------------------------------

func newPlanK8sCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "kubernetes",
		Short:   "List Kubernetes plans",
		Aliases: []string{"k8s"},
		Example: `  zcp plan kubernetes --region yow-1
  zcp plan k8s --region yow-1 --output json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			_, client, printer, err := buildClientAndPrinter(cmd)
			if err != nil {
				return err
			}
			svc := plan.NewService(client)
			ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
			defer cancel()

			region, err := planRegion(cmd)
			if err != nil {
				return err
			}
			plans, err := svc.List(ctx, plan.ServiceKubernetes, region)
			if err != nil {
				return fmt.Errorf("plan kubernetes: %w", err)
			}

			headers := []string{"ID", "SLUG", "NAME", "CPU", "MEMORY", "HOURLY", "MONTHLY", "ACTIVE"}
			rows := make([][]string, 0, len(plans))
			for _, p := range plans {
				rows = append(rows, []string{
					p.ID,
					p.Slug,
					p.Name,
					p.Attribute.FormattedCPU.String(),
					p.Attribute.FormattedMemory,
					formatPrice(p.HourlyPrice),
					formatPrice(p.MonthlyPrice),
					strconv.FormatBool(p.Status),
				})
			}
			return printer.PrintTable(headers, rows)
		},
	}
}

// ---------------------------------------------------------------------------
// IP Address
// ---------------------------------------------------------------------------

func newPlanIPCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "ip",
		Short: "List IP Address plans",
		Example: `  zcp plan ip --region yow-1
  zcp plan ip --region yow-1 --output json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			_, client, printer, err := buildClientAndPrinter(cmd)
			if err != nil {
				return err
			}
			svc := plan.NewService(client)
			ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
			defer cancel()

			region, err := planRegion(cmd)
			if err != nil {
				return err
			}
			plans, err := svc.List(ctx, plan.ServiceIPAddress, region)
			if err != nil {
				return fmt.Errorf("plan ip: %w", err)
			}

			headers := []string{"ID", "SLUG", "NAME", "TAG", "HOURLY", "MONTHLY", "ACTIVE"}
			rows := make([][]string, 0, len(plans))
			for _, p := range plans {
				rows = append(rows, []string{
					p.ID,
					p.Slug,
					p.Name,
					p.ParsedTag(),
					formatPrice(p.HourlyPrice),
					formatPrice(p.MonthlyPrice),
					strconv.FormatBool(p.Status),
				})
			}
			return printer.PrintTable(headers, rows)
		},
	}
}

// ---------------------------------------------------------------------------
// VM Snapshot
// ---------------------------------------------------------------------------

func newPlanVMSnapshotCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "vm-snapshot",
		Short: "List VM Snapshot plans",
		Example: `  zcp plan vm-snapshot --region yow-1
  zcp plan vm-snapshot --region yow-1 --output json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			_, client, printer, err := buildClientAndPrinter(cmd)
			if err != nil {
				return err
			}
			svc := plan.NewService(client)
			ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
			defer cancel()

			region, err := planRegion(cmd)
			if err != nil {
				return err
			}
			plans, err := svc.List(ctx, plan.ServiceVMSnapshot, region)
			if err != nil {
				return fmt.Errorf("plan vm-snapshot: %w", err)
			}

			headers := []string{"ID", "SLUG", "NAME", "HOURLY", "MONTHLY", "ACTIVE"}
			rows := make([][]string, 0, len(plans))
			for _, p := range plans {
				rows = append(rows, []string{
					p.ID,
					p.Slug,
					p.Name,
					formatPrice(p.HourlyPrice),
					formatPrice(p.MonthlyPrice),
					strconv.FormatBool(p.Status),
				})
			}
			return printer.PrintTable(headers, rows)
		},
	}
}

// ---------------------------------------------------------------------------
// My Template
// ---------------------------------------------------------------------------

func newPlanTemplateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "template",
		Short: "List My Template plans",
		Example: `  zcp plan template --region yow-1
  zcp plan template --region yow-1 --output json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			_, client, printer, err := buildClientAndPrinter(cmd)
			if err != nil {
				return err
			}
			svc := plan.NewService(client)
			ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
			defer cancel()

			region, err := planRegion(cmd)
			if err != nil {
				return err
			}
			plans, err := svc.List(ctx, plan.ServiceMyTemplate, region)
			if err != nil {
				return fmt.Errorf("plan template: %w", err)
			}

			headers := []string{"ID", "SLUG", "NAME", "TAG", "HOURLY", "MONTHLY", "ACTIVE"}
			rows := make([][]string, 0, len(plans))
			for _, p := range plans {
				rows = append(rows, []string{
					p.ID,
					p.Slug,
					p.Name,
					p.ParsedTag(),
					formatPrice(p.HourlyPrice),
					formatPrice(p.MonthlyPrice),
					strconv.FormatBool(p.Status),
				})
			}
			return printer.PrintTable(headers, rows)
		},
	}
}

// ---------------------------------------------------------------------------
// ISO
// ---------------------------------------------------------------------------

func newPlanObjectStorageCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "object-storage",
		Short: "List Object Storage plans (slugs for object-storage create --plan)",
		Example: `  zcp plan object-storage --region os-yul
  zcp plan object-storage --region os-yul --output json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			_, client, printer, err := buildClientAndPrinter(cmd)
			if err != nil {
				return err
			}
			svc := plan.NewService(client)
			ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
			defer cancel()

			region, err := planRegion(cmd)
			if err != nil {
				return err
			}
			plans, err := svc.List(ctx, plan.ServiceObjectStorage, region)
			if err != nil {
				return fmt.Errorf("plan object-storage: %w", err)
			}

			headers := []string{"SLUG", "NAME", "STORAGE", "HOURLY", "MONTHLY", "ACTIVE"}
			rows := make([][]string, 0, len(plans))
			for _, p := range plans {
				storage := p.Attribute.FormattedStorage
				if storage == "" {
					storage = p.Attribute.Storage.String() + " " + p.Attribute.StorageUnit
				}
				rows = append(rows, []string{
					p.Slug,
					p.Name,
					storage,
					formatPrice(p.HourlyPrice),
					formatPrice(p.MonthlyPrice),
					strconv.FormatBool(p.Status),
				})
			}
			return printer.PrintTable(headers, rows)
		},
	}
}

func newPlanISOCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "iso",
		Short: "List ISO plans",
		Example: `  zcp plan iso --region yow-1
  zcp plan iso --region yow-1 --output json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			_, client, printer, err := buildClientAndPrinter(cmd)
			if err != nil {
				return err
			}
			svc := plan.NewService(client)
			ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
			defer cancel()

			region, err := planRegion(cmd)
			if err != nil {
				return err
			}
			plans, err := svc.List(ctx, plan.ServiceISO, region)
			if err != nil {
				return fmt.Errorf("plan iso: %w", err)
			}

			headers := []string{"ID", "SLUG", "NAME", "TAG", "HOURLY", "MONTHLY", "ACTIVE"}
			rows := make([][]string, 0, len(plans))
			for _, p := range plans {
				rows = append(rows, []string{
					p.ID,
					p.Slug,
					p.Name,
					p.ParsedTag(),
					formatPrice(p.HourlyPrice),
					formatPrice(p.MonthlyPrice),
					strconv.FormatBool(p.Status),
				})
			}
			return printer.PrintTable(headers, rows)
		},
	}
}

// ---------------------------------------------------------------------------
// Backups
// ---------------------------------------------------------------------------

func newPlanBackupCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "backup",
		Short: "List Backup plans",
		Example: `  zcp plan backup --region yow-1
  zcp plan backup --region yow-1 --output json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			_, client, printer, err := buildClientAndPrinter(cmd)
			if err != nil {
				return err
			}
			svc := plan.NewService(client)
			ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
			defer cancel()

			region, err := planRegion(cmd)
			if err != nil {
				return err
			}
			plans, err := svc.List(ctx, plan.ServiceBackups, region)
			if err != nil {
				return fmt.Errorf("plan backup: %w", err)
			}

			headers := []string{"ID", "SLUG", "NAME", "TAG", "HOURLY", "MONTHLY", "ACTIVE"}
			rows := make([][]string, 0, len(plans))
			for _, p := range plans {
				rows = append(rows, []string{
					p.ID,
					p.Slug,
					p.Name,
					p.ParsedTag(),
					formatPrice(p.HourlyPrice),
					formatPrice(p.MonthlyPrice),
					strconv.FormatBool(p.Status),
				})
			}
			return printer.PrintTable(headers, rows)
		},
	}
}

// formatPrice renders a float price as a string with up to 4 decimal places,
// trimming trailing zeros for readability.
func formatPrice(v float64) string {
	return strconv.FormatFloat(v, 'f', -1, 64)
}
