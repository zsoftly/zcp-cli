package zone_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/zsoftly/zcp-cli/internal/api/zone"
	"github.com/zsoftly/zcp-cli/internal/httpclient"
)

type listZoneResponse struct {
	Count            int         `json:"count"`
	ListZoneResponse []zone.Zone `json:"listZoneResponse"`
}

func TestZoneList(t *testing.T) {
	expected := []zone.Zone{
		{UUID: "uuid-1", Name: "Zone A", CountryName: "Canada", IsActive: true},
		{UUID: "uuid-2", Name: "Zone B", CountryName: "US", IsActive: false},
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/restapi/zone/zonelist" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(listZoneResponse{
			Count:            len(expected),
			ListZoneResponse: expected,
		})
	}))
	defer srv.Close()

	client := httpclient.New(httpclient.Options{
		BaseURL:   srv.URL,
		APIKey:    "k",
		SecretKey: "s",
		Timeout:   5 * time.Second,
	})

	svc := zone.NewService(client)
	zones, err := svc.List(context.Background(), "")
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	if len(zones) != 2 {
		t.Fatalf("List() returned %d zones, want 2", len(zones))
	}
	if zones[0].UUID != "uuid-1" {
		t.Errorf("zones[0].UUID = %q, want %q", zones[0].UUID, "uuid-1")
	}
	if zones[1].Name != "Zone B" {
		t.Errorf("zones[1].Name = %q, want %q", zones[1].Name, "Zone B")
	}
}

func TestZoneListWithUUIDFilter(t *testing.T) {
	var gotQuery string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.Query().Get("uuid")
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(listZoneResponse{Count: 0, ListZoneResponse: nil})
	}))
	defer srv.Close()

	client := httpclient.New(httpclient.Options{
		BaseURL:   srv.URL,
		APIKey:    "k",
		SecretKey: "s",
		Timeout:   5 * time.Second,
	})

	svc := zone.NewService(client)
	svc.List(context.Background(), "target-uuid")

	if gotQuery != "target-uuid" {
		t.Errorf("uuid query param = %q, want %q", gotQuery, "target-uuid")
	}
}

func TestZoneListAPIError(t *testing.T) {
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

	client := httpclient.New(httpclient.Options{
		BaseURL:   srv.URL,
		APIKey:    "bad",
		SecretKey: "creds",
		Timeout:   5 * time.Second,
	})

	svc := zone.NewService(client)
	_, err := svc.List(context.Background(), "")
	if err == nil {
		t.Fatal("expected error for 401, got nil")
	}
}
