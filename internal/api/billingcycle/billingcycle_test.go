package billingcycle_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/zsoftly/zcp-cli/internal/api/billingcycle"
	"github.com/zsoftly/zcp-cli/internal/httpclient"
)

func TestBillingCycleList(t *testing.T) {
	expected := []billingcycle.BillingCycle{
		{
			ID:        "bc-1",
			Name:      "Hourly",
			Slug:      "hourly",
			Duration:  1,
			Unit:      "hour",
			IsEnabled: true,
			PaymentModes: []billingcycle.PaymentMode{
				{ID: "pm-1", Name: "PREPAID", Slug: "prepaid", DisplayName: "Online Payment", Status: true},
			},
		},
		{
			ID:        "bc-2",
			Name:      "Monthly",
			Slug:      "monthly",
			Duration:  1,
			Unit:      "month",
			IsEnabled: true,
		},
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/billing-cycles" {
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

	svc := billingcycle.NewService(client)
	cycles, err := svc.List(context.Background())
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	if len(cycles) != 2 {
		t.Fatalf("List() returned %d cycles, want 2", len(cycles))
	}
	if cycles[0].Slug != "hourly" {
		t.Errorf("cycles[0].Slug = %q, want %q", cycles[0].Slug, "hourly")
	}
	if cycles[0].Unit != "hour" {
		t.Errorf("cycles[0].Unit = %q, want %q", cycles[0].Unit, "hour")
	}
	if len(cycles[0].PaymentModes) != 1 {
		t.Fatalf("cycles[0].PaymentModes has %d items, want 1", len(cycles[0].PaymentModes))
	}
	if cycles[0].PaymentModes[0].DisplayName != "Online Payment" {
		t.Errorf("cycles[0].PaymentModes[0].DisplayName = %q, want %q",
			cycles[0].PaymentModes[0].DisplayName, "Online Payment")
	}
}

func TestBillingCycleListAPIError(t *testing.T) {
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

	svc := billingcycle.NewService(client)
	_, err := svc.List(context.Background())
	if err == nil {
		t.Fatal("expected error for 401, got nil")
	}
}
