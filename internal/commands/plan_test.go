package commands

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestPlanVMDisplaysTag(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/plans/service/Virtual Machine" {
			http.NotFound(w, r)
			return
		}
		fmt.Fprint(w, `{"status":"Success","data":[{"id":"plan-1","slug":"compute-small","name":"Compute Small","attribute":{"formatted_cpu":1,"formatted_memory":"1 GB"},"tag":{"tag":"Recommended"},"status":true}]}`)
	}))
	defer srv.Close()

	t.Setenv("ZCP_BEARER_TOKEN", "test-token")
	root := newTestRoot()
	root.PersistentFlags().String("region", "", "")
	root.AddCommand(newPlanVMCmd())
	var stdout bytes.Buffer
	root.SetOut(&stdout)
	root.SetArgs([]string{"vm", "--region", "test-region", "--api-url", srv.URL, "--output", "json"})
	if err := root.Execute(); err != nil {
		t.Fatalf("plan vm error = %v", err)
	}

	output := stdout.String()

	var rows []map[string]string
	if err := json.Unmarshal([]byte(output), &rows); err != nil {
		t.Fatalf("decode output: %v\noutput: %s", err, output)
	}
	if len(rows) != 1 {
		t.Fatalf("rows = %d, want 1", len(rows))
	}
	if got := rows[0]["tag"]; got != "Recommended" {
		t.Errorf("tag = %q, want %q", got, "Recommended")
	}
}
