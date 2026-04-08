package currency_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/zsoftly/zcp-cli/internal/api/currency"
	"github.com/zsoftly/zcp-cli/internal/httpclient"
)

func TestCurrencyList(t *testing.T) {
	expected := []currency.Currency{
		{
			ID:           "cur-1",
			Name:         "INR",
			Slug:         "inr",
			Locale:       "en_IN",
			CurrencyName: "Rupees",
			Fraction:     "Paise",
			Status:       true,
			Default:      true,
			DecimalPlace: 4,
		},
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/currencies" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		data, _ := json.Marshal(expected)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  "Success",
			"message": "OK",
			"data":    json.RawMessage(data),
		})
	}))
	defer srv.Close()

	client := httpclient.New(httpclient.Options{
		BaseURL:     srv.URL,
		BearerToken: "test-token",
		Timeout:     5 * time.Second,
	})

	svc := currency.NewService(client)
	currencies, err := svc.List(context.Background())
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	if len(currencies) != 1 {
		t.Fatalf("List() returned %d currencies, want 1", len(currencies))
	}
	if currencies[0].Name != "INR" {
		t.Errorf("currencies[0].Name = %q, want %q", currencies[0].Name, "INR")
	}
	if currencies[0].CurrencyName != "Rupees" {
		t.Errorf("currencies[0].CurrencyName = %q, want %q", currencies[0].CurrencyName, "Rupees")
	}
	if !currencies[0].Default {
		t.Errorf("currencies[0].Default = false, want true")
	}
}

func TestCurrencyListAPIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(401)
		json.NewEncoder(w).Encode(map[string]string{
			"message": "Unauthenticated.",
		})
	}))
	defer srv.Close()

	client := httpclient.New(httpclient.Options{
		BaseURL:     srv.URL,
		BearerToken: "test-token",
		Timeout:     5 * time.Second,
	})

	svc := currency.NewService(client)
	_, err := svc.List(context.Background())
	if err == nil {
		t.Fatal("expected error for 401, got nil")
	}
}
