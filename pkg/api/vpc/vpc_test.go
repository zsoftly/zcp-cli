package vpc_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/zsoftly/zcp-cli/pkg/api/vpc"
	"github.com/zsoftly/zcp-cli/pkg/httpclient"
)

func newClient(baseURL string) *httpclient.Client {
	return httpclient.New(httpclient.Options{
		BaseURL:     baseURL,
		BearerToken: "test-token",
		Timeout:     5 * time.Second,
	})
}

// apiEnvelope mirrors the ZCP response envelope used by the service.
type apiEnvelope struct {
	Status string      `json:"status"`
	Data   interface{} `json:"data"`
}

func makeVPC(slug, name string) vpc.VPC {
	return vpc.VPC{
		Slug:        slug,
		Name:        name,
		Status:      "Enabled",
		CIDR:        "10.0.0.0/8",
		ZoneName:    "TestZone",
		DomainName:  "testdomain.com",
		Description: "",
	}
}

// TestVPCList verifies URL path, optional zoneSlug param, and response parsing.
func TestVPCList(t *testing.T) {
	vpcs := []vpc.VPC{
		makeVPC("production-vpc", "production-vpc"),
		makeVPC("staging-vpc", "staging-vpc"),
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/vpcs" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(apiEnvelope{
			Status: "ok",
			Data:   vpcs,
		})
	}))
	defer srv.Close()

	svc := vpc.NewService(newClient(srv.URL))

	result, err := svc.List(context.Background(), "")
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(result) != 2 {
		t.Fatalf("List() returned %d VPCs, want 2", len(result))
	}
	if result[0].Slug != "production-vpc" {
		t.Errorf("result[0].Slug = %q, want %q", result[0].Slug, "production-vpc")
	}
	if result[1].Name != "staging-vpc" {
		t.Errorf("result[1].Name = %q, want %q", result[1].Name, "staging-vpc")
	}
}

// TestVPCGet verifies that Get filters by slug from the list.
func TestVPCGet(t *testing.T) {
	allVPCs := []vpc.VPC{
		makeVPC("other-vpc", "other-vpc"),
		makeVPC("target-vpc", "target-vpc"),
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/vpcs" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(apiEnvelope{
			Status: "ok",
			Data:   allVPCs,
		})
	}))
	defer srv.Close()

	svc := vpc.NewService(newClient(srv.URL))

	v, err := svc.Get(context.Background(), "target-vpc")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if v.Slug != "target-vpc" {
		t.Errorf("v.Slug = %q, want %q", v.Slug, "target-vpc")
	}
	if v.Name != "target-vpc" {
		t.Errorf("v.Name = %q, want %q", v.Name, "target-vpc")
	}
}

// TestVPCCreate verifies POST body and response parsing.
func TestVPCCreate(t *testing.T) {
	created := makeVPC("my-vpc", "my-vpc")

	var gotBody map[string]interface{}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "expected POST", http.StatusMethodNotAllowed)
			return
		}
		if r.URL.Path != "/vpcs" {
			http.NotFound(w, r)
			return
		}
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Errorf("decoding request body: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(apiEnvelope{
			Status: "ok",
			Data:   created,
		})
	}))
	defer srv.Close()

	svc := vpc.NewService(newClient(srv.URL))

	req := vpc.CreateRequest{
		Name:            "my-vpc",
		CloudProvider:   "nimbo",
		Region:          "noida",
		Project:         "default-124",
		Type:            "Vpc",
		BillingCycle:    "hourly",
		CIDR:            "10.0.0.1",
		Size:            "24",
		Plan:            "vpc-1",
		StorageCategory: "nvme",
	}

	v, err := svc.Create(context.Background(), req)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if v.Slug != "my-vpc" {
		t.Errorf("v.Slug = %q, want %q", v.Slug, "my-vpc")
	}
	if gotBody["name"] != "my-vpc" {
		t.Errorf("body[name] = %v, want %q", gotBody["name"], "my-vpc")
	}
	if gotBody["cloud_provider"] != "nimbo" {
		t.Errorf("body[cloud_provider] = %v, want %q", gotBody["cloud_provider"], "nimbo")
	}
	if gotBody["cidr"] != "10.0.0.1" {
		t.Errorf("body[cidr] = %v, want %q", gotBody["cidr"], "10.0.0.1")
	}
	if gotBody["region"] != "noida" {
		t.Errorf("body[region] = %v, want %q", gotBody["region"], "noida")
	}
	if gotBody["project"] != "default-124" {
		t.Errorf("body[project] = %v, want %q", gotBody["project"], "default-124")
	}
	if gotBody["type"] != "Vpc" {
		t.Errorf("body[type] = %v, want %q", gotBody["type"], "Vpc")
	}
	if gotBody["billing_cycle"] != "hourly" {
		t.Errorf("body[billing_cycle] = %v, want %q", gotBody["billing_cycle"], "hourly")
	}
	if gotBody["plan"] != "vpc-1" {
		t.Errorf("body[plan] = %v, want %q", gotBody["plan"], "vpc-1")
	}
	if gotBody["storage_category"] != "nvme" {
		t.Errorf("body[storage_category] = %v, want %q", gotBody["storage_category"], "nvme")
	}
	if gotBody["size"] != "24" {
		t.Errorf("body[size] = %v, want %q", gotBody["size"], "24")
	}
}

// TestVPCUpdate verifies PUT path/body and that Update falls back to GET when the
// PUT response carries data:null (the production API behaviour).
func TestVPCUpdate(t *testing.T) {
	updated := makeVPC("vpc-upd-1", "renamed-vpc")

	var gotPUTPath string
	var gotBody map[string]interface{}
	var sawGET bool

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.Method {
		case http.MethodPut:
			gotPUTPath = r.URL.Path
			if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
				t.Errorf("decoding request body: %v", err)
			}
			// Return data:null to exercise the GET fallback in Update().
			json.NewEncoder(w).Encode(apiEnvelope{Status: "ok", Data: nil})
		case http.MethodGet:
			// Update.Get() calls List(), which GETs /vpcs.
			sawGET = true
			if r.URL.Path != "/vpcs" {
				http.NotFound(w, r)
				return
			}
			json.NewEncoder(w).Encode(apiEnvelope{Status: "ok", Data: []vpc.VPC{updated}})
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	svc := vpc.NewService(newClient(srv.URL))

	desc := "updated description"
	req := vpc.UpdateRequest{
		Name:        "renamed-vpc",
		Description: &desc,
	}

	v, err := svc.Update(context.Background(), "vpc-upd-1", req)
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if gotPUTPath != "/vpcs/vpc-upd-1" {
		t.Errorf("PUT path = %q, want %q", gotPUTPath, "/vpcs/vpc-upd-1")
	}
	if !sawGET {
		t.Error("Update() did not fall back to GET after null PUT response")
	}
	if v.Slug != "vpc-upd-1" {
		t.Errorf("v.Slug = %q, want %q", v.Slug, "vpc-upd-1")
	}
	if gotBody["name"] != "renamed-vpc" {
		t.Errorf("body[name] = %v, want %q", gotBody["name"], "renamed-vpc")
	}
	if gotBody["description"] != "updated description" {
		t.Errorf("body[description] = %v, want %q", gotBody["description"], "updated description")
	}
}

// TestVPCDelete verifies DELETE path includes slug.
func TestVPCDelete(t *testing.T) {
	var gotPath string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			http.Error(w, "expected DELETE", http.StatusMethodNotAllowed)
			return
		}
		gotPath = r.URL.Path
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	svc := vpc.NewService(newClient(srv.URL))

	err := svc.Delete(context.Background(), "vpc-del-1")
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if gotPath != "/vpcs/vpc-del-1" {
		t.Errorf("path = %q, want %q", gotPath, "/vpcs/vpc-del-1")
	}
}

// TestVPCGetDetailEndpoint verifies Get uses GET /vpcs/{slug} and maps the
// CloudStack meta block (state, cidr, zone) that the list endpoint omits.
func TestVPCGetDetailEndpoint(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/vpcs/my-vpc" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"status":"Success","data":{
			"slug":"my-vpc","name":"my-vpc","description":"prod",
			"meta":{"state":"Enabled","cidr":"10.30.0.0/16","zone_name":"yul-1","network_domain":"cs1cloud.internal"}
		}}`)
	}))
	defer srv.Close()

	svc := vpc.NewService(newClient(srv.URL))

	v, err := svc.Get(context.Background(), "my-vpc")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if v.CIDR != "10.30.0.0/16" {
		t.Errorf("v.CIDR = %q, want %q", v.CIDR, "10.30.0.0/16")
	}
	if v.Status != "Enabled" {
		t.Errorf("v.Status = %q, want %q", v.Status, "Enabled")
	}
	if v.ZoneName != "yul-1" {
		t.Errorf("v.ZoneName = %q, want %q", v.ZoneName, "yul-1")
	}
	if v.DomainName != "cs1cloud.internal" {
		t.Errorf("v.DomainName = %q, want %q", v.DomainName, "cs1cloud.internal")
	}
}

// TestVPCGetFallsBackToList verifies Get still works when the detail
// endpoint is unavailable (older deployments).
func TestVPCGetFallsBackToList(t *testing.T) {
	target := makeVPC("target-vpc", "target-vpc")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/vpcs" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(apiEnvelope{Status: "ok", Data: []vpc.VPC{target}})
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()

	svc := vpc.NewService(newClient(srv.URL))

	v, err := svc.Get(context.Background(), "target-vpc")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if v.Slug != "target-vpc" {
		t.Errorf("v.Slug = %q, want %q", v.Slug, "target-vpc")
	}
}
