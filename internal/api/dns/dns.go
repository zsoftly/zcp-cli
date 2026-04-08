// Package dns provides ZCP DNS domain and record API operations (STKCNSL).
package dns

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"

	"github.com/zsoftly/zcp-cli/internal/httpclient"
)

// Domain represents a DNS domain.
type Domain struct {
	ID          string          `json:"id"`
	Name        string          `json:"name"`
	Slug        string          `json:"slug"`
	AccountID   string          `json:"account_id"`
	ProjectID   int             `json:"project_id"`
	DNSProvider string          `json:"dns_provider"`
	Status      string          `json:"status"`
	CreatedAt   string          `json:"created_at"`
	UpdatedAt   string          `json:"updated_at"`
	Records     []Record        `json:"records,omitempty"`
	Project     json.RawMessage `json:"project,omitempty"`
}

// Record represents a single DNS record within a domain.
type Record struct {
	ID        string `json:"id"`
	DomainID  int    `json:"domain_id"`
	Name      string `json:"name"`
	Type      string `json:"type"`
	Content   string `json:"content"`
	TTL       int    `json:"ttl"`
	Priority  int    `json:"priority,omitempty"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

// CreateDomainRequest holds parameters for creating a DNS domain.
type CreateDomainRequest struct {
	Name        string `json:"name"`
	Project     string `json:"project"`
	DNSProvider string `json:"dns_provider"`
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

// DeleteRecord removes a DNS record under the given domain slug.
// The record is identified by its ID in the request body.
func (s *Service) DeleteRecord(ctx context.Context, domainSlug string, recordID int) error {
	path := "/dns/domains/" + domainSlug + "/records"
	q := url.Values{}
	q.Set("record_id", strconv.Itoa(recordID))
	if err := s.client.Delete(ctx, path, q); err != nil {
		return fmt.Errorf("deleting DNS record %d on domain %s: %w", recordID, domainSlug, err)
	}
	return nil
}
