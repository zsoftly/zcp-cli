package response_test

import (
	"encoding/json"
	"testing"

	"github.com/zsoftly/zcp-cli/pkg/api/response"
)

func TestParseFlexInt(t *testing.T) {
	tests := []struct {
		name    string
		raw     string
		want    int
		wantErr bool
	}{
		{name: "json number", raw: `3`, want: 3},
		{name: "quoted numeric string", raw: `"3"`, want: 3},
		{name: "null", raw: `null`, want: 0},
		{name: "empty", raw: ``, want: 0},
		{name: "empty quoted string", raw: `""`, want: 0},
		{name: "integral float", raw: `3.0`, want: 3},
		{name: "quoted integral float", raw: `"3.0"`, want: 3},
		{name: "non-integral float", raw: `3.5`, wantErr: true},
		{name: "quoted non-integral float", raw: `"3.5"`, wantErr: true},
		{name: "non-numeric string", raw: `"soon"`, wantErr: true},
		{name: "invalid quoted json", raw: `"unterminated`, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := response.ParseFlexInt(json.RawMessage(tt.raw))
			if tt.wantErr {
				if err == nil {
					t.Fatalf("ParseFlexInt(%q) expected error, got nil", tt.raw)
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseFlexInt(%q) unexpected error: %v", tt.raw, err)
			}
			if got != tt.want {
				t.Errorf("ParseFlexInt(%q) = %d, want %d", tt.raw, got, tt.want)
			}
		})
	}
}
