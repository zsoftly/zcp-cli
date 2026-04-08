package monitoring_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/zsoftly/zcp-cli/internal/api/monitoring"
	"github.com/zsoftly/zcp-cli/internal/httpclient"
)

func newTestClient(t *testing.T, srv *httptest.Server) *httpclient.Client {
	t.Helper()
	return httpclient.New(httpclient.Options{
		BaseURL:     srv.URL,
		BearerToken: "test-token",
		Timeout:     5 * time.Second,
	})
}

// wrapEnvelope returns a STKCNSL-style {"status":"Success","data":...} JSON body.
func wrapEnvelope(t *testing.T, data interface{}) []byte {
	t.Helper()
	d, err := json.Marshal(data)
	if err != nil {
		t.Fatalf("marshal data: %v", err)
	}
	env := map[string]interface{}{
		"status": "Success",
		"data":   json.RawMessage(d),
	}
	b, err := json.Marshal(env)
	if err != nil {
		t.Fatalf("marshal envelope: %v", err)
	}
	return b
}

func TestGlobal(t *testing.T) {
	resources := []monitoring.GlobalResource{
		{Name: "CPU", Total: 64, Used: 32, Free: 32, Unit: "cores", Percentage: 50},
		{Name: "Memory", Total: 128, Used: 96, Free: 32, Unit: "GB", Percentage: 75},
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/monitoring/global" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(wrapEnvelope(t, resources))
	}))
	defer srv.Close()

	svc := monitoring.NewService(newTestClient(t, srv))
	result, err := svc.Global(context.Background())
	if err != nil {
		t.Fatalf("Global() error = %v", err)
	}
	if len(result) != 2 {
		t.Fatalf("Global() returned %d resources, want 2", len(result))
	}
	if result[0].Name != "CPU" {
		t.Errorf("result[0].Name = %q, want %q", result[0].Name, "CPU")
	}
	if result[1].Percentage != 75 {
		t.Errorf("result[1].Percentage = %v, want 75", result[1].Percentage)
	}
}

func TestCPUUsage(t *testing.T) {
	points := []monitoring.MetricPoint{
		{Timestamp: "2026-04-06T10:00:00Z", Value: 45.2, Unit: "%"},
		{Timestamp: "2026-04-06T10:05:00Z", Value: 52.1, Unit: "%"},
	}

	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		w.Write(wrapEnvelope(t, points))
	}))
	defer srv.Close()

	svc := monitoring.NewService(newTestClient(t, srv))
	result, err := svc.CPUUsage(context.Background(), "my-vm-slug")
	if err != nil {
		t.Fatalf("CPUUsage() error = %v", err)
	}
	if gotPath != "/monitoring/my-vm-slug/cpu-usage" {
		t.Errorf("path = %q, want %q", gotPath, "/monitoring/my-vm-slug/cpu-usage")
	}
	if len(result) != 2 {
		t.Fatalf("CPUUsage() returned %d points, want 2", len(result))
	}
	if result[0].Value != 45.2 {
		t.Errorf("result[0].Value = %v, want 45.2", result[0].Value)
	}
}

func TestMemoryUsage(t *testing.T) {
	points := []monitoring.MetricPoint{
		{Timestamp: "2026-04-06T10:00:00Z", Value: 78.5, Unit: "%"},
	}

	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		w.Write(wrapEnvelope(t, points))
	}))
	defer srv.Close()

	svc := monitoring.NewService(newTestClient(t, srv))
	result, err := svc.MemoryUsage(context.Background(), "vm-1")
	if err != nil {
		t.Fatalf("MemoryUsage() error = %v", err)
	}
	if gotPath != "/monitoring/vm-1/memory-usage" {
		t.Errorf("path = %q, want %q", gotPath, "/monitoring/vm-1/memory-usage")
	}
	if len(result) != 1 {
		t.Fatalf("MemoryUsage() returned %d points, want 1", len(result))
	}
	if result[0].Value != 78.5 {
		t.Errorf("result[0].Value = %v, want 78.5", result[0].Value)
	}
}

func TestDiskReadWrite(t *testing.T) {
	points := []monitoring.DiskMetricPoint{
		{Timestamp: "2026-04-06T10:00:00Z", Read: 1024, Write: 512, Unit: "KB/s"},
	}

	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		w.Write(wrapEnvelope(t, points))
	}))
	defer srv.Close()

	svc := monitoring.NewService(newTestClient(t, srv))
	result, err := svc.DiskReadWrite(context.Background(), "vm-2")
	if err != nil {
		t.Fatalf("DiskReadWrite() error = %v", err)
	}
	if gotPath != "/monitoring/vm-2/disk-read-write" {
		t.Errorf("path = %q, want %q", gotPath, "/monitoring/vm-2/disk-read-write")
	}
	if len(result) != 1 {
		t.Fatalf("DiskReadWrite() returned %d points, want 1", len(result))
	}
	if result[0].Read != 1024 {
		t.Errorf("result[0].Read = %v, want 1024", result[0].Read)
	}
	if result[0].Write != 512 {
		t.Errorf("result[0].Write = %v, want 512", result[0].Write)
	}
}

func TestDiskIOReadWrite(t *testing.T) {
	points := []monitoring.DiskMetricPoint{
		{Timestamp: "2026-04-06T10:00:00Z", Read: 200, Write: 100, Unit: "IOPS"},
	}

	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		w.Write(wrapEnvelope(t, points))
	}))
	defer srv.Close()

	svc := monitoring.NewService(newTestClient(t, srv))
	result, err := svc.DiskIOReadWrite(context.Background(), "vm-3")
	if err != nil {
		t.Fatalf("DiskIOReadWrite() error = %v", err)
	}
	if gotPath != "/monitoring/vm-3/disk-io-read-write" {
		t.Errorf("path = %q, want %q", gotPath, "/monitoring/vm-3/disk-io-read-write")
	}
	if len(result) != 1 {
		t.Fatalf("DiskIOReadWrite() returned %d points, want 1", len(result))
	}
	if result[0].Read != 200 {
		t.Errorf("result[0].Read = %v, want 200", result[0].Read)
	}
}

func TestNetworkTraffic(t *testing.T) {
	points := []monitoring.NetworkMetricPoint{
		{Timestamp: "2026-04-06T10:00:00Z", Incoming: 5000, Outgoing: 3000, Unit: "KB/s"},
	}

	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		w.Write(wrapEnvelope(t, points))
	}))
	defer srv.Close()

	svc := monitoring.NewService(newTestClient(t, srv))
	result, err := svc.NetworkTraffic(context.Background(), "vm-4")
	if err != nil {
		t.Fatalf("NetworkTraffic() error = %v", err)
	}
	if gotPath != "/monitoring/vm-4/network-traffic" {
		t.Errorf("path = %q, want %q", gotPath, "/monitoring/vm-4/network-traffic")
	}
	if len(result) != 1 {
		t.Fatalf("NetworkTraffic() returned %d points, want 1", len(result))
	}
	if result[0].Incoming != 5000 {
		t.Errorf("result[0].Incoming = %v, want 5000", result[0].Incoming)
	}
	if result[0].Outgoing != 3000 {
		t.Errorf("result[0].Outgoing = %v, want 3000", result[0].Outgoing)
	}
}

func TestCharts(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/monitoring/charts" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(wrapEnvelope(t, map[string]string{"chart": "data"}))
	}))
	defer srv.Close()

	svc := monitoring.NewService(newTestClient(t, srv))
	result, err := svc.Charts(context.Background())
	if err != nil {
		t.Fatalf("Charts() error = %v", err)
	}
	if len(result) == 0 {
		t.Error("Charts() returned empty result")
	}
	var v interface{}
	if err := json.Unmarshal(result, &v); err != nil {
		t.Errorf("Charts() result is not valid JSON: %v", err)
	}
}

func TestGlobalAPIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"listErrorResponse": map[string]string{
				"errorCode": "UNAUTHORIZED",
				"errorMsg":  "Invalid API key",
			},
		})
	}))
	defer srv.Close()

	svc := monitoring.NewService(newTestClient(t, srv))
	_, err := svc.Global(context.Background())
	if err == nil {
		t.Fatal("Global() expected error for 401, got nil")
	}
}

func TestGlobalNonSuccessStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "Error",
			"data":   nil,
		})
	}))
	defer srv.Close()

	svc := monitoring.NewService(newTestClient(t, srv))
	_, err := svc.Global(context.Background())
	if err == nil {
		t.Fatal("Global() expected error for non-Success status, got nil")
	}
}
