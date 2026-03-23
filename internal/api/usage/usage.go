// Package usage provides ZCP usage and consumption API operations.
package usage

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/zsoftly/zcp-cli/internal/httpclient"
)

// RawResult holds an untyped API response for usage endpoints with undefined schemas.
type RawResult = json.RawMessage

// CreditBalance holds the user's credit and billing info.
type CreditBalance struct {
	UserEmail     string  `json:"userEmail"`
	UserType      string  `json:"userType"`
	BalanceAmount float64 `json:"balanceAmount"`
	Type          string  `json:"type"`
}

// Service provides usage API operations.
type Service struct {
	client *httpclient.Client
}

// NewService creates a new usage Service.
func NewService(client *httpclient.Client) *Service { return &Service{client: client} }

// ConsumptionList returns raw usage consumption data for a billing period.
// period format: "YYYY-MM". customer is optional (email).
func (s *Service) ConsumptionList(ctx context.Context, period, customer string) (json.RawMessage, error) {
	q := url.Values{"period": {period}}
	if customer != "" {
		q.Set("customer", customer)
	}
	var result json.RawMessage
	if err := s.client.Get(ctx, "/restapi/usage/usageConsumptionList", q, &result); err != nil {
		return nil, fmt.Errorf("listing usage consumption: %w", err)
	}
	return result, nil
}

// ReportList returns raw usage report data for a date range.
func (s *Service) ReportList(ctx context.Context, periodFrom, periodTo, customer string) (json.RawMessage, error) {
	q := url.Values{"periodFrom": {periodFrom}, "periodTo": {periodTo}}
	if customer != "" {
		q.Set("customer", customer)
	}
	var result json.RawMessage
	if err := s.client.Get(ctx, "/restapi/usage/usageReportList", q, &result); err != nil {
		return nil, fmt.Errorf("listing usage report: %w", err)
	}
	return result, nil
}

// ProgressStatus returns current billing progress status.
func (s *Service) ProgressStatus(ctx context.Context) (json.RawMessage, error) {
	var result json.RawMessage
	if err := s.client.Get(ctx, "/restapi/usage/usageProgressStatus", url.Values{}, &result); err != nil {
		return nil, fmt.Errorf("getting usage progress status: %w", err)
	}
	return result, nil
}

// CreditBalance returns the authenticated user's credit balance.
func (s *Service) CreditBalance(ctx context.Context) (*CreditBalance, error) {
	var balance CreditBalance
	if err := s.client.Get(ctx, "/restapi/user/creditBalance", url.Values{}, &balance); err != nil {
		return nil, fmt.Errorf("getting credit balance: %w", err)
	}
	return &balance, nil
}
