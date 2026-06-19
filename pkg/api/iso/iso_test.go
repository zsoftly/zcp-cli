package iso_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/zsoftly/zcp-cli/pkg/api/iso"
	"github.com/zsoftly/zcp-cli/pkg/httpclient"
)

func newClient(baseURL string) *httpclient.Client {
	return httpclient.New(httpclient.Options{
		BaseURL:     baseURL,
		BearerToken: "test-token",
		Timeout:     5 * time.Second,
	})
}

// TestISOListBoolStatus verifies that a list response with status as a boolean
// (the real API shape) decodes without error.
func TestISOListBoolStatus(t *testing.T) {
	payload := `{"status":"Success","message":"OK","current_page":1,"total":1,"data":[{"id":"iso-1","name":"ubuntu-22.iso","slug":"ubuntu-22-iso","status":true,"state":"Ready","image_type":"Operating System","is_bootable":true,"is_extractable":false,"password_enabled":false,"created_at":"2026-01-01T00:00:00.000000Z","updated_at":"2026-01-01T00:00:00.000000Z"}]}`

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(payload))
	}))
	defer srv.Close()

	svc := iso.NewService(newClient(srv.URL))
	isos, err := svc.List(context.Background(), "")
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(isos) != 1 {
		t.Fatalf("got %d isos, want 1", len(isos))
	}
	if !isos[0].Status {
		t.Errorf("Status = false, want true")
	}
	if isos[0].Name != "ubuntu-22.iso" {
		t.Errorf("Name = %q, want %q", isos[0].Name, "ubuntu-22.iso")
	}
}
