// Package cost provides ZCP cost estimate API operations.
package cost

import (
	"context"
	"fmt"
	"net/url"

	"github.com/zsoftly/zcp-cli/internal/httpclient"
)

// Currency represents a supported billing currency.
type Currency struct {
	UUID              string  `json:"uuid"`
	Currency          string  `json:"currency"`
	CurrencySymbol    string  `json:"currencySymbol"`
	Cost              float64 `json:"cost"`
	IsDefaultCurrency bool    `json:"isDefaultCurrency"`
}

// TaxInfo holds tax configuration for the organization.
type TaxInfo struct {
	Name            string  `json:"name"`
	TaxPercentage   float64 `json:"taxPercentage"`
	OrganizationTax float64 `json:"organizationTax"`
	IndividualTax   float64 `json:"individualTax"`
}

// MultiCurrencyResponse wraps the currency list response.
type MultiCurrencyResponse struct {
	OrganizationName  string     `json:"organizationName"`
	Count             int        `json:"count"`
	ListMultiCurrency []Currency `json:"listMultiCurrency"`
}

type taxResponse struct {
	TaxResponse []TaxInfo `json:"taxResponse"`
}

// Service provides cost API operations.
type Service struct {
	client *httpclient.Client
}

// NewService creates a new cost Service.
func NewService(client *httpclient.Client) *Service { return &Service{client: client} }

// ListCurrencies returns all supported billing currencies and their rates.
func (s *Service) ListCurrencies(ctx context.Context) (*MultiCurrencyResponse, error) {
	var resp MultiCurrencyResponse
	if err := s.client.Get(ctx, "/restapi/costestimate/multicurrency", url.Values{}, &resp); err != nil {
		return nil, fmt.Errorf("listing currencies: %w", err)
	}
	return &resp, nil
}

// GetTax returns tax configuration for the organization.
func (s *Service) GetTax(ctx context.Context) ([]TaxInfo, error) {
	var resp taxResponse
	if err := s.client.Get(ctx, "/restapi/costestimate/tax", url.Values{}, &resp); err != nil {
		return nil, fmt.Errorf("getting tax info: %w", err)
	}
	return resp.TaxResponse, nil
}
