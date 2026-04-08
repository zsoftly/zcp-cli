package billing_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/zsoftly/zcp-cli/internal/api/billing"
	"github.com/zsoftly/zcp-cli/internal/httpclient"
)

func newClient(baseURL string) *httpclient.Client {
	return httpclient.New(httpclient.Options{
		BaseURL:     baseURL,
		BearerToken: "test-token",
		Timeout:     5 * time.Second,
	})
}

func envelope(data interface{}) []byte {
	d, _ := json.Marshal(data)
	resp := map[string]interface{}{
		"status":  "Success",
		"message": "OK",
		"data":    json.RawMessage(d),
	}
	b, _ := json.Marshal(resp)
	return b
}

func paginatedEnvelope(data interface{}, total int) []byte {
	d, _ := json.Marshal(data)
	resp := map[string]interface{}{
		"status":       "Success",
		"message":      "OK",
		"current_page": 1,
		"data":         json.RawMessage(d),
		"total":        total,
		"last_page":    1,
	}
	b, _ := json.Marshal(resp)
	return b
}

func TestGetBalance(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/account/balance" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(envelope(map[string]interface{}{
			"available_balance":   3644.36,
			"deposited":           5000.0,
			"current_hourly_rate": 14.20,
			"current_month_usage": 2343.95,
		}))
	}))
	defer srv.Close()

	svc := billing.NewService(newClient(srv.URL))
	bal, err := svc.GetBalance(context.Background())
	if err != nil {
		t.Fatalf("GetBalance() error = %v", err)
	}
	if bal.AvailableBalance != 3644.36 {
		t.Errorf("AvailableBalance = %v, want 3644.36", bal.AvailableBalance)
	}
	if bal.Deposited != 5000.0 {
		t.Errorf("Deposited = %v, want 5000", bal.Deposited)
	}
}

func TestGetBalanceError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
	}))
	defer srv.Close()

	svc := billing.NewService(newClient(srv.URL))
	_, err := svc.GetBalance(context.Background())
	if err == nil {
		t.Fatal("GetBalance() expected error on 401, got nil")
	}
}

func TestListServiceCosts(t *testing.T) {
	costs := []billing.ServiceCost{
		{Name: "Virtual Machine", DisplayName: "Instances", TotalCost: 554.54},
		{Name: "Block Storage", DisplayName: "Volume", TotalCost: 102.77},
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/analytics/services/costs" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(envelope(costs))
	}))
	defer srv.Close()

	svc := billing.NewService(newClient(srv.URL))
	result, err := svc.ListServiceCosts(context.Background())
	if err != nil {
		t.Fatalf("ListServiceCosts() error = %v", err)
	}
	if len(result) != 2 {
		t.Fatalf("ListServiceCosts() returned %d items, want 2", len(result))
	}
	if result[0].Name != "Virtual Machine" {
		t.Errorf("result[0].Name = %q, want %q", result[0].Name, "Virtual Machine")
	}
	if result[0].TotalCost != 554.54 {
		t.Errorf("result[0].TotalCost = %v, want 554.54", result[0].TotalCost)
	}
}

func TestListServiceCostsError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "server error", http.StatusInternalServerError)
	}))
	defer srv.Close()

	svc := billing.NewService(newClient(srv.URL))
	_, err := svc.ListServiceCosts(context.Background())
	if err == nil {
		t.Fatal("ListServiceCosts() expected error on 500, got nil")
	}
}

func TestListMonthlyUsage(t *testing.T) {
	usage := []map[string]interface{}{
		{"month": "Jan", "year": "2026", "cost": "2343.95"},
		{"month": "Feb", "year": "2026", "cost": 0},
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/analytics/month-wise-usage" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(envelope(usage))
	}))
	defer srv.Close()

	svc := billing.NewService(newClient(srv.URL))
	result, err := svc.ListMonthlyUsage(context.Background())
	if err != nil {
		t.Fatalf("ListMonthlyUsage() error = %v", err)
	}
	if len(result) != 2 {
		t.Fatalf("ListMonthlyUsage() returned %d items, want 2", len(result))
	}
	if result[0].Month != "Jan" {
		t.Errorf("result[0].Month = %q, want %q", result[0].Month, "Jan")
	}
}

func TestGetServiceCounts(t *testing.T) {
	counts := map[string]int{
		"Virtual Machine": 1,
		"Block Storage":   1,
		"Network":         2,
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/analytics/account/services/counts" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(envelope(counts))
	}))
	defer srv.Close()

	svc := billing.NewService(newClient(srv.URL))
	result, err := svc.GetServiceCounts(context.Background())
	if err != nil {
		t.Fatalf("GetServiceCounts() error = %v", err)
	}
	if result["Virtual Machine"] != 1 {
		t.Errorf("Virtual Machine count = %d, want 1", result["Virtual Machine"])
	}
	if result["Network"] != 2 {
		t.Errorf("Network count = %d, want 2", result["Network"])
	}
}

func TestGetCreditLimit(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/billing/credit-limit" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(envelope(map[string]interface{}{
			"limit":              "1000",
			"usage_amount":       0,
			"available_to_spend": 1000,
		}))
	}))
	defer srv.Close()

	svc := billing.NewService(newClient(srv.URL))
	result, err := svc.GetCreditLimit(context.Background())
	if err != nil {
		t.Fatalf("GetCreditLimit() error = %v", err)
	}
	if result.Limit != "1000" {
		t.Errorf("Limit = %q, want %q", result.Limit, "1000")
	}
	if result.AvailableToSpend != 1000 {
		t.Errorf("AvailableToSpend = %v, want 1000", result.AvailableToSpend)
	}
}

func TestListInvoices(t *testing.T) {
	invoices := []billing.Invoice{
		{
			ID:           "inv-1",
			Number:       1611,
			CustomNumber: "INV-2026-1611",
			Amount:       "5900",
			Status:       "PAID",
			Type:         "PAYABLE",
		},
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/billing/invoices" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(paginatedEnvelope(invoices, 1))
	}))
	defer srv.Close()

	svc := billing.NewService(newClient(srv.URL))
	result, total, err := svc.ListInvoices(context.Background(), 0)
	if err != nil {
		t.Fatalf("ListInvoices() error = %v", err)
	}
	if total != 1 {
		t.Errorf("total = %d, want 1", total)
	}
	if len(result) != 1 {
		t.Fatalf("ListInvoices() returned %d items, want 1", len(result))
	}
	if result[0].CustomNumber != "INV-2026-1611" {
		t.Errorf("result[0].CustomNumber = %q, want %q", result[0].CustomNumber, "INV-2026-1611")
	}
}

func TestGetInvoiceCount(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/billing/invoices-count" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(envelope(1))
	}))
	defer srv.Close()

	svc := billing.NewService(newClient(srv.URL))
	count, err := svc.GetInvoiceCount(context.Background())
	if err != nil {
		t.Fatalf("GetInvoiceCount() error = %v", err)
	}
	if count != 1 {
		t.Errorf("count = %d, want 1", count)
	}
}

func TestListActiveSubscriptions(t *testing.T) {
	subs := []billing.Subscription{
		{
			ID:                 "sub-1",
			Name:               "demo-vm",
			Product:            "Virtual Machine",
			ProductDisplayName: "Instances",
			Price:              "9.40",
			TotalUsage:         "554.54",
		},
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/billing/subscriptions/active" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(paginatedEnvelope(subs, 1))
	}))
	defer srv.Close()

	svc := billing.NewService(newClient(srv.URL))
	result, total, err := svc.ListActiveSubscriptions(context.Background(), 0)
	if err != nil {
		t.Fatalf("ListActiveSubscriptions() error = %v", err)
	}
	if total != 1 {
		t.Errorf("total = %d, want 1", total)
	}
	if len(result) != 1 {
		t.Fatalf("ListActiveSubscriptions() returned %d items, want 1", len(result))
	}
	if result[0].Product != "Virtual Machine" {
		t.Errorf("result[0].Product = %q, want %q", result[0].Product, "Virtual Machine")
	}
}

func TestListInactiveSubscriptions(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/billing/subscriptions/inactive" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(paginatedEnvelope([]billing.Subscription{}, 0))
	}))
	defer srv.Close()

	svc := billing.NewService(newClient(srv.URL))
	result, total, err := svc.ListInactiveSubscriptions(context.Background(), 0)
	if err != nil {
		t.Fatalf("ListInactiveSubscriptions() error = %v", err)
	}
	if total != 0 {
		t.Errorf("total = %d, want 0", total)
	}
	if len(result) != 0 {
		t.Fatalf("ListInactiveSubscriptions() returned %d items, want 0", len(result))
	}
}

func TestRedeemCoupon(t *testing.T) {
	var gotCode string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/account/coupons" || r.Method != http.MethodPost {
			http.NotFound(w, r)
			return
		}
		var req billing.RedeemCouponRequest
		json.NewDecoder(r.Body).Decode(&req)
		gotCode = req.Code
		w.Header().Set("Content-Type", "application/json")
		w.Write(envelope(map[string]string{"message": "Coupon applied"}))
	}))
	defer srv.Close()

	svc := billing.NewService(newClient(srv.URL))
	_, err := svc.RedeemCoupon(context.Background(), "SAVE50")
	if err != nil {
		t.Fatalf("RedeemCoupon() error = %v", err)
	}
	if gotCode != "SAVE50" {
		t.Errorf("code = %q, want %q", gotCode, "SAVE50")
	}
}

func TestCancelService(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		w.Write(envelope(map[string]string{"message": "Cancellation scheduled"}))
	}))
	defer srv.Close()

	svc := billing.NewService(newClient(srv.URL))
	err := svc.CancelService(context.Background(), "my-subscription-slug", billing.CancelServiceRequest{ServiceName: "Virtual Machine", Reason: "not_needed_anymore", Type: "Immediate"})
	if err != nil {
		t.Fatalf("CancelService() error = %v", err)
	}
	if gotPath != "/billing/service-cancel-requests/my-subscription-slug" {
		t.Errorf("path = %q, want %q", gotPath, "/billing/service-cancel-requests/my-subscription-slug")
	}
}

func TestSetBudgetAlert(t *testing.T) {
	var gotReq billing.SetBudgetAlertRequest
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/billing/budget-alert-settings" || r.Method != http.MethodPost {
			http.NotFound(w, r)
			return
		}
		json.NewDecoder(r.Body).Decode(&gotReq)
		w.Header().Set("Content-Type", "application/json")
		w.Write(envelope(map[string]string{"message": "Budget alert updated"}))
	}))
	defer srv.Close()

	svc := billing.NewService(newClient(srv.URL))
	_, err := svc.SetBudgetAlert(context.Background(), billing.SetBudgetAlertRequest{
		Amount:    500.0,
		Threshold: 80.0,
		IsEnabled: true,
	})
	if err != nil {
		t.Fatalf("SetBudgetAlert() error = %v", err)
	}
	if gotReq.Amount != 500.0 {
		t.Errorf("Amount = %v, want 500.0", gotReq.Amount)
	}
	if gotReq.Threshold != 80.0 {
		t.Errorf("Threshold = %v, want 80.0", gotReq.Threshold)
	}
	if !gotReq.IsEnabled {
		t.Error("IsEnabled = false, want true")
	}
}
