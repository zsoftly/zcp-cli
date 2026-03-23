package vpn_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/zsoftly/zcp-cli/internal/api/vpn"
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

// ---------------------------------------------------------------------------
// Gateway tests
// ---------------------------------------------------------------------------

type listVpnGatewayResponse struct {
	Count                  int           `json:"count"`
	ListVpnGatewayResponse []vpn.Gateway `json:"listVpnGatewayResponse"`
}

func TestGatewayList(t *testing.T) {
	expected := []vpn.Gateway{
		{UUID: "gw-1", VPCUUID: "vpc-1", ZoneUUID: "zone-1", Status: "Enabled"},
		{UUID: "gw-2", VPCUUID: "vpc-2", ZoneUUID: "zone-1", Status: "Enabled"},
	}

	var gotZone string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/restapi/vpngateway/vpnGatewayList" {
			http.NotFound(w, r)
			return
		}
		gotZone = r.URL.Query().Get("zoneUuid")
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(listVpnGatewayResponse{Count: len(expected), ListVpnGatewayResponse: expected})
	}))
	defer srv.Close()

	svc := vpn.NewGatewayService(newClient(srv.URL))
	gws, err := svc.List(context.Background(), "zone-1", "", "")
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(gws) != 2 {
		t.Fatalf("List() returned %d gateways, want 2", len(gws))
	}
	if gotZone != "zone-1" {
		t.Errorf("zoneUuid query param = %q, want %q", gotZone, "zone-1")
	}
	if gws[0].UUID != "gw-1" {
		t.Errorf("gws[0].UUID = %q, want %q", gws[0].UUID, "gw-1")
	}
}

func TestGatewayListWithFilters(t *testing.T) {
	var gotUUID, gotVPCUUID string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUUID = r.URL.Query().Get("uuid")
		gotVPCUUID = r.URL.Query().Get("vpcUuid")
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(listVpnGatewayResponse{Count: 0})
	}))
	defer srv.Close()

	svc := vpn.NewGatewayService(newClient(srv.URL))
	_, err := svc.List(context.Background(), "zone-1", "gw-1", "vpc-1")
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if gotUUID != "gw-1" {
		t.Errorf("uuid query param = %q, want %q", gotUUID, "gw-1")
	}
	if gotVPCUUID != "vpc-1" {
		t.Errorf("vpcUuid query param = %q, want %q", gotVPCUUID, "vpc-1")
	}
}

func TestGatewayCreate(t *testing.T) {
	created := vpn.Gateway{UUID: "gw-new", VPCUUID: "vpc-1", Status: "Enabled"}

	var gotBody map[string]interface{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "expected POST", http.StatusMethodNotAllowed)
			return
		}
		if r.URL.Path != "/restapi/vpngateway/addVpnGateway" {
			http.NotFound(w, r)
			return
		}
		json.NewDecoder(r.Body).Decode(&gotBody)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(listVpnGatewayResponse{Count: 1, ListVpnGatewayResponse: []vpn.Gateway{created}})
	}))
	defer srv.Close()

	svc := vpn.NewGatewayService(newClient(srv.URL))
	result, err := svc.Create(context.Background(), "vpc-1")
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if result.UUID != "gw-new" {
		t.Errorf("result.UUID = %q, want %q", result.UUID, "gw-new")
	}
	if gotBody["vpcUuid"] != "vpc-1" {
		t.Errorf("body vpcUuid = %v, want %q", gotBody["vpcUuid"], "vpc-1")
	}
}

func TestGatewayCreateEmptyResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(listVpnGatewayResponse{Count: 0})
	}))
	defer srv.Close()

	svc := vpn.NewGatewayService(newClient(srv.URL))
	_, err := svc.Create(context.Background(), "vpc-1")
	if err == nil {
		t.Fatal("Create() expected error on empty response, got nil")
	}
}

func TestGatewayDelete(t *testing.T) {
	var gotPath, gotMethod string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	svc := vpn.NewGatewayService(newClient(srv.URL))
	err := svc.Delete(context.Background(), "gw-del-1")
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if gotMethod != http.MethodDelete {
		t.Errorf("method = %q, want %q", gotMethod, http.MethodDelete)
	}
	if gotPath != "/restapi/vpngateway/deleteVpnGateway/gw-del-1" {
		t.Errorf("path = %q, want %q", gotPath, "/restapi/vpngateway/deleteVpnGateway/gw-del-1")
	}
}

func TestGatewayListError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "server error", http.StatusInternalServerError)
	}))
	defer srv.Close()

	svc := vpn.NewGatewayService(newClient(srv.URL))
	_, err := svc.List(context.Background(), "zone-1", "", "")
	if err == nil {
		t.Fatal("List() expected error on 500, got nil")
	}
}

// ---------------------------------------------------------------------------
// Connection tests
// ---------------------------------------------------------------------------

type listVpnConnectionResponse struct {
	Count                     int              `json:"count"`
	ListVpnConnectionResponse []vpn.Connection `json:"listVpnConnectionResponse"`
}

func TestConnectionList(t *testing.T) {
	expected := []vpn.Connection{
		{UUID: "conn-1", VPNGatewayUUID: "gw-1", CustomerGatewayUUID: "cgw-1", ZoneUUID: "zone-1"},
	}

	var gotZone string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/restapi/vpnconnection/vpnConnectionList" {
			http.NotFound(w, r)
			return
		}
		gotZone = r.URL.Query().Get("zoneUuid")
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(listVpnConnectionResponse{Count: len(expected), ListVpnConnectionResponse: expected})
	}))
	defer srv.Close()

	svc := vpn.NewConnectionService(newClient(srv.URL))
	conns, err := svc.List(context.Background(), "zone-1", "", "")
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(conns) != 1 {
		t.Fatalf("List() returned %d connections, want 1", len(conns))
	}
	if gotZone != "zone-1" {
		t.Errorf("zoneUuid query param = %q, want %q", gotZone, "zone-1")
	}
	if conns[0].UUID != "conn-1" {
		t.Errorf("conns[0].UUID = %q, want %q", conns[0].UUID, "conn-1")
	}
}

func TestConnectionCreate(t *testing.T) {
	created := vpn.Connection{UUID: "conn-new", VPNGatewayUUID: "gw-1", CustomerGatewayUUID: "cgw-1"}

	var gotBody map[string]interface{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "expected POST", http.StatusMethodNotAllowed)
			return
		}
		if r.URL.Path != "/restapi/vpnconnection/addVpnConnection" {
			http.NotFound(w, r)
			return
		}
		json.NewDecoder(r.Body).Decode(&gotBody)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(listVpnConnectionResponse{Count: 1, ListVpnConnectionResponse: []vpn.Connection{created}})
	}))
	defer srv.Close()

	svc := vpn.NewConnectionService(newClient(srv.URL))
	req := vpn.ConnectionCreateRequest{
		VPCUUID:             "vpc-1",
		CustomerGatewayUUID: "cgw-1",
		Passive:             false,
	}
	result, err := svc.Create(context.Background(), req)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if result.UUID != "conn-new" {
		t.Errorf("result.UUID = %q, want %q", result.UUID, "conn-new")
	}
	if gotBody["vpcUuid"] != "vpc-1" {
		t.Errorf("body vpcUuid = %v, want %q", gotBody["vpcUuid"], "vpc-1")
	}
	if gotBody["customerGatewayUuid"] != "cgw-1" {
		t.Errorf("body customerGatewayUuid = %v, want %q", gotBody["customerGatewayUuid"], "cgw-1")
	}
}

func TestConnectionCreateEmptyResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(listVpnConnectionResponse{Count: 0})
	}))
	defer srv.Close()

	svc := vpn.NewConnectionService(newClient(srv.URL))
	_, err := svc.Create(context.Background(), vpn.ConnectionCreateRequest{VPCUUID: "vpc-1", CustomerGatewayUUID: "cgw-1"})
	if err == nil {
		t.Fatal("Create() expected error on empty response, got nil")
	}
}

func TestConnectionReset(t *testing.T) {
	resetConn := vpn.Connection{UUID: "conn-1", State: "Connected"}

	var gotPath, gotMethod string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(listVpnConnectionResponse{Count: 1, ListVpnConnectionResponse: []vpn.Connection{resetConn}})
	}))
	defer srv.Close()

	svc := vpn.NewConnectionService(newClient(srv.URL))
	result, err := svc.Reset(context.Background(), "conn-1")
	if err != nil {
		t.Fatalf("Reset() error = %v", err)
	}
	if result.UUID != "conn-1" {
		t.Errorf("result.UUID = %q, want %q", result.UUID, "conn-1")
	}
	if gotMethod != http.MethodPut {
		t.Errorf("method = %q, want %q", gotMethod, http.MethodPut)
	}
	if gotPath != "/restapi/vpnconnection/resetVpnConnection/conn-1" {
		t.Errorf("path = %q, want %q", gotPath, "/restapi/vpnconnection/resetVpnConnection/conn-1")
	}
}

func TestConnectionDelete(t *testing.T) {
	var gotPath, gotMethod string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	svc := vpn.NewConnectionService(newClient(srv.URL))
	err := svc.Delete(context.Background(), "conn-del-1")
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if gotMethod != http.MethodDelete {
		t.Errorf("method = %q, want %q", gotMethod, http.MethodDelete)
	}
	if gotPath != "/restapi/vpnconnection/deleteVpnConnection/conn-del-1" {
		t.Errorf("path = %q, want %q", gotPath, "/restapi/vpnconnection/deleteVpnConnection/conn-del-1")
	}
}

// ---------------------------------------------------------------------------
// CustomerGateway tests
// ---------------------------------------------------------------------------

type listVpnCustomerGatewayResponse struct {
	Count                          int                   `json:"count"`
	ListVpnCustomerGatewayResponse []vpn.CustomerGateway `json:"listVpnCustomerGatewayResponse"`
}

func TestCustomerGatewayList(t *testing.T) {
	expected := []vpn.CustomerGateway{
		{UUID: "cgw-1", CIDRList: "10.0.0.0/8", IKEPolicy: "aes-128-sha1-modp1536"},
		{UUID: "cgw-2", CIDRList: "192.168.0.0/16", IKEPolicy: "aes-256-sha256-modp2048"},
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/restapi/vpncustomergateway/vpnCustomerGatewayList" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(listVpnCustomerGatewayResponse{Count: len(expected), ListVpnCustomerGatewayResponse: expected})
	}))
	defer srv.Close()

	svc := vpn.NewCustomerGatewayService(newClient(srv.URL))
	cgws, err := svc.List(context.Background(), "")
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(cgws) != 2 {
		t.Fatalf("List() returned %d customer gateways, want 2", len(cgws))
	}
	if cgws[0].UUID != "cgw-1" {
		t.Errorf("cgws[0].UUID = %q, want %q", cgws[0].UUID, "cgw-1")
	}
}

func TestCustomerGatewayListWithUUID(t *testing.T) {
	var gotUUID string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUUID = r.URL.Query().Get("uuid")
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(listVpnCustomerGatewayResponse{Count: 0})
	}))
	defer srv.Close()

	svc := vpn.NewCustomerGatewayService(newClient(srv.URL))
	_, err := svc.List(context.Background(), "cgw-1")
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if gotUUID != "cgw-1" {
		t.Errorf("uuid query param = %q, want %q", gotUUID, "cgw-1")
	}
}

func TestCustomerGatewayCreate(t *testing.T) {
	created := vpn.CustomerGateway{UUID: "cgw-new", CIDRList: "10.0.0.0/8"}

	var gotBody map[string]interface{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "expected POST", http.StatusMethodNotAllowed)
			return
		}
		if r.URL.Path != "/restapi/vpncustomergateway/addVpnCustomerGateway" {
			http.NotFound(w, r)
			return
		}
		json.NewDecoder(r.Body).Decode(&gotBody)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(listVpnCustomerGatewayResponse{Count: 1, ListVpnCustomerGatewayResponse: []vpn.CustomerGateway{created}})
	}))
	defer srv.Close()

	svc := vpn.NewCustomerGatewayService(newClient(srv.URL))
	req := vpn.CustomerGatewayRequest{
		Name:     "my-cgw",
		Gateway:  "203.0.113.1",
		CIDRList: "10.0.0.0/8",
		IPSecPSK: "s3cr3t",
	}
	result, err := svc.Create(context.Background(), req)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if result.UUID != "cgw-new" {
		t.Errorf("result.UUID = %q, want %q", result.UUID, "cgw-new")
	}
	if gotBody["name"] != "my-cgw" {
		t.Errorf("body name = %v, want %q", gotBody["name"], "my-cgw")
	}
	if gotBody["gateway"] != "203.0.113.1" {
		t.Errorf("body gateway = %v, want %q", gotBody["gateway"], "203.0.113.1")
	}
}

func TestCustomerGatewayCreateEmptyResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(listVpnCustomerGatewayResponse{Count: 0})
	}))
	defer srv.Close()

	svc := vpn.NewCustomerGatewayService(newClient(srv.URL))
	_, err := svc.Create(context.Background(), vpn.CustomerGatewayRequest{Name: "x"})
	if err == nil {
		t.Fatal("Create() expected error on empty response, got nil")
	}
}

func TestCustomerGatewayUpdate(t *testing.T) {
	updated := vpn.CustomerGateway{UUID: "cgw-1", CIDRList: "172.16.0.0/12"}

	var gotBody map[string]interface{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			http.Error(w, "expected PUT", http.StatusMethodNotAllowed)
			return
		}
		if r.URL.Path != "/restapi/vpncustomergateway/updateVpnCustomerGateway" {
			http.NotFound(w, r)
			return
		}
		json.NewDecoder(r.Body).Decode(&gotBody)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(listVpnCustomerGatewayResponse{Count: 1, ListVpnCustomerGatewayResponse: []vpn.CustomerGateway{updated}})
	}))
	defer srv.Close()

	svc := vpn.NewCustomerGatewayService(newClient(srv.URL))
	req := vpn.CustomerGatewayUpdateRequest{
		UUID: "cgw-1",
		CustomerGatewayRequest: vpn.CustomerGatewayRequest{
			Name:     "updated-cgw",
			CIDRList: "172.16.0.0/12",
		},
	}
	result, err := svc.Update(context.Background(), req)
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if result.UUID != "cgw-1" {
		t.Errorf("result.UUID = %q, want %q", result.UUID, "cgw-1")
	}
	if gotBody["uuid"] != "cgw-1" {
		t.Errorf("body uuid = %v, want %q", gotBody["uuid"], "cgw-1")
	}
}

func TestCustomerGatewayDelete(t *testing.T) {
	var gotPath, gotMethod string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	svc := vpn.NewCustomerGatewayService(newClient(srv.URL))
	err := svc.Delete(context.Background(), "cgw-del-1")
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if gotMethod != http.MethodDelete {
		t.Errorf("method = %q, want %q", gotMethod, http.MethodDelete)
	}
	if gotPath != "/restapi/vpncustomergateway/deleteVpnCustomerGateway/cgw-del-1" {
		t.Errorf("path = %q, want %q", gotPath, "/restapi/vpncustomergateway/deleteVpnCustomerGateway/cgw-del-1")
	}
}

// ---------------------------------------------------------------------------
// User tests
// ---------------------------------------------------------------------------

type listVpnUserResponse struct {
	Count               int        `json:"count"`
	ListVpnUserResponse []vpn.User `json:"listVpnUserResponse"`
}

func TestVPNUserList(t *testing.T) {
	expected := []vpn.User{
		{UUID: "usr-1", UserName: "alice", IsActive: true},
		{UUID: "usr-2", UserName: "bob", IsActive: false},
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/restapi/vpnuser/vpnUserlist" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(listVpnUserResponse{Count: len(expected), ListVpnUserResponse: expected})
	}))
	defer srv.Close()

	svc := vpn.NewUserService(newClient(srv.URL))
	users, err := svc.List(context.Background(), "")
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(users) != 2 {
		t.Fatalf("List() returned %d users, want 2", len(users))
	}
	if users[0].UUID != "usr-1" {
		t.Errorf("users[0].UUID = %q, want %q", users[0].UUID, "usr-1")
	}
	if users[0].UserName != "alice" {
		t.Errorf("users[0].UserName = %q, want %q", users[0].UserName, "alice")
	}
}

func TestVPNUserListWithUUID(t *testing.T) {
	var gotUUID string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUUID = r.URL.Query().Get("uuid")
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(listVpnUserResponse{Count: 0})
	}))
	defer srv.Close()

	svc := vpn.NewUserService(newClient(srv.URL))
	_, err := svc.List(context.Background(), "usr-1")
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if gotUUID != "usr-1" {
		t.Errorf("uuid query param = %q, want %q", gotUUID, "usr-1")
	}
}

func TestVPNUserCreate(t *testing.T) {
	created := vpn.User{UUID: "usr-new", UserName: "carol", IsActive: true}

	var gotBody map[string]interface{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "expected POST", http.StatusMethodNotAllowed)
			return
		}
		if r.URL.Path != "/restapi/vpnuser/addVpnUser" {
			http.NotFound(w, r)
			return
		}
		json.NewDecoder(r.Body).Decode(&gotBody)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(listVpnUserResponse{Count: 1, ListVpnUserResponse: []vpn.User{created}})
	}))
	defer srv.Close()

	svc := vpn.NewUserService(newClient(srv.URL))
	result, err := svc.Create(context.Background(), "carol", "p@ssw0rd")
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if result.UUID != "usr-new" {
		t.Errorf("result.UUID = %q, want %q", result.UUID, "usr-new")
	}
	if gotBody["username"] != "carol" {
		t.Errorf("body username = %v, want %q", gotBody["username"], "carol")
	}
	if gotBody["password"] != "p@ssw0rd" {
		t.Errorf("body password = %v, want %q", gotBody["password"], "p@ssw0rd")
	}
}

func TestVPNUserCreateEmptyResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(listVpnUserResponse{Count: 0})
	}))
	defer srv.Close()

	svc := vpn.NewUserService(newClient(srv.URL))
	_, err := svc.Create(context.Background(), "dave", "secret")
	if err == nil {
		t.Fatal("Create() expected error on empty response, got nil")
	}
}

func TestVPNUserDelete(t *testing.T) {
	var gotPath, gotMethod, gotUserName string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		gotUserName = r.URL.Query().Get("userName")
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	svc := vpn.NewUserService(newClient(srv.URL))
	err := svc.Delete(context.Background(), "alice")
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if gotMethod != http.MethodDelete {
		t.Errorf("method = %q, want %q", gotMethod, http.MethodDelete)
	}
	if gotPath != "/restapi/vpnuser/deleteVpnUser" {
		t.Errorf("path = %q, want %q", gotPath, "/restapi/vpnuser/deleteVpnUser")
	}
	if gotUserName != "alice" {
		t.Errorf("userName query param = %q, want %q", gotUserName, "alice")
	}
}

func TestVPNUserDeleteError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "not found", http.StatusNotFound)
	}))
	defer srv.Close()

	svc := vpn.NewUserService(newClient(srv.URL))
	err := svc.Delete(context.Background(), "nonexistent")
	if err == nil {
		t.Fatal("Delete() expected error on 404, got nil")
	}
}

func TestVPNUserListError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "server error", http.StatusInternalServerError)
	}))
	defer srv.Close()

	svc := vpn.NewUserService(newClient(srv.URL))
	_, err := svc.List(context.Background(), "")
	if err == nil {
		t.Fatal("List() expected error on 500, got nil")
	}
}
