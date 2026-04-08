// Package currency provides ZCP currency API operations (STKCNSL).
package currency

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/zsoftly/zcp-cli/internal/httpclient"
)

// Currency represents a STKCNSL currency.
type Currency struct {
	ID                string `json:"id"`
	Name              string `json:"name"`
	Slug              string `json:"slug"`
	Locale            string `json:"locale"`
	CurrencyName      string `json:"currency_name"`
	Fraction          string `json:"fraction"`
	Status            bool   `json:"status"`
	Default           bool   `json:"default"`
	DecimalPlace      int    `json:"decimal_place"`
	ResellerThreshold string `json:"reseller_threshold"`
	CustomerThreshold string `json:"customer_threshold"`
	CreatedAt         string `json:"created_at"`
	UpdatedAt         string `json:"updated_at"`
}

// envelope is the STKCNSL response wrapper.
type envelope struct {
	Status  string          `json:"status"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

// Service provides currency API operations.
type Service struct {
	client *httpclient.Client
}

// NewService creates a new currency Service.
func NewService(client *httpclient.Client) *Service {
	return &Service{client: client}
}

// List returns all currencies.
func (s *Service) List(ctx context.Context) ([]Currency, error) {
	var env envelope
	if err := s.client.Get(ctx, "/currencies", nil, &env); err != nil {
		return nil, fmt.Errorf("listing currencies: %w", err)
	}

	var currencies []Currency
	if err := json.Unmarshal(env.Data, &currencies); err != nil {
		return nil, fmt.Errorf("decoding currencies: %w", err)
	}

	return currencies, nil
}
