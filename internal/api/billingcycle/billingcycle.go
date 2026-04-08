// Package billingcycle provides ZCP billing cycle API operations (STKCNSL).
package billingcycle

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/zsoftly/zcp-cli/internal/httpclient"
)

// PaymentMode represents a payment mode associated with a billing cycle.
type PaymentMode struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Slug        string `json:"slug"`
	DisplayName string `json:"display_name"`
	Status      bool   `json:"status"`
}

// BillingCycle represents a STKCNSL billing cycle
// (e.g. Hourly, Monthly, Quarterly, Yearly).
type BillingCycle struct {
	ID           string        `json:"id"`
	Name         string        `json:"name"`
	Slug         string        `json:"slug"`
	Description  string        `json:"description"`
	Duration     int           `json:"duration"`
	Unit         string        `json:"unit"`
	IsEnabled    bool          `json:"is_enabled"`
	SortOrder    int           `json:"sort_order"`
	CreatedAt    string        `json:"created_at"`
	UpdatedAt    string        `json:"updated_at"`
	PaymentModes []PaymentMode `json:"payment_modes"`
}

// envelope is the STKCNSL response wrapper.
type envelope struct {
	Status  string          `json:"status"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

// Service provides billing cycle API operations.
type Service struct {
	client *httpclient.Client
}

// NewService creates a new billing cycle Service.
func NewService(client *httpclient.Client) *Service {
	return &Service{client: client}
}

// List returns all billing cycles.
func (s *Service) List(ctx context.Context) ([]BillingCycle, error) {
	var env envelope
	if err := s.client.Get(ctx, "/billing-cycles", nil, &env); err != nil {
		return nil, fmt.Errorf("listing billing cycles: %w", err)
	}

	var cycles []BillingCycle
	if err := json.Unmarshal(env.Data, &cycles); err != nil {
		return nil, fmt.Errorf("decoding billing cycles: %w", err)
	}

	return cycles, nil
}
