package commands

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/zsoftly/zcp-cli/pkg/api/dns"
	"gopkg.in/yaml.v3"
)

func TestDNSRecordCreatePrintsOnlyCreatedValue(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/dns/domains/example-com-1/records" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"status":"Success","data":{"name":"example.com","records":[{"name":"other.example.com.","type":"A","ttl":300,"contents":["192.0.2.9"]},{"name":"www.example.com.","type":"A","ttl":3600,"contents":["192.0.2.1","192.0.2.2"]}]}}`)
	}))
	defer srv.Close()
	t.Setenv("ZCP_BEARER_TOKEN", "test-token")

	out, _, err := execCmd(t, NewDNSCmd(), "record-create", "--domain", "example-com-1", "--name", "www", "--type", "A", "--content", "192.0.2.2", "--api-url", srv.URL, "--output", "json")
	if err != nil {
		t.Fatalf("record-create error = %v", err)
	}
	var rows []map[string]string
	if err := json.Unmarshal([]byte(out), &rows); err != nil {
		t.Fatalf("JSON output = %q: %v", out, err)
	}
	if len(rows) != 1 || rows[0]["content"] != "192.0.2.2" || rows[0]["name"] != "www.example.com." {
		t.Errorf("rows = %#v, want only created value", rows)
	}
	if rows[0]["priority"] != "-" {
		t.Errorf("priority = %q, want '-'", rows[0]["priority"])
	}
	if _, ok := rows[0]["source"]; ok {
		t.Errorf("source = %q, want absent for created record", rows[0]["source"])
	}
}

func TestDNSRecordCreateNormalizesTXTAndMXOutput(t *testing.T) {
	tests := []struct {
		name string
		args []string
		body string
		want map[string]string
	}{
		{
			name: "TXT quoted response and request",
			args: []string{"--name", "txt", "--type", "TXT", "--content", `"hello"`},
			body: `{"name":"example.com","records":[{"name":"txt.example.com.","type":"TXT","ttl":3600,"contents":["\"hello\""]}]}`,
			want: map[string]string{"content": `"hello"`, "priority": "-"},
		},
		{
			name: "MX zero priority",
			args: []string{"--name", "@", "--type", "MX", "--content", "mail.example.com.", "--priority", "0"},
			body: `{"name":"example.com","records":[{"name":"example.com.","type":"MX","ttl":3600,"contents":["0 MAIL.EXAMPLE.COM."]}]}`,
			want: map[string]string{"content": "MAIL.EXAMPLE.COM.", "priority": "0"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPost {
					http.NotFound(w, r)
					return
				}
				w.Header().Set("Content-Type", "application/json")
				fmt.Fprintf(w, `{"status":"Success","data":%s}`, tt.body)
			}))
			defer srv.Close()
			t.Setenv("ZCP_BEARER_TOKEN", "test-token")

			args := append([]string{"record-create", "--domain", "example-com-1"}, tt.args...)
			args = append(args, "--api-url", srv.URL, "--output", "yaml")
			out, _, err := execCmd(t, NewDNSCmd(), args...)
			if err != nil {
				t.Fatalf("record-create error = %v", err)
			}
			var rows []map[string]string
			if err := yaml.Unmarshal([]byte(out), &rows); err != nil {
				t.Fatalf("YAML output = %q: %v", out, err)
			}
			if len(rows) != 1 || rows[0]["content"] != tt.want["content"] || rows[0]["priority"] != tt.want["priority"] {
				t.Errorf("rows = %#v, want %#v", rows, tt.want)
				return
			}
			if _, hasSource := rows[0]["source"]; hasSource {
				t.Errorf("rows = %#v, want no source for created record", rows)
			}
		})
	}
}

func TestDNSRecordMatchesRequestNormalizesDomainNamesAndAddresses(t *testing.T) {
	tests := []struct {
		name    string
		record  string
		req     string
		recordT string
	}{
		{"CNAME trailing dot and case", "Target.Example.COM.", "target.example.com", "CNAME"},
		{"IPv6 canonical form", "2001:0db8:0000:0000:0000:0000:0000:0001", "2001:db8::1", "AAAA"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !dnsRecordMatchesRequest(dnsRecord(tt.recordT), tt.record, dnsRequest(tt.recordT, tt.req)) {
				t.Errorf("dnsRecordMatchesRequest(%q, %q) = false, want true", tt.record, tt.req)
			}
		})
	}
}

func TestDNSRecordMatchesRequestDecodesQuotedTXT(t *testing.T) {
	tests := []struct {
		name     string
		actual   string
		expected string
		match    bool
	}{
		{"escaped quotes", `"This \"is\" valid"`, `This "is" valid`, true},
		{"escaped backslash", `"This is also \\ valid"`, `This is also \ valid`, true},
		{"decimal UTF-8", `"\195\167"`, "ç", true},
		{"quoted chunks", `"one" "two"`, "onetwo", true},
		{"whitespace outside quoted chunks", "  \t\"one\" \n \"two\"  ", "onetwo", true},
		{"comma", `"one,two"`, "one,two", true},
		{"whitespace inside quote", `" leading and trailing "`, " leading and trailing ", true},
		{"unquoted whitespace preserved", " leading and trailing ", " leading and trailing ", true},
		{"different values do not match", `"one"`, "two", false},
		{"invalid quoted value remains raw", `"one`, "one", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := dnsRecordMatchesRequest(dnsRecord("TXT"), tt.actual, dnsRequest("TXT", tt.expected))
			if got != tt.match {
				t.Errorf("dnsRecordMatchesRequest(%q, %q) = %v, want %v", tt.actual, tt.expected, got, tt.match)
			}
		})
	}
}

func TestDNSRecordMatchesRequestNormalizesCAA(t *testing.T) {
	tests := []struct {
		name     string
		actual   string
		expected string
		match    bool
	}{
		{"quoted API value, unquoted request", `0 ISSUE "letsencrypt.org"`, "000 issue letsencrypt.org", true},
		{"value preserves internal spaces", `128 iodef "mailto:security@example.com urgent"`, `128 IODEF "mailto:security@example.com urgent"`, true},
		{"different flag", `128 issue "letsencrypt.org"`, "0 issue letsencrypt.org", false},
		{"different tag", `0 iodef "letsencrypt.org"`, "0 issue letsencrypt.org", false},
		{"different value case", `0 issue "LetsEncrypt.org"`, "0 issue letsencrypt.org", false},
		{"different value", `0 issue "other.example"`, "0 issue letsencrypt.org", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := dnsRecordMatchesRequest(dnsRecord("CAA"), tt.actual, dnsRequest("CAA", tt.expected))
			if got != tt.match {
				t.Errorf("dnsRecordMatchesRequest(%q, %q) = %v, want %v", tt.actual, tt.expected, got, tt.match)
			}
		})
	}
}

func TestDNSRecordCreateNormalizesCAAOutput(t *testing.T) {
	for _, format := range []string{"table", "json", "yaml"} {
		t.Run(format, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPost {
					http.NotFound(w, r)
					return
				}
				w.Header().Set("Content-Type", "application/json")
				fmt.Fprint(w, `{"status":"Success","data":{"name":"example.com","records":[{"name":"caa.example.com.","type":"CAA","ttl":3600,"contents":["128 issue \"letsencrypt.org\"","0 iodef \"letsencrypt.org\"","0 issue \"LetsEncrypt.org\"","0 issue \"other.example\"","0 ISSUE \"letsencrypt.org\""]}]}}`)
			}))
			defer srv.Close()
			t.Setenv("ZCP_BEARER_TOKEN", "test-token")

			out, _, err := execCmd(t, NewDNSCmd(), "record-create", "--domain", "example-com-1", "--name", "caa", "--type", "CAA", "--content", `000 issue "letsencrypt.org"`, "--api-url", srv.URL, "--output", format)
			if err != nil {
				t.Fatalf("record-create error = %v", err)
			}
			const want = `0 ISSUE "letsencrypt.org"`
			switch format {
			case "table":
				if !strings.Contains(out, want) || strings.Contains(out, "other.example") || strings.Contains(out, "LetsEncrypt.org") || strings.Contains(out, "Pre-delete snapshot") {
					t.Errorf("table output = %q, want only %q", out, want)
				}
			case "json":
				var rows []map[string]string
				if err := json.Unmarshal([]byte(out), &rows); err != nil {
					t.Fatalf("JSON output = %q: %v", out, err)
				}
				if len(rows) != 1 || rows[0]["content"] != want {
					t.Errorf("JSON output = %q rows=%#v error=%v, want only %q", out, rows, err, want)
					return
				}
				if _, hasSource := rows[0]["source"]; hasSource {
					t.Errorf("JSON output = %q rows=%#v, want no source", out, rows)
				}
			case "yaml":
				var rows []map[string]string
				if err := yaml.Unmarshal([]byte(out), &rows); err != nil {
					t.Fatalf("YAML output = %q: %v", out, err)
				}
				if len(rows) != 1 || rows[0]["content"] != want {
					t.Errorf("YAML output = %q rows=%#v error=%v, want only %q", out, rows, err, want)
					return
				}
				if _, hasSource := rows[0]["source"]; hasSource {
					t.Errorf("YAML output = %q rows=%#v, want no source", out, rows)
				}
			}
		})
	}
}

func dnsRecord(recordType string) dns.Record {
	return dns.Record{Type: recordType}
}

func dnsRequest(recordType, content string) dns.CreateRecordRequest {
	return dns.CreateRecordRequest{Type: recordType, Content: content}
}

func TestDNSRecordCreateShowZoneAndFallback(t *testing.T) {
	var showCalls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.Method {
		case http.MethodPost:
			fmt.Fprint(w, `{"status":"Success","data":{"name":"example.com"}}`)
		case http.MethodGet:
			showCalls++
			fmt.Fprint(w, `{"status":"Success","data":{"name":"example.com","records":[{"name":"www.example.com.","type":"A","ttl":60,"contents":["192.0.2.1"]},{"name":"other.example.com.","type":"A","ttl":60,"contents":["192.0.2.2"]}]}}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	t.Setenv("ZCP_BEARER_TOKEN", "test-token")

	out, _, err := execCmd(t, NewDNSCmd(), "record-create", "--domain", "example-com-1", "--name", "www", "--type", "A", "--content", "192.0.2.1", "--show-zone", "--api-url", srv.URL, "--output", "json")
	if err != nil {
		t.Fatalf("record-create --show-zone error = %v", err)
	}
	var rows []map[string]string
	if err := json.Unmarshal([]byte(out), &rows); err != nil {
		t.Fatalf("JSON output = %q: %v", out, err)
	}
	if showCalls != 1 || len(rows) != 2 {
		t.Errorf("show calls = %d, rows = %#v; want fallback and full zone", showCalls, rows)
	}
}

func TestDNSRecordCreateFallsBackOnceAndSelectsCreatedValue(t *testing.T) {
	var postCalls, showCalls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.Method {
		case http.MethodPost:
			postCalls++
			fmt.Fprint(w, `{"status":"Success","data":{"name":"example.com"}}`)
		case http.MethodGet:
			showCalls++
			fmt.Fprint(w, `{"status":"Success","data":{"name":"example.com","records":[{"name":"created.example.com.","type":"A","ttl":60,"contents":["192.0.2.1"]},{"name":"other.example.com.","type":"A","ttl":60,"contents":["192.0.2.2"]}]}}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	t.Setenv("ZCP_BEARER_TOKEN", "test-token")

	out, _, err := execCmd(t, NewDNSCmd(), "record-create", "--domain", "example-com-1", "--name", "created", "--type", "A", "--content", "192.0.2.1", "--api-url", srv.URL, "--output", "json")
	if err != nil {
		t.Fatalf("record-create error = %v", err)
	}
	var rows []map[string]string
	if err := json.Unmarshal([]byte(out), &rows); err != nil {
		t.Fatalf("JSON output = %q: %v", out, err)
	}
	if postCalls != 1 || showCalls != 1 || len(rows) != 1 || rows[0]["name"] != "created.example.com." {
		t.Errorf("post=%d show=%d rows=%#v, want one write and selected fallback value", postCalls, showCalls, rows)
	}
}

func TestDNSRecordCreateUnverifiedOutputWarnsAndReturnsEmptyStructuredRows(t *testing.T) {
	for _, format := range []string{"json", "yaml"} {
		t.Run(format, func(t *testing.T) {
			var postCalls, showCalls int
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				switch r.Method {
				case http.MethodPost:
					postCalls++
					fmt.Fprint(w, `{"status":"Success","data":{"name":"example.com"}}`)
				case http.MethodGet:
					showCalls++
					fmt.Fprint(w, `{"status":"Success","data":{"name":"example.com","records":[{"name":"other.example.com.","type":"A","ttl":60,"contents":["192.0.2.2"]}]}}`)
				default:
					http.NotFound(w, r)
				}
			}))
			defer srv.Close()
			t.Setenv("ZCP_BEARER_TOKEN", "test-token")

			out, stderr, err := execCaptureStdio(t, NewDNSCmd(), "record-create", "--domain", "example-com-1", "--name", "created", "--type", "A", "--content", "192.0.2.1", "--api-url", srv.URL, "--output", format)
			if err != nil {
				t.Fatalf("record-create error = %v", err)
			}
			if postCalls != 1 || showCalls != 1 || !strings.Contains(stderr, "could not be matched") {
				t.Errorf("post=%d show=%d stderr=%q, want one write, one lookup, and warning", postCalls, showCalls, stderr)
			}
			if format == "json" && strings.TrimSpace(out) != "[]" {
				t.Errorf("JSON output = %q, want []", out)
			}
			if format == "yaml" {
				var rows []map[string]string
				if err := yaml.Unmarshal([]byte(out), &rows); err != nil || len(rows) != 0 {
					t.Errorf("YAML output = %q rows=%#v error=%v, want empty list", out, rows, err)
				}
			}
		})
	}
}

func TestDNSRecordDeleteReportsSnapshotAndShowZone(t *testing.T) {
	var getCalls, deleteCalls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.Method {
		case http.MethodGet:
			getCalls++
			if getCalls == 1 {
				fmt.Fprint(w, `{"status":"Success","data":{"name":"example.com","records":[{"name":"www.example.com.","type":"A","ttl":3600,"contents":["192.0.2.1","192.0.2.2"]},{"name":"other.example.com.","type":"A","ttl":300,"contents":["192.0.2.9"]}]}}`)
				return
			}
			fmt.Fprint(w, `{"status":"Success","data":{"name":"example.com","records":[{"name":"other.example.com.","type":"A","ttl":300,"contents":["192.0.2.9"]}]}}`)
		case http.MethodDelete:
			deleteCalls++
			fmt.Fprint(w, `{"status":"Success"}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	t.Setenv("ZCP_BEARER_TOKEN", "test-token")

	out, stderr, err := execCaptureStdio(t, NewDNSCmd(), "record-delete", "--domain", "example-com-1", "--name", "www", "--type", "A", "--yes", "--api-url", srv.URL, "--output", "json")
	if err != nil {
		t.Fatalf("record-delete error = %v", err)
	}
	var snapshot []map[string]string
	if err := json.Unmarshal([]byte(out), &snapshot); err != nil {
		t.Fatalf("JSON output = %q: %v", out, err)
	}
	if deleteCalls != 2 || getCalls != 1 || len(snapshot) != 2 || snapshot[1]["content"] != "192.0.2.2" || snapshot[0]["source"] != "pre_delete_snapshot" || snapshot[1]["source"] != "pre_delete_snapshot" || !strings.Contains(stderr, "snapshot read before deletion") {
		t.Errorf("get=%d delete=%d snapshot=%#v, want marked pre-delete values", getCalls, deleteCalls, snapshot)
	}

	getCalls = 0
	out, stderr, err = execCaptureStdio(t, NewDNSCmd(), "record-delete", "--domain", "example-com-1", "--name", "www", "--type", "A", "--yes", "--show-zone", "--api-url", srv.URL, "--output", "json")
	if err != nil {
		t.Fatalf("record-delete --show-zone error = %v", err)
	}
	var remaining []map[string]string
	if err := json.Unmarshal([]byte(out), &remaining); err != nil {
		t.Fatalf("JSON output = %q: %v", out, err)
	}
	if len(remaining) != 1 || remaining[0]["name"] != "other.example.com." || strings.Contains(stderr, "snapshot read before deletion") {
		t.Errorf("remaining = %#v, want resulting zone", remaining)
	} else if _, hasSource := remaining[0]["source"]; hasSource {
		t.Errorf("remaining = %#v, want no source for resulting zone", remaining)
	}
}

func TestDNSRecordDeleteMissingAndFailureDoNotFabricateRows(t *testing.T) {
	t.Run("stale pre-read still deletes", func(t *testing.T) {
		var deleteCalls int
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodGet {
				w.Header().Set("Content-Type", "application/json")
				fmt.Fprint(w, `{"status":"Success","data":{"name":"example.com","records":[]}}`)
				return
			}
			deleteCalls++
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"status":"Success"}`)
		}))
		defer srv.Close()
		t.Setenv("ZCP_BEARER_TOKEN", "test-token")

		out, stderr, err := execCaptureStdio(t, NewDNSCmd(), "record-delete", "--domain", "example-com-1", "--name", "www", "--type", "A", "--yes", "--api-url", srv.URL, "--output", "json")
		if err != nil || strings.TrimSpace(out) != "[]" || deleteCalls != 0 || stderr != "" {
			t.Errorf("error=%v output=%q stderr=%q deleteCalls=%d, want empty result without a delete request", err, out, stderr, deleteCalls)
		}
	})

	t.Run("already deleted", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			if r.Method == http.MethodGet {
				fmt.Fprint(w, `{"status":"Success","data":{"name":"example.com","records":[{"name":"www.example.com.","type":"A","ttl":60,"contents":["192.0.2.1"]}]}}`)
				return
			}
			http.NotFound(w, r)
		}))
		defer srv.Close()
		t.Setenv("ZCP_BEARER_TOKEN", "test-token")

		out, stderr, err := execCaptureStdio(t, NewDNSCmd(), "record-delete", "--domain", "example-com-1", "--name", "www", "--type", "A", "--yes", "--api-url", srv.URL, "--output", "json")
		if err != nil || !strings.Contains(out, "192.0.2.1") || !strings.Contains(stderr, "snapshot read before deletion") {
			t.Errorf("error=%v output=%q stderr=%q, want the pre-delete snapshot when the value disappears after the read", err, out, stderr)
		}
	})

	t.Run("delete failure", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodGet {
				w.Header().Set("Content-Type", "application/json")
				fmt.Fprint(w, `{"status":"Success","data":{"name":"example.com","records":[{"name":"www.example.com.","type":"A","ttl":60,"contents":["192.0.2.1"]}]}}`)
				return
			}
			http.Error(w, "unavailable", http.StatusServiceUnavailable)
		}))
		defer srv.Close()
		t.Setenv("ZCP_BEARER_TOKEN", "test-token")

		out, stderr, err := execCaptureStdio(t, NewDNSCmd(), "record-delete", "--domain", "example-com-1", "--name", "www", "--type", "A", "--yes", "--api-url", srv.URL, "--output", "json")
		if err == nil || strings.Contains(out, "192.0.2.1") || strings.Contains(stderr, "previous values are unavailable") || strings.Contains(stderr, "snapshot read before deletion") {
			t.Errorf("error=%v output=%q stderr=%q, want failure with no record result or missing-details warning", err, out, stderr)
		}
	})
}

func TestDNSRecordDeletePrintsPreDeleteSnapshot(t *testing.T) {
	for _, format := range []string{"table", "json", "yaml"} {
		t.Run(format, func(t *testing.T) {
			var deleteCalls int
			currentContent := "192.0.2.1"
			deletedContent := ""
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch r.Method {
				case http.MethodGet:
					w.Header().Set("Content-Type", "application/json")
					fmt.Fprintf(w, `{"status":"Success","data":{"name":"example.com","records":[{"name":"www.example.com.","type":"A","ttl":60,"contents":[%q]}]}}`, currentContent)
				case http.MethodDelete:
					deleteCalls++
					// The record set changes after the pre-delete read and before DELETE.
					// Delete responses do not include the value that was removed.
					currentContent = "192.0.2.2"
					deletedContent = currentContent
					w.Header().Set("Content-Type", "application/json")
					fmt.Fprint(w, `{"status":"Success"}`)
				default:
					http.NotFound(w, r)
				}
			}))
			defer srv.Close()
			t.Setenv("ZCP_BEARER_TOKEN", "test-token")

			out, stderr, err := execCaptureStdio(t, NewDNSCmd(), "record-delete", "--domain", "example-com-1", "--name", "www", "--type", "A", "--yes", "--api-url", srv.URL, "--output", format)
			if err != nil || deleteCalls != 1 || deletedContent != "192.0.2.2" || strings.Contains(out, "192.0.2.2") || !strings.Contains(stderr, "snapshot read before deletion") || strings.Contains(out, "snapshot read before deletion") {
				t.Errorf("error=%v delete=%d deleted=%q output=%q stderr=%q, want pre-delete snapshot and warning", err, deleteCalls, deletedContent, out, stderr)
			}
			switch format {
			case "table":
				if !strings.Contains(out, "192.0.2.1") || !strings.Contains(out, "Pre-delete snapshot") {
					t.Errorf("table output = %q, want marked pre-delete snapshot", out)
				}
			case "json":
				var rows []map[string]string
				if err := json.Unmarshal([]byte(out), &rows); err != nil || len(rows) != 1 || rows[0]["content"] != "192.0.2.1" || rows[0]["source"] != "pre_delete_snapshot" {
					t.Errorf("JSON output = %q rows=%#v error=%v, want marked pre-delete snapshot", out, rows, err)
				}
			case "yaml":
				var rows []map[string]string
				if err := yaml.Unmarshal([]byte(out), &rows); err != nil || len(rows) != 1 || rows[0]["content"] != "192.0.2.1" || rows[0]["source"] != "pre_delete_snapshot" {
					t.Errorf("YAML output = %q rows=%#v error=%v, want marked pre-delete snapshot", out, rows, err)
				}
			}
		})
	}
}

func TestDNSRecordDeleteByNameUsesAbsoluteFQDNWhenPreReadUnavailable(t *testing.T) {
	tests := []struct {
		name         string
		showStatus   int
		deleteStatus int
		wantDelete   bool
		wantError    string
	}{
		{"unavailable", http.StatusServiceUnavailable, http.StatusOK, true, ""},
		{"missing", http.StatusNotFound, http.StatusOK, true, ""},
		{"unauthorized", http.StatusUnauthorized, http.StatusOK, false, "resolving domain"},
		{"forbidden", http.StatusForbidden, http.StatusOK, false, "resolving domain"},
		{"failed delete", http.StatusServiceUnavailable, http.StatusServiceUnavailable, true, "dns record-delete"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var deleteCalls int
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch r.Method {
				case http.MethodGet:
					http.Error(w, "pre-read unavailable", tt.showStatus)
				case http.MethodDelete:
					deleteCalls++
					if r.URL.Query().Get("name") != "www.example.com." || r.URL.Query().Get("type") != "A" {
						http.Error(w, "unexpected delete target", http.StatusBadRequest)
						return
					}
					if tt.deleteStatus != http.StatusOK {
						http.Error(w, "delete unavailable", tt.deleteStatus)
						return
					}
					w.Header().Set("Content-Type", "application/json")
					fmt.Fprint(w, `{"status":"Success"}`)
				default:
					http.NotFound(w, r)
				}
			}))
			defer srv.Close()
			t.Setenv("ZCP_BEARER_TOKEN", "test-token")

			out, stderr, err := execCaptureStdio(t, NewDNSCmd(), "record-delete", "--domain", "example-com-1", "--name", "WWW.Example.COM.", "--type", "A", "--yes", "--api-url", srv.URL, "--output", "json")
			if tt.wantError != "" {
				wantDeleteCalls := 0
				if tt.wantDelete {
					wantDeleteCalls = 1
				}
				if err == nil || !strings.Contains(err.Error(), tt.wantError) || deleteCalls != wantDeleteCalls || strings.Contains(stderr, "previous values are unavailable") || strings.Contains(stderr, "snapshot read before deletion") {
					t.Errorf("error=%v delete=%d output=%q stderr=%q, want %q without a successful-delete warning", err, deleteCalls, out, stderr, tt.wantError)
				}
				return
			}
			if err != nil || deleteCalls != 1 || strings.TrimSpace(out) != "[]" || !strings.Contains(stderr, "previous values are unavailable") || strings.Contains(stderr, "snapshot read before deletion") {
				t.Errorf("error=%v delete=%d output=%q stderr=%q, want successful delete with unavailable-details warning", err, deleteCalls, out, stderr)
			}
		})
	}
}

func TestDNSRecordDeleteByNameRequiresLookupForRelativeNames(t *testing.T) {
	for _, name := range []string{"www", "@"} {
		t.Run(name, func(t *testing.T) {
			var deleteCalls int
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method == http.MethodGet {
					http.Error(w, "pre-read unavailable", http.StatusServiceUnavailable)
					return
				}
				deleteCalls++
				http.Error(w, "delete must not be called", http.StatusInternalServerError)
			}))
			defer srv.Close()
			t.Setenv("ZCP_BEARER_TOKEN", "test-token")

			_, _, err := execCaptureStdio(t, NewDNSCmd(), "record-delete", "--domain", "example-com-1", "--name", name, "--type", "A", "--yes", "--api-url", srv.URL)
			if err == nil || !strings.Contains(err.Error(), "use a fully qualified name ending in a dot") || deleteCalls != 0 {
				t.Errorf("error=%v delete=%d, want actionable lookup error without delete", err, deleteCalls)
			}
		})
	}
}

func TestDNSRecordDeleteByNameGetsFreshTimeoutAfterPreRead(t *testing.T) {
	var deleteCalls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			<-r.Context().Done()
		case http.MethodDelete:
			deleteCalls++
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"status":"Success"}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	t.Setenv("ZCP_BEARER_TOKEN", "test-token")

	out, stderr, err := execCaptureStdio(t, NewDNSCmd(), "record-delete", "--domain", "example-com-1", "--name", "www.example.com.", "--type", "A", "--yes", "--timeout", "1", "--api-url", srv.URL, "--output", "json")
	if err != nil || deleteCalls != 1 || strings.TrimSpace(out) != "[]" || !strings.Contains(stderr, "previous values are unavailable") || strings.Contains(stderr, "snapshot read before deletion") {
		t.Errorf("error=%v delete=%d output=%q stderr=%q, want successful delete with fresh timeout and unavailable-details warning", err, deleteCalls, out, stderr)
	}
}

func TestDNSRecordDeleteAlreadyDeletedShowZone(t *testing.T) {
	tests := []struct {
		name          string
		args          []string
		showZone      bool
		postShowError bool
		wantSnapshot  bool
	}{
		{
			name:         "name and type without show zone",
			args:         []string{"--name", "www", "--type", "A"},
			wantSnapshot: true,
		},
		{
			name:     "name and type with show zone",
			args:     []string{"--name", "www", "--type", "A"},
			showZone: true,
		},
		{
			name:          "name and type with show zone post read failure",
			args:          []string{"--name", "www", "--type", "A"},
			showZone:      true,
			postShowError: true,
		},
		{
			name: "record ID without show zone",
			args: []string{"--record-id", "42"},
		},
		{
			name:     "record ID with show zone",
			args:     []string{"--record-id", "42"},
			showZone: true,
		},
		{
			name:          "record ID with show zone post read failure",
			args:          []string{"--record-id", "42"},
			showZone:      true,
			postShowError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var showCalls int
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch r.Method {
				case http.MethodGet:
					showCalls++
					if showCalls > 1 && tt.postShowError {
						http.Error(w, "post-read unavailable", http.StatusServiceUnavailable)
						return
					}
					w.Header().Set("Content-Type", "application/json")
					if showCalls > 1 {
						fmt.Fprint(w, `{"status":"Success","data":{"name":"example.com","records":[{"name":"remaining.example.com.","type":"A","ttl":60,"contents":["192.0.2.2"]}]}}`)
						return
					}
					fmt.Fprint(w, `{"status":"Success","data":{"name":"example.com","records":[{"id":"42","name":"www.example.com.","type":"A","ttl":60,"contents":["192.0.2.1"]}]}}`)
				case http.MethodDelete:
					http.NotFound(w, r)
				default:
					http.NotFound(w, r)
				}
			}))
			defer srv.Close()
			t.Setenv("ZCP_BEARER_TOKEN", "test-token")

			args := append([]string{"record-delete", "--domain", "example-com-1", "--yes"}, tt.args...)
			if tt.showZone {
				args = append(args, "--show-zone")
			}
			args = append(args, "--api-url", srv.URL, "--output", "json")
			out, stderr, err := execCaptureStdio(t, NewDNSCmd(), args...)

			if tt.postShowError {
				if err == nil || showCalls < 2 || strings.Contains(out, "remaining.example.com") || strings.Contains(stderr, "previous values are unavailable") || strings.Contains(stderr, "snapshot read before deletion") {
					t.Errorf("error=%v show=%d output=%q stderr=%q, want post-read error without rows or missing-details warning", err, showCalls, out, stderr)
				}
				return
			}
			if err != nil {
				t.Fatalf("record-delete error = %v", err)
			}
			if tt.showZone {
				if showCalls != 2 || !strings.Contains(out, "remaining.example.com") || strings.Contains(stderr, "previous values are unavailable") || strings.Contains(stderr, "snapshot read before deletion") {
					t.Errorf("show=%d output=%q stderr=%q, want remaining zone without missing-details warning", showCalls, out, stderr)
				}
				return
			}
			if tt.wantSnapshot {
				var rows []map[string]string
				if json.Unmarshal([]byte(out), &rows) != nil || showCalls != 1 || len(rows) != 1 || rows[0]["content"] != "192.0.2.1" || rows[0]["source"] != "pre_delete_snapshot" || strings.Contains(stderr, "previous values are unavailable") || !strings.Contains(stderr, "snapshot read before deletion") {
					t.Errorf("show=%d output=%q stderr=%q, want pre-delete snapshot and warning when the value was already deleted", showCalls, out, stderr)
				}
				return
			}
			if showCalls != 1 || strings.TrimSpace(out) != "[]" || strings.Contains(stderr, "previous values are unavailable") || strings.Contains(stderr, "snapshot read before deletion") {
				t.Errorf("show=%d output=%q stderr=%q, want empty output without missing-details warning", showCalls, out, stderr)
			}
		})
	}
}

func TestDNSRecordDeleteLegacyIDReportsRecord(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodGet {
			fmt.Fprint(w, `{"status":"Success","data":{"name":"example.com","records":[{"id":"42","name":"www.example.com.","type":"A","ttl":60,"content":"192.0.2.1"}]}}`)
			return
		}
		fmt.Fprint(w, `{"status":"Success"}`)
	}))
	defer srv.Close()
	t.Setenv("ZCP_BEARER_TOKEN", "test-token")

	out, stderr, err := execCaptureStdio(t, NewDNSCmd(), "record-delete", "--domain", "example-com-1", "--record-id", "42", "--yes", "--api-url", srv.URL, "--output", "json")
	var rows []map[string]string
	if json.Unmarshal([]byte(out), &rows) != nil || len(rows) != 1 || rows[0]["content"] != "192.0.2.1" || rows[0]["source"] != "pre_delete_snapshot" || err != nil || !strings.Contains(stderr, "snapshot read before deletion") {
		t.Errorf("error=%v output=%q stderr=%q, want legacy record snapshot and warning", err, out, stderr)
	}
}

func TestDNSRecordDeleteLegacyIDPreReadIsBestEffort(t *testing.T) {
	tests := []struct {
		name       string
		firstShow  int
		showZone   bool
		deleteCode int
		postShow   int
	}{
		{"show error", http.StatusForbidden, false, http.StatusOK, http.StatusOK},
		{"record ID missing", http.StatusOK, false, http.StatusOK, http.StatusOK},
		{"show zone after failed pre-read", http.StatusForbidden, true, http.StatusOK, http.StatusOK},
		{"show zone post-read error", http.StatusOK, true, http.StatusOK, http.StatusForbidden},
		{"delete error remains error", http.StatusForbidden, false, http.StatusServiceUnavailable, http.StatusOK},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var showCalls, deleteCalls int
			deleted := false
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch r.Method {
				case http.MethodGet:
					showCalls++
					if !deleted && tt.firstShow != http.StatusOK {
						http.Error(w, "pre-read unavailable", tt.firstShow)
						return
					}
					w.Header().Set("Content-Type", "application/json")
					if tt.showZone && deleted {
						if tt.postShow != http.StatusOK {
							http.Error(w, "post-read unavailable", tt.postShow)
							return
						}
						fmt.Fprint(w, `{"status":"Success","data":{"name":"example.com","records":[{"id":"43","name":"remaining.example.com.","type":"A","ttl":60,"contents":["192.0.2.2"]}]}}`)
						return
					}
					fmt.Fprint(w, `{"status":"Success","data":{"name":"example.com","records":[{"id":"43","name":"other.example.com.","type":"A","ttl":60,"contents":["192.0.2.2"]}]}}`)
				case http.MethodDelete:
					deleteCalls++
					if tt.deleteCode != http.StatusOK {
						http.Error(w, "delete unavailable", tt.deleteCode)
						return
					}
					w.Header().Set("Content-Type", "application/json")
					deleted = true
					fmt.Fprint(w, `{"status":"Success"}`)
				default:
					http.NotFound(w, r)
				}
			}))
			defer srv.Close()
			t.Setenv("ZCP_BEARER_TOKEN", "test-token")

			args := []string{"record-delete", "--domain", "example-com-1", "--record-id", "42", "--yes", "--api-url", srv.URL, "--output", "json"}
			if tt.showZone {
				args = append(args, "--show-zone")
			}
			out, stderr, err := execCaptureStdio(t, NewDNSCmd(), args...)
			if tt.deleteCode != http.StatusOK || tt.postShow != http.StatusOK {
				if err == nil || deleteCalls != 1 || strings.Contains(out, "remaining.example.com") || strings.Contains(stderr, "previous values are unavailable") || strings.Contains(stderr, "snapshot read before deletion") {
					t.Errorf("error=%v delete=%d output=%q stderr=%q, want error without rows or missing-details warning", err, deleteCalls, out, stderr)
				}
				return
			}
			if err != nil || deleteCalls != 1 {
				t.Fatalf("error=%v delete=%d, want successful delete", err, deleteCalls)
			}
			if tt.showZone {
				if showCalls < 2 || !strings.Contains(out, "remaining.example.com") || strings.Contains(stderr, "previous values are unavailable") || strings.Contains(stderr, "snapshot read before deletion") {
					t.Errorf("show=%d output=%q stderr=%q, want post-delete zone without missing-details warning", showCalls, out, stderr)
				}
			} else if strings.TrimSpace(out) != "[]" || !strings.Contains(stderr, "previous values are unavailable") || strings.Contains(stderr, "snapshot read before deletion") {
				t.Errorf("output=%q stderr=%q, want empty rows and warning when pre-read could not identify record", out, stderr)
			}
		})
	}
}

func TestDNSRecordDeleteLegacyIDGetsFreshTimeoutAfterPreRead(t *testing.T) {
	var deleteCalls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			<-r.Context().Done()
		case http.MethodDelete:
			deleteCalls++
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"status":"Success"}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	t.Setenv("ZCP_BEARER_TOKEN", "test-token")

	out, stderr, err := execCaptureStdio(t, NewDNSCmd(), "record-delete", "--domain", "example-com-1", "--record-id", "42", "--yes", "--timeout", "1", "--api-url", srv.URL, "--output", "json")
	if err != nil || deleteCalls != 1 || strings.TrimSpace(out) != "[]" || !strings.Contains(stderr, "previous values are unavailable") || strings.Contains(stderr, "snapshot read before deletion") {
		t.Errorf("error=%v delete=%d output=%q stderr=%q, want successful delete with empty rows and warning", err, deleteCalls, out, stderr)
	}
}

func TestDNSRecordDeleteUsesRawInitialSnapshotValues(t *testing.T) {
	tests := []struct {
		name       string
		recordType string
		contents   []string
	}{
		{"A", "A", []string{"192.0.2.10", "192.0.2.11"}},
		{"MX", "MX", []string{"0 mail.example.com.", "10 backup.example.com."}},
		{"TXT", "TXT", []string{`"one"`, `"two"`}},
		{"CAA", "CAA", []string{`0 issue "letsencrypt.org"`, `0 iodef "mailto:security@example.com"`}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var contents []string
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				switch r.Method {
				case http.MethodGet:
					payload, err := json.Marshal(tt.contents)
					if err != nil {
						t.Fatal(err)
					}
					fmt.Fprintf(w, `{"status":"Success","data":{"name":"example.com","records":[{"name":"value.example.com.","type":%q,"ttl":60,"contents":%s}]}}`, tt.recordType, payload)
				case http.MethodDelete:
					content := r.URL.Query().Get("content")
					if content == "" {
						w.WriteHeader(http.StatusInternalServerError)
						fmt.Fprint(w, `{"status":"Error","message":"Content is required to delete a record from a multi-value record set."}`)
						return
					}
					contents = append(contents, content)
					fmt.Fprint(w, `{"status":"Success"}`)
				default:
					http.NotFound(w, r)
				}
			}))
			defer srv.Close()
			t.Setenv("ZCP_BEARER_TOKEN", "test-token")

			out, _, err := execCaptureStdio(t, NewDNSCmd(), "record-delete", "--domain", "example-com-1", "--name", "value", "--type", tt.recordType, "--yes", "--api-url", srv.URL, "--output", "json")
			if err != nil {
				t.Fatalf("record-delete error = %v", err)
			}
			if got, want := strings.Join(contents, "|"), strings.Join(tt.contents, "|"); got != want {
				t.Errorf("delete content = %q, want %q", got, want)
			}
			if !strings.Contains(out, "pre_delete_snapshot") {
				t.Errorf("output = %q, want snapshot rows", out)
			}
		})
	}
}

func TestDNSRecordDeleteStopsOnPartialFailureWithoutSnapshotOutput(t *testing.T) {
	var contents []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.Method {
		case http.MethodGet:
			fmt.Fprint(w, `{"status":"Success","data":{"name":"example.com","records":[{"name":"www.example.com.","type":"A","ttl":60,"contents":["192.0.2.10","192.0.2.11"]}]}}`)
		case http.MethodDelete:
			contents = append(contents, r.URL.Query().Get("content"))
			if len(contents) == 2 {
				http.Error(w, "backend unavailable", http.StatusServiceUnavailable)
				return
			}
			fmt.Fprint(w, `{"status":"Success"}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	t.Setenv("ZCP_BEARER_TOKEN", "test-token")

	out, stderr, err := execCaptureStdio(t, NewDNSCmd(), "record-delete", "--domain", "example-com-1", "--name", "www", "--type", "A", "--yes", "--api-url", srv.URL, "--output", "json")
	if err == nil || !strings.Contains(stderr, "stopped after deleting 1 of 2 values") || strings.Contains(out, "pre_delete_snapshot") {
		t.Errorf("error=%v output=%q stderr=%q, want partial failure without success output", err, out, stderr)
	}
	if got := strings.Join(contents, "|"); got != "192.0.2.10|192.0.2.11" {
		t.Errorf("delete content = %q, want both initial values in order", got)
	}
}

func TestDNSRecordDeleteDoesNotDrainConcurrentValues(t *testing.T) {
	var deleteCalls, showCalls int
	remaining := []string{"192.0.2.10"}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.Method {
		case http.MethodGet:
			showCalls++
			payload, err := json.Marshal(remaining)
			if err != nil {
				t.Fatal(err)
			}
			fmt.Fprintf(w, `{"status":"Success","data":{"name":"example.com","records":[{"name":"www.example.com.","type":"A","ttl":60,"contents":%s}]}}`, payload)
		case http.MethodDelete:
			deleteCalls++
			if r.URL.Query().Get("content") != "192.0.2.10" {
				http.Error(w, "unexpected value", http.StatusBadRequest)
				return
			}
			remaining = []string{"192.0.2.99"}
			fmt.Fprint(w, `{"status":"Success"}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	t.Setenv("ZCP_BEARER_TOKEN", "test-token")

	out, _, err := execCaptureStdio(t, NewDNSCmd(), "record-delete", "--domain", "example-com-1", "--name", "www", "--type", "A", "--yes", "--show-zone", "--api-url", srv.URL, "--output", "json")
	if err != nil || deleteCalls != 1 || showCalls != 2 || !strings.Contains(out, "192.0.2.99") {
		t.Errorf("error=%v deleteCalls=%d showCalls=%d output=%q, want only the initial value deleted and the concurrent value preserved", err, deleteCalls, showCalls, out)
	}
}

func TestDNSRecordDeleteTreatsOnlyExactDNS500AsAbsent(t *testing.T) {
	for _, tt := range []struct {
		name      string
		message   string
		wantError bool
	}{
		{"verified absent response", "DNS record not found.", false},
		{"unrelated server failure", "database unavailable", true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch r.Method {
				case http.MethodGet:
					http.Error(w, "pre-read unavailable", http.StatusServiceUnavailable)
				case http.MethodDelete:
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusInternalServerError)
					fmt.Fprintf(w, `{"status":"Error","message":%q}`, tt.message)
				default:
					http.NotFound(w, r)
				}
			}))
			defer srv.Close()
			t.Setenv("ZCP_BEARER_TOKEN", "test-token")

			out, stderr, err := execCaptureStdio(t, NewDNSCmd(), "record-delete", "--domain", "example-com-1", "--name", "www.example.com.", "--type", "A", "--yes", "--api-url", srv.URL, "--output", "json")
			if tt.wantError {
				if err == nil || strings.Contains(out, "pre_delete_snapshot") {
					t.Errorf("error=%v output=%q, want unrelated 500 to fail", err, out)
				}
				return
			}
			if err != nil || strings.TrimSpace(out) != "[]" || !strings.Contains(stderr, "not found (already deleted)") {
				t.Errorf("error=%v output=%q stderr=%q, want idempotent absent result", err, out, stderr)
			}
		})
	}
}

func TestDNSRecordCreateRejectsUnquotedCAABeforeHTTP(t *testing.T) {
	var calls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		http.Error(w, "must not be called", http.StatusInternalServerError)
	}))
	defer srv.Close()
	t.Setenv("ZCP_BEARER_TOKEN", "test-token")

	_, _, err := execCaptureStdio(t, NewDNSCmd(), "record-create", "--domain", "example-com-1", "--name", "caa", "--type", "CAA", "--content", "0 issue letsencrypt.org", "--api-url", srv.URL)
	if err == nil || !strings.Contains(err.Error(), `0 issue "letsencrypt.org"`) || calls != 0 {
		t.Errorf("error=%v calls=%d, want local quoted-CAA validation without HTTP", err, calls)
	}
}
