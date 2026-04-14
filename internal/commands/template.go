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
	"github.com/zsoftly/zcp-cli/internal/api/template"
)

// NewTemplateCmd returns the 'template' cobra command.
func NewTemplateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "template",
		Short: "Manage VM templates",
	}
	cmd.AddCommand(newTemplateListCmd())
	cmd.AddCommand(newTemplateAccountListCmd())
	cmd.AddCommand(newTemplateAccountCreateCmd())
	cmd.AddCommand(newTemplateAccountDeleteCmd())
	return cmd
}

func newTemplateListCmd() *cobra.Command {
	var region string

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List available public templates",
		Example: `  zcp template list
  zcp template list --region yow-1
  zcp template list --output json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runTemplateList(cmd, region)
		},
	}
	cmd.Flags().StringVar(&region, "region", "", "Filter by region slug")
	return cmd
}

func runTemplateList(cmd *cobra.Command, region string) error {
	_, client, printer, err := buildClientAndPrinter(cmd)
	if err != nil {
		return err
	}

	svc := template.NewService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	templates, err := svc.List(ctx, region)
	if err != nil {
		return fmt.Errorf("template list: %w", err)
	}

	headers := []string{"SLUG", "NAME", "TYPE", "OS", "VERSION", "IMAGE TYPE", "PASSWORD"}
	rows := make([][]string, 0, len(templates))
	for _, t := range templates {
		osName := ""
		osVersion := ""
		if t.OperatingSystem != nil {
			osName = t.OperatingSystem.Name
		}
		if t.OperatingSystemVersion != nil {
			osVersion = t.OperatingSystemVersion.Version
		}
		rows = append(rows, []string{
			t.Slug,
			t.Name,
			t.Type,
			osName,
			osVersion,
			t.ImageType,
			strconv.FormatBool(t.PasswordEnabled),
		})
	}
	return printer.PrintTable(headers, rows)
}

func newTemplateAccountListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "account-list",
		Short: "List account (user-created) templates",
		Example: `  zcp template account-list
  zcp template account-list --output json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runTemplateAccountList(cmd)
		},
	}
	return cmd
}

func runTemplateAccountList(cmd *cobra.Command) error {
	_, client, printer, err := buildClientAndPrinter(cmd)
	if err != nil {
		return err
	}

	svc := template.NewService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	templates, err := svc.ListAccount(ctx)
	if err != nil {
		return fmt.Errorf("template account-list: %w", err)
	}

	headers := []string{"SLUG", "NAME", "FORMAT", "IMAGE TYPE", "STATUS", "CREATED"}
	rows := make([][]string, 0, len(templates))
	for _, t := range templates {
		rows = append(rows, []string{
			t.Slug,
			t.Name,
			t.Format,
			t.ImageType,
			t.Status,
			t.CreatedAt,
		})
	}
	return printer.PrintTable(headers, rows)
}

func newTemplateAccountCreateCmd() *cobra.Command {
	var (
		name            string
		description     string
		templateURL     string
		cloudProvider   string
		region          string
		project         string
		osTypeID        string
		imageType       string
		operatingSystem string
		osVersion       string
		passwordEnabled bool
		billingCycle    string
		plan            string
		format          string
	)

	cmd := &cobra.Command{
		Use:   "account-create",
		Short: "Create an account template",
		Example: `  zcp template account-create --name my-template --cloud-provider zcp \
    --region yow-1 --project my-project --os-type-id <uuid> \
    --image-type "Operating System" --os ubuntu --os-version "22.04 LTS" \
    --billing-cycle hourly --url https://example.com/image.qcow2 --format QCOW2`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if name == "" {
				return fmt.Errorf("--name is required")
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
			req := template.CreateAccountTemplateRequest{
				Name:                   name,
				Description:            description,
				URL:                    templateURL,
				CloudProvider:          cloudProvider,
				Region:                 region,
				Project:                project,
				OSTypeID:               osTypeID,
				ImageType:              imageType,
				OperatingSystem:        operatingSystem,
				OperatingSystemVersion: osVersion,
				PasswordEnabled:        passwordEnabled,
				BillingCycle:           billingCycle,
				Plan:                   plan,
				Format:                 format,
			}
			return runTemplateAccountCreate(cmd, req)
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "Template name (required)")
	cmd.Flags().StringVar(&description, "description", "", "Template description")
	cmd.Flags().StringVar(&templateURL, "url", "", "URL to the template image")
	cmd.Flags().StringVar(&cloudProvider, "cloud-provider", "", "Cloud provider slug (required)")
	cmd.Flags().StringVar(&region, "region", "", "Region slug (required)")
	cmd.Flags().StringVar(&project, "project", "", "Project slug (required)")
	cmd.Flags().StringVar(&osTypeID, "os-type-id", "", "OS type UUID")
	cmd.Flags().StringVar(&imageType, "image-type", "Operating System", "Image type")
	cmd.Flags().StringVar(&operatingSystem, "os", "", "Operating system name")
	cmd.Flags().StringVar(&osVersion, "os-version", "", "Operating system version")
	cmd.Flags().BoolVar(&passwordEnabled, "password-enabled", true, "Enable password login")
	cmd.Flags().StringVar(&billingCycle, "billing-cycle", "hourly", "Billing cycle (hourly, monthly)")
	cmd.Flags().StringVar(&plan, "plan", "", "Plan slug")
	cmd.Flags().StringVar(&format, "format", "", "Image format (QCOW2, RAW, etc.)")
	return cmd
}

func runTemplateAccountCreate(cmd *cobra.Command, req template.CreateAccountTemplateRequest) error {
	_, client, printer, err := buildClientAndPrinter(cmd)
	if err != nil {
		return err
	}

	svc := template.NewService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	t, err := svc.CreateAccount(ctx, req)
	if err != nil {
		return fmt.Errorf("template account-create: %w", err)
	}

	headers := []string{"FIELD", "VALUE"}
	rows := [][]string{
		{"Slug", t.Slug},
		{"Name", t.Name},
		{"Format", t.Format},
		{"Image Type", t.ImageType},
		{"Status", t.Status},
		{"Created", t.CreatedAt},
	}
	return printer.PrintTable(headers, rows)
}

func newTemplateAccountDeleteCmd() *cobra.Command {
	var yes bool

	cmd := &cobra.Command{
		Use:   "account-delete <slug>",
		Short: "Delete an account template",
		Args:  cobra.ExactArgs(1),
		Example: `  zcp template account-delete my-template
  zcp template account-delete my-template --yes`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runTemplateAccountDelete(cmd, args[0], yes)
		},
	}
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "Skip confirmation prompt")
	return cmd
}

func runTemplateAccountDelete(cmd *cobra.Command, slug string, yes bool) error {
	if !yes && !autoApproved(cmd) {
		fmt.Fprintf(os.Stderr, "Delete account template %q? [y/N]: ", slug)
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

	svc := template.NewService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	if err := svc.DeleteAccount(ctx, slug); err != nil {
		return fmt.Errorf("template account-delete: %w", err)
	}

	printer.Fprintf("Account template %q deleted.\n", slug)
	return nil
}
