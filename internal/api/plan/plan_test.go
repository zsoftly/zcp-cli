package plan_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/zsoftly/zcp-cli/internal/api/plan"
	"github.com/zsoftly/zcp-cli/internal/httpclient"
)

func TestListVMPlans(t *testing.T) {
	expected := map[string]interface{}{
		"status":  "Success",
		"message": "OK",
		"data": []map[string]interface{}{
			{
				"id":   "plan-1",
				"name": "BP_4vC-8GB",
				"slug": "bp-4vc-8gb",
				"attribute": map[string]interface{}{
					"cpu":              4,
					"memory":           8192,
					"storage":          0,
					"formatted_memory": "8.0 (GB)",
					"formatted_cpu":    4,
				},
				"status":        true,
				"is_custom":     false,
				"hourly_price":  9.39,
				"monthly_price": 3440,
				"prices":        []interface{}{},
				"tag":           []interface{}{},
			},
			{
				"id":   "plan-2",
				"name": "BP_2vC-4GB",
				"slug": "bp-2vc-4gb",
				"attribute": map[string]interface{}{
					"cpu":              2,
					"memory":           4096,
					"storage":          0,
					"formatted_memory": "4.0 (GB)",
					"formatted_cpu":    2,
				},
				"status":        true,
				"is_custom":     false,
				"hourly_price":  4.7,
				"monthly_price": 1720,
				"prices":        []interface{}{},
				"tag": map[string]interface{}{
					"label": "Recommended",
					"value": "Recommended",
					"color": "red",
				},
			},
		},
	}

	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(expected)
	}))
	defer srv.Close()

	client := httpclient.New(httpclient.Options{
		BaseURL:     srv.URL,
		BearerToken: "test-token",
		Timeout:     5 * time.Second,
	})

	svc := plan.NewService(client)
	plans, err := svc.List(context.Background(), plan.ServiceVM)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	if gotPath != "/plans/service/Virtual Machine" {
		t.Errorf("request path = %q, want /plans/service/Virtual Machine", gotPath)
	}

	if len(plans) != 2 {
		t.Fatalf("List() returned %d plans, want 2", len(plans))
	}
	if plans[0].ID != "plan-1" {
		t.Errorf("plans[0].ID = %q, want %q", plans[0].ID, "plan-1")
	}
	if plans[0].Name != "BP_4vC-8GB" {
		t.Errorf("plans[0].Name = %q, want %q", plans[0].Name, "BP_4vC-8GB")
	}
	if plans[0].MonthlyPrice != 3440 {
		t.Errorf("plans[0].MonthlyPrice = %v, want 3440", plans[0].MonthlyPrice)
	}
	if plans[0].Status != true {
		t.Errorf("plans[0].Status = %v, want true", plans[0].Status)
	}
}

func TestListBlockStoragePlans(t *testing.T) {
	expected := map[string]interface{}{
		"status":  "Success",
		"message": "OK",
		"data": []map[string]interface{}{
			{
				"id":   "storage-1",
				"name": "50 GB",
				"slug": "50-gb",
				"attribute": map[string]interface{}{
					"size":           50,
					"formatted_size": "50.0 (GB)",
				},
				"status":        true,
				"is_custom":     false,
				"hourly_price":  1.74,
				"monthly_price": 425,
				"prices":        []interface{}{},
				"tag":           []interface{}{},
			},
		},
	}

	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(expected)
	}))
	defer srv.Close()

	client := httpclient.New(httpclient.Options{
		BaseURL:     srv.URL,
		BearerToken: "test-token",
		Timeout:     5 * time.Second,
	})

	svc := plan.NewService(client)
	plans, err := svc.List(context.Background(), plan.ServiceBlockStorage)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	if gotPath != "/plans/service/Block Storage" {
		t.Errorf("request path = %q, want /plans/service/Block Storage", gotPath)
	}

	if len(plans) != 1 {
		t.Fatalf("List() returned %d plans, want 1", len(plans))
	}
	if plans[0].Name != "50 GB" {
		t.Errorf("plans[0].Name = %q, want %q", plans[0].Name, "50 GB")
	}
}

func TestParsedTag(t *testing.T) {
	tests := []struct {
		name string
		tag  string
		want string
	}{
		{"empty array", `[]`, "-"},
		{"object with label", `{"label":"Recommended","value":"Recommended","color":"red"}`, "Recommended"},
		{"empty object", `{}`, "-"},
		{"null-like", ``, "-"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := plan.Plan{Tag: json.RawMessage(tt.tag)}
			if got := p.ParsedTag(); got != tt.want {
				t.Errorf("ParsedTag() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestListAPIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(401)
		w.Write([]byte(`{"status":"Error","message":"Unauthorized"}`))
	}))
	defer srv.Close()

	client := httpclient.New(httpclient.Options{
		BaseURL:     srv.URL,
		BearerToken: "test-token",
		Timeout:     5 * time.Second,
	})

	svc := plan.NewService(client)
	_, err := svc.List(context.Background(), plan.ServiceVM)
	if err == nil {
		t.Fatal("expected error for 401, got nil")
	}
}
