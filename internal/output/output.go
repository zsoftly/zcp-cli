// Package output provides consistent table, JSON, and YAML output for ZCP CLI commands.
package output

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/olekukonko/tablewriter"
	"gopkg.in/yaml.v3"
)

// Format represents the output format.
type Format string

const (
	FormatTable Format = "table"
	FormatJSON  Format = "json"
	FormatYAML  Format = "yaml"
)

// ParseFormat parses a format string, defaulting to table on unknown values.
func ParseFormat(s string) Format {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "json":
		return FormatJSON
	case "yaml":
		return FormatYAML
	default:
		return FormatTable
	}
}

// Printer renders CLI output in the specified format.
type Printer struct {
	w       io.Writer
	format  Format
	noColor bool
}

// NewPrinter creates a new Printer writing to w.
func NewPrinter(w io.Writer, format Format, noColor bool) *Printer {
	return &Printer{w: w, format: format, noColor: noColor}
}

// PrintTable renders a table with the given headers and rows.
// In JSON mode, it emits an array of objects keyed by header name.
// In YAML mode, it emits a YAML list.
func (p *Printer) PrintTable(headers []string, rows [][]string) error {
	switch p.format {
	case FormatJSON:
		return p.printTableAsJSON(headers, rows)
	case FormatYAML:
		return p.printTableAsYAML(headers, rows)
	default:
		return p.printTableAsText(headers, rows)
	}
}

func (p *Printer) printTableAsText(headers []string, rows [][]string) error {
	table := tablewriter.NewWriter(p.w)
	table.SetHeader(headers)
	table.SetBorder(false)
	table.SetHeaderLine(false)
	table.SetColumnSeparator("  ")
	table.SetHeaderAlignment(tablewriter.ALIGN_LEFT)
	table.SetAlignment(tablewriter.ALIGN_LEFT)
	table.SetTablePadding("  ")
	table.SetNoWhiteSpace(true)
	table.SetAutoWrapText(false)

	for _, row := range rows {
		table.Append(row)
	}
	table.Render()
	return nil
}

func (p *Printer) printTableAsJSON(headers []string, rows [][]string) error {
	records := make([]map[string]string, 0, len(rows))
	for _, row := range rows {
		rec := make(map[string]string, len(headers))
		for i, h := range headers {
			key := normalizeKey(h)
			if i < len(row) {
				rec[key] = row[i]
			}
		}
		records = append(records, rec)
	}
	enc := json.NewEncoder(p.w)
	enc.SetIndent("", "  ")
	return enc.Encode(records)
}

func (p *Printer) printTableAsYAML(headers []string, rows [][]string) error {
	records := make([]map[string]string, 0, len(rows))
	for _, row := range rows {
		rec := make(map[string]string, len(headers))
		for i, h := range headers {
			key := normalizeKey(h)
			if i < len(row) {
				rec[key] = row[i]
			}
		}
		records = append(records, rec)
	}
	return yaml.NewEncoder(p.w).Encode(records)
}

// Print renders an arbitrary value as JSON or YAML.
// In table mode it falls back to JSON.
func (p *Printer) Print(v interface{}) error {
	switch p.format {
	case FormatYAML:
		return yaml.NewEncoder(p.w).Encode(v)
	default: // table and json both use JSON for structured data
		enc := json.NewEncoder(p.w)
		enc.SetIndent("", "  ")
		return enc.Encode(v)
	}
}

// Fprintf writes formatted text to the output writer (useful for simple messages).
func (p *Printer) Fprintf(format string, args ...interface{}) {
	fmt.Fprintf(p.w, format, args...)
}

// normalizeKey converts a header like "CPU Cores" -> "cpu_cores".
func normalizeKey(h string) string {
	h = strings.ToLower(h)
	h = strings.ReplaceAll(h, " ", "_")
	h = strings.ReplaceAll(h, "-", "_")
	return h
}
