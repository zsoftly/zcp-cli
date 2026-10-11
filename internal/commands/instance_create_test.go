package commands

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func instanceCreateArgs(apiURL string) []string {
	return []string{
		"--name", "new-vm",
		"--cloud-provider", "test-cloud",
		"--project", "test-project",
		"--region", "test-region",
		"--template", "ubuntu",
		"--billing-cycle", "hourly",
		"--storage-category", "premium-ssd",
		"--network-plan", "network-plan",
		"--api-url", apiURL,
	}
}

func TestInstanceCreateRequiresPlan(t *testing.T) {
	_, _, err := execCmd(t, newInstanceCreateCmd(), instanceCreateArgs("http://example.invalid")...)
	if err == nil {
		t.Fatal("expected missing-plan error")
	}
	if !strings.Contains(err.Error(), "--plan is required") {
		t.Fatalf("error = %q, want missing-plan error", err)
	}
}

func TestInstanceCreateRejectsRetiredSizingFlags(t *testing.T) {
	for _, flag := range []string{"--cpu", "--memory", "--disk"} {
		t.Run(flag, func(t *testing.T) {
			args := append(instanceCreateArgs("http://example.invalid"), "--plan", "ca2sxs", flag, "2")
			_, _, err := execCmd(t, newInstanceCreateCmd(), args...)
			if err == nil {
				t.Fatalf("expected %s to be rejected", flag)
			}
			if !strings.Contains(err.Error(), "unknown flag") || !strings.Contains(err.Error(), flag) {
				t.Fatalf("error = %q, want unknown %s flag", err, flag)
			}
		})
	}
}

func TestInstanceCreateSendsNamedPlansAndRootDiskCapacity(t *testing.T) {
	t.Setenv("ZCP_BEARER_TOKEN", "test-token")

	var request map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/virtual-machines" {
			http.NotFound(w, r)
			return
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			http.Error(w, "invalid request", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"Success","data":{"slug":"new-vm","name":"new-vm","state":"Creating"}}`))
	}))
	defer srv.Close()

	args := append(instanceCreateArgs(srv.URL), "--plan", "ca2sxs", "--blockstorage-plan", "b2g1", "--root-disk-size", "100")
	_, _, err := execCmd(t, newInstanceCreateCmd(), args...)
	if err != nil {
		t.Fatalf("instance create: %v", err)
	}
	if got := request["plan"]; got != "ca2sxs" {
		t.Errorf("plan = %#v, want %q", got, "ca2sxs")
	}
	if got, ok := request["custom_plan"]; !ok || got != nil {
		t.Errorf("custom_plan = %#v, want null", got)
	}
	if got := request["blockstorage_plan"]; got != "b2g1" {
		t.Errorf("blockstorage_plan = %#v, want %q", got, "b2g1")
	}
	rootDisk, ok := request["blockstorage_custom_plan"].(map[string]any)
	if !ok {
		t.Fatalf("blockstorage_custom_plan = %#v, want object", request["blockstorage_custom_plan"])
	}
	if got := rootDisk["storage"]; got != float64(100) {
		t.Errorf("blockstorage_custom_plan.storage = %#v, want %d", got, 100)
	}
}

func TestInstanceCreateRequiresNamedRootStorageAndCapacity(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{name: "both missing", want: "--blockstorage-plan and --root-disk-size are required"},
		{name: "tier only", args: []string{"--blockstorage-plan", "b2g1"}, want: "--root-disk-size is required"},
		{name: "capacity only", args: []string{"--root-disk-size", "100"}, want: "--blockstorage-plan is required"},
		{name: "zero capacity", args: []string{"--blockstorage-plan", "b2g1", "--root-disk-size", "0"}, want: "--root-disk-size must be > 0"},
		{name: "unnamed tier", args: []string{"--blockstorage-plan", "custom_plan", "--root-disk-size", "100"}, want: "must name a storage tier"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := append(instanceCreateArgs("http://example.invalid"), "--plan", "ca2sxs")
			args = append(args, tt.args...)
			_, _, err := execCmd(t, newInstanceCreateCmd(), args...)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error = %v, want %q", err, tt.want)
			}
		})
	}
}
