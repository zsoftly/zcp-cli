// Package store provides ZCP store API operations.
package store

import (
	"context"
	"fmt"
	"net/url"

	"github.com/zsoftly/zcp-cli/internal/httpclient"
)

// Item represents a store item.
type Item struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Slug        string  `json:"slug"`
	Description string  `json:"description"`
	Status      string  `json:"status"`
	Price       float64 `json:"price"`
	Quantity    int     `json:"quantity"`
	CreatedAt   string  `json:"created_at"`
	UpdatedAt   string  `json:"updated_at"`
}

// CheckoutProduct describes a product within a checkout request.
type CheckoutProduct struct {
	Description string `json:"description"`
	Product     string `json:"product"`
	Quantity    int    `json:"quantity"`
	Status      string `json:"status"`
	UserID      string `json:"user_id,omitempty"`
}

// CheckoutRequest holds the body for POST /store/checkout.
type CheckoutRequest struct {
	Service      string            `json:"service"`
	Products     []CheckoutProduct `json:"products"`
	BillingCycle string            `json:"billing_cycle"`
	Coupon       *string           `json:"coupon"`
}

// listItemsResponse wraps the paginated store items response.
type listItemsResponse struct {
	Status      string `json:"status"`
	Message     string `json:"message"`
	CurrentPage int    `json:"current_page"`
	Data        []Item `json:"data"`
	LastPage    int    `json:"last_page"`
	PerPage     int    `json:"per_page"`
	Total       int    `json:"total"`
}

// checkoutResponse wraps the checkout response.
type checkoutResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

// Service provides store API operations.
type Service struct {
	client *httpclient.Client
}

// NewService creates a new store Service.
func NewService(client *httpclient.Client) *Service {
	return &Service{client: client}
}

// ListItems returns store items with optional sorting and pagination.
func (s *Service) ListItems(ctx context.Context, sort string, page, limit int) ([]Item, int, error) {
	q := url.Values{}
	if sort != "" {
		q.Set("sort", sort)
	}
	if page > 0 {
		q.Set("page", fmt.Sprintf("%d", page))
	}
	if limit > 0 {
		q.Set("limit", fmt.Sprintf("%d", limit))
	}

	var resp listItemsResponse
	if err := s.client.Get(ctx, "/store/items", q, &resp); err != nil {
		return nil, 0, fmt.Errorf("listing store items: %w", err)
	}

	return resp.Data, resp.Total, nil
}

// Checkout submits a checkout/purchase request.
func (s *Service) Checkout(ctx context.Context, req CheckoutRequest) error {
	var resp checkoutResponse
	if err := s.client.Post(ctx, "/store/checkout", req, &resp); err != nil {
		return fmt.Errorf("store checkout: %w", err)
	}
	if resp.Status != "" && resp.Status != "Success" {
		return fmt.Errorf("store checkout failed: %s", resp.Message)
	}
	return nil
}
