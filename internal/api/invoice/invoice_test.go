package invoice_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/zsoftly/zcp-cli/internal/api/invoice"
	"github.com/zsoftly/zcp-cli/internal/httpclient"
)

func newClient(baseURL string) *httpclient.Client {
	return httpclient.New(httpclient.Options{
		BaseURL:     baseURL,
		BearerToken: "test-token",
		Timeout:     5 * time.Second,
	})
}

type listInvoiceResponse struct {
	Count               int               `json:"count"`
	ListInvoiceResponse []invoice.Invoice `json:"listInvoiceResponse"`
}

func TestInvoiceList(t *testing.T) {
	expected := []invoice.Invoice{
		{InvoiceNumber: "INV-001", ClientEmail: "user@example.com", TotalCost: 100.0, BillPeriod: "2024-01"},
		{InvoiceNumber: "INV-002", ClientEmail: "user@example.com", TotalCost: 200.0, BillPeriod: "2024-02"},
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/restapi/invoice/listByClient" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(listInvoiceResponse{Count: len(expected), ListInvoiceResponse: expected})
	}))
	defer srv.Close()

	svc := invoice.NewService(newClient(srv.URL))
	invoices, err := svc.List(context.Background(), "", "", "")
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(invoices) != 2 {
		t.Fatalf("List() returned %d invoices, want 2", len(invoices))
	}
	if invoices[0].InvoiceNumber != "INV-001" {
		t.Errorf("invoices[0].InvoiceNumber = %q, want %q", invoices[0].InvoiceNumber, "INV-001")
	}
}

func TestInvoiceListWithFilters(t *testing.T) {
	var gotEmail, gotStatus, gotPeriod string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotEmail = r.URL.Query().Get("clientEmail")
		gotStatus = r.URL.Query().Get("status")
		gotPeriod = r.URL.Query().Get("billPeriod")
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(listInvoiceResponse{Count: 0, ListInvoiceResponse: nil})
	}))
	defer srv.Close()

	svc := invoice.NewService(newClient(srv.URL))
	_, err := svc.List(context.Background(), "user@example.com", "paid", "2024-01")
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if gotEmail != "user@example.com" {
		t.Errorf("clientEmail query param = %q, want %q", gotEmail, "user@example.com")
	}
	if gotStatus != "paid" {
		t.Errorf("status query param = %q, want %q", gotStatus, "paid")
	}
	if gotPeriod != "2024-01" {
		t.Errorf("billPeriod query param = %q, want %q", gotPeriod, "2024-01")
	}
}

func TestInvoiceListError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "server error", http.StatusInternalServerError)
	}))
	defer srv.Close()

	svc := invoice.NewService(newClient(srv.URL))
	_, err := svc.List(context.Background(), "", "", "")
	if err == nil {
		t.Fatal("List() expected error on 500, got nil")
	}
}

func TestInvoiceGenerate(t *testing.T) {
	expected := invoice.GenerateResponse{
		InvoiceNumber: 1001,
		Message:       "Invoice generated successfully",
		Status:        true,
	}

	var gotInvoiceNumber string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/restapi/invoice/generateInvoice" {
			http.NotFound(w, r)
			return
		}
		gotInvoiceNumber = r.URL.Query().Get("invoiceNumber")
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(expected)
	}))
	defer srv.Close()

	svc := invoice.NewService(newClient(srv.URL))
	resp, err := svc.Generate(context.Background(), "INV-001")
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	if resp.InvoiceNumber != 1001 {
		t.Errorf("resp.InvoiceNumber = %d, want 1001", resp.InvoiceNumber)
	}
	if !resp.Status {
		t.Errorf("resp.Status = false, want true")
	}
	if gotInvoiceNumber != "INV-001" {
		t.Errorf("invoiceNumber query param = %q, want %q", gotInvoiceNumber, "INV-001")
	}
}

func TestInvoiceGenerateError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "not found", http.StatusNotFound)
	}))
	defer srv.Close()

	svc := invoice.NewService(newClient(srv.URL))
	_, err := svc.Generate(context.Background(), "NONEXISTENT")
	if err == nil {
		t.Fatal("Generate() expected error on 404, got nil")
	}
}
