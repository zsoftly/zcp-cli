package commands

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/zsoftly/zcp-cli/pkg/api/apierrors"
	"github.com/zsoftly/zcp-cli/pkg/api/volume"
)

// formatSize normalizes a json.Number size into a clean display string.
func formatSize(size json.Number) string {
	s := size.String()
	if s == "" {
		return "-"
	}
	// Strip trailing .0 for whole numbers (e.g. "50.0" -> "50")
	s = strings.TrimSuffix(s, ".0")
	return s
}

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
	cmd.AddCommand(newVolumeDeleteCmd())
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

			region, project := scopedRegionProject(cmd)
			volumes, err := svc.List(ctx, region, project)
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
					formatSize(v.Size),
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
	var size int
	var isFreeTrial bool

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new block storage volume",
		Example: `  zcp volume create --name my-disk --project default-9 --region yul-1 --billing-cycle hourly --storage-category pro-nvme --plan b2g1
  zcp volume create --name my-disk --project default-9 --region yul-1 --billing-cycle hourly --storage-category pro-nvme --size 50
  zcp volume create --name my-disk --project default-9 --region yul-1 --billing-cycle hourly --storage-category pro-nvme --plan b2g1 --vm vm-slug`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if name == "" {
				return fmt.Errorf("--name is required")
			}
			project = resolveProject(project)
			if project == "" {
				return fmt.Errorf("--project is required")
			}
			cloudProvider = resolveCloudProvider(cmd, cloudProvider)
			if cloudProvider == "" {
				return fmt.Errorf("could not determine cloud provider — run 'zcp auth validate' to detect it, or pass --cloud-provider (see 'zcp cloud-provider list')")
			}
			region = resolveRegion(region)
			if region == "" {
				return fmt.Errorf("--region is required")
			}
			if billingCycle == "" {
				return fmt.Errorf("--billing-cycle is required")
			}
			if storageCategory == "" {
				return fmt.Errorf("--storage-category is required")
			}
			sizeChanged := cmd.Flags().Changed("size")
			if plan != "" && sizeChanged {
				return fmt.Errorf("--plan and --size are mutually exclusive")
			}
			if plan == "" && !sizeChanged {
				return fmt.Errorf("--plan or --size is required")
			}
			if sizeChanged && size <= 0 {
				return fmt.Errorf("--size must be > 0")
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
				IsCustomPlan:    size > 0,
				VirtualMachine:  vmSlug,
				Coupon:          coupon,
				IsFreeTrial:     isFreeTrial,
			}
			if size > 0 {
				req.CustomPlan = &volume.CustomPlanStorage{Storage: size}
			}
			vol, err := svc.Create(ctx, req)
			if err != nil {
				return fmt.Errorf("volume create: %w", err)
			}

			headers := []string{"SLUG", "NAME", "SIZE", "TYPE", "CREATED"}
			rows := [][]string{{
				vol.Slug,
				vol.Name,
				formatSize(vol.Size),
				vol.VolumeType,
				vol.CreatedAt,
			}}
			return printer.PrintTable(headers, rows)
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "Volume name (required)")
	cmd.Flags().StringVar(&project, "project", "", "Project slug (required)")
	cmd.Flags().StringVar(&cloudProvider, "cloud-provider", "", "Cloud provider slug (optional; auto-detected, override only)")
	cmd.Flags().StringVar(&region, "region", "", "Region slug (required)")
	cmd.Flags().StringVar(&billingCycle, "billing-cycle", "", "Billing cycle slug, e.g. hourly (required)")
	cmd.Flags().StringVar(&storageCategory, "storage-category", "", "Storage category slug, e.g. pro-nvme (required)")
	cmd.Flags().StringVar(&plan, "plan", "", "Plan slug, e.g. b2g1 (mutually exclusive with --size)")
	cmd.Flags().IntVar(&size, "size", 0, "Storage size in GB for custom-tier plans (mutually exclusive with --plan)")
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
		Args:    exactArgs(1),
		Example: `  zcp volume attach bs-001001-0042 --vm my-vm`,
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
				formatSize(vol.Size),
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
		Args:    exactArgs(1),
		Example: `  zcp volume detach bs-001001-0042`,
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
				if apierrors.IsResourceNotFound(err) {
					fmt.Fprintf(os.Stderr, "Volume %q not found — already detached or deleted.\n", volumeSlug)
					return nil
				}
				return fmt.Errorf("volume detach: %w", err)
			}

			headers := []string{"SLUG", "NAME", "SIZE"}
			rows := [][]string{{
				vol.Slug,
				vol.Name,
				formatSize(vol.Size),
			}}
			return printer.PrintTable(headers, rows)
		},
	}
	return cmd
}

func newVolumeDeleteCmd() *cobra.Command {
	var yes bool

	cmd := &cobra.Command{
		Use:   "delete <volume-slug>",
		Short: "Permanently delete a block storage volume",
		Args:  exactArgs(1),
		Example: `  zcp volume delete bs-001001-0042
  zcp volume delete bs-001001-0042 --yes`,
		RunE: func(cmd *cobra.Command, args []string) error {
			slug := args[0]
			if !yes && !autoApproved(cmd) {
				fmt.Fprintf(os.Stderr, "Delete volume %q? This cannot be undone. Detach the volume first if it is attached. [y/N]: ", slug)
				scanner := bufio.NewScanner(os.Stdin)
				scanner.Scan()
				answer := strings.TrimSpace(strings.ToLower(scanner.Text()))
				if answer != "y" && answer != "yes" {
					fmt.Fprintln(os.Stderr, "Aborted.")
					return nil
				}
			}
			_, client, _, err := buildClientAndPrinter(cmd)
			if err != nil {
				return err
			}
			svc := volume.NewService(client)
			ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
			defer cancel()
			if err := svc.Delete(ctx, slug); err != nil {
				if apierrors.IsResourceNotFound(err) {
					fmt.Fprintf(os.Stderr, "Volume %q not found — already deleted.\n", slug)
					return nil
				}
				return fmt.Errorf("volume delete: %w", err)
			}
			fmt.Fprintf(os.Stdout, "Volume %q deleted.\n", slug)
			return nil
		},
	}
	cmd.Flags().BoolVar(&yes, "yes", false, "Skip confirmation prompt")
	return cmd
}
