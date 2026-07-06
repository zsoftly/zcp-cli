// Package dns provides ZCP DNS domain and record API operations (STKCNSL).
package dns

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/zsoftly/zcp-cli/pkg/httpclient"
)

// Domain represents a DNS domain.
type Domain struct {
	ID          string          `json:"id"`
	Name        string          `json:"name"`
	Slug        string          `json:"slug"`
	AccountID   string          `json:"account_id"`
	ProjectID   string          `json:"project_id"`
	DNSProvider string          `json:"dns_provider"`
	Status      bool            `json:"status"`
	CreatedAt   string          `json:"created_at"`
	UpdatedAt   string          `json:"updated_at"`
	Records     []Record        `json:"records,omitempty"`
	Project     json.RawMessage `json:"project,omitempty"`
}

// Record represents a single DNS record within a domain.
//
// The live PowerDNS-backed API returns record SETS (RRsets): no id, and the
// values under a "contents" array rather than a "content" string (verified
// live 2026-07-05). UnmarshalJSON accepts both shapes; Content carries the
// joined values for display and Contents the individual values.
type Record struct {
	ID        string   `json:"id"`
	DomainID  int      `json:"domain_id"`
	Name      string   `json:"name"`
	Type      string   `json:"type"`
	Content   string   `json:"content"`
	Contents  []string `json:"contents"`
	TTL       int      `json:"ttl"`
	Priority  int      `json:"priority,omitempty"`
	CreatedAt string   `json:"created_at"`
	UpdatedAt string   `json:"updated_at"`
}

// UnmarshalJSON tolerates both record shapes: legacy rows with id/content and
// PowerDNS RRsets with a contents array and no id.
func (r *Record) UnmarshalJSON(b []byte) error {
	type recordAlias Record
	var v recordAlias
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}
	*r = Record(v)
	if r.Content == "" && len(r.Contents) > 0 {
		r.Content = strings.Join(r.Contents, ", ")
	}
	return nil
}

// CreateDomainRequest holds parameters for creating a DNS domain.
type CreateDomainRequest struct {
	Name          string `json:"name"`
	Project       string `json:"project"`
	DNSProvider   string `json:"dns_provider"`
	CloudProvider string `json:"cloud_provider"`
	Region        string `json:"region"`
}

// CreateRecordRequest holds parameters for creating a DNS record.
type CreateRecordRequest struct {
	Name    string `json:"name"`
	Type    string `json:"type"`
	Content string `json:"content"`
	TTL     int    `json:"ttl"`
}

// DeleteRecordRequest holds parameters for deleting a DNS record.
type DeleteRecordRequest struct {
	RecordID int `json:"record_id"`
}

// envelopeList wraps the standard paginated list response.
type envelopeList struct {
	Status      string   `json:"status"`
	Message     string   `json:"message"`
	CurrentPage int      `json:"current_page"`
	Data        []Domain `json:"data"`
	Total       int      `json:"total"`
}

// envelopeSingle wraps a single-object response.
type envelopeSingle struct {
	Status  string `json:"status"`
	Message string `json:"message"`
	Data    Domain `json:"data"`
}

// Service provides DNS API operations.
type Service struct {
	client *httpclient.Client
}

// NewService creates a new DNS Service.
func NewService(client *httpclient.Client) *Service {
	return &Service{client: client}
}

// List returns all DNS domains.
func (s *Service) List(ctx context.Context) ([]Domain, error) {
	q := url.Values{}
	q.Set("include", "dns_provider")
	var resp envelopeList
	if err := s.client.Get(ctx, "/dns/domains", q, &resp); err != nil {
		return nil, fmt.Errorf("listing DNS domains: %w", err)
	}
	return resp.Data, nil
}

// Show returns details for a single DNS domain by slug, including records.
func (s *Service) Show(ctx context.Context, slug string) (*Domain, error) {
	q := url.Values{}
	q.Set("dns_provider", "PowerDNS")
	var resp envelopeSingle
	if err := s.client.Get(ctx, "/dns/domains/"+slug, q, &resp); err != nil {
		return nil, fmt.Errorf("showing DNS domain %s: %w", slug, err)
	}
	return &resp.Data, nil
}

// Create creates a new DNS domain.
func (s *Service) Create(ctx context.Context, req CreateDomainRequest) (*Domain, error) {
	var resp envelopeSingle
	if err := s.client.Post(ctx, "/dns/domains", req, &resp); err != nil {
		return nil, fmt.Errorf("creating DNS domain: %w", err)
	}
	return &resp.Data, nil
}

// Delete removes a DNS domain by slug.
func (s *Service) Delete(ctx context.Context, slug string) error {
	if err := s.client.Delete(ctx, "/dns/domains/"+slug, nil); err != nil {
		return fmt.Errorf("deleting DNS domain %s: %w", slug, err)
	}
	return nil
}

// CreateRecord creates a DNS record under the given domain slug.
func (s *Service) CreateRecord(ctx context.Context, domainSlug string, req CreateRecordRequest) (*Domain, error) {
	var resp envelopeSingle
	if err := s.client.Post(ctx, "/dns/domains/"+domainSlug+"/records", req, &resp); err != nil {
		return nil, fmt.Errorf("creating DNS record on %s: %w", domainSlug, err)
	}
	return &resp.Data, nil
}

// DeleteRecord removes a DNS record under the given domain slug by numeric ID.
//
// Deprecated: the live PowerDNS-backed API does not expose record IDs, so this
// cannot work there; use DeleteRecordByName instead.
func (s *Service) DeleteRecord(ctx context.Context, domainSlug string, recordID int) error {
	path := "/dns/domains/" + domainSlug + "/records"
	q := url.Values{}
	q.Set("record_id", strconv.Itoa(recordID))
	if err := s.client.Delete(ctx, path, q); err != nil {
		return fmt.Errorf("deleting DNS record %d on domain %s: %w", recordID, domainSlug, err)
	}
	return nil
}

// DeleteRecordByName removes a DNS record set identified by its fully
// qualified name and type — the addressing scheme the live PowerDNS-backed
// API uses (record sets carry no IDs). name should be the stored FQDN with a
// trailing dot (e.g. "www.example.com.").
func (s *Service) DeleteRecordByName(ctx context.Context, domainSlug, name, recordType string) error {
	q := url.Values{}
	q.Set("name", name)
	q.Set("type", recordType)
	if err := s.client.Delete(ctx, "/dns/domains/"+domainSlug+"/records", q); err != nil {
		return fmt.Errorf("deleting DNS record %s %s on domain %s: %w", recordType, name, domainSlug, err)
	}
	return nil
}

// CanonicalRecordFQDN builds the backend's stored record name for a record in
// the given zone: relative labels get the zone appended, absolute names are
// normalized, and the result always carries a trailing dot.
func CanonicalRecordFQDN(name, zoneName string) string {
	n := strings.ToLower(strings.TrimSuffix(strings.TrimSpace(name), "."))
	zone := strings.ToLower(strings.TrimSuffix(strings.TrimSpace(zoneName), "."))
	if n == "" || n == "@" || n == zone {
		return zone + "."
	}
	if strings.HasSuffix(n, "."+zone) {
		return n + "."
	}
	return n + "." + zone + "."
}
