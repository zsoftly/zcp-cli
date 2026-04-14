package commands

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/zsoftly/zcp-cli/internal/api/iso"
)

// NewISOCmd returns the 'iso' cobra command.
func NewISOCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "iso",
		Short: "Manage ISO images",
	}
	cmd.AddCommand(newISOListCmd())
	cmd.AddCommand(newISOCreateCmd())
	cmd.AddCommand(newISOUpdateCmd())
	cmd.AddCommand(newISODeleteCmd())
	return cmd
}

func newISOListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List ISO images",
		Example: `  zcp iso list
  zcp iso list --output json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runISOList(cmd)
		},
	}
	return cmd
}

func runISOList(cmd *cobra.Command) error {
	_, client, printer, err := buildClientAndPrinter(cmd)
	if err != nil {
		return err
	}

	svc := iso.NewService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	isos, err := svc.List(ctx)
	if err != nil {
		return fmt.Errorf("iso list: %w", err)
	}

	headers := []string{"SLUG", "NAME", "IMAGE TYPE", "BOOTABLE", "EXTRACTABLE", "PASSWORD", "CREATED"}
	rows := make([][]string, 0, len(isos))
	for _, i := range isos {
		rows = append(rows, []string{
			i.Slug,
			i.Name,
			i.ImageType,
			strconv.FormatBool(i.IsBootable),
			strconv.FormatBool(i.IsExtractable),
			strconv.FormatBool(i.PasswordEnabled),
			i.CreatedAt,
		})
	}
	return printer.PrintTable(headers, rows)
}

func newISOCreateCmd() *cobra.Command {
	var (
		name            string
		description     string
		isoURL          string
		cloudProvider   string
		project         string
		region          string
		osTypeID        string
		imageType       string
		operatingSystem string
		osVersion       string
		billingCycle    string
		passwordEnabled bool
		isExtractable   bool
		isBootable      bool
	)

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create (register) an ISO image",
		Example: `  zcp iso create --name my-iso --url https://example.com/my.iso \
    --cloud-provider zcp --project my-project --region yow-1 \
    --os-type-id <uuid> --image-type "Operating System" \
    --os ubuntu --os-version "22.04 LTS" --billing-cycle hourly`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if name == "" {
				return fmt.Errorf("--name is required")
			}
			if isoURL == "" {
				return fmt.Errorf("--url is required")
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
			req := iso.CreateRequest{
				Name:                   name,
				Description:            description,
				URL:                    isoURL,
				CloudProvider:          cloudProvider,
				Project:                project,
				Region:                 region,
				OSTypeID:               osTypeID,
				ImageType:              imageType,
				OperatingSystem:        operatingSystem,
				OperatingSystemVersion: osVersion,
				BillingCycle:           billingCycle,
				PasswordEnabled:        passwordEnabled,
				IsExtractable:          isExtractable,
				IsBootable:             isBootable,
				Service:                "ISO",
			}
			return runISOCreate(cmd, req)
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "ISO name (required)")
	cmd.Flags().StringVar(&description, "description", "", "ISO description")
	cmd.Flags().StringVar(&isoURL, "url", "", "URL to the ISO file (required)")
	cmd.Flags().StringVar(&cloudProvider, "cloud-provider", "", "Cloud provider slug (required)")
	cmd.Flags().StringVar(&project, "project", "", "Project slug (required)")
	cmd.Flags().StringVar(&region, "region", "", "Region slug (required)")
	cmd.Flags().StringVar(&osTypeID, "os-type-id", "", "OS type UUID")
	cmd.Flags().StringVar(&imageType, "image-type", "Operating System", "Image type")
	cmd.Flags().StringVar(&operatingSystem, "os", "", "Operating system name")
	cmd.Flags().StringVar(&osVersion, "os-version", "", "Operating system version")
	cmd.Flags().StringVar(&billingCycle, "billing-cycle", "hourly", "Billing cycle (hourly, monthly)")
	cmd.Flags().BoolVar(&passwordEnabled, "password-enabled", true, "Enable password on the ISO")
	cmd.Flags().BoolVar(&isExtractable, "extractable", false, "ISO is extractable")
	cmd.Flags().BoolVar(&isBootable, "bootable", false, "ISO is bootable")
	return cmd
}

func runISOCreate(cmd *cobra.Command, req iso.CreateRequest) error {
	_, client, printer, err := buildClientAndPrinter(cmd)
	if err != nil {
		return err
	}

	svc := iso.NewService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	i, err := svc.Create(ctx, req)
	if err != nil {
		return fmt.Errorf("iso create: %w", err)
	}

	headers := []string{"FIELD", "VALUE"}
	rows := [][]string{
		{"Slug", i.Slug},
		{"Name", i.Name},
		{"Image Type", i.ImageType},
		{"Bootable", strconv.FormatBool(i.IsBootable)},
		{"Extractable", strconv.FormatBool(i.IsExtractable)},
		{"Password Enabled", strconv.FormatBool(i.PasswordEnabled)},
		{"Created", i.CreatedAt},
	}
	return printer.PrintTable(headers, rows)
}

func newISOUpdateCmd() *cobra.Command {
	var (
		passwordEnabled bool
		isExtractable   bool
		isBootable      bool
	)

	cmd := &cobra.Command{
		Use:   "update <slug>",
		Short: "Update ISO permissions",
		Args:  cobra.ExactArgs(1),
		Example: `  zcp iso update my-iso --bootable --password-enabled
  zcp iso update my-iso --extractable=false`,
		RunE: func(cmd *cobra.Command, args []string) error {
			req := iso.UpdateRequest{
				PasswordEnabled: passwordEnabled,
				IsExtractable:   isExtractable,
				IsBootable:      isBootable,
			}
			return runISOUpdate(cmd, args[0], req)
		},
	}
	cmd.Flags().BoolVar(&passwordEnabled, "password-enabled", true, "Enable password")
	cmd.Flags().BoolVar(&isExtractable, "extractable", false, "ISO is extractable")
	cmd.Flags().BoolVar(&isBootable, "bootable", false, "ISO is bootable")
	return cmd
}

func runISOUpdate(cmd *cobra.Command, slug string, req iso.UpdateRequest) error {
	_, client, printer, err := buildClientAndPrinter(cmd)
	if err != nil {
		return err
	}

	svc := iso.NewService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	if err := svc.Update(ctx, slug, req); err != nil {
		return fmt.Errorf("iso update: %w", err)
	}

	printer.Fprintf("ISO %q updated.\n", slug)
	return nil
}

func newISODeleteCmd() *cobra.Command {
	var yes bool

	cmd := &cobra.Command{
		Use:   "delete <slug>",
		Short: "Delete an ISO image",
		Args:  cobra.ExactArgs(1),
		Example: `  zcp iso delete my-iso
  zcp iso delete my-iso --yes`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runISODelete(cmd, args[0], yes)
		},
	}
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "Skip confirmation prompt")
	return cmd
}

func runISODelete(cmd *cobra.Command, slug string, yes bool) error {
	if !yes && !autoApproved(cmd) {
		fmt.Fprintf(os.Stderr, "Delete ISO %q? [y/N]: ", slug)
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

	svc := iso.NewService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	if err := svc.Delete(ctx, slug); err != nil {
		return fmt.Errorf("iso delete: %w", err)
	}

	printer.Fprintf("ISO %q deleted.\n", slug)
	return nil
}
