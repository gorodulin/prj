package project

import "testing"

func TestIsAnyValidID(t *testing.T) {
	tests := []struct {
		name string
		id   string
		want bool
	}{
		{"aYMDb single prefix", "p20250101a", true},
		{"aYMDb triple prefix", "prj20250101a", true},
		{"aYMDb with dash separator", "prj-20250101a", true},
		{"aYMDb with underscore separator", "prj_20250101a", true},
		{"UUIDv7", "01932c07-a9c3-7b2a-8b1c-432f0696a585", true},
		{"ULID", "01ARYZ6S41TSV4RRFFQ69G5FAV", true},
		{"KSUID", "2E8JwMKbBEgHvAsDFNNeyamgmCi", true},
		{"random string", "not-a-project", false},
		{"empty", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsAnyValidID(tt.id)
			if got != tt.want {
				t.Errorf("IsAnyValidID(%q) = %v, want %v", tt.id, got, tt.want)
			}
		})
	}
}

func TestIsValidID(t *testing.T) {
	tests := []struct {
		name   string
		id     string
		format string
		prefix string
		want   bool
	}{
		// aYYYYMMDDb format — prefix matching
		{"aYMDb valid single prefix", "p20250101a", FormatAYMDb, "p", true},
		{"aYMDb valid double prefix", "ab20250101a", FormatAYMDb, "ab", true},
		{"aYMDb valid triple prefix", "prj20250101a", FormatAYMDb, "prj", true},
		{"aYMDb valid five-letter prefix", "abcde20250101a", FormatAYMDb, "abcde", true},
		{"aYMDb valid triple suffix", "p20250101abc", FormatAYMDb, "p", true},
		{"aYMDb prefix with dash", "prj-20250101a", FormatAYMDb, "prj-", true},
		{"aYMDb prefix with underscore", "prj_20250101a", FormatAYMDb, "prj_", true},
		{"aYMDb prefix mismatch", "p20250101a", FormatAYMDb, "prj", false},
		{"aYMDb no suffix letter", "p20250101", FormatAYMDb, "p", false},
		{"aYMDb too many suffix letters", "p20250101abcd", FormatAYMDb, "p", false},
		{"aYMDb uppercase ID rejected", "P20250101a", FormatAYMDb, "p", false},
		{"aYMDb short digits", "p2025011a", FormatAYMDb, "p", false},
		{"aYMDb non-project name", "metadata", FormatAYMDb, "p", false},

		// UUIDv7 — prefix ignored
		{"UUIDv7 valid", "01932c07-a9c3-7b2a-8b1c-432f0696a585", FormatUUIDv7, "", true},
		{"UUIDv7 wrong version", "01932c07-a9c3-4b2a-8b1c-432f0696a585", FormatUUIDv7, "", false},
		{"UUIDv7 plain string", "not-a-uuid", FormatUUIDv7, "", false},

		// ULID — prefix ignored
		{"ULID valid", "01ARYZ6S41TSV4RRFFQ69G5FAV", FormatULID, "", true},
		{"ULID wrong length", "01ARYZ6S41TSV4RRFFQ69G5FA", FormatULID, "", false},
		{"ULID invalid char I", "01ARYZ6S41TSV4RRFFQI9G5FAV", FormatULID, "", false},

		// KSUID — prefix ignored
		{"KSUID valid", "2E8JwMKbBEgHvAsDFNNeyamgmCi", FormatKSUID, "", true},
		{"KSUID wrong length", "2E8JwMKbBEgHvAsDFNNeyamgm", FormatKSUID, "", false},

		// Unknown format
		{"unknown format", "anything", "unknown", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsValidID(tt.id, tt.format, tt.prefix)
			if got != tt.want {
				t.Errorf("IsValidID(%q, %q, %q) = %v, want %v", tt.id, tt.format, tt.prefix, got, tt.want)
			}
		})
	}
}
