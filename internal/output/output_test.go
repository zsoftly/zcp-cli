package output_test

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/zsoftly/zcp-cli/internal/output"
	"gopkg.in/yaml.v3"
)

var testHeaders = []string{"UUID", "NAME", "ACTIVE"}
var testRows = [][]string{
	{"abc-123", "zone-1", "true"},
	{"def-456", "zone-2", "false"},
}

func TestPrintTableText(t *testing.T) {
	var buf bytes.Buffer
	p := output.NewPrinter(&buf, output.FormatTable, true)

	if err := p.PrintTable(testHeaders, testRows); err != nil {
		t.Fatalf("PrintTable() error = %v", err)
	}

	got := buf.String()
	for _, h := range testHeaders {
		if !strings.Contains(got, h) {
			t.Errorf("output missing header %q", h)
		}
	}
	if !strings.Contains(got, "zone-1") {
		t.Error("output missing row data 'zone-1'")
	}
}

func TestPrintTableJSON(t *testing.T) {
	var buf bytes.Buffer
	p := output.NewPrinter(&buf, output.FormatJSON, true)

	if err := p.PrintTable(testHeaders, testRows); err != nil {
		t.Fatalf("PrintTable() JSON error = %v", err)
	}

	var records []map[string]string
	if err := json.Unmarshal(buf.Bytes(), &records); err != nil {
		t.Fatalf("JSON output not parseable: %v\noutput: %s", err, buf.String())
	}

	if len(records) != 2 {
		t.Errorf("expected 2 records, got %d", len(records))
	}
	if records[0]["name"] != "zone-1" {
		t.Errorf("records[0][name] = %q, want %q", records[0]["name"], "zone-1")
	}
}

func TestPrintTableYAML(t *testing.T) {
	var buf bytes.Buffer
	p := output.NewPrinter(&buf, output.FormatYAML, true)

	if err := p.PrintTable(testHeaders, testRows); err != nil {
		t.Fatalf("PrintTable() YAML error = %v", err)
	}

	var records []map[string]string
	if err := yaml.Unmarshal(buf.Bytes(), &records); err != nil {
		t.Fatalf("YAML output not parseable: %v\noutput: %s", err, buf.String())
	}

	if len(records) != 2 {
		t.Errorf("expected 2 records, got %d", len(records))
	}
}

func TestParseFormat(t *testing.T) {
	tests := []struct {
		input string
		want  output.Format
	}{
		{"table", output.FormatTable},
		{"TABLE", output.FormatTable},
		{"json", output.FormatJSON},
		{"JSON", output.FormatJSON},
		{"yaml", output.FormatYAML},
		{"YAML", output.FormatYAML},
		{"", output.FormatTable},
		{"unknown", output.FormatTable},
	}
	for _, tt := range tests {
		got := output.ParseFormat(tt.input)
		if got != tt.want {
			t.Errorf("ParseFormat(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestPrintJSON(t *testing.T) {
	var buf bytes.Buffer
	p := output.NewPrinter(&buf, output.FormatJSON, true)

	type item struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}
	if err := p.Print(item{ID: "1", Name: "test"}); err != nil {
		t.Fatalf("Print() error = %v", err)
	}

	var got item
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("Print() output not valid JSON: %v", err)
	}
	if got.Name != "test" {
		t.Errorf("Name = %q, want %q", got.Name, "test")
	}
}
