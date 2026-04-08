// Package product provides ZCP product and product category API operations.
package product

import (
	"context"
	"fmt"
	"net/url"

	"github.com/zsoftly/zcp-cli/internal/httpclient"
)

// Category represents a product category.
type Category struct {
	ID          string `json:"id"`
	UserID      string `json:"user_id"`
	Name        string `json:"name"`
	Slug        string `json:"slug"`
	Description string `json:"description"`
	Status      bool   `json:"status"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

// Product represents a product in the store.
type Product struct {
	ID                string    `json:"id"`
	UserID            string    `json:"user_id"`
	Name              string    `json:"name"`
	Slug              string    `json:"slug"`
	Description       string    `json:"description"`
	Status            bool      `json:"status"`
	Price             float64   `json:"price"`
	ProductCategoryID string    `json:"product_category_id"`
	ProductCategory   *Category `json:"product_category,omitempty"`
	CreatedAt         string    `json:"created_at"`
	UpdatedAt         string    `json:"updated_at"`
}

// listCategoriesResponse wraps the paginated product categories response.
type listCategoriesResponse struct {
	Status      string     `json:"status"`
	Message     string     `json:"message"`
	CurrentPage int        `json:"current_page"`
	Data        []Category `json:"data"`
	LastPage    int        `json:"last_page"`
	PerPage     int        `json:"per_page"`
	Total       int        `json:"total"`
}

// listProductsResponse wraps the paginated products response.
type listProductsResponse struct {
	Status      string    `json:"status"`
	Message     string    `json:"message"`
	CurrentPage int       `json:"current_page"`
	Data        []Product `json:"data"`
	LastPage    int       `json:"last_page"`
	PerPage     int       `json:"per_page"`
	Total       int       `json:"total"`
}

// Service provides product API operations.
type Service struct {
	client *httpclient.Client
}

// NewService creates a new product Service.
func NewService(client *httpclient.Client) *Service {
	return &Service{client: client}
}

// ListCategories returns all product categories.
func (s *Service) ListCategories(ctx context.Context) ([]Category, error) {
	var resp listCategoriesResponse
	if err := s.client.Get(ctx, "/list-products-categories", nil, &resp); err != nil {
		return nil, fmt.Errorf("listing product categories: %w", err)
	}
	return resp.Data, nil
}

// ListAll returns all products with optional filters.
func (s *Service) ListAll(ctx context.Context, cardType, cardSlug, include string) ([]Product, error) {
	q := url.Values{}
	if cardType != "" {
		q.Set("card_type", cardType)
	}
	if cardSlug != "" {
		q.Set("card_slug", cardSlug)
	}
	if include != "" {
		q.Set("include", include)
	}

	var resp listProductsResponse
	if err := s.client.Get(ctx, "/list-all-products", q, &resp); err != nil {
		return nil, fmt.Errorf("listing products: %w", err)
	}
	return resp.Data, nil
}
