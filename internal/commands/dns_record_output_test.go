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

func TestDNSRecordDeleteReportsRemovedValuesAndShowZone(t *testing.T) {
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

	out, _, err := execCmd(t, NewDNSCmd(), "record-delete", "--domain", "example-com-1", "--name", "www", "--type", "A", "--yes", "--api-url", srv.URL, "--output", "json")
	if err != nil {
		t.Fatalf("record-delete error = %v", err)
	}
	var removed []map[string]string
	if err := json.Unmarshal([]byte(out), &removed); err != nil {
		t.Fatalf("JSON output = %q: %v", out, err)
	}
	if deleteCalls != 1 || getCalls != 1 || len(removed) != 2 || removed[1]["content"] != "192.0.2.2" {
		t.Errorf("get=%d delete=%d removed=%#v, want both values", getCalls, deleteCalls, removed)
	}

	getCalls = 0
	out, _, err = execCmd(t, NewDNSCmd(), "record-delete", "--domain", "example-com-1", "--name", "www", "--type", "A", "--yes", "--show-zone", "--api-url", srv.URL, "--output", "json")
	if err != nil {
		t.Fatalf("record-delete --show-zone error = %v", err)
	}
	var remaining []map[string]string
	if err := json.Unmarshal([]byte(out), &remaining); err != nil {
		t.Fatalf("JSON output = %q: %v", out, err)
	}
	if len(remaining) != 1 || remaining[0]["name"] != "other.example.com." {
		t.Errorf("remaining = %#v, want resulting zone", remaining)
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

		out, _, err := execCmd(t, NewDNSCmd(), "record-delete", "--domain", "example-com-1", "--name", "www", "--type", "A", "--yes", "--api-url", srv.URL, "--output", "json")
		if err != nil || strings.TrimSpace(out) != "[]" || deleteCalls != 1 {
			t.Errorf("error=%v output=%q deleteCalls=%d, want empty result after delete", err, out, deleteCalls)
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

		out, _, err := execCmd(t, NewDNSCmd(), "record-delete", "--domain", "example-com-1", "--name", "www", "--type", "A", "--yes", "--api-url", srv.URL, "--output", "json")
		if err != nil || strings.TrimSpace(out) != "[]" {
			t.Errorf("error=%v output=%q, want empty result for an already deleted record", err, out)
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

		out, _, err := execCmd(t, NewDNSCmd(), "record-delete", "--domain", "example-com-1", "--name", "www", "--type", "A", "--yes", "--api-url", srv.URL, "--output", "json")
		if err == nil || strings.Contains(out, "192.0.2.1") {
			t.Errorf("error=%v output=%q, want failure with no record result", err, out)
		}
	})
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

	out, _, err := execCmd(t, NewDNSCmd(), "record-delete", "--domain", "example-com-1", "--record-id", "42", "--yes", "--api-url", srv.URL, "--output", "json")
	if err != nil || !strings.Contains(out, "192.0.2.1") {
		t.Errorf("error=%v output=%q, want legacy record details", err, out)
	}
}
