package commands

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"github.com/zsoftly/zcp-cli/internal/api/volume"
)

// NewVolumeCmd returns the 'volume' cobra command.
func NewVolumeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "volume",
		Short: "Manage block storage volumes",
	}
	cmd.AddCommand(newVolumeListCmd())
	cmd.AddCommand(newVolumeCreateCmd())
	cmd.AddCommand(newVolumeAttachCmd())
	cmd.AddCommand(newVolumeDetachCmd())
	return cmd
}

func newVolumeListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List block storage volumes",
		Example: `  zcp volume list
  zcp volume list --output json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			_, client, printer, err := buildClientAndPrinter(cmd)
			if err != nil {
				return err
			}
			svc := volume.NewService(client)
			ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
			defer cancel()

			volumes, err := svc.List(ctx)
			if err != nil {
				return fmt.Errorf("volume list: %w", err)
			}

			headers := []string{"SLUG", "NAME", "SIZE", "TYPE", "REGION", "STORAGE", "CREATED"}
			rows := make([][]string, 0, len(volumes))
			for _, v := range volumes {
				regionName := ""
				if v.Region != nil {
					regionName = v.Region.Name
				}
				storageName := ""
				if v.StorageSetting != nil {
					storageName = v.StorageSetting.Name
				}
				rows = append(rows, []string{
					v.Slug,
					v.Name,
					v.Size,
					v.VolumeType,
					regionName,
					storageName,
					v.CreatedAt,
				})
			}
			return printer.PrintTable(headers, rows)
		},
	}
	return cmd
}

func newVolumeCreateCmd() *cobra.Command {
	var name, project, cloudProvider, region, billingCycle, storageCategory, plan string
	var vmSlug, coupon string
	var isCustomPlan, isFreeTrial bool

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new block storage volume",
		Example: `  zcp volume create --name my-disk --project default-73 --cloud-provider nimbo --region noida --billing-cycle hourly --storage-category nvme --plan 50-gb-2
  zcp volume create --name my-disk --project default-73 --cloud-provider nimbo --region noida --billing-cycle hourly --storage-category nvme --plan 50-gb-2 --vm vm-slug`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if name == "" {
				return fmt.Errorf("--name is required")
			}
			if project == "" {
				return fmt.Errorf("--project is required")
			}
			if cloudProvider == "" {
				return fmt.Errorf("--cloud-provider is required")
			}
			if region == "" {
				return fmt.Errorf("--region is required")
			}
			if billingCycle == "" {
				return fmt.Errorf("--billing-cycle is required")
			}
			if storageCategory == "" {
				return fmt.Errorf("--storage-category is required")
			}
			if plan == "" {
				return fmt.Errorf("--plan is required")
			}
			_, client, printer, err := buildClientAndPrinter(cmd)
			if err != nil {
				return err
			}
			svc := volume.NewService(client)
			ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
			defer cancel()

			req := volume.CreateRequest{
				Name:            name,
				Project:         project,
				CloudProvider:   cloudProvider,
				Region:          region,
				BillingCycle:    billingCycle,
				StorageCategory: storageCategory,
				Plan:            plan,
				IsCustomPlan:    isCustomPlan,
				VirtualMachine:  vmSlug,
				Coupon:          coupon,
				IsFreeTrial:     isFreeTrial,
			}
			vol, err := svc.Create(ctx, req)
			if err != nil {
				return fmt.Errorf("volume create: %w", err)
			}

			headers := []string{"SLUG", "NAME", "SIZE", "TYPE", "CREATED"}
			rows := [][]string{{
				vol.Slug,
				vol.Name,
				vol.Size,
				vol.VolumeType,
				vol.CreatedAt,
			}}
			return printer.PrintTable(headers, rows)
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "Volume name (required)")
	cmd.Flags().StringVar(&project, "project", "", "Project slug (required)")
	cmd.Flags().StringVar(&cloudProvider, "cloud-provider", "", "Cloud provider slug (required)")
	cmd.Flags().StringVar(&region, "region", "", "Region slug (required)")
	cmd.Flags().StringVar(&billingCycle, "billing-cycle", "", "Billing cycle slug, e.g. hourly (required)")
	cmd.Flags().StringVar(&storageCategory, "storage-category", "", "Storage category slug, e.g. nvme (required)")
	cmd.Flags().StringVar(&plan, "plan", "", "Plan slug, e.g. 50-gb-2 (required)")
	cmd.Flags().BoolVar(&isCustomPlan, "custom-plan", false, "Use a custom plan")
	cmd.Flags().StringVar(&vmSlug, "vm", "", "Virtual machine slug to attach on creation")
	cmd.Flags().StringVar(&coupon, "coupon", "", "Coupon code")
	cmd.Flags().BoolVar(&isFreeTrial, "free-trial", false, "Use a free trial plan")
	return cmd
}

func newVolumeAttachCmd() *cobra.Command {
	var vmSlug string

	cmd := &cobra.Command{
		Use:     "attach <volume-slug>",
		Short:   "Attach a volume to a virtual machine",
		Args:    cobra.ExactArgs(1),
		Example: `  zcp volume attach <volume-slug> --vm <vm-slug>`,
		RunE: func(cmd *cobra.Command, args []string) error {
			volumeSlug := args[0]
			if vmSlug == "" {
				return fmt.Errorf("--vm is required")
			}
			_, client, printer, err := buildClientAndPrinter(cmd)
			if err != nil {
				return err
			}
			svc := volume.NewService(client)
			ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
			defer cancel()

			vol, err := svc.Attach(ctx, volumeSlug, vmSlug)
			if err != nil {
				return fmt.Errorf("volume attach: %w", err)
			}

			headers := []string{"SLUG", "NAME", "SIZE", "VM ID"}
			rows := [][]string{{
				vol.Slug,
				vol.Name,
				vol.Size,
				vol.VirtualMachineID,
			}}
			return printer.PrintTable(headers, rows)
		},
	}
	cmd.Flags().StringVar(&vmSlug, "vm", "", "Virtual machine slug to attach to (required)")
	return cmd
}

func newVolumeDetachCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "detach <volume-slug>",
		Short:   "Detach a volume from its virtual machine",
		Args:    cobra.ExactArgs(1),
		Example: `  zcp volume detach <volume-slug>`,
		RunE: func(cmd *cobra.Command, args []string) error {
			volumeSlug := args[0]
			_, client, printer, err := buildClientAndPrinter(cmd)
			if err != nil {
				return err
			}
			svc := volume.NewService(client)
			ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
			defer cancel()

			vol, err := svc.Detach(ctx, volumeSlug)
			if err != nil {
				return fmt.Errorf("volume detach: %w", err)
			}

			headers := []string{"SLUG", "NAME", "SIZE"}
			rows := [][]string{{
				vol.Slug,
				vol.Name,
				vol.Size,
			}}
			return printer.PrintTable(headers, rows)
		},
	}
	return cmd
}
