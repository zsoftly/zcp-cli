// Package billing provides ZCP billing, analytics, and account API operations.
package billing

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/zsoftly/zcp-cli/internal/httpclient"
)

// Balance represents account balance information.
type Balance struct {
	AvailableBalance     float64           `json:"available_balance"`
	AvailableNetBalance  float64           `json:"available_net_balance"`
	Deposited            float64           `json:"deposited"`
	Charged              float64           `json:"charged"`
	Due                  float64           `json:"due"`
	Usage                float64           `json:"usage"`
	CurrentUsage         float64           `json:"current_usage"`
	HourlyUsage          float64           `json:"hourly_usage"`
	CurrentHourlyRate    float64           `json:"current_hourly_rate"`
	AllTimeUsage         float64           `json:"all_time_usage"`
	EstimatedHourlyUsage float64           `json:"estimated_hourly_usage"`
	CurrentMonthUsage    float64           `json:"current_month_usage"`
	AvailableFreeCredits float64           `json:"available_free_credits"`
	FreeCreditBalance    float64           `json:"free_credit_balance"`
	TotalPayouts         float64           `json:"total_payouts"`
	UnpaidInvoices       float64           `json:"unpaid_invoices"`
	BillingCycleUsage    map[string]string `json:"billing_cycle_usage"`
	DepositedPayments    float64           `json:"deposited_payments"`
	SubscriptionAmount   float64           `json:"subscription_amount"`
}

// ServiceCost represents cost for a single service category.
type ServiceCost struct {
	Name        string  `json:"name"`
	DisplayName string  `json:"display_name"`
	TotalCost   float64 `json:"total_cost"`
}

// MonthlyUsage represents usage for a single month.
type MonthlyUsage struct {
	Month string      `json:"month"`
	Year  string      `json:"year"`
	Cost  json.Number `json:"cost"`
}

// CreditLimit represents the account credit limit.
type CreditLimit struct {
	Limit            string  `json:"limit"`
	UsageAmount      float64 `json:"usage_amount"`
	AvailableToSpend float64 `json:"available_to_spend"`
}

// BillingCycle holds billing cycle info within a subscription.
type BillingCycle struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Slug        string `json:"slug"`
	Description string `json:"description"`
	Duration    int    `json:"duration"`
	Unit        string `json:"unit"`
	IsEnabled   bool   `json:"is_enabled"`
}

// SubscriptionProject holds project info within a subscription.
type SubscriptionProject struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Slug        string `json:"slug"`
	Purpose     string `json:"purpose"`
	Description string `json:"description"`
}

// Subscription represents an active or inactive service subscription.
type Subscription struct {
	ID                 string              `json:"id"`
	Name               string              `json:"name"`
	Description        *string             `json:"description"`
	Product            string              `json:"product"`
	ProductDisplayName string              `json:"product_display_name"`
	CustomerName       string              `json:"customer_name"`
	CustomerID         string              `json:"customer_id"`
	BillingCycle       BillingCycle        `json:"billing_cycle"`
	InvoiceItemsCount  int                 `json:"invoice_items_count"`
	RenewAt            string              `json:"renew_at"`
	Quantity           string              `json:"quantity"`
	Price              string              `json:"price"`
	TotalUsage         string              `json:"total_usage"`
	TotalUsageWithTax  string              `json:"total_usage_with_tax"`
	Project            SubscriptionProject `json:"project"`
	AccountID          string              `json:"account_id"`
	ProjectID          string              `json:"project_id"`
	RegionID           string              `json:"region_id"`
	Rule               string              `json:"rule"`
	HasContract        bool                `json:"has_contract"`
	CreatedAt          string              `json:"created_at"`
	AccountCRN         string              `json:"account_crn"`
}

// InvoiceItem represents a line item on an invoice.
type InvoiceItem struct {
	ID              string `json:"id"`
	InvoiceID       string `json:"invoice_id"`
	Item            string `json:"item"`
	Quantity        string `json:"quantity"`
	Description     string `json:"description"`
	Rate            string `json:"rate"`
	SubAmount       string `json:"sub_amount"`
	Amount          string `json:"amount"`
	ServiceName     string `json:"service_name"`
	ItemDisplayName string `json:"item_display_name"`
}

// Invoice represents a ZCP billing invoice.
type Invoice struct {
	ID                 string        `json:"id"`
	Number             int           `json:"number"`
	CustomNumber       string        `json:"custom_number"`
	Amount             string        `json:"amount"`
	SubAmount          string        `json:"sub_amount"`
	TaxPercent         string        `json:"tax_percent"`
	TaxAmount          float64       `json:"tax_amount"`
	Type               string        `json:"type"`
	Status             string        `json:"status"`
	InvoiceAt          string        `json:"invoice_at"`
	DueAt              string        `json:"due_at"`
	PaidAt             string        `json:"paid_at"`
	CreatedAt          string        `json:"created_at"`
	Items              []InvoiceItem `json:"items"`
	InvoiceViewURL     string        `json:"invoice_view_url"`
	InvoiceDownloadURL string        `json:"invoice_download_url"`
	InvoiceValue       string        `json:"invoice_value"`
	Total              string        `json:"total"`
	PaymentMethods     string        `json:"payment_methods"`
	GeneratedBy        string        `json:"generated_by"`
}

// envelope is the standard STKCNSL API response wrapper.
type envelope struct {
	Status  string          `json:"status"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

// paginatedEnvelope extends envelope with pagination fields.
type paginatedEnvelope struct {
	Status      string          `json:"status"`
	Message     string          `json:"message"`
	CurrentPage int             `json:"current_page"`
	Data        json.RawMessage `json:"data"`
	Total       int             `json:"total"`
	LastPage    int             `json:"last_page"`
}

// Service provides billing API operations.
type Service struct {
	client *httpclient.Client
}

// NewService creates a new billing Service.
func NewService(client *httpclient.Client) *Service { return &Service{client: client} }

// GetBalance returns the account balance summary.
func (s *Service) GetBalance(ctx context.Context) (*Balance, error) {
	var env envelope
	if err := s.client.Get(ctx, "/account/balance", nil, &env); err != nil {
		return nil, fmt.Errorf("getting account balance: %w", err)
	}
	var bal Balance
	if err := json.Unmarshal(env.Data, &bal); err != nil {
		return nil, fmt.Errorf("decoding account balance: %w", err)
	}
	return &bal, nil
}

// ListServiceCosts returns per-service cost breakdown.
func (s *Service) ListServiceCosts(ctx context.Context) ([]ServiceCost, error) {
	var env envelope
	if err := s.client.Get(ctx, "/analytics/services/costs", nil, &env); err != nil {
		return nil, fmt.Errorf("listing service costs: %w", err)
	}
	var costs []ServiceCost
	if err := json.Unmarshal(env.Data, &costs); err != nil {
		return nil, fmt.Errorf("decoding service costs: %w", err)
	}
	return costs, nil
}

// ListMonthlyUsage returns month-by-month usage data.
func (s *Service) ListMonthlyUsage(ctx context.Context) ([]MonthlyUsage, error) {
	var env envelope
	if err := s.client.Get(ctx, "/analytics/month-wise-usage", nil, &env); err != nil {
		return nil, fmt.Errorf("listing monthly usage: %w", err)
	}
	var usage []MonthlyUsage
	if err := json.Unmarshal(env.Data, &usage); err != nil {
		return nil, fmt.Errorf("decoding monthly usage: %w", err)
	}
	return usage, nil
}

// GetServiceCounts returns a map of service name to count.
func (s *Service) GetServiceCounts(ctx context.Context) (map[string]int, error) {
	var env envelope
	if err := s.client.Get(ctx, "/analytics/account/services/counts", nil, &env); err != nil {
		return nil, fmt.Errorf("getting service counts: %w", err)
	}
	var counts map[string]int
	if err := json.Unmarshal(env.Data, &counts); err != nil {
		return nil, fmt.Errorf("decoding service counts: %w", err)
	}
	return counts, nil
}

// GetCreditLimit returns the account credit limit information.
func (s *Service) GetCreditLimit(ctx context.Context) (*CreditLimit, error) {
	var env envelope
	if err := s.client.Get(ctx, "/billing/credit-limit", nil, &env); err != nil {
		return nil, fmt.Errorf("getting credit limit: %w", err)
	}
	var limit CreditLimit
	if err := json.Unmarshal(env.Data, &limit); err != nil {
		return nil, fmt.Errorf("decoding credit limit: %w", err)
	}
	return &limit, nil
}

// ListInvoices returns a paginated list of invoices.
func (s *Service) ListInvoices(ctx context.Context, page int) ([]Invoice, int, error) {
	q := url.Values{}
	if page > 0 {
		q.Set("page", fmt.Sprintf("%d", page))
	}
	var env paginatedEnvelope
	if err := s.client.Get(ctx, "/billing/invoices", q, &env); err != nil {
		return nil, 0, fmt.Errorf("listing invoices: %w", err)
	}
	var invoices []Invoice
	if err := json.Unmarshal(env.Data, &invoices); err != nil {
		return nil, 0, fmt.Errorf("decoding invoices: %w", err)
	}
	return invoices, env.Total, nil
}

// GetInvoiceCount returns the total number of invoices.
func (s *Service) GetInvoiceCount(ctx context.Context) (int, error) {
	var env envelope
	if err := s.client.Get(ctx, "/billing/invoices-count", nil, &env); err != nil {
		return 0, fmt.Errorf("getting invoice count: %w", err)
	}
	var count int
	if err := json.Unmarshal(env.Data, &count); err != nil {
		return 0, fmt.Errorf("decoding invoice count: %w", err)
	}
	return count, nil
}

// ListActiveSubscriptions returns active service subscriptions.
func (s *Service) ListActiveSubscriptions(ctx context.Context, page int) ([]Subscription, int, error) {
	q := url.Values{}
	if page > 0 {
		q.Set("page", fmt.Sprintf("%d", page))
	}
	var env paginatedEnvelope
	if err := s.client.Get(ctx, "/billing/subscriptions/active", q, &env); err != nil {
		return nil, 0, fmt.Errorf("listing active subscriptions: %w", err)
	}
	var subs []Subscription
	if err := json.Unmarshal(env.Data, &subs); err != nil {
		return nil, 0, fmt.Errorf("decoding active subscriptions: %w", err)
	}
	return subs, env.Total, nil
}

// ListInactiveSubscriptions returns inactive service subscriptions.
func (s *Service) ListInactiveSubscriptions(ctx context.Context, page int) ([]Subscription, int, error) {
	q := url.Values{}
	if page > 0 {
		q.Set("page", fmt.Sprintf("%d", page))
	}
	var env paginatedEnvelope
	if err := s.client.Get(ctx, "/billing/subscriptions/inactive", q, &env); err != nil {
		return nil, 0, fmt.Errorf("listing inactive subscriptions: %w", err)
	}
	var subs []Subscription
	if err := json.Unmarshal(env.Data, &subs); err != nil {
		return nil, 0, fmt.Errorf("decoding inactive subscriptions: %w", err)
	}
	return subs, env.Total, nil
}

// GetAccountUsage returns raw billing account usage data.
func (s *Service) GetAccountUsage(ctx context.Context) (json.RawMessage, error) {
	var env envelope
	if err := s.client.Get(ctx, "/billing/account/usage", nil, &env); err != nil {
		return nil, fmt.Errorf("getting account usage: %w", err)
	}
	return env.Data, nil
}

// GetFreeCredits returns free credits information.
func (s *Service) GetFreeCredits(ctx context.Context) (json.RawMessage, error) {
	var env envelope
	if err := s.client.Get(ctx, "/account/free-credits", nil, &env); err != nil {
		return nil, fmt.Errorf("getting free credits: %w", err)
	}
	return env.Data, nil
}

// ListServiceContracts returns service contracts.
func (s *Service) ListServiceContracts(ctx context.Context) (json.RawMessage, error) {
	var env envelope
	if err := s.client.Get(ctx, "/billing/subscriptions/service-contracts", nil, &env); err != nil {
		return nil, fmt.Errorf("listing service contracts: %w", err)
	}
	return env.Data, nil
}

// ListServiceTrials returns active free trials.
func (s *Service) ListServiceTrials(ctx context.Context) (json.RawMessage, error) {
	var env envelope
	if err := s.client.Get(ctx, "/billing/subscriptions/service-trials", nil, &env); err != nil {
		return nil, fmt.Errorf("listing service trials: %w", err)
	}
	return env.Data, nil
}

// ListCancelRequests returns scheduled service cancellation requests.
func (s *Service) ListCancelRequests(ctx context.Context) (json.RawMessage, error) {
	var env envelope
	if err := s.client.Get(ctx, "/billing/service-cancel-requests", nil, &env); err != nil {
		return nil, fmt.Errorf("listing cancel requests: %w", err)
	}
	return env.Data, nil
}

// CancelServiceRequest holds parameters for service cancellation.
type CancelServiceRequest struct {
	ServiceName string `json:"service_name"`
	Reason      string `json:"reason"`
	Type        string `json:"type"`
	Description string `json:"description,omitempty"`
}

// CancelService submits a cancellation request for a service by subscription slug.
func (s *Service) CancelService(ctx context.Context, slug string, req CancelServiceRequest) error {
	var env envelope
	if err := s.client.Post(ctx, "/billing/service-cancel-requests/"+slug, req, &env); err != nil {
		return fmt.Errorf("cancelling service %s: %w", slug, err)
	}
	return nil
}

// ListPayments returns payment transactions for the account.
func (s *Service) ListPayments(ctx context.Context, page int) (json.RawMessage, error) {
	q := url.Values{}
	if page > 0 {
		q.Set("page", fmt.Sprintf("%d", page))
	}
	var env paginatedEnvelope
	if err := s.client.Get(ctx, "/account/payments", q, &env); err != nil {
		return nil, fmt.Errorf("listing payments: %w", err)
	}
	return env.Data, nil
}

// ListCoupons returns coupons associated with the account.
func (s *Service) ListCoupons(ctx context.Context) (json.RawMessage, error) {
	var env envelope
	if err := s.client.Get(ctx, "/account/coupons", nil, &env); err != nil {
		return nil, fmt.Errorf("listing coupons: %w", err)
	}
	return env.Data, nil
}

// RedeemCouponRequest holds coupon redemption parameters.
type RedeemCouponRequest struct {
	Code string `json:"code"`
}

// RedeemCoupon applies a coupon code to the account.
func (s *Service) RedeemCoupon(ctx context.Context, code string) (json.RawMessage, error) {
	req := RedeemCouponRequest{Code: code}
	var env envelope
	if err := s.client.Post(ctx, "/account/coupons", req, &env); err != nil {
		return nil, fmt.Errorf("redeeming coupon %s: %w", code, err)
	}
	return env.Data, nil
}

// BudgetAlert represents budget alert settings.
type BudgetAlert struct {
	Amount    float64 `json:"amount"`
	Threshold float64 `json:"threshold"`
	IsEnabled bool    `json:"is_enabled"`
}

// GetBudgetAlert returns the current budget alert settings.
func (s *Service) GetBudgetAlert(ctx context.Context) (json.RawMessage, error) {
	var env envelope
	if err := s.client.Get(ctx, "/billing/budget-alert-settings", nil, &env); err != nil {
		return nil, fmt.Errorf("getting budget alert settings: %w", err)
	}
	return env.Data, nil
}

// SetBudgetAlertRequest holds budget alert configuration.
type SetBudgetAlertRequest struct {
	Amount    float64 `json:"amount"`
	Threshold float64 `json:"threshold"`
	IsEnabled bool    `json:"is_enabled"`
}

// SetBudgetAlert updates budget alert settings.
func (s *Service) SetBudgetAlert(ctx context.Context, req SetBudgetAlertRequest) (json.RawMessage, error) {
	var env envelope
	if err := s.client.Post(ctx, "/billing/budget-alert-settings", req, &env); err != nil {
		return nil, fmt.Errorf("setting budget alert: %w", err)
	}
	return env.Data, nil
}
