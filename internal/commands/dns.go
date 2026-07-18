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
	"github.com/zsoftly/zcp-cli/pkg/api/apierrors"
	"github.com/zsoftly/zcp-cli/pkg/api/dns"
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

	headers := []string{"SLUG", "NAME", "STATUS", "CREATED"}
	rows := make([][]string, 0, len(domains))
	for _, d := range domains {
		rows = append(rows, []string{
			d.Slug,
			d.Name,
			strconv.FormatBool(d.Status),
			d.CreatedAt,
		})
	}
	return printer.PrintTable(headers, rows)
}

func newDNSShowCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "show <slug>",
		Short: "Show DNS domain details and records",
		Args:  exactArgs(1),
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
		{"Status", strconv.FormatBool(domain.Status)},
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
		// The live DNS backend returns record sets without IDs, so the table
		// is keyed by name and type.
		recHeaders := []string{"NAME", "TYPE", "CONTENT", "TTL"}
		recRows := make([][]string, 0, len(domain.Records))
		for _, r := range domain.Records {
			recRows = append(recRows, []string{
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
		Use:     "create",
		Short:   "Create a DNS domain",
		Example: `  zcp dns create --name example.com --project default-9`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if name == "" {
				return fmt.Errorf("--name is required")
			}
			project = resolveProject(project)
			if project == "" {
				return fmt.Errorf("--project is required")
			}
			// DNS is served by the dedicated "dns" cloud provider, which has a
			// single region ("default"). Both are verified against the live
			// /cloud-providers and /regions endpoints (region "default" maps to
			// provider "dns"). Default to them so DNS is hands-off; an explicit
			// --cloud-provider / --region still overrides. ZCP_REGION is ignored
			// here because it targets compute regions, which DNS cannot use.
			cloudProvider = cloudProviderFlagOrEnv(cloudProvider)
			if cloudProvider == "" {
				cloudProvider = "dns"
			}
			if region == "" {
				region = "default"
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
	cmd.Flags().StringVar(&project, "project", "", "Project slug (required, e.g. default-9)")
	// --dns-provider selects the DNS backend; it is an internal detail with a
	// working default, so it is hidden from help (still usable as an override).
	cmd.Flags().StringVar(&dnsProvider, "dns-provider", "powerdns", "DNS backend (internal; optional override)")
	_ = cmd.Flags().MarkHidden("dns-provider")
	cmd.Flags().StringVar(&cloudProvider, "cloud-provider", "", "Cloud provider slug (default: dns)")
	cmd.Flags().StringVar(&region, "region", "", "Region slug (default: default, the DNS region)")
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
		{"Status", strconv.FormatBool(domain.Status)},
		{"Created", domain.CreatedAt},
	}
	return printer.PrintTable(headers, rows)
}

func newDNSDeleteCmd() *cobra.Command {
	var yes bool

	cmd := &cobra.Command{
		Use:   "delete <slug>",
		Short: "Delete a DNS domain",
		Args:  exactArgs(1),
		Example: `  zcp dns delete example-com-1
  zcp dns delete example-com-1 --yes`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDNSDelete(cmd, args[0], yes)
		},
	}
	cmd.Flags().BoolVar(&yes, "yes", false, "Skip confirmation prompt")
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
		if apierrors.IsResourceNotFound(err) {
			fmt.Fprintf(os.Stderr, "DNS domain %q not found (already deleted).\n", slug)
			return nil
		}
		return fmt.Errorf("dns delete: %w", err)
	}

	printer.Fprintf("DNS domain %q deleted.\n", slug)
	return nil
}

func newDNSRecordCreateCmd() *cobra.Command {
	var domain, name, recordType, content string
	var ttl, priority int

	cmd := &cobra.Command{
		Use:   "record-create",
		Short: "Create a DNS record",
		Example: `  zcp dns record-create --domain example-com-1 --name www --type A --content 192.0.2.1
  zcp dns record-create --domain example-com-1 --name @ --type MX --content mail.example.com. --priority 10 --ttl 3600`,
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
			req := dns.CreateRecordRequest{
				Name:    name,
				Type:    recordType,
				Content: content,
				TTL:     ttl,
			}
			// Priority applies only to MX records, which carry their preference
			// in a separate field the backend requires; sent without one they
			// are rejected. Require --priority for MX and reject it for every
			// other type, so both failures are clear CLI messages rather than
			// server-side errors. SRV also uses priority/weight/port/target,
			// but the ZCP CMP DNS API rejects SRV (and LOC) in every request
			// shape while accepting A/AAAA/CNAME/MX/TXT/CAA. PowerDNS itself
			// stores and serves SRV fine (verified live 2026-07-18), so the gap
			// is in the CMP layer; SRV stays unsupported here until CMP adds it.
			prioritySet := cmd.Flags().Changed("priority")
			isMX := strings.EqualFold(recordType, "MX")
			if isMX && !prioritySet {
				return fmt.Errorf("--priority is required for MX records (e.g. --priority 10)")
			}
			if prioritySet {
				if !isMX {
					return fmt.Errorf("--priority is only valid for MX records")
				}
				if priority < 0 || priority > 65535 {
					return fmt.Errorf("--priority must be between 0 and 65535")
				}
				p := priority
				req.Priority = &p
			}
			return runDNSRecordCreate(cmd, domain, req)
		},
	}
	cmd.Flags().StringVar(&domain, "domain", "", "Domain slug (required)")
	cmd.Flags().StringVar(&name, "name", "", "Relative record name, e.g. www. The backend appends the zone (required)")
	cmd.Flags().StringVar(&recordType, "type", "", "Record type: A, AAAA, CNAME, MX, TXT, etc. (required)")
	cmd.Flags().StringVar(&content, "content", "", "Record content / value (required)")
	cmd.Flags().IntVar(&ttl, "ttl", 14400, "Time-to-live in seconds (default: 14400)")
	cmd.Flags().IntVar(&priority, "priority", 0, "Preference for MX records, 0-65535 (required for MX)")
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
		headers := []string{"NAME", "TYPE", "CONTENT", "TTL"}
		rows := make([][]string, 0, len(domain.Records))
		for _, r := range domain.Records {
			rows = append(rows, []string{
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
	var domain, name, recordType string
	var recordID int
	var yes bool

	cmd := &cobra.Command{
		Use:   "record-delete",
		Short: "Delete a DNS record set by name and type",
		Example: `  zcp dns record-delete --domain example-com-1 --name www --type A
  zcp dns record-delete --domain example-com-1 --name www --type A --yes`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if domain == "" {
				return fmt.Errorf("--domain is required (use the domain slug)")
			}
			if recordID > 0 {
				// Legacy numeric-ID path; only works on deployments whose DNS
				// backend exposes record IDs.
				return runDNSRecordDelete(cmd, domain, recordID, yes)
			}
			if name == "" || recordType == "" {
				return fmt.Errorf("--name and --type are required (records are addressed by name and type)")
			}
			return runDNSRecordDeleteByName(cmd, domain, name, recordType, yes)
		},
	}
	cmd.Flags().StringVar(&domain, "domain", "", "Domain slug (required)")
	cmd.Flags().StringVar(&name, "name", "", "Record name, relative (www) or fully qualified (www.example.com.)")
	cmd.Flags().StringVar(&recordType, "type", "", "Record type: A, AAAA, CNAME, MX, TXT, etc.")
	cmd.Flags().IntVar(&recordID, "record-id", 0, "Legacy: numeric record ID (most deployments do not expose IDs)")
	cmd.Flags().BoolVar(&yes, "yes", false, "Skip confirmation prompt")
	return cmd
}

func runDNSRecordDeleteByName(cmd *cobra.Command, domainSlug, name, recordType string, yes bool) error {
	if !yes && !autoApproved(cmd) {
		fmt.Fprintf(os.Stderr, "Delete DNS record %s %q on domain %q? [y/N]: ", recordType, name, domainSlug)
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

	// Resolve the stored FQDN: the backend appends the zone to relative names.
	domain, err := svc.Show(ctx, domainSlug)
	if err != nil {
		return fmt.Errorf("dns record-delete: resolving domain %s: %w", domainSlug, err)
	}
	fqdn := dns.CanonicalRecordFQDN(name, domain.Name)

	if err := svc.DeleteRecordByName(ctx, domainSlug, fqdn, recordType); err != nil {
		if apierrors.IsResourceNotFound(err) {
			fmt.Fprintf(os.Stderr, "DNS record %s %q not found (already deleted).\n", recordType, fqdn)
			return nil
		}
		return fmt.Errorf("dns record-delete: %w", err)
	}

	printer.Fprintf("DNS record %s %q deleted from domain %q.\n", recordType, fqdn, domainSlug)
	return nil
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
		if apierrors.IsResourceNotFound(err) {
			fmt.Fprintf(os.Stderr, "DNS record %d not found (already deleted).\n", recordID)
			return nil
		}
		return fmt.Errorf("dns record-delete: %w", err)
	}

	printer.Fprintf("DNS record %d deleted from domain %q.\n", recordID, domainSlug)
	return nil
}
