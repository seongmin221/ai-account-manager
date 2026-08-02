package config

import "testing"

func TestValidateIdentifier(t *testing.T) {
	tests := []struct {
		name  string
		value string
		valid bool
	}{
		{name: "work", value: "work", valid: true},
		{name: "hyphen and underscore", value: "team_1-prod", valid: true},
		{name: "must start with letter", value: "1work", valid: false},
		{name: "uppercase rejected", value: "Work", valid: false},
		{name: "space rejected", value: "work profile", valid: false},
		{name: "empty rejected", value: "", valid: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ValidateIdentifier(tt.value) == nil; got != tt.valid {
				t.Fatalf("ValidateIdentifier(%q) valid = %v, want %v", tt.value, got, tt.valid)
			}
		})
	}
}

func TestDocumentOverallMode(t *testing.T) {
	tests := []struct {
		name   string
		active map[ProviderID]ProfileID
		want   string
	}{
		{name: "empty", active: nil, want: "unconfigured"},
		{name: "same profile", active: map[ProviderID]ProfileID{"github": "work", "codex": "work"}, want: "work"},
		{name: "mixed profiles", active: map[ProviderID]ProfileID{"github": "personal", "codex": "work"}, want: "mixed"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc := Document{Active: tt.active}
			if got := doc.OverallMode(); got != tt.want {
				t.Fatalf("OverallMode() = %q, want %q", got, tt.want)
			}
		})
	}
}
