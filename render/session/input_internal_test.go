package session

import "testing"

func TestParseModelName(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		wantBase    string
		wantVersion string
	}{
		{name: "name with version", input: "Opus 4.5", wantBase: "Opus", wantVersion: "4.5"},
		{name: "name without version", input: "Claude", wantBase: "Claude", wantVersion: ""},
		{name: "name with long version", input: "Sonnet 3.5.2", wantBase: "Sonnet", wantVersion: "3.5.2"},
		{name: "empty string", input: "", wantBase: "", wantVersion: ""},
		{name: "multiple spaces", input: "Model Name Version", wantBase: "Model", wantVersion: "Name Version"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			base, version := splitModelName(tt.input)
			if base != tt.wantBase || version != tt.wantVersion {
				t.Errorf("splitModelName(%q) = (%q, %q), want (%q, %q)", tt.input, base, version, tt.wantBase, tt.wantVersion)
			}
		})
	}
}
