package cost_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/zsoftly/zcp-cli/internal/api/cost"
	"github.com/zsoftly/zcp-cli/internal/httpclient"
)

func newClient(baseURL string) *httpclient.Client {
	return httpclient.New(httpclient.Options{
		BaseURL:   baseURL,
		APIKey:    "testkey",
		SecretKey: "testsecret",
		Timeout:   5 * time.Second,
	})
}

func TestCostListCurrencies(t *testing.T) {
	expected := cost.MultiCurrencyResponse{
		OrganizationName: "Test Org",
		Count:            2,
		ListMultiCurrency: []cost.Currency{
			{UUID: "cur-1", Currency: "USD", CurrencySymbol: "$", Cost: 1.0, IsDefaultCurrency: true},
			{UUID: "cur-2", Currency: "EUR", CurrencySymbol: "€", Cost: 0.92},
		},
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/restapi/costestimate/multicurrency" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(expected)
	}))
	defer srv.Close()

	svc := cost.NewService(newClient(srv.URL))
	resp, err := svc.ListCurrencies(context.Background())
	if err != nil {
		t.Fatalf("ListCurrencies() error = %v", err)
	}
	if resp.OrganizationName != "Test Org" {
		t.Errorf("OrganizationName = %q, want %q", resp.OrganizationName, "Test Org")
	}
	if len(resp.ListMultiCurrency) != 2 {
		t.Fatalf("ListMultiCurrency length = %d, want 2", len(resp.ListMultiCurrency))
	}
	if resp.ListMultiCurrency[0].UUID != "cur-1" {
		t.Errorf("ListMultiCurrency[0].UUID = %q, want %q", resp.ListMultiCurrency[0].UUID, "cur-1")
	}
	if resp.ListMultiCurrency[0].Currency != "USD" {
		t.Errorf("ListMultiCurrency[0].Currency = %q, want %q", resp.ListMultiCurrency[0].Currency, "USD")
	}
}

func TestCostListCurrenciesError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "server error", http.StatusInternalServerError)
	}))
	defer srv.Close()

	svc := cost.NewService(newClient(srv.URL))
	_, err := svc.ListCurrencies(context.Background())
	if err == nil {
		t.Fatal("ListCurrencies() expected error on 500, got nil")
	}
}

func TestCostGetTax(t *testing.T) {
	type taxResponse struct {
		TaxResponse []cost.TaxInfo `json:"taxResponse"`
	}
	expected := taxResponse{
		TaxResponse: []cost.TaxInfo{
			{Name: "VAT", TaxPercentage: 20.0, OrganizationTax: 18.0, IndividualTax: 20.0},
		},
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/restapi/costestimate/tax" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(expected)
	}))
	defer srv.Close()

	svc := cost.NewService(newClient(srv.URL))
	taxes, err := svc.GetTax(context.Background())
	if err != nil {
		t.Fatalf("GetTax() error = %v", err)
	}
	if len(taxes) != 1 {
		t.Fatalf("GetTax() returned %d items, want 1", len(taxes))
	}
	if taxes[0].Name != "VAT" {
		t.Errorf("taxes[0].Name = %q, want %q", taxes[0].Name, "VAT")
	}
	if taxes[0].TaxPercentage != 20.0 {
		t.Errorf("taxes[0].TaxPercentage = %v, want 20.0", taxes[0].TaxPercentage)
	}
}

func TestCostGetTaxError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
	}))
	defer srv.Close()

	svc := cost.NewService(newClient(srv.URL))
	_, err := svc.GetTax(context.Background())
	if err == nil {
		t.Fatal("GetTax() expected error on 401, got nil")
	}
}
