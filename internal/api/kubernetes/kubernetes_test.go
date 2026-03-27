package kubernetes_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/zsoftly/zcp-cli/internal/api/kubernetes"
	"github.com/zsoftly/zcp-cli/internal/httpclient"
)

func newTestClient(srv *httptest.Server) *httpclient.Client {
	return httpclient.New(httpclient.Options{
		BaseURL: srv.URL, APIKey: "k", SecretKey: "s", Timeout: 5 * time.Second,
	})
}

func TestKubernetesListCluster(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/restapi/kubernetes/listCluster" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != http.MethodGet {
			t.Errorf("unexpected method: %s", r.Method)
		}
		json.NewEncoder(w).Encode(map[string]interface{}{
			"uuid":         "cluster-1",
			"name":         "my-cluster",
			"state":        "Running",
			"size":         3,
			"controlNodes": 1,
		})
	}))
	defer srv.Close()

	svc := kubernetes.NewService(newTestClient(srv))
	clusters, err := svc.List(context.Background(), "")
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(clusters) != 1 {
		t.Fatalf("expected 1 cluster, got %d", len(clusters))
	}
	if clusters[0].UUID != "cluster-1" {
		t.Errorf("UUID = %q, want %q", clusters[0].UUID, "cluster-1")
	}
	if clusters[0].Name != "my-cluster" {
		t.Errorf("Name = %q, want %q", clusters[0].Name, "my-cluster")
	}
	if clusters[0].State != "Running" {
		t.Errorf("State = %q, want %q", clusters[0].State, "Running")
	}
}

func TestKubernetesListClusterWithFilter(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got := r.URL.Query().Get("clusterUuid")
		if got != "filter-uuid" {
			t.Errorf("clusterUuid param = %q, want %q", got, "filter-uuid")
		}
		json.NewEncoder(w).Encode(map[string]interface{}{
			"uuid":  "filter-uuid",
			"name":  "filtered-cluster",
			"state": "Stopped",
		})
	}))
	defer srv.Close()

	svc := kubernetes.NewService(newTestClient(srv))
	clusters, err := svc.List(context.Background(), "filter-uuid")
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(clusters) != 1 {
		t.Fatalf("expected 1 cluster, got %d", len(clusters))
	}
	if clusters[0].UUID != "filter-uuid" {
		t.Errorf("UUID = %q, want %q", clusters[0].UUID, "filter-uuid")
	}
}

func TestKubernetesListClusterEmpty(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Return an empty cluster object (no UUID)
		json.NewEncoder(w).Encode(map[string]interface{}{})
	}))
	defer srv.Close()

	svc := kubernetes.NewService(newTestClient(srv))
	clusters, err := svc.List(context.Background(), "")
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
		if r.URL.Path != "/restapi/kubernetes/createKubernetes" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Errorf("unexpected method: %s", r.Method)
		}
		json.NewDecoder(r.Body).Decode(&gotBody)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"uuid":  "new-cluster",
			"name":  "test-cluster",
			"state": "Starting",
		})
	}))
	defer srv.Close()

	svc := kubernetes.NewService(newTestClient(srv))
	req := kubernetes.CreateRequest{
		Name:                "test-cluster",
		ZoneUUID:            "zone-1",
		VersionUUID:         "version-1",
		ComputeOfferingUUID: "offering-1",
		TransNetworkUUID:    "network-1",
		Size:                3,
		ControlNodes:        1,
		SSHKeyName:          "mykey",
		HAEnabled:           false,
	}
	cluster, err := svc.Create(context.Background(), req)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if cluster.UUID != "new-cluster" {
		t.Errorf("UUID = %q, want %q", cluster.UUID, "new-cluster")
	}
	if cluster.Name != "test-cluster" {
		t.Errorf("Name = %q, want %q", cluster.Name, "test-cluster")
	}
	// Verify body fields were sent
	if gotBody["name"] != "test-cluster" {
		t.Errorf("body name = %v, want %q", gotBody["name"], "test-cluster")
	}
	if gotBody["zoneUuid"] != "zone-1" {
		t.Errorf("body zoneUuid = %v, want %q", gotBody["zoneUuid"], "zone-1")
	}
	if gotBody["sshKeyName"] != "mykey" {
		t.Errorf("body sshKeyName = %v, want %q", gotBody["sshKeyName"], "mykey")
	}
}

func TestKubernetesDelete(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/restapi/kubernetes/destroyKubernetes" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != http.MethodDelete {
			t.Errorf("unexpected method: %s", r.Method)
		}
		got := r.URL.Query().Get("uuid")
		if got != "cluster-del" {
			t.Errorf("uuid param = %q, want %q", got, "cluster-del")
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	svc := kubernetes.NewService(newTestClient(srv))
	err := svc.Delete(context.Background(), "cluster-del")
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
}

func TestKubernetesStart(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/restapi/kubernetes/startKubernetes" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != http.MethodPut {
			t.Errorf("unexpected method: %s, want PUT", r.Method)
		}
		got := r.URL.Query().Get("uuid")
		if got != "cluster-start" {
			t.Errorf("uuid param = %q, want %q", got, "cluster-start")
		}
		json.NewEncoder(w).Encode(map[string]interface{}{
			"uuid":  "cluster-start",
			"name":  "my-cluster",
			"state": "Running",
		})
	}))
	defer srv.Close()

	svc := kubernetes.NewService(newTestClient(srv))
	cluster, err := svc.Start(context.Background(), "cluster-start")
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	if cluster.State != "Running" {
		t.Errorf("State = %q, want %q", cluster.State, "Running")
	}
}

func TestKubernetesStop(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/restapi/kubernetes/stopKubernetes" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != http.MethodPut {
			t.Errorf("unexpected method: %s, want PUT", r.Method)
		}
		got := r.URL.Query().Get("uuid")
		if got != "cluster-stop" {
			t.Errorf("uuid param = %q, want %q", got, "cluster-stop")
		}
		json.NewEncoder(w).Encode(map[string]interface{}{
			"uuid":  "cluster-stop",
			"name":  "my-cluster",
			"state": "Stopped",
		})
	}))
	defer srv.Close()

	svc := kubernetes.NewService(newTestClient(srv))
	cluster, err := svc.Stop(context.Background(), "cluster-stop")
	if err != nil {
		t.Fatalf("Stop() error = %v", err)
	}
	if cluster.State != "Stopped" {
		t.Errorf("State = %q, want %q", cluster.State, "Stopped")
	}
}

func TestKubernetesScale(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/restapi/kubernetes/scaleKubernetes" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != http.MethodPut {
			t.Errorf("unexpected method: %s, want PUT", r.Method)
		}
		q := r.URL.Query()
		if q.Get("uuid") != "cluster-scale" {
			t.Errorf("uuid param = %q, want %q", q.Get("uuid"), "cluster-scale")
		}
		if q.Get("size") != "5" {
			t.Errorf("size param = %q, want %q", q.Get("size"), "5")
		}
		if q.Get("autoscalingEnabled") != "true" {
			t.Errorf("autoscalingEnabled param = %q, want %q", q.Get("autoscalingEnabled"), "true")
		}
		json.NewEncoder(w).Encode(map[string]interface{}{
			"uuid":  "cluster-scale",
			"name":  "my-cluster",
			"state": "Running",
			"size":  5,
		})
	}))
	defer srv.Close()

	svc := kubernetes.NewService(newTestClient(srv))
	cluster, err := svc.Scale(context.Background(), "cluster-scale", 5, true)
	if err != nil {
		t.Fatalf("Scale() error = %v", err)
	}
	if cluster.Size != 5 {
		t.Errorf("Size = %d, want %d", cluster.Size, 5)
	}
}

func TestKubernetesScaleNoAutoscaling(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if q.Get("autoscalingEnabled") != "" {
			t.Errorf("autoscalingEnabled should not be set, got %q", q.Get("autoscalingEnabled"))
		}
		json.NewEncoder(w).Encode(map[string]interface{}{
			"uuid":  "cluster-scale",
			"state": "Running",
			"size":  3,
		})
	}))
	defer srv.Close()

	svc := kubernetes.NewService(newTestClient(srv))
	_, err := svc.Scale(context.Background(), "cluster-scale", 3, false)
	if err != nil {
		t.Fatalf("Scale() error = %v", err)
	}
}

func TestKubernetesListNodes(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/restapi/kubernetes/listNodes" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != http.MethodGet {
			t.Errorf("unexpected method: %s", r.Method)
		}
		got := r.URL.Query().Get("clusterUuid")
		if got != "cluster-1" {
			t.Errorf("clusterUuid param = %q, want %q", got, "cluster-1")
		}
		json.NewEncoder(w).Encode(map[string]interface{}{
			"count": 2,
			"listInstanceResponse": []map[string]interface{}{
				{
					"uuid":              "node-1",
					"name":              "k8s-worker-1",
					"state":             "Running",
					"memory":            "4096",
					"instancePrivateIp": "10.0.0.10",
					"zoneUuid":          "zone-1",
					"isActive":          true,
				},
				{
					"uuid":              "node-2",
					"name":              "k8s-worker-2",
					"state":             "Running",
					"memory":            "4096",
					"instancePrivateIp": "10.0.0.11",
					"zoneUuid":          "zone-1",
					"isActive":          true,
				},
			},
		})
	}))
	defer srv.Close()

	svc := kubernetes.NewService(newTestClient(srv))
	nodes, err := svc.ListNodes(context.Background(), "cluster-1")
	if err != nil {
		t.Fatalf("ListNodes() error = %v", err)
	}
	if len(nodes) != 2 {
		t.Fatalf("expected 2 nodes, got %d", len(nodes))
	}
	if nodes[0].UUID != "node-1" {
		t.Errorf("nodes[0].UUID = %q, want %q", nodes[0].UUID, "node-1")
	}
	if nodes[0].PrivateIP != "10.0.0.10" {
		t.Errorf("nodes[0].PrivateIP = %q, want %q", nodes[0].PrivateIP, "10.0.0.10")
	}
	if !nodes[0].IsActive {
		t.Errorf("nodes[0].IsActive = false, want true")
	}
}

func TestKubernetesListVersions(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/restapi/costestimate/kubernetes-version-list" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != http.MethodGet {
			t.Errorf("unexpected method: %s", r.Method)
		}
		got := r.URL.Query().Get("zoneUuid")
		if got != "zone-1" {
			t.Errorf("zoneUuid param = %q, want %q", got, "zone-1")
		}
		json.NewEncoder(w).Encode(map[string]interface{}{
			"count": 2,
			"listKubernetesVersion": []map[string]interface{}{
				{
					"uuid":         "ver-1",
					"name":         "1.28",
					"description":  "Kubernetes 1.28",
					"isActive":     true,
					"minMemory":    4096,
					"minCpuNumber": 2,
				},
				{
					"uuid":         "ver-2",
					"name":         "1.27",
					"description":  "Kubernetes 1.27",
					"isActive":     false,
					"minMemory":    4096,
					"minCpuNumber": 2,
				},
			},
		})
	}))
	defer srv.Close()

	svc := kubernetes.NewService(newTestClient(srv))
	versions, err := svc.ListVersions(context.Background(), "zone-1")
	if err != nil {
		t.Fatalf("ListVersions() error = %v", err)
	}
	if len(versions) != 2 {
		t.Fatalf("expected 2 versions, got %d", len(versions))
	}
	if versions[0].UUID != "ver-1" {
		t.Errorf("versions[0].UUID = %q, want %q", versions[0].UUID, "ver-1")
	}
	if versions[0].Name != "1.28" {
		t.Errorf("versions[0].Name = %q, want %q", versions[0].Name, "1.28")
	}
	if !versions[0].IsActive {
		t.Errorf("versions[0].IsActive = false, want true")
	}
	if versions[1].IsActive {
		t.Errorf("versions[1].IsActive = true, want false")
	}
}
