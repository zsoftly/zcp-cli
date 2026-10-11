package commands

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/zsoftly/zcp-cli/pkg/api/kubernetes"
)

func TestK8sWorkerNodeCount(t *testing.T) {
	tests := []struct {
		name    string
		cluster kubernetes.Cluster
		want    int
	}{
		{
			name:    "worker node size",
			cluster: kubernetes.Cluster{NodeSize: 3, WorkerNodeSize: 4},
			want:    4,
		},
		{
			name:    "legacy node size",
			cluster: kubernetes.Cluster{NodeSize: 3},
			want:    3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := k8sWorkerNodeCount(tt.cluster); got != tt.want {
				t.Errorf("k8sWorkerNodeCount() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestK8sScaleComplete(t *testing.T) {
	tests := []struct {
		name                string
		cluster             kubernetes.Cluster
		workers             int
		autoscalingDisabled bool
		want                bool
	}{
		{
			name:    "matching worker count",
			cluster: kubernetes.Cluster{State: "Running", WorkerNodeSize: 3},
			workers: 3,
			want:    true,
		},
		{
			name:    "running before worker count updates",
			cluster: kubernetes.Cluster{State: "Running", WorkerNodeSize: 2},
			workers: 3,
			want:    false,
		},
		{
			name:    "platform worker count",
			cluster: kubernetes.Cluster{State: "Running", NodeSize: 1, Meta: &kubernetes.ClusterMeta{Size: "3"}},
			workers: 3,
			want:    true,
		},
		{
			name:                "autoscaling still enabled",
			cluster:             kubernetes.Cluster{State: "Running", WorkerNodeSize: 3, Meta: &kubernetes.ClusterMeta{AutoscalingEnabled: json.RawMessage("true")}},
			workers:             3,
			autoscalingDisabled: true,
			want:                false,
		},
		{
			name:                "autoscaling disabled",
			cluster:             kubernetes.Cluster{State: "Running", WorkerNodeSize: 3, Meta: &kubernetes.ClusterMeta{AutoscalingEnabled: json.RawMessage("false")}},
			workers:             3,
			autoscalingDisabled: true,
			want:                true,
		},
		{
			name:                "autoscaling relation still present",
			cluster:             kubernetes.Cluster{State: "Running", WorkerNodeSize: 3, Autoscale: &kubernetes.Autoscale{}},
			workers:             3,
			autoscalingDisabled: true,
			want:                false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := k8sScaleComplete(&tt.cluster, tt.workers, tt.autoscalingDisabled); got != tt.want {
				t.Errorf("k8sScaleComplete() = %t, want %t", got, tt.want)
			}
		})
	}
}

func TestK8sCreateSendsSeparateResourcePlans(t *testing.T) {
	t.Setenv("ZCP_BEARER_TOKEN", "test-token")

	var (
		request    map[string]interface{}
		requestErr error
		requestMu  sync.Mutex
	)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/kubernetes-clusters" {
			http.NotFound(w, r)
			return
		}
		decoded := map[string]interface{}{}
		err := json.NewDecoder(r.Body).Decode(&decoded)
		requestMu.Lock()
		request, requestErr = decoded, err
		requestMu.Unlock()
		if err != nil {
			http.Error(w, "invalid request", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"Success","data":{"slug":"new-cluster","name":"new-cluster","state":"Creating","version":"v1.36.1","worker_node_size":3,"control_nodes":1}}`))
	}))
	defer srv.Close()

	_, _, err := execCmd(t, NewKubernetesCmd(),
		"create",
		"--name", "new-cluster",
		"--version", "v1.36.1",
		"--control-plane-plan", "k8s-cpi-yul",
		"--worker-plan", "k8s-li-yul",
		"--storage-plan", "b2g1",
		"--root-disk-size", "100",
		"--storage-category", "pro-nvme",
		"--cloud-provider", "nimbo",
		"--region", "yul-1",
		"--project", "test",
		"--billing-cycle", "hourly",
		"--workers", "3",
		"--enable-csi",
		"--ssh-key", "test-key",
		"--api-url", srv.URL,
	)
	if err != nil {
		t.Fatalf("kubernetes create: %v", err)
	}
	requestMu.Lock()
	defer requestMu.Unlock()
	if requestErr != nil {
		t.Fatalf("decode request: %v", requestErr)
	}

	for field, want := range map[string]interface{}{
		"master_plan":               "k8s-cpi-yul",
		"worker_plan":               "k8s-li-yul",
		"blockstorage_plan":         "b2g1",
		"billing_cycle":             "hourly",
		"node_size":                 float64(3),
		"worker_node_size":          float64(3),
		"control_nodes":             float64(1),
		"enable_csi":                true,
		"is_k8s_custom_plan":        false,
		"is_k8s_master_custom_plan": false,
		"is_k8s_worker_custom_plan": false,
	} {
		if got := request[field]; got != want {
			t.Errorf("%s = %#v, want %#v", field, got, want)
		}
	}
	if _, ok := request["plan"]; ok {
		t.Error("legacy plan must be omitted from a modern Kubernetes create request")
	}
	rootDisk, ok := request["blockstorage_custom_plan"].(map[string]interface{})
	if !ok || rootDisk["storage"] != float64(100) {
		t.Errorf("blockstorage_custom_plan = %#v, want storage=100", request["blockstorage_custom_plan"])
	}
	for _, field := range []string{"master_custom_plan", "worker_custom_plan"} {
		if value, ok := request[field]; !ok || value != nil {
			t.Errorf("%s = %#v, want null", field, value)
		}
	}
}

func TestK8sGetShowsSafeResourceOverview(t *testing.T) {
	t.Setenv("ZCP_BEARER_TOKEN", "test-token")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/kubernetes-clusters/cluster-1" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"status":"Success",
			"data":{
				"slug":"cluster-1","name":"cluster-1","state":"Running","node_size":2,"worker_node_size":2,"control_nodes":1,
				"meta":{"size":"2","control_nodes":"1","cpu_number":"6","memory":"12288","autoscaling_enabled":true,"min_size":1,"max_size":4,"kubernetes_version_name":"Kubernetes 1.37.0","config":{"configdata":"must-not-print"}},
				"offering":{"master_plan":{"name":"control","attribute":{"formatted_cpu":4,"cpu":4,"formatted_memory":"8 GB","formatted_storage":"0 GB"}},"worker_plan":null,"master_custom_cpu":null,"master_custom_memory":null,"master_custom_storage":null,"worker_custom_cpu":null,"worker_custom_memory":null,"worker_custom_storage":null,"cpu":2,"memory":4096,"storage":0,"formatted_memory":"4 GB","formatted_storage":"0 GB"},
				"blockstorages":[{"name":"root-1","is_root":true,"size":"100"},{"name":"root-2","is_root":true,"size":50},{"name":"root-3","is_root":true},{"name":"data-1","is_root":false,"size":500}],
				"network":{"name":"cluster-network","slug":"cluster-network"}
			}
		}`))
	}))
	defer srv.Close()

	stdout, _, err := execCmd(t, NewKubernetesCmd(), "get", "cluster-1", "--api-url", srv.URL)
	if err != nil {
		t.Fatalf("kubernetes get: %v", err)
	}
	for _, want := range []string{
		"Kubernetes 1.37.0", "Total CPU", "6", "Total RAM", "12 GB",
		"Autoscaling", "Enabled", "Minimum workers", "Maximum workers",
		"Custom", "2 Cores", "4 Cores", "4 GB", "Control-plane plan storage", "0 GB",
		"Root volume root-1", "100 GB", "Root volume root-2", "50 GB", "Root volume root-3", "Root volumes", "3", "cluster-network",
	} {
		if !strings.Contains(stdout, want) {
			t.Errorf("output missing %q:\n%s", want, stdout)
		}
	}
	if strings.Contains(stdout, "must-not-print") {
		t.Errorf("output leaks kubeconfig: %q", stdout)
	}
	if strings.Count(stdout, "Root volume ") != 3 {
		t.Errorf("root volume rows = %d, want 3:\n%s", strings.Count(stdout, "Root volume "), stdout)
	}
	missingSize := false
	for _, line := range strings.Split(stdout, "\n") {
		if strings.Join(strings.Fields(line), " ") == "Root volume root-3 -" {
			missingSize = true
			break
		}
	}
	if !missingSize {
		t.Errorf("missing root-volume capacity must render as -:\n%s", stdout)
	}
}

func TestK8sDeleteSubmitsServiceCancellation(t *testing.T) {
	t.Setenv("ZCP_BEARER_TOKEN", "test-token")
	var getCount, postCount, deleteCount int
	var request map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/kubernetes-clusters/cluster-1":
			getCount++
			_, _ = w.Write([]byte(`{"status":"Success","data":{"slug":"cluster-1","offering":{"billing_cycle":{"unit":"hour","slug":"monthly","name":"Monthly"}}}}`))
		case r.Method == http.MethodPost && r.URL.Path == "/billing/service-cancel-requests/cluster-1":
			postCount++
			if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
				t.Errorf("decode cancellation request: %v", err)
			}
			_, _ = w.Write([]byte(`{"status":"Success"}`))
		case r.Method == http.MethodDelete:
			deleteCount++
			http.Error(w, "direct delete must not be used", http.StatusInternalServerError)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	stdout, _, err := execCmd(t, NewKubernetesCmd(), "delete", "cluster-1", "--yes", "--api-url", srv.URL)
	if err != nil {
		t.Fatalf("kubernetes delete: %v", err)
	}
	if getCount != 1 || postCount != 1 || deleteCount != 0 {
		t.Fatalf("requests GET=%d POST=%d DELETE=%d, want 1, 1, 0", getCount, postCount, deleteCount)
	}
	for field, want := range map[string]any{
		"service_name":     "Kubernetes",
		"reason":           "not_needed_anymore",
		"status":           "Pending",
		"type":             "Immediate",
		"billing_cycle":    "hour",
		"delete_public_ip": true,
	} {
		if got := request[field]; got != want {
			t.Errorf("%s = %#v, want %#v", field, got, want)
		}
	}
	if !strings.Contains(stdout, "Deletion requested") || strings.Contains(stdout, "deleted") {
		t.Errorf("output = %q, want asynchronous deletion request", stdout)
	}
}

func TestK8sDeleteOmitsUnknownBillingCycle(t *testing.T) {
	t.Setenv("ZCP_BEARER_TOKEN", "test-token")
	var postCount, deleteCount int
	var request map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/kubernetes-clusters/cluster-1":
			_, _ = w.Write([]byte(`{"status":"Success","data":{"slug":"cluster-1"}}`))
		case r.Method == http.MethodPost && r.URL.Path == "/billing/service-cancel-requests/cluster-1":
			postCount++
			_ = json.NewDecoder(r.Body).Decode(&request)
			_, _ = w.Write([]byte(`{"status":"Success"}`))
		case r.Method == http.MethodDelete:
			deleteCount++
			http.Error(w, "direct delete must not be used", http.StatusInternalServerError)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	if _, _, err := execCmd(t, NewKubernetesCmd(), "delete", "cluster-1", "--yes", "--api-url", srv.URL); err != nil {
		t.Fatalf("kubernetes delete: %v", err)
	}
	if postCount != 1 || deleteCount != 0 {
		t.Fatalf("requests POST=%d DELETE=%d, want 1, 0", postCount, deleteCount)
	}
	if _, ok := request["billing_cycle"]; ok {
		t.Errorf("billing_cycle = %#v, want omitted", request["billing_cycle"])
	}
}

func TestK8sDeleteStopsWhenGetFails(t *testing.T) {
	t.Setenv("ZCP_BEARER_TOKEN", "test-token")
	var postCount, deleteCount int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			http.Error(w, "unavailable", http.StatusInternalServerError)
			return
		}
		if r.Method == http.MethodPost {
			postCount++
		}
		if r.Method == http.MethodDelete {
			deleteCount++
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()

	if _, _, err := execCmd(t, NewKubernetesCmd(), "delete", "cluster-1", "--yes", "--api-url", srv.URL); err == nil {
		t.Fatal("kubernetes delete succeeded after cluster lookup failed")
	}
	if postCount != 0 || deleteCount != 0 {
		t.Errorf("requests POST=%d DELETE=%d, want 0, 0", postCount, deleteCount)
	}
}

func TestK8sDeleteNotFoundIsIdempotent(t *testing.T) {
	t.Setenv("ZCP_BEARER_TOKEN", "test-token")
	var postCount, deleteCount int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		if r.Method == http.MethodPost {
			postCount++
		}
		if r.Method == http.MethodDelete {
			deleteCount++
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()

	if _, _, err := execCmd(t, NewKubernetesCmd(), "delete", "cluster-1", "--yes", "--api-url", srv.URL); err != nil {
		t.Fatalf("kubernetes delete: %v", err)
	}
	if postCount != 0 || deleteCount != 0 {
		t.Errorf("requests POST=%d DELETE=%d, want 0, 0", postCount, deleteCount)
	}
}

func TestK8sCancelBillingCycle(t *testing.T) {
	tests := []struct {
		name    string
		cluster kubernetes.Cluster
		want    string
	}{
		{name: "offering unit wins", cluster: kubernetes.Cluster{BillingCycle: &kubernetes.BillingCycle{Unit: "month"}, Offering: &kubernetes.Offering{BillingCycle: &kubernetes.BillingCycle{Unit: "hour"}}}, want: "hour"},
		{name: "top level slug", cluster: kubernetes.Cluster{BillingCycle: &kubernetes.BillingCycle{Slug: "monthly"}}, want: "month"},
		{name: "top level name", cluster: kubernetes.Cluster{BillingCycle: &kubernetes.BillingCycle{Name: "Hourly"}}, want: "hour"},
		{name: "unknown omitted", cluster: kubernetes.Cluster{}, want: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := k8sCancelBillingCycle(&tt.cluster); got != tt.want {
				t.Errorf("k8sCancelBillingCycle() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestK8sGetShowsUnknownAutoscalingWhenOmitted(t *testing.T) {
	t.Setenv("ZCP_BEARER_TOKEN", "test-token")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/kubernetes-clusters/cluster-1" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"Success","data":{"slug":"cluster-1","state":"Running","meta":{}}}`))
	}))
	defer srv.Close()

	stdout, _, err := execCmd(t, NewKubernetesCmd(), "get", "cluster-1", "--api-url", srv.URL)
	if err != nil {
		t.Fatalf("kubernetes get: %v", err)
	}
	if !strings.Contains(stdout, "Autoscaling") || !strings.Contains(stdout, "Unknown") {
		t.Errorf("output must show unknown autoscaling status when omitted:\n%s", stdout)
	}
}

func TestK8sAutoscalingStatus(t *testing.T) {
	tests := []struct {
		name string
		raw  json.RawMessage
		want string
	}{
		{name: "omitted", want: "Unknown"},
		{name: "null", raw: json.RawMessage("null"), want: "Unknown"},
		{name: "empty string", raw: json.RawMessage(`""`), want: "Disabled"},
		{name: "enabled", raw: json.RawMessage("true"), want: "Enabled"},
		{name: "disabled", raw: json.RawMessage("false"), want: "Disabled"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := k8sAutoscalingStatus(tt.raw, false); got != tt.want {
				t.Errorf("k8sAutoscalingStatus() = %q, want %q", got, tt.want)
			}
		})
	}
	if got := k8sAutoscalingStatus(nil, true); got != "Enabled" {
		t.Errorf("k8sAutoscalingStatus() = %q with an autoscale relation, want Enabled", got)
	}
}

func TestK8sPlanDetailsDoesNotUseLegacyFallbackForNamedPlan(t *testing.T) {
	name, cpu, memory, storage := k8sPlanDetails(
		&kubernetes.NodePlan{Name: "modern-plan"},
		nil, nil, nil, "", "",
		json.RawMessage("2"), json.RawMessage("4096"), json.RawMessage("100"), "4 GB", "100 GB",
	)
	if name != "modern-plan" || cpu != "-" || memory != "-" || storage != "-" {
		t.Errorf("k8sPlanDetails() = (%q, %q, %q, %q), want modern plan with unavailable attributes", name, cpu, memory, storage)
	}
}

func TestK8sScaleAutoscalingModes(t *testing.T) {
	t.Setenv("ZCP_BEARER_TOKEN", "test-token")
	for _, tt := range []struct {
		name string
		args []string
		want map[string]interface{}
	}{
		{
			name: "enable",
			args: []string{"--enable-autoscaling", "--min-workers", "2", "--max-workers", "5"},
			want: map[string]interface{}{"autoscale": float64(1), "min_cluster_size": float64(2), "max_cluster_size": float64(5)},
		},
		{
			name: "disable",
			args: []string{"--disable-autoscaling", "--workers", "3"},
			want: map[string]interface{}{"autoscale": float64(0), "node_size": float64(3)},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var (
				body map[string]interface{}
				mu   sync.Mutex
			)
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				switch {
				case r.Method == http.MethodGet && r.URL.Path == "/kubernetes-clusters/cluster-1":
					_, _ = w.Write([]byte(`{"status":"Success","data":{"slug":"cluster-1","state":"Running","node_size":2}}`))
				case r.Method == http.MethodPut && r.URL.Path == "/kubernetes-clusters/cluster-1/scale":
					decoded := map[string]interface{}{}
					if err := json.NewDecoder(r.Body).Decode(&decoded); err != nil {
						t.Errorf("decode request: %v", err)
					}
					mu.Lock()
					body = decoded
					mu.Unlock()
					_, _ = w.Write([]byte(`{"status":"Success"}`))
				default:
					http.NotFound(w, r)
				}
			}))
			defer srv.Close()

			args := append([]string{"scale", "cluster-1", "--api-url", srv.URL}, tt.args...)
			if _, _, err := execCmd(t, NewKubernetesCmd(), args...); err != nil {
				t.Fatalf("kubernetes scale: %v", err)
			}
			mu.Lock()
			defer mu.Unlock()
			for field, want := range tt.want {
				if got := body[field]; got != want {
					t.Errorf("%s = %#v, want %#v", field, got, want)
				}
			}
			if tt.name == "enable" {
				if _, ok := body["node_size"]; ok {
					t.Errorf("enable payload includes node_size = %#v", body["node_size"])
				}
			}
		})
	}
}

func TestK8sCreateRequiresSeparateResourcePlans(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{
			name: "control plane",
			args: []string{"--worker-plan", "k8s-worker-1", "--storage-plan", "k8s-root-volume-1"},
			want: "--control-plane-plan is required",
		},
		{
			name: "worker",
			args: []string{"--control-plane-plan", "k8s-control-1", "--storage-plan", "k8s-root-volume-1"},
			want: "--worker-plan is required",
		},
		{
			name: "storage",
			args: []string{"--control-plane-plan", "k8s-control-1", "--worker-plan", "k8s-worker-1"},
			want: "--storage-plan is required",
		},
		{
			name: "high availability control nodes",
			args: []string{"--control-plane-plan", "k8s-control-1", "--worker-plan", "k8s-worker-1", "--storage-plan", "k8s-root-volume-1", "--ha"},
			want: "--control-nodes must be >= 2 when --ha is set",
		},
	}

	baseArgs := []string{
		"kubernetes", "create",
		"--name", "test-cluster",
		"--version", "v1.36.1",
		"--cloud-provider", "nimbo",
		"--region", "yul-1",
		"--project", "test",
		"--billing-cycle", "hourly",
		"--workers", "1",
		"--root-disk-size", "100",
		"--storage-category", "pro-nvme",
		"--ssh-key", "mykey",
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := newTestRoot()
			root.AddCommand(NewKubernetesCmd())
			root.SetOut(&bytes.Buffer{})
			root.SetErr(&bytes.Buffer{})
			root.SetArgs(append(baseArgs, tt.args...))

			err := root.Execute()
			if err == nil {
				t.Fatal("expected validation error")
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Errorf("error = %q, want %q", err, tt.want)
			}
		})
	}
}

func TestK8sCreateRequiresNamedRootStorageAndCapacity(t *testing.T) {
	baseArgs := []string{
		"kubernetes", "create",
		"--name", "test-cluster",
		"--version", "v1.36.1",
		"--control-plane-plan", "k8s-control-1",
		"--worker-plan", "k8s-worker-1",
		"--cloud-provider", "nimbo",
		"--region", "yul-1",
		"--project", "test",
		"--billing-cycle", "hourly",
		"--workers", "1",
		"--storage-category", "pro-nvme",
		"--ssh-key", "mykey",
	}
	tests := []struct {
		name string
		args []string
		want string
	}{
		{name: "missing capacity", args: []string{"--storage-plan", "b2g1"}, want: "--root-disk-size is required"},
		{name: "zero capacity", args: []string{"--storage-plan", "b2g1", "--root-disk-size", "0"}, want: "--root-disk-size must be > 0"},
		{name: "negative capacity", args: []string{"--storage-plan", "b2g1", "--root-disk-size", "-1"}, want: "--root-disk-size must be > 0"},
		{name: "unnamed tier", args: []string{"--storage-plan", "custom_plan", "--root-disk-size", "100"}, want: "must name a storage tier"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := newTestRoot()
			root.AddCommand(NewKubernetesCmd())
			root.SetOut(&bytes.Buffer{})
			root.SetErr(&bytes.Buffer{})
			root.SetArgs(append(baseArgs, tt.args...))

			err := root.Execute()
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error = %v, want %q", err, tt.want)
			}
		})
	}
}
