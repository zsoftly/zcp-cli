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
				"name": "b1.g1",
				"slug": "b1g1",
				"attribute": map[string]interface{}{
					"size":           50,
					"formatted_size": "50.0 (GB)",
					"storage_tags":   "rbd-fast",
				},
				"storage_category_id": "cat-nvme-id",
				"status":              true,
				"is_custom":           true,
				"hourly_price":        0.00019,
				"monthly_price":       0.14,
				"prices":              []interface{}{},
				"tag":                 []interface{}{},
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

	p := plans[0]
	if p.Name != "b1.g1" {
		t.Errorf("Name = %q, want %q", p.Name, "b1.g1")
	}
	if p.StorageCategoryID != "cat-nvme-id" {
		t.Errorf("StorageCategoryID = %q, want %q", p.StorageCategoryID, "cat-nvme-id")
	}
	if p.Attribute.StorageTags != "rbd-fast" {
		t.Errorf("Attribute.StorageTags = %q, want %q", p.Attribute.StorageTags, "rbd-fast")
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

// TestFlexNumberUnmarshal verifies that FlexNumber accepts numbers and quoted
// numeric strings and rejects booleans, objects, arrays, and non-numeric strings.
func TestFlexNumberUnmarshal(t *testing.T) {
	cases := []struct {
		input string
		want  string
		ok    bool
	}{
		{`200`, "200", true},
		{`0`, "0", true},
		{`-1`, "-1", true},
		{`"200"`, "200", true},
		{`null`, "", true},
		{`true`, "", false},
		{`false`, "", false},
		{`{}`, "", false},
		{`[]`, "", false},
		{`"abc"`, "", false},
	}
	for _, tc := range cases {
		var n plan.FlexNumber
		err := json.Unmarshal([]byte(tc.input), &n)
		if tc.ok {
			if err != nil {
				t.Errorf("input=%s: unexpected error: %v", tc.input, err)
			} else if string(n) != tc.want {
				t.Errorf("input=%s: got %q, want %q", tc.input, string(n), tc.want)
			}
		} else {
			if err == nil {
				t.Errorf("input=%s: expected error, got nil (FlexNumber=%q)", tc.input, string(n))
			}
		}
	}
}

// TestNetworkRateAsNumber verifies that a plan with network_rate as a bare JSON
// number (as returned by the Virtual Router plan endpoint) decodes without error.
func TestNetworkRateAsNumber(t *testing.T) {
	payload := `{"status":"Success","message":"OK","data":[{"id":"vr-1","name":"VR-1GB","slug":"vr-1gb","attribute":{"cpu":1,"memory":1024,"network_rate":200,"formatted_memory":"1.0 (GB)","formatted_cpu":1},"status":true,"is_custom":false,"hourly_price":1.5,"monthly_price":550,"prices":[],"tag":[]}]}`

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(payload))
	}))
	defer srv.Close()

	client := httpclient.New(httpclient.Options{
		BaseURL:     srv.URL,
		BearerToken: "test-token",
		Timeout:     5 * time.Second,
	})

	svc := plan.NewService(client)
	plans, err := svc.List(context.Background(), plan.ServiceVirtualRouter)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(plans) != 1 {
		t.Fatalf("got %d plans, want 1", len(plans))
	}
	if plans[0].Attribute.NetworkRate.String() != "200" {
		t.Errorf("NetworkRate = %q, want %q", plans[0].Attribute.NetworkRate.String(), "200")
	}
}
