package permission_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/zsoftly/zcp-cli/pkg/api/permission"
	"github.com/zsoftly/zcp-cli/pkg/httpclient"
)

func newClient(baseURL string) *httpclient.Client {
	return httpclient.New(httpclient.Options{
		BaseURL:     baseURL,
		BearerToken: "test-token",
		Timeout:     5 * time.Second,
	})
}

func TestList(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/permissions" {
			t.Errorf("path = %q", r.URL.Path)
		}
		perms := []permission.Permission{
			{ID: "p1", Name: "Virtual Machine Read", Slug: "virtual-machine-read", Category: "Virtual Machine", Status: true},
			{ID: "p2", Name: "DNS Manage", Slug: "dns-manage", Category: "DNS", Status: true},
		}
		data, _ := json.Marshal(perms)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "Success", "data": json.RawMessage(data),
		})
	}))
	defer srv.Close()

	svc := permission.NewService(newClient(srv.URL))
	perms, err := svc.List(context.Background())
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(perms) != 2 {
		t.Fatalf("got %d permissions, want 2", len(perms))
	}
	if perms[0].Slug != "virtual-machine-read" || perms[0].Category != "Virtual Machine" {
		t.Errorf("perms[0] = %+v", perms[0])
	}
}
