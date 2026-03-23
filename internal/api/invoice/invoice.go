// Package invoice provides ZCP invoice API operations.
package invoice

import (
	"context"
	"fmt"
	"net/url"

	"github.com/zsoftly/zcp-cli/internal/httpclient"
)

// Invoice represents a ZCP billing invoice.
type Invoice struct {
	DomainUUID          string  `json:"domainUuid"`
	AdjustmentCost      float64 `json:"adjustmentCost"`
	ClientEmail         string  `json:"clientEmail"`
	UsageCost           float64 `json:"usageCost"`
	ReferenceNumber     string  `json:"referenceNumber"`
	InvoiceNumber       string  `json:"invoiceNumber"`
	GeneratedDate       string  `json:"generatedDate"`
	TotalAdjustmentCost float64 `json:"totalAdjustmentCost"`
	Currency            string  `json:"currency"`
	BandwidthFreeUsage  string  `json:"bandwidthFreeUsage"`
	TotalCost           float64 `json:"totalCost"`
	BillPeriod          string  `json:"billPeriod"`
}

// GenerateResponse is the response from generating an invoice.
type GenerateResponse struct {
	InvoiceNumber int    `json:"invoiceNumber"`
	Message       string `json:"message"`
	Status        bool   `json:"status"`
}

type listInvoiceResponse struct {
	Count               int       `json:"count"`
	ListInvoiceResponse []Invoice `json:"listInvoiceResponse"`
}

// Service provides invoice API operations.
type Service struct {
	client *httpclient.Client
}

// NewService creates a new invoice Service.
func NewService(client *httpclient.Client) *Service { return &Service{client: client} }

// List returns invoices filtered by optional clientEmail, status, and billPeriod.
func (s *Service) List(ctx context.Context, clientEmail, status, billPeriod string) ([]Invoice, error) {
	q := url.Values{}
	if clientEmail != "" {
		q.Set("clientEmail", clientEmail)
	}
	if status != "" {
		q.Set("status", status)
	}
	if billPeriod != "" {
		q.Set("billPeriod", billPeriod)
	}
	var resp listInvoiceResponse
	if err := s.client.Get(ctx, "/restapi/invoice/listByClient", q, &resp); err != nil {
		return nil, fmt.Errorf("listing invoices: %w", err)
	}
	return resp.ListInvoiceResponse, nil
}

// Generate triggers invoice generation for the given invoice number.
func (s *Service) Generate(ctx context.Context, invoiceNumber string) (*GenerateResponse, error) {
	q := url.Values{"invoiceNumber": {invoiceNumber}}
	var resp GenerateResponse
	if err := s.client.Get(ctx, "/restapi/invoice/generateInvoice", q, &resp); err != nil {
		return nil, fmt.Errorf("generating invoice %s: %w", invoiceNumber, err)
	}
	return &resp, nil
}
