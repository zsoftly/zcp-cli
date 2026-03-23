package acl_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/zsoftly/zcp-cli/internal/api/acl"
	"github.com/zsoftly/zcp-cli/internal/httpclient"
)

func newClient(baseURL string) *httpclient.Client {
	t := httpclient.Options{
		BaseURL:   baseURL,
		APIKey:    "testkey",
		SecretKey: "testsecret",
		Timeout:   5 * time.Second,
	}
	return httpclient.New(t)
}

type listNetworkAclListResponse struct {
	Count                      int              `json:"count"`
	ListNetworkAclListResponse []acl.NetworkACL `json:"listNetworkAclListResponse"`
}

type listNetworkResponse struct {
	Count               int           `json:"count"`
	ListNetworkResponse []acl.Network `json:"listNetworkResponse"`
}

func TestACLList(t *testing.T) {
	expected := []acl.NetworkACL{
		{UUID: "acl-1", Name: "default-acl", ZoneUUID: "zone-1", VPCUUID: "vpc-1"},
		{UUID: "acl-2", Name: "strict-acl", ZoneUUID: "zone-1", VPCUUID: "vpc-1"},
	}

	var gotZone string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/restapi/networkacllist/networkAclList" {
			http.NotFound(w, r)
			return
		}
		gotZone = r.URL.Query().Get("zoneUuid")
		if gotZone == "" {
			http.Error(w, "zoneUuid required", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(listNetworkAclListResponse{Count: len(expected), ListNetworkAclListResponse: expected})
	}))
	defer srv.Close()

	svc := acl.NewService(newClient(srv.URL))
	acls, err := svc.List(context.Background(), "zone-1", "", "")
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(acls) != 2 {
		t.Fatalf("List() returned %d ACLs, want 2", len(acls))
	}
	if gotZone != "zone-1" {
		t.Errorf("zoneUuid query param = %q, want %q", gotZone, "zone-1")
	}
	if acls[0].UUID != "acl-1" {
		t.Errorf("acls[0].UUID = %q, want %q", acls[0].UUID, "acl-1")
	}
}

func TestACLListWithFilters(t *testing.T) {
	expected := []acl.NetworkACL{
		{UUID: "acl-1", Name: "default-acl", ZoneUUID: "zone-1", VPCUUID: "vpc-1"},
	}

	var gotUUID, gotVPCUUID string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUUID = r.URL.Query().Get("uuid")
		gotVPCUUID = r.URL.Query().Get("vpcUuid")
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(listNetworkAclListResponse{Count: 1, ListNetworkAclListResponse: expected})
	}))
	defer srv.Close()

	svc := acl.NewService(newClient(srv.URL))
	_, err := svc.List(context.Background(), "zone-1", "acl-1", "vpc-1")
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if gotUUID != "acl-1" {
		t.Errorf("uuid query param = %q, want %q", gotUUID, "acl-1")
	}
	if gotVPCUUID != "vpc-1" {
		t.Errorf("vpcUuid query param = %q, want %q", gotVPCUUID, "vpc-1")
	}
}

func TestACLCreate(t *testing.T) {
	created := acl.NetworkACL{
		UUID:    "acl-new",
		Name:    "my-acl",
		VPCUUID: "vpc-1",
	}

	var gotBody map[string]interface{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "expected POST", http.StatusMethodNotAllowed)
			return
		}
		if r.URL.Path != "/restapi/networkacllist/createNetworkAcl" {
			http.NotFound(w, r)
			return
		}
		json.NewDecoder(r.Body).Decode(&gotBody)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(listNetworkAclListResponse{Count: 1, ListNetworkAclListResponse: []acl.NetworkACL{created}})
	}))
	defer srv.Close()

	svc := acl.NewService(newClient(srv.URL))
	req := acl.CreateRequest{
		Name:    "my-acl",
		VPCUUID: "vpc-1",
	}
	result, err := svc.Create(context.Background(), req)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if result.UUID != "acl-new" {
		t.Errorf("result.UUID = %q, want %q", result.UUID, "acl-new")
	}
	if gotBody["name"] != "my-acl" {
		t.Errorf("body name = %v, want %q", gotBody["name"], "my-acl")
	}
	if gotBody["vpcUuid"] != "vpc-1" {
		t.Errorf("body vpcUuid = %v, want %q", gotBody["vpcUuid"], "vpc-1")
	}
}

func TestACLCreateEmptyResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(listNetworkAclListResponse{Count: 0, ListNetworkAclListResponse: nil})
	}))
	defer srv.Close()

	svc := acl.NewService(newClient(srv.URL))
	_, err := svc.Create(context.Background(), acl.CreateRequest{Name: "x", VPCUUID: "vpc-1"})
	if err == nil {
		t.Fatal("Create() expected error on empty response, got nil")
	}
}

func TestACLDelete(t *testing.T) {
	var gotPath, gotMethod string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	svc := acl.NewService(newClient(srv.URL))
	err := svc.Delete(context.Background(), "acl-del-1")
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if gotMethod != http.MethodDelete {
		t.Errorf("method = %q, want %q", gotMethod, http.MethodDelete)
	}
	if gotPath != "/restapi/networkacllist/deleteNetworkAcl/acl-del-1" {
		t.Errorf("path = %q, want %q", gotPath, "/restapi/networkacllist/deleteNetworkAcl/acl-del-1")
	}
}

func TestACLDeleteError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "not found", http.StatusNotFound)
	}))
	defer srv.Close()

	svc := acl.NewService(newClient(srv.URL))
	err := svc.Delete(context.Background(), "missing")
	if err == nil {
		t.Fatal("Delete() expected error on 404, got nil")
	}
}

func TestACLReplaceNetworkACL(t *testing.T) {
	expected := []acl.Network{
		{UUID: "net-1", Name: "my-net"},
	}

	var gotNetUUID, gotACLUUID string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/restapi/network/replaceAcl" {
			http.NotFound(w, r)
			return
		}
		gotNetUUID = r.URL.Query().Get("uuid")
		gotACLUUID = r.URL.Query().Get("aclUuid")
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(listNetworkResponse{Count: 1, ListNetworkResponse: expected})
	}))
	defer srv.Close()

	svc := acl.NewService(newClient(srv.URL))
	nets, err := svc.ReplaceNetworkACL(context.Background(), "net-1", "acl-1")
	if err != nil {
		t.Fatalf("ReplaceNetworkACL() error = %v", err)
	}
	if len(nets) != 1 {
		t.Fatalf("ReplaceNetworkACL() returned %d networks, want 1", len(nets))
	}
	if gotNetUUID != "net-1" {
		t.Errorf("uuid query param = %q, want %q", gotNetUUID, "net-1")
	}
	if gotACLUUID != "acl-1" {
		t.Errorf("aclUuid query param = %q, want %q", gotACLUUID, "acl-1")
	}
}

func TestACLListError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "internal error", http.StatusInternalServerError)
	}))
	defer srv.Close()

	svc := acl.NewService(newClient(srv.URL))
	_, err := svc.List(context.Background(), "zone-1", "", "")
	if err == nil {
		t.Fatal("List() expected error on 500, got nil")
	}
}
