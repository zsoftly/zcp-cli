package kubernetes_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/zsoftly/zcp-cli/internal/api/kubernetes"
	"github.com/zsoftly/zcp-cli/internal/httpclient"
)

func newTestClient(srv *httptest.Server) *httpclient.Client {
	return httpclient.New(httpclient.Options{
		BaseURL:     srv.URL,
		BearerToken: "test-token",
	})
}

func TestKubernetesListClusters(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/kubernetes-clusters" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != http.MethodGet {
			t.Errorf("unexpected method: %s", r.Method)
		}
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  "Success",
			"message": "OK",
			"data": []map[string]interface{}{
				{
					"id":            "abc-123",
					"name":          "my-cluster",
					"slug":          "my-cluster",
					"state":         "Running",
					"version":       "v1.28.4",
					"node_size":     3,
					"control_nodes": 1,
					"enable_ha":     false,
					"created_at":    "2026-04-04T17:09:26.000000Z",
					"updated_at":    "2026-04-04T17:10:20.000000Z",
				},
			},
			"current_page": 1,
			"last_page":    1,
			"total":        1,
		})
	}))
	defer srv.Close()

	svc := kubernetes.NewService(newTestClient(srv))
	clusters, err := svc.List(context.Background())
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(clusters) != 1 {
		t.Fatalf("expected 1 cluster, got %d", len(clusters))
	}
	if clusters[0].ID != "abc-123" {
		t.Errorf("ID = %q, want %q", clusters[0].ID, "abc-123")
	}
	if clusters[0].Name != "my-cluster" {
		t.Errorf("Name = %q, want %q", clusters[0].Name, "my-cluster")
	}
	if clusters[0].Slug != "my-cluster" {
		t.Errorf("Slug = %q, want %q", clusters[0].Slug, "my-cluster")
	}
	if clusters[0].State != "Running" {
		t.Errorf("State = %q, want %q", clusters[0].State, "Running")
	}
	if clusters[0].Version != "v1.28.4" {
		t.Errorf("Version = %q, want %q", clusters[0].Version, "v1.28.4")
	}
	if clusters[0].NodeSize != 3 {
		t.Errorf("NodeSize = %d, want %d", clusters[0].NodeSize, 3)
	}
	if clusters[0].ControlNodes != 1 {
		t.Errorf("ControlNodes = %d, want %d", clusters[0].ControlNodes, 1)
	}
}

func TestKubernetesListClustersEmpty(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":       "Success",
			"message":      "OK",
			"data":         []interface{}{},
			"current_page": 1,
			"last_page":    1,
			"total":        0,
		})
	}))
	defer srv.Close()

	svc := kubernetes.NewService(newTestClient(srv))
	clusters, err := svc.List(context.Background())
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(clusters) != 0 {
		t.Errorf("expected 0 clusters, got %d", len(clusters))
	}
}

func TestKubernetesCreate(t *testing.T) {
	var gotBody map[string]interface{}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/kubernetes-clusters" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Errorf("unexpected method: %s", r.Method)
		}
		json.NewDecoder(r.Body).Decode(&gotBody)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  "Success",
			"message": "OK",
			"data": map[string]interface{}{
				"id":            "new-123",
				"name":          "test-cluster",
				"slug":          "test-cluster",
				"state":         "Starting",
				"version":       "v1.28.4",
				"node_size":     3,
				"control_nodes": 1,
				"enable_ha":     false,
				"created_at":    "2026-04-04T17:09:26.000000Z",
				"updated_at":    "2026-04-04T17:09:26.000000Z",
			},
		})
	}))
	defer srv.Close()

	svc := kubernetes.NewService(newTestClient(srv))
	req := kubernetes.CreateRequest{
		Name:          "test-cluster",
		Version:       "v1.28.4",
		NodeSize:      3,
		ControlNodes:  1,
		CloudProvider: "nimbo",
		Region:        "noida",
		Project:       "default-59",
		BillingCycle:  "monthly",
		EnableHA:      false,
		Networks:      []string{},
		Plan:          "k8s-plan-1",
		SSHKey:        "mykey",
		AuthMethod:    "ssh-key",
	}
	cluster, err := svc.Create(context.Background(), req)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if cluster.ID != "new-123" {
		t.Errorf("ID = %q, want %q", cluster.ID, "new-123")
	}
	if cluster.Name != "test-cluster" {
		t.Errorf("Name = %q, want %q", cluster.Name, "test-cluster")
	}
	if cluster.Slug != "test-cluster" {
		t.Errorf("Slug = %q, want %q", cluster.Slug, "test-cluster")
	}
	// Verify body fields were sent
	if gotBody["name"] != "test-cluster" {
		t.Errorf("body name = %v, want %q", gotBody["name"], "test-cluster")
	}
	if gotBody["version"] != "v1.28.4" {
		t.Errorf("body version = %v, want %q", gotBody["version"], "v1.28.4")
	}
	if gotBody["region"] != "noida" {
		t.Errorf("body region = %v, want %q", gotBody["region"], "noida")
	}
	if gotBody["plan"] != "k8s-plan-1" {
		t.Errorf("body plan = %v, want %q", gotBody["plan"], "k8s-plan-1")
	}
	if gotBody["ssh_key"] != "mykey" {
		t.Errorf("body ssh_key = %v, want %q", gotBody["ssh_key"], "mykey")
	}
}

func TestKubernetesStart(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/kubernetes-clusters/my-cluster/start" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != http.MethodPut {
			t.Errorf("unexpected method: %s, want PUT", r.Method)
		}
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  "Success",
			"message": "Kubernetes cluster start initiated.",
		})
	}))
	defer srv.Close()

	svc := kubernetes.NewService(newTestClient(srv))
	err := svc.Start(context.Background(), "my-cluster")
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}
}

func TestKubernetesStop(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/kubernetes-clusters/my-cluster/stop" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != http.MethodPut {
			t.Errorf("unexpected method: %s, want PUT", r.Method)
		}
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  "Success",
			"message": "Kubernetes cluster stop initiated.",
		})
	}))
	defer srv.Close()

	svc := kubernetes.NewService(newTestClient(srv))
	err := svc.Stop(context.Background(), "my-cluster")
	if err != nil {
		t.Fatalf("Stop() error = %v", err)
	}
}

func TestKubernetesUpgrade(t *testing.T) {
	var gotBody map[string]interface{}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/kubernetes-clusters/my-cluster/change-plan" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != http.MethodPut {
			t.Errorf("unexpected method: %s, want PUT", r.Method)
		}
		json.NewDecoder(r.Body).Decode(&gotBody)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  "Success",
			"message": "Kubernetes cluster upgrade initiated.",
		})
	}))
	defer srv.Close()

	svc := kubernetes.NewService(newTestClient(srv))
	req := kubernetes.UpgradeRequest{
		Plan:         "k8s-plan-2",
		Slug:         "my-cluster",
		BillingCycle: "hourly",
		IsCustomPlan: false,
		CustomPlan:   nil,
	}
	err := svc.Upgrade(context.Background(), "my-cluster", req)
	if err != nil {
		t.Fatalf("Upgrade() error = %v", err)
	}
	if gotBody["plan"] != "k8s-plan-2" {
		t.Errorf("body plan = %v, want %q", gotBody["plan"], "k8s-plan-2")
	}
	if gotBody["slug"] != "my-cluster" {
		t.Errorf("body slug = %v, want %q", gotBody["slug"], "my-cluster")
	}
	if gotBody["billing_cycle"] != "hourly" {
		t.Errorf("body billing_cycle = %v, want %q", gotBody["billing_cycle"], "hourly")
	}
}
