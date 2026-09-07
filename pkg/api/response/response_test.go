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
		{name: "negative", raw: `-1`, want: -1},
		{name: "large integer exact", raw: `9007199254740993`, want: 9007199254740993},
		{name: "quoted large integer exact", raw: `"9007199254740993"`, want: 9007199254740993},
		{name: "exponent form", raw: `3e0`, want: 3},
		{name: "integer out of range", raw: `99999999999999999999`, wantErr: true},
		{name: "float out of range", raw: `1e30`, wantErr: true},
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
