package project

import "testing"

func TestIsValidID(t *testing.T) {
	tests := []struct {
		name   string
		id     string
		format string
		want   bool
	}{
		// aYYYYMMDDb format
		{"aYMDb valid single prefix", "p20250101a", FormatAYMDb, true},
		{"aYMDb valid double prefix", "ab20250101a", FormatAYMDb, true},
		{"aYMDb valid triple suffix", "p20250101abc", FormatAYMDb, true},
		{"aYMDb no prefix letter", "20250101a", FormatAYMDb, false},
		{"aYMDb no suffix letter", "p20250101", FormatAYMDb, false},
		{"aYMDb too many prefix letters", "abc20250101a", FormatAYMDb, false},
		{"aYMDb too many suffix letters", "p20250101abcd", FormatAYMDb, false},
		{"aYMDb uppercase rejected", "P20250101a", FormatAYMDb, false},
		{"aYMDb short digits", "p2025011a", FormatAYMDb, false},
		{"aYMDb non-project name", "metadata", FormatAYMDb, false},

		// UUIDv7
		{"UUIDv7 valid", "01932c07-a9c3-7b2a-8b1c-432f0696a585", FormatUUIDv7, true},
		{"UUIDv7 wrong version", "01932c07-a9c3-4b2a-8b1c-432f0696a585", FormatUUIDv7, false},
		{"UUIDv7 plain string", "not-a-uuid", FormatUUIDv7, false},

		// ULID
		{"ULID valid", "01ARYZ6S41TSV4RRFFQ69G5FAV", FormatULID, true},
		{"ULID wrong length", "01ARYZ6S41TSV4RRFFQ69G5FA", FormatULID, false},
		{"ULID invalid char I", "01ARYZ6S41TSV4RRFFQI9G5FAV", FormatULID, false},

		// KSUID
		{"KSUID valid", "2E8JwMKbBEgHvAsDFNNeyamgmCi", FormatKSUID, true},
		{"KSUID wrong length", "2E8JwMKbBEgHvAsDFNNeyamgm", FormatKSUID, false},

		// Unknown format
		{"unknown format", "anything", "unknown", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsValidID(tt.id, tt.format)
			if got != tt.want {
				t.Errorf("IsValidID(%q, %q) = %v, want %v", tt.id, tt.format, got, tt.want)
			}
		})
	}
}
