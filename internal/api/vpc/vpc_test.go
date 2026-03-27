package vpc_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/zsoftly/zcp-cli/internal/api/vpc"
	"github.com/zsoftly/zcp-cli/internal/httpclient"
)

func newClient(baseURL string) *httpclient.Client {
	return httpclient.New(httpclient.Options{
		BaseURL:   baseURL,
		APIKey:    "testkey",
		SecretKey: "testsecret",
		Timeout:   5 * time.Second,
	})
}

type listVpcResponse struct {
	Count           int       `json:"count"`
	ListVpcResponse []vpc.VPC `json:"listVpcResponse"`
}

func makeVPC(uuid, name string) vpc.VPC {
	return vpc.VPC{
		UUID:       uuid,
		Name:       name,
		Status:     "Enabled",
		IsActive:   true,
		CIDR:       "10.0.0.0/8",
		ZoneUUID:   "zone-uuid-1",
		ZoneName:   "TestZone",
		DomainName: "testdomain.com",
	}
}

// TestVPCList verifies URL path, required zoneUuid param, and response parsing.
func TestVPCList(t *testing.T) {
	vpcs := []vpc.VPC{
		makeVPC("vpc-1", "production-vpc"),
		makeVPC("vpc-2", "staging-vpc"),
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/restapi/vpc/vpcList" {
			http.NotFound(w, r)
			return
		}
		zoneUUID := r.URL.Query().Get("zoneUuid")
		if zoneUUID == "" {
			http.Error(w, "zoneUuid required", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(listVpcResponse{
			Count:           len(vpcs),
			ListVpcResponse: vpcs,
		})
	}))
	defer srv.Close()

	svc := vpc.NewService(newClient(srv.URL))

	result, err := svc.List(context.Background(), "zone-uuid-1", "")
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(result) != 2 {
		t.Fatalf("List() returned %d VPCs, want 2", len(result))
	}
	if result[0].UUID != "vpc-1" {
		t.Errorf("result[0].UUID = %q, want %q", result[0].UUID, "vpc-1")
	}
	if result[1].Name != "staging-vpc" {
		t.Errorf("result[1].Name = %q, want %q", result[1].Name, "staging-vpc")
	}
}

// TestVPCGet verifies uuid param is sent and a single result is returned.
func TestVPCGet(t *testing.T) {
	expected := makeVPC("vpc-99", "target-vpc")

	var gotUUID string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/restapi/vpc/vpcId" {
			http.NotFound(w, r)
			return
		}
		gotUUID = r.URL.Query().Get("uuid")
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(listVpcResponse{
			Count:           1,
			ListVpcResponse: []vpc.VPC{expected},
		})
	}))
	defer srv.Close()

	svc := vpc.NewService(newClient(srv.URL))

	v, err := svc.Get(context.Background(), "vpc-99")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if gotUUID != "vpc-99" {
		t.Errorf("uuid query param = %q, want %q", gotUUID, "vpc-99")
	}
	if v.UUID != "vpc-99" {
		t.Errorf("v.UUID = %q, want %q", v.UUID, "vpc-99")
	}
	if v.Name != "target-vpc" {
		t.Errorf("v.Name = %q, want %q", v.Name, "target-vpc")
	}
}

// TestVPCCreate verifies POST body and response parsing.
func TestVPCCreate(t *testing.T) {
	created := makeVPC("new-vpc-1", "my-vpc")

	var gotBody map[string]interface{}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "expected POST", http.StatusMethodNotAllowed)
			return
		}
		if r.URL.Path != "/restapi/vpc/createVpc" {
			http.NotFound(w, r)
			return
		}
		json.NewDecoder(r.Body).Decode(&gotBody)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(listVpcResponse{
			Count:           1,
			ListVpcResponse: []vpc.VPC{created},
		})
	}))
	defer srv.Close()

	svc := vpc.NewService(newClient(srv.URL))

	req := vpc.CreateRequest{
		Name:            "my-vpc",
		ZoneUUID:        "zone-1",
		VPCOfferingUUID: "offering-1",
		CIDR:            "10.0.0.0/8",
	}

	v, err := svc.Create(context.Background(), req)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if v.UUID != "new-vpc-1" {
		t.Errorf("v.UUID = %q, want %q", v.UUID, "new-vpc-1")
	}
	if gotBody["name"] != "my-vpc" {
		t.Errorf("body[name] = %v, want %q", gotBody["name"], "my-vpc")
	}
	if gotBody["zoneUuid"] != "zone-1" {
		t.Errorf("body[zoneUuid] = %v, want %q", gotBody["zoneUuid"], "zone-1")
	}
	if gotBody["vpcOfferingUuid"] != "offering-1" {
		t.Errorf("body[vpcOfferingUuid] = %v, want %q", gotBody["vpcOfferingUuid"], "offering-1")
	}
	if gotBody["cIDR"] != "10.0.0.0/8" {
		t.Errorf("body[cIDR] = %v, want %q", gotBody["cIDR"], "10.0.0.0/8")
	}
}

// TestVPCUpdate verifies that PUT method is used and body is sent correctly.
func TestVPCUpdate(t *testing.T) {
	updated := makeVPC("vpc-upd-1", "renamed-vpc")

	var gotMethod string
	var gotBody map[string]interface{}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/restapi/vpc/updateVpc" {
			http.NotFound(w, r)
			return
		}
		gotMethod = r.Method
		json.NewDecoder(r.Body).Decode(&gotBody)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(listVpcResponse{
			Count:           1,
			ListVpcResponse: []vpc.VPC{updated},
		})
	}))
	defer srv.Close()

	svc := vpc.NewService(newClient(srv.URL))

	req := vpc.UpdateRequest{
		UUID:        "vpc-upd-1",
		Name:        "renamed-vpc",
		Description: "updated description",
	}

	v, err := svc.Update(context.Background(), req)
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if gotMethod != http.MethodPut {
		t.Errorf("HTTP method = %q, want %q", gotMethod, http.MethodPut)
	}
	if v.UUID != "vpc-upd-1" {
		t.Errorf("v.UUID = %q, want %q", v.UUID, "vpc-upd-1")
	}
	if gotBody["uuid"] != "vpc-upd-1" {
		t.Errorf("body[uuid] = %v, want %q", gotBody["uuid"], "vpc-upd-1")
	}
	if gotBody["name"] != "renamed-vpc" {
		t.Errorf("body[name] = %v, want %q", gotBody["name"], "renamed-vpc")
	}
}

// TestVPCDelete verifies DELETE path includes uuid.
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
	if gotPath != "/restapi/vpc/deleteVpc/vpc-del-1" {
		t.Errorf("path = %q, want %q", gotPath, "/restapi/vpc/deleteVpc/vpc-del-1")
	}
}
