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
	"github.com/zsoftly/zcp-cli/internal/api/dns"
)

// NewDNSCmd returns the 'dns' cobra command.
func NewDNSCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "dns",
		Short: "Manage DNS domains and records",
	}
	cmd.AddCommand(newDNSListCmd())
	cmd.AddCommand(newDNSShowCmd())
	cmd.AddCommand(newDNSCreateCmd())
	cmd.AddCommand(newDNSDeleteCmd())
	cmd.AddCommand(newDNSRecordCreateCmd())
	cmd.AddCommand(newDNSRecordDeleteCmd())
	return cmd
}

func newDNSListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List DNS domains",
		Example: `  zcp dns list
  zcp dns list --output json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDNSList(cmd)
		},
	}
	return cmd
}

func runDNSList(cmd *cobra.Command) error {
	_, client, printer, err := buildClientAndPrinter(cmd)
	if err != nil {
		return err
	}

	svc := dns.NewService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	domains, err := svc.List(ctx)
	if err != nil {
		return fmt.Errorf("dns list: %w", err)
	}

	headers := []string{"SLUG", "NAME", "DNS PROVIDER", "STATUS", "CREATED"}
	rows := make([][]string, 0, len(domains))
	for _, d := range domains {
		rows = append(rows, []string{
			d.Slug,
			d.Name,
			d.DNSProvider,
			d.Status,
			d.CreatedAt,
		})
	}
	return printer.PrintTable(headers, rows)
}

func newDNSShowCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "show <slug>",
		Short: "Show DNS domain details and records",
		Args:  cobra.ExactArgs(1),
		Example: `  zcp dns show example-com-1
  zcp dns show example-com-1 --output json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDNSShow(cmd, args[0])
		},
	}
	return cmd
}

func runDNSShow(cmd *cobra.Command, slug string) error {
	_, client, printer, err := buildClientAndPrinter(cmd)
	if err != nil {
		return err
	}

	svc := dns.NewService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	domain, err := svc.Show(ctx, slug)
	if err != nil {
		return fmt.Errorf("dns show: %w", err)
	}

	// Print domain details
	detailHeaders := []string{"FIELD", "VALUE"}
	detailRows := [][]string{
		{"Slug", domain.Slug},
		{"Name", domain.Name},
		{"DNS Provider", domain.DNSProvider},
		{"Status", domain.Status},
		{"Created", domain.CreatedAt},
		{"Updated", domain.UpdatedAt},
	}
	if err := printer.PrintTable(detailHeaders, detailRows); err != nil {
		return err
	}

	// Print records if any
	if len(domain.Records) > 0 {
		fmt.Fprintln(os.Stderr)
		fmt.Fprintf(os.Stderr, "Records (%d):\n", len(domain.Records))
		recHeaders := []string{"ID", "NAME", "TYPE", "CONTENT", "TTL"}
		recRows := make([][]string, 0, len(domain.Records))
		for _, r := range domain.Records {
			recRows = append(recRows, []string{
				r.ID,
				r.Name,
				r.Type,
				r.Content,
				strconv.Itoa(r.TTL),
			})
		}
		return printer.PrintTable(recHeaders, recRows)
	}

	return nil
}

func newDNSCreateCmd() *cobra.Command {
	var name, project, dnsProvider string
	var cloudProvider, region string

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a DNS domain",
		Example: `  zcp dns create --name example.com --project my-project --dns-provider dns-provider --cloud-provider <slug> --region <slug>
  zcp dns create --name example.com --project my-project --cloud-provider <slug> --region <slug>`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if name == "" {
				return fmt.Errorf("--name is required")
			}
			project = resolveProject(project)
			if project == "" {
				return fmt.Errorf("--project is required")
			}
			cloudProvider = resolveCloudProvider(cloudProvider)
			if cloudProvider == "" {
				return fmt.Errorf("--cloud-provider is required")
			}
			region = resolveRegion(region)
			if region == "" {
				return fmt.Errorf("--region is required")
			}
			if dnsProvider == "" {
				dnsProvider = "powerdns"
			}
			return runDNSCreate(cmd, dns.CreateDomainRequest{
				Name:          name,
				Project:       project,
				DNSProvider:   dnsProvider,
				CloudProvider: cloudProvider,
				Region:        region,
			})
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "Domain name (required, e.g. example.com)")
	cmd.Flags().StringVar(&project, "project", "", "Project slug (required, e.g. my-project)")
	cmd.Flags().StringVar(&dnsProvider, "dns-provider", "powerdns", "DNS provider (default: powerdns)")
	cmd.Flags().StringVar(&cloudProvider, "cloud-provider", "", "Cloud provider slug (required)")
	cmd.Flags().StringVar(&region, "region", "", "Region slug (required)")
	return cmd
}

func runDNSCreate(cmd *cobra.Command, req dns.CreateDomainRequest) error {
	_, client, printer, err := buildClientAndPrinter(cmd)
	if err != nil {
		return err
	}

	svc := dns.NewService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	domain, err := svc.Create(ctx, req)
	if err != nil {
		return fmt.Errorf("dns create: %w", err)
	}

	headers := []string{"FIELD", "VALUE"}
	rows := [][]string{
		{"Slug", domain.Slug},
		{"Name", domain.Name},
		{"DNS Provider", domain.DNSProvider},
		{"Status", domain.Status},
		{"Created", domain.CreatedAt},
	}
	return printer.PrintTable(headers, rows)
}

func newDNSDeleteCmd() *cobra.Command {
	var yes bool

	cmd := &cobra.Command{
		Use:   "delete <slug>",
		Short: "Delete a DNS domain",
		Args:  cobra.ExactArgs(1),
		Example: `  zcp dns delete example-com-1
  zcp dns delete example-com-1 --yes`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDNSDelete(cmd, args[0], yes)
		},
	}
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "Skip confirmation prompt")
	return cmd
}

func runDNSDelete(cmd *cobra.Command, slug string, yes bool) error {
	if !yes && !autoApproved(cmd) {
		fmt.Fprintf(os.Stderr, "Delete DNS domain %q? This will remove all records. [y/N]: ", slug)
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

	svc := dns.NewService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	if err := svc.Delete(ctx, slug); err != nil {
		return fmt.Errorf("dns delete: %w", err)
	}

	printer.Fprintf("DNS domain %q deleted.\n", slug)
	return nil
}

func newDNSRecordCreateCmd() *cobra.Command {
	var domain, name, recordType, content string
	var ttl int

	cmd := &cobra.Command{
		Use:   "record-create",
		Short: "Create a DNS record",
		Example: `  zcp dns record-create --domain example-com-1 --name www --type A --content 192.0.2.1
  zcp dns record-create --domain example-com-1 --name mail --type MX --content mail.example.com --ttl 3600`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if domain == "" {
				return fmt.Errorf("--domain is required (use the domain slug)")
			}
			if name == "" {
				return fmt.Errorf("--name is required")
			}
			if recordType == "" {
				return fmt.Errorf("--type is required (e.g. A, AAAA, CNAME, MX, TXT)")
			}
			if content == "" {
				return fmt.Errorf("--content is required")
			}
			if ttl <= 0 {
				ttl = 14400
			}
			return runDNSRecordCreate(cmd, domain, dns.CreateRecordRequest{
				Name:    name,
				Type:    recordType,
				Content: content,
				TTL:     ttl,
			})
		},
	}
	cmd.Flags().StringVar(&domain, "domain", "", "Domain slug (required)")
	cmd.Flags().StringVar(&name, "name", "", "Record name / subdomain (required)")
	cmd.Flags().StringVar(&recordType, "type", "", "Record type: A, AAAA, CNAME, MX, TXT, etc. (required)")
	cmd.Flags().StringVar(&content, "content", "", "Record content / value (required)")
	cmd.Flags().IntVar(&ttl, "ttl", 14400, "Time-to-live in seconds (default: 14400)")
	return cmd
}

func runDNSRecordCreate(cmd *cobra.Command, domainSlug string, req dns.CreateRecordRequest) error {
	_, client, printer, err := buildClientAndPrinter(cmd)
	if err != nil {
		return err
	}

	svc := dns.NewService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	domain, err := svc.CreateRecord(ctx, domainSlug, req)
	if err != nil {
		return fmt.Errorf("dns record-create: %w", err)
	}

	// Show the record table for the domain
	if len(domain.Records) > 0 {
		headers := []string{"ID", "NAME", "TYPE", "CONTENT", "TTL"}
		rows := make([][]string, 0, len(domain.Records))
		for _, r := range domain.Records {
			rows = append(rows, []string{
				r.ID,
				r.Name,
				r.Type,
				r.Content,
				strconv.Itoa(r.TTL),
			})
		}
		return printer.PrintTable(headers, rows)
	}

	printer.Fprintf("Record created on domain %q.\n", domainSlug)
	return nil
}

func newDNSRecordDeleteCmd() *cobra.Command {
	var domain string
	var recordID int
	var yes bool

	cmd := &cobra.Command{
		Use:   "record-delete",
		Short: "Delete a DNS record",
		Example: `  zcp dns record-delete --domain example-com-1 --record-id 42
  zcp dns record-delete --domain example-com-1 --record-id 42 --yes`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if domain == "" {
				return fmt.Errorf("--domain is required (use the domain slug)")
			}
			if recordID <= 0 {
				return fmt.Errorf("--record-id is required")
			}
			return runDNSRecordDelete(cmd, domain, recordID, yes)
		},
	}
	cmd.Flags().StringVar(&domain, "domain", "", "Domain slug (required)")
	cmd.Flags().IntVar(&recordID, "record-id", 0, "Record ID to delete (required; use 'dns show' to find IDs)")
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "Skip confirmation prompt")
	return cmd
}

func runDNSRecordDelete(cmd *cobra.Command, domainSlug string, recordID int, yes bool) error {
	if !yes && !autoApproved(cmd) {
		fmt.Fprintf(os.Stderr, "Delete DNS record %d on domain %q? [y/N]: ", recordID, domainSlug)
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

	svc := dns.NewService(client)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	if err := svc.DeleteRecord(ctx, domainSlug, recordID); err != nil {
		return fmt.Errorf("dns record-delete: %w", err)
	}

	printer.Fprintf("DNS record %d deleted from domain %q.\n", recordID, domainSlug)
	return nil
}
