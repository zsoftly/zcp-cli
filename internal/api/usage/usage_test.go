package usage_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/zsoftly/zcp-cli/internal/api/usage"
	"github.com/zsoftly/zcp-cli/internal/httpclient"
)

func newTestClient(t *testing.T, srv *httptest.Server) *httpclient.Client {
	t.Helper()
	return httpclient.New(httpclient.Options{
		BaseURL:     srv.URL,
		BearerToken: "tok",
		Timeout:     5 * time.Second,
	})
}

func TestCreditBalance(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/restapi/user/creditBalance" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"userEmail":     "user@example.com",
			"userType":      "customer",
			"balanceAmount": 150.75,
			"type":          "USD",
		})
	}))
	defer srv.Close()

	svc := usage.NewService(newTestClient(t, srv))
	bal, err := svc.CreditBalance(context.Background())
	if err != nil {
		t.Fatalf("CreditBalance() error = %v", err)
	}
	if bal.UserEmail != "user@example.com" {
		t.Errorf("UserEmail = %q, want %q", bal.UserEmail, "user@example.com")
	}
	if bal.BalanceAmount != 150.75 {
		t.Errorf("BalanceAmount = %v, want 150.75", bal.BalanceAmount)
	}
}

func TestUsageConsumptionList(t *testing.T) {
	var gotPeriod string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/restapi/usage/usageConsumptionList" {
			http.NotFound(w, r)
			return
		}
		gotPeriod = r.URL.Query().Get("period")
		w.Header().Set("Content-Type", "application/json")
		// Return raw JSON with no defined schema
		w.Write([]byte(`{"items":[{"resource":"vm","cost":10.5}]}`))
	}))
	defer srv.Close()

	svc := usage.NewService(newTestClient(t, srv))
	raw, err := svc.ConsumptionList(context.Background(), "2025-01", "")
	if err != nil {
		t.Fatalf("ConsumptionList() error = %v", err)
	}

	if gotPeriod != "2025-01" {
		t.Errorf("period query param = %q, want %q", gotPeriod, "2025-01")
	}

	if len(raw) == 0 {
		t.Error("ConsumptionList() returned empty raw response")
	}

	// Verify it is valid JSON
	var v interface{}
	if err := json.Unmarshal(raw, &v); err != nil {
		t.Errorf("ConsumptionList() raw response is not valid JSON: %v", err)
	}
}

func TestUsageConsumptionListWithCustomer(t *testing.T) {
	var gotCustomer string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotCustomer = r.URL.Query().Get("customer")
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	svc := usage.NewService(newTestClient(t, srv))
	_, err := svc.ConsumptionList(context.Background(), "2025-01", "admin@example.com")
	if err != nil {
		t.Fatalf("ConsumptionList() error = %v", err)
	}

	if gotCustomer != "admin@example.com" {
		t.Errorf("customer query param = %q, want %q", gotCustomer, "admin@example.com")
	}
}

func TestUsageProgressStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/restapi/usage/usageProgressStatus" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"active","progress":75}`))
	}))
	defer srv.Close()

	svc := usage.NewService(newTestClient(t, srv))
	raw, err := svc.ProgressStatus(context.Background())
	if err != nil {
		t.Fatalf("ProgressStatus() error = %v", err)
	}

	if len(raw) == 0 {
		t.Error("ProgressStatus() returned empty raw response")
	}

	var v interface{}
	if err := json.Unmarshal(raw, &v); err != nil {
		t.Errorf("ProgressStatus() raw response is not valid JSON: %v", err)
	}
}

func TestCreditBalanceAPIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(401)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"listErrorResponse": map[string]string{
				"errorCode": "UNAUTHORIZED",
				"errorMsg":  "Invalid API key",
			},
		})
	}))
	defer srv.Close()

	svc := usage.NewService(newTestClient(t, srv))
	_, err := svc.CreditBalance(context.Background())
	if err == nil {
		t.Fatal("expected error for 401, got nil")
	}
}
