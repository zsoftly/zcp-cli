package quota_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/zsoftly/zcp-cli/internal/api/quota"
	"github.com/zsoftly/zcp-cli/internal/httpclient"
)

func newClient(baseURL string) *httpclient.Client {
	return httpclient.New(httpclient.Options{
		BaseURL:     baseURL,
		BearerToken: "test-token",
		Timeout:     5 * time.Second,
	})
}

type listResourceQuotaResponse struct {
	Count                     int                   `json:"count"`
	ListResourceQuotaResponse []quota.ResourceQuota `json:"listResourceQuotaResponse"`
}

func TestQuotaList(t *testing.T) {
	expected := []quota.ResourceQuota{
		{UnitType: "Count", QuotaType: "Instance", AvailableLimit: "10", DomainUUID: "dom-1", UsedLimit: "3", MaximumLimit: "20"},
		{UnitType: "GB", QuotaType: "Volume", AvailableLimit: "500", DomainUUID: "dom-1", UsedLimit: "100", MaximumLimit: "1000"},
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/restapi/resource-quota/get-resource-limit" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(listResourceQuotaResponse{Count: len(expected), ListResourceQuotaResponse: expected})
	}))
	defer srv.Close()

	svc := quota.NewService(newClient(srv.URL))
	quotas, err := svc.List(context.Background(), "")
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(quotas) != 2 {
		t.Fatalf("List() returned %d quotas, want 2", len(quotas))
	}
	if quotas[0].QuotaType != "Instance" {
		t.Errorf("quotas[0].QuotaType = %q, want %q", quotas[0].QuotaType, "Instance")
	}
	if quotas[0].MaximumLimit != "20" {
		t.Errorf("quotas[0].MaximumLimit = %q, want %q", quotas[0].MaximumLimit, "20")
	}
}

func TestQuotaListWithDomainUUID(t *testing.T) {
	var gotDomainUUID string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotDomainUUID = r.URL.Query().Get("domainUuid")
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(listResourceQuotaResponse{Count: 0, ListResourceQuotaResponse: nil})
	}))
	defer srv.Close()

	svc := quota.NewService(newClient(srv.URL))
	_, err := svc.List(context.Background(), "dom-1")
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if gotDomainUUID != "dom-1" {
		t.Errorf("domainUuid query param = %q, want %q", gotDomainUUID, "dom-1")
	}
}

func TestQuotaListNoDomainParam(t *testing.T) {
	var gotDomainUUID string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotDomainUUID = r.URL.Query().Get("domainUuid")
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(listResourceQuotaResponse{Count: 0, ListResourceQuotaResponse: nil})
	}))
	defer srv.Close()

	svc := quota.NewService(newClient(srv.URL))
	_, err := svc.List(context.Background(), "")
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if gotDomainUUID != "" {
		t.Errorf("domainUuid query param should be empty, got %q", gotDomainUUID)
	}
}

func TestQuotaListError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "server error", http.StatusInternalServerError)
	}))
	defer srv.Close()

	svc := quota.NewService(newClient(srv.URL))
	_, err := svc.List(context.Background(), "")
	if err == nil {
		t.Fatal("List() expected error on 500, got nil")
	}
}
