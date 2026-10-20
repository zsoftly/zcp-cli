package commands

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"net/netip"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/zsoftly/zcp-cli/internal/output"
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
			domainStatusCell(&d),
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

// domainStatusCell renders a domain's status for display. A missing status is
// displayed as "-" rather than being confused with an explicit false.
func domainStatusCell(d *dns.Domain) string {
	if !d.StatusKnown {
		return "-"
	}
	return strconv.FormatBool(d.Status)
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
	if !domain.StatusKnown {
		if err := resolveDNSDomainStatus(ctx, svc, domain); err != nil {
			return fmt.Errorf("dns show: %w", err)
		}
	}

	// Print domain details
	detailHeaders := []string{"FIELD", "VALUE"}
	detailRows := [][]string{
		{"Slug", domain.Slug},
		{"Name", domain.Name},
		{"Status", domainStatusCell(domain)},
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

// resolveDNSDomainStatus fills a status omitted by the detail endpoint from a
// matching list entry that reports it. A missing matching entry remains unknown
// rather than being guessed from another field.
func resolveDNSDomainStatus(ctx context.Context, svc *dns.Service, domain *dns.Domain) error {
	listed, err := svc.FindBySlug(ctx, domain.Slug)
	if err != nil {
		return fmt.Errorf("resolving status for DNS domain %s: %w", domain.Slug, err)
	}
	if listed != nil && listed.StatusKnown {
		domain.Status = listed.Status
		domain.StatusKnown = true
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
		{"Status", domainStatusCell(domain)},
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
	var showZone bool

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
			if strings.EqualFold(recordType, "CAA") {
				if err := validateCAARecordContent(content); err != nil {
					return err
				}
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
			return runDNSRecordCreate(cmd, domain, req, showZone)
		},
	}
	cmd.Flags().StringVar(&domain, "domain", "", "Domain slug (required)")
	cmd.Flags().StringVar(&name, "name", "", "Relative record name, e.g. www. The backend appends the zone (required)")
	cmd.Flags().StringVar(&recordType, "type", "", "Record type: A, AAAA, CNAME, MX, TXT, etc. (required)")
	cmd.Flags().StringVar(&content, "content", "", "Record content / value (required)")
	cmd.Flags().IntVar(&ttl, "ttl", 14400, "Time-to-live in seconds (default: 14400)")
	cmd.Flags().IntVar(&priority, "priority", 0, "Preference for MX records, 0-65535 (required for MX)")
	cmd.Flags().BoolVar(&showZone, "show-zone", false, "Show all records in the resulting zone")
	return cmd
}

func runDNSRecordCreate(cmd *cobra.Command, domainSlug string, req dns.CreateRecordRequest, showZone bool) error {
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

	if showZone {
		if len(domain.Records) == 0 {
			domain, err = svc.Show(ctx, domainSlug)
			if err != nil {
				return fmt.Errorf("dns record-create: record created but could not verify resulting zone: %w", err)
			}
		}
		return printDNSRecordRows(printer, domain.Records)
	}

	created := findCreatedDNSRecord(domain, req)
	if created == nil {
		domain, err = svc.Show(ctx, domainSlug)
		if err != nil {
			return fmt.Errorf("dns record-create: record created but output verification failed: %w", err)
		}
		created = findCreatedDNSRecord(domain, req)
	}
	if created == nil {
		fmt.Fprintf(os.Stderr, "DNS record created, but its value could not be matched. Use dns show %q to inspect the zone.\n", domainSlug)
		return printDNSRecordRows(printer, nil)
	}
	return printDNSRecordRows(printer, []dns.Record{*created})
}

func newDNSRecordDeleteCmd() *cobra.Command {
	var domain, name, recordType string
	var recordID int
	var yes bool
	var showZone bool

	cmd := &cobra.Command{
		Use:   "record-delete",
		Short: "Delete a DNS record set by name and type",
		Long:  "Delete a DNS record set by name and type. By default, successful deletion output is a pre-delete snapshot, marked with SOURCE. Use --show-zone to show the resulting zone.",
		Example: `  zcp dns record-delete --domain example-com-1 --name www --type A
  zcp dns record-delete --domain example-com-1 --name www --type A --yes`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if domain == "" {
				return fmt.Errorf("--domain is required (use the domain slug)")
			}
			if recordID > 0 {
				// Legacy numeric-ID path; only works on deployments whose DNS
				// backend exposes record IDs.
				return runDNSRecordDelete(cmd, domain, recordID, yes, showZone)
			}
			if name == "" || recordType == "" {
				return fmt.Errorf("--name and --type are required (records are addressed by name and type)")
			}
			return runDNSRecordDeleteByName(cmd, domain, name, recordType, yes, showZone)
		},
	}
	cmd.Flags().StringVar(&domain, "domain", "", "Domain slug (required)")
	cmd.Flags().StringVar(&name, "name", "", "Record name, relative (www) or fully qualified (www.example.com.)")
	cmd.Flags().StringVar(&recordType, "type", "", "Record type: A, AAAA, CNAME, MX, TXT, etc.")
	cmd.Flags().IntVar(&recordID, "record-id", 0, "Legacy: numeric record ID (most deployments do not expose IDs)")
	cmd.Flags().BoolVar(&yes, "yes", false, "Skip confirmation prompt")
	cmd.Flags().BoolVar(&showZone, "show-zone", false, "Show all records in the resulting zone")
	return cmd
}

func runDNSRecordDeleteByName(cmd *cobra.Command, domainSlug, name, recordType string, yes, showZone bool) error {
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

	// Read the record set before deleting it so successful commands can report
	// a pre-delete snapshot. A fully qualified name does not need the zone name
	// to address the DELETE endpoint, so preserve that operation when this best-
	// effort read is unavailable. Relative names and the zone apex require the
	// read to determine their FQDN and must therefore fail closed.
	preReadCtx, cancelPreRead := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	domain, showErr := svc.Show(preReadCtx, domainSlug)
	cancelPreRead()

	fqdn, suppliedFQDN := suppliedDNSRecordFQDN(name)
	var snapshot []dns.Record
	if showErr == nil {
		if !suppliedFQDN {
			fqdn = dns.CanonicalRecordFQDN(name, domain.Name)
		}
		snapshot = findDNSRecordsByNameAndType(domain.Records, domain.Name, fqdn, recordType)
	} else {
		if apierrors.IsUnauthorized(showErr) || apierrors.IsForbidden(showErr) {
			return fmt.Errorf("dns record-delete: resolving domain %s: %w", domainSlug, showErr)
		}
		if !suppliedFQDN {
			return fmt.Errorf("dns record-delete: could not resolve relative record name %q because reading domain %q failed: %w; use a fully qualified name ending in a dot", name, domainSlug, showErr)
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	if showErr == nil && len(snapshot) == 0 {
		if showZone {
			return printDNSRecordRows(printer, domain.Records)
		}
		return printDNSRecordRows(printer, nil)
	}

	if len(snapshot) == 0 {
		if err := svc.DeleteRecordByName(ctx, domainSlug, fqdn, recordType); err != nil {
			if isDNSRecordNotFound(err) {
				fmt.Fprintf(os.Stderr, "DNS record %s %q not found (already deleted).\n", recordType, fqdn)
				if !showZone {
					return printDNSRecordRows(printer, nil)
				}
			} else {
				return fmt.Errorf("dns record-delete: %w", err)
			}
		}
	} else {
		values := dnsRecordValues(snapshot)
		deleted := 0
		for _, value := range values {
			if err := svc.DeleteRecordValueByName(ctx, domainSlug, fqdn, recordType, value); err != nil {
				if isDNSRecordNotFound(err) {
					continue
				}
				fmt.Fprintf(os.Stderr, "DNS record-delete stopped after deleting %d of %d values from the pre-delete snapshot; remaining snapshot values may still exist.\n", deleted, len(values))
				return fmt.Errorf("dns record-delete: %w", err)
			}
			deleted++
		}
	}

	if showZone {
		domain, err = svc.Show(ctx, domainSlug)
		if err != nil {
			return fmt.Errorf("dns record-delete: record deleted but could not verify resulting zone: %w", err)
		}
		return printDNSRecordRows(printer, domain.Records)
	}
	if len(snapshot) == 0 {
		fmt.Fprintf(os.Stderr, "DNS record deleted, but its previous values are unavailable. Use dns show %q to inspect the remaining zone.\n", domainSlug)
	} else {
		fmt.Fprintln(os.Stderr, "DNS record deleted; output is a snapshot read before deletion and may not match values removed if the record changed concurrently.")
	}
	return printDNSRecordSnapshotRows(printer, snapshot)
}

// isDNSRecordNotFound handles the DNS endpoint's verified idempotency
// response. Unlike the normal API not-found status, it reports an absent DNS
// record as HTTP 500 with this exact message. Do not broaden this exception:
// other 500 responses remain failures.
func isDNSRecordNotFound(err error) bool {
	if apierrors.IsResourceNotFound(err) {
		return true
	}
	var apiErr *apierrors.APIError
	return errors.As(err, &apiErr) && apiErr.StatusCode == 500 && strings.TrimSpace(apiErr.Message) == "DNS record not found."
}

// suppliedDNSRecordFQDN returns a normalized FQDN only when the caller made
// its absolute form explicit with a trailing dot. It never derives a name from
// a domain slug, because slugs are not DNS zone names.
func suppliedDNSRecordFQDN(name string) (string, bool) {
	name = strings.TrimSpace(name)
	if !strings.HasSuffix(name, ".") {
		return "", false
	}
	name = strings.TrimSuffix(name, ".")
	if name == "" || name == "@" {
		return "", false
	}
	return strings.ToLower(name) + ".", true
}

func runDNSRecordDelete(cmd *cobra.Command, domainSlug string, recordID int, yes, showZone bool) error {
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

	var snapshot *dns.Record
	preReadCtx, cancelPreRead := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	domain, showErr := svc.Show(preReadCtx, domainSlug)
	cancelPreRead()
	if showErr == nil {
		for i := range domain.Records {
			if domain.Records[i].ID == strconv.Itoa(recordID) {
				snapshot = &domain.Records[i]
				break
			}
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(getTimeout(cmd))*time.Second)
	defer cancel()

	//lint:ignore SA1019 --record-id intentionally supports legacy DNS deployments that expose numeric record IDs.
	if err := svc.DeleteRecord(ctx, domainSlug, recordID); err != nil {
		if apierrors.IsResourceNotFound(err) {
			fmt.Fprintf(os.Stderr, "DNS record %d not found (already deleted).\n", recordID)
			if !showZone {
				return printDNSRecordRows(printer, nil)
			}
		} else {
			return fmt.Errorf("dns record-delete: %w", err)
		}
	}

	if showZone {
		domain, err = svc.Show(ctx, domainSlug)
		if err != nil {
			return fmt.Errorf("dns record-delete: record deleted but could not verify resulting zone: %w", err)
		}
		return printDNSRecordRows(printer, domain.Records)
	}
	if snapshot == nil {
		fmt.Fprintf(os.Stderr, "DNS record deleted, but its previous values are unavailable. Use dns show %q to inspect the remaining zone.\n", domainSlug)
		return printDNSRecordRows(printer, nil)
	}
	fmt.Fprintln(os.Stderr, "DNS record deleted; output is a snapshot read before deletion and may not match values removed if the record changed concurrently.")
	return printDNSRecordSnapshotRows(printer, []dns.Record{*snapshot})
}

var dnsRecordHeaders = []string{"NAME", "TYPE", "CONTENT", "TTL", "PRIORITY"}
var dnsRecordSnapshotHeaders = append(append([]string(nil), dnsRecordHeaders...), "SOURCE")

func printDNSRecordRows(printer *output.Printer, records []dns.Record) error {
	rows := make([][]string, 0, len(records))
	for _, record := range records {
		rows = append(rows, dnsRecordRows(record)...)
	}
	return printer.PrintTable(dnsRecordHeaders, rows)
}

func printDNSRecordSnapshotRows(printer *output.Printer, records []dns.Record) error {
	source := "Pre-delete snapshot"
	if printer.Format() == output.FormatJSON || printer.Format() == output.FormatYAML {
		source = "pre_delete_snapshot"
	}
	rows := make([][]string, 0, len(records))
	for _, record := range records {
		for _, row := range dnsRecordRows(record) {
			rows = append(rows, append(row, source))
		}
	}
	return printer.PrintTable(dnsRecordSnapshotHeaders, rows)
}

func dnsRecordRows(record dns.Record) [][]string {
	rows := make([][]string, 0, len(recordContents(&record)))
	for _, content := range recordContents(&record) {
		content, priority := dnsRecordContentAndPriority(record, content)
		rows = append(rows, []string{record.Name, record.Type, content, strconv.Itoa(record.TTL), priority})
	}
	return rows
}

func dnsRecordContentAndPriority(record dns.Record, content string) (string, string) {
	if !strings.EqualFold(record.Type, "MX") {
		return content, "-"
	}
	fields := strings.Fields(content)
	if len(fields) > 1 {
		if priority, err := strconv.Atoi(fields[0]); err == nil {
			return strings.Join(fields[1:], " "), strconv.Itoa(priority)
		}
	}
	return content, strconv.Itoa(record.Priority)
}

func findCreatedDNSRecord(domain *dns.Domain, req dns.CreateRecordRequest) *dns.Record {
	fqdn := dns.CanonicalRecordFQDN(req.Name, domain.Name)
	for i := range domain.Records {
		record := &domain.Records[i]
		if !strings.EqualFold(dns.CanonicalRecordFQDN(record.Name, domain.Name), fqdn) || !strings.EqualFold(record.Type, req.Type) {
			continue
		}
		for _, content := range recordContents(record) {
			if dnsRecordMatchesRequest(*record, content, req) {
				copy := *record
				copy.Contents = []string{content}
				copy.Content = content
				return &copy
			}
		}
	}
	return nil
}

func recordContents(record *dns.Record) []string {
	if len(record.Contents) > 0 {
		return record.Contents
	}
	return []string{record.Content}
}

func dnsRecordValues(records []dns.Record) []string {
	values := make([]string, 0)
	for i := range records {
		values = append(values, recordContents(&records[i])...)
	}
	return values
}

func dnsRecordMatchesRequest(record dns.Record, content string, req dns.CreateRecordRequest) bool {
	if strings.EqualFold(record.Type, "CAA") {
		return caaRecordMatches(content, req.Content)
	}

	actual, actualPriority := dnsRecordContentAndPriority(record, content)
	actual = comparableDNSRecordContent(record.Type, actual)
	expected := comparableDNSRecordContent(record.Type, req.Content)
	if actual != expected {
		return false
	}
	if req.Priority == nil {
		return true
	}
	return actualPriority == strconv.Itoa(*req.Priority)
}

func caaRecordMatches(actual, expected string) bool {
	actualFlags, actualTag, actualValue, actualOK := parseCAAContent(actual)
	expectedFlags, expectedTag, expectedValue, expectedOK := parseCAAContent(expected)
	if !actualOK || !expectedOK {
		return actual == expected
	}
	return actualFlags == expectedFlags && strings.EqualFold(actualTag, expectedTag) && actualValue == expectedValue
}

func parseCAAContent(content string) (uint8, string, string, bool) {
	content = strings.TrimSpace(content)
	flagsEnd := strings.IndexAny(content, " \t\n\r")
	if flagsEnd <= 0 {
		return 0, "", "", false
	}
	flags, err := strconv.ParseUint(content[:flagsEnd], 10, 8)
	if err != nil {
		return 0, "", "", false
	}

	content = strings.TrimLeft(content[flagsEnd:], " \t\n\r")
	tagEnd := strings.IndexAny(content, " \t\n\r")
	if tagEnd <= 0 {
		return 0, "", "", false
	}
	tag := content[:tagEnd]
	if !isCAATag(tag) {
		return 0, "", "", false
	}

	value := strings.TrimSpace(content[tagEnd:])
	if value == "" {
		return 0, "", "", false
	}
	if decoded, ok := decodeQuotedTXTContent(value); ok {
		return uint8(flags), tag, decoded, true
	}
	if strings.ContainsAny(value, " \t\n\r") {
		return 0, "", "", false
	}
	return uint8(flags), tag, value, true
}

func validateCAARecordContent(content string) error {
	original := strings.TrimSpace(content)
	flagsEnd := strings.IndexAny(original, " \t\n\r")
	if flagsEnd <= 0 {
		return fmt.Errorf(`CAA --content must use a quoted value (e.g. 0 issue "letsencrypt.org")`)
	}
	remainder := strings.TrimLeft(original[flagsEnd:], " \t\n\r")
	tagEnd := strings.IndexAny(remainder, " \t\n\r")
	if tagEnd <= 0 {
		return fmt.Errorf(`CAA --content must use a quoted value (e.g. 0 issue "letsencrypt.org")`)
	}
	value := strings.TrimSpace(remainder[tagEnd:])
	if _, ok := decodeQuotedTXTContent(value); !ok {
		return fmt.Errorf(`CAA --content value must be quoted (e.g. 0 issue "letsencrypt.org")`)
	}
	if _, _, _, ok := parseCAAContent(original); !ok {
		return fmt.Errorf(`CAA --content must use the form <flags> <tag> "value" (e.g. 0 issue "letsencrypt.org")`)
	}
	return nil
}

func isCAATag(tag string) bool {
	for i := range tag {
		if (tag[i] < 'a' || tag[i] > 'z') && (tag[i] < 'A' || tag[i] > 'Z') && (tag[i] < '0' || tag[i] > '9') {
			return false
		}
	}
	return true
}

func normalizeTXTContent(content string) string {
	decoded, ok := decodeQuotedTXTContent(content)
	if ok {
		return decoded
	}
	return content
}

// decodeQuotedTXTContent decodes PowerDNS TXT presentation syntax only when
// the entire value is a sequence of quoted chunks. Whitespace outside chunks
// is ignored; whitespace inside chunks is part of the value. Invalid or
// unquoted values are left to callers unchanged.
func decodeQuotedTXTContent(content string) (string, bool) {
	var decoded strings.Builder
	sawChunk := false
	for i := 0; i < len(content); {
		for i < len(content) && (content[i] == ' ' || content[i] == '\t' || content[i] == '\n' || content[i] == '\r') {
			i++
		}
		if i == len(content) {
			return decoded.String(), sawChunk
		}
		if content[i] != '"' {
			return content, false
		}
		sawChunk = true
		i++
		for i < len(content) && content[i] != '"' {
			if content[i] != '\\' {
				decoded.WriteByte(content[i])
				i++
				continue
			}
			i++
			if i == len(content) {
				return content, false
			}
			if i+2 < len(content) && isASCIIDigit(content[i]) && isASCIIDigit(content[i+1]) && isASCIIDigit(content[i+2]) {
				octet, err := strconv.Atoi(content[i : i+3])
				if err != nil || octet > 255 {
					return content, false
				}
				decoded.WriteByte(byte(octet))
				i += 3
				continue
			}
			decoded.WriteByte(content[i])
			i++
		}
		if i == len(content) {
			return content, false
		}
		i++
	}
	return decoded.String(), true
}

func isASCIIDigit(value byte) bool {
	return value >= '0' && value <= '9'
}

func comparableDNSRecordContent(recordType, content string) string {
	switch strings.ToUpper(recordType) {
	case "TXT":
		return normalizeTXTContent(content)
	case "MX", "CNAME", "NS":
		return strings.ToLower(strings.TrimSuffix(strings.TrimSpace(content), "."))
	case "A", "AAAA":
		if address, err := netip.ParseAddr(strings.TrimSpace(content)); err == nil {
			return address.String()
		}
	}
	return content
}

func findDNSRecordsByNameAndType(records []dns.Record, zoneName, fqdn, recordType string) []dns.Record {
	matched := make([]dns.Record, 0)
	for _, record := range records {
		if strings.EqualFold(dns.CanonicalRecordFQDN(record.Name, zoneName), fqdn) && strings.EqualFold(record.Type, recordType) {
			matched = append(matched, record)
		}
	}
	return matched
}
