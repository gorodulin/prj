package project

import "testing"

func TestCompareIDs(t *testing.T) {
	tests := []struct {
		a, b string
		want int
	}{
		{"p20260402a", "p20260402b", -1},
		{"p20260402z", "p20260402aa", -1},
		{"p20260402aa", "p20260402z", 1},
		{"p20260402zz", "p20260402aaa", -1},
		{"p20260401z", "p20260402a", -1},
		{"p20260102aa", "p20260103a", -1},
		{"p20260103a", "p20260102aa", 1},
		{"p20260402a", "p20260402a", 0},
		{"01J9ZQ6X3H4K5M6N7P8Q9R0S1T", "01JA0000000000000000000000", -1},
		{"0192d4c1-0000-7000-8000-000000000000", "0192d4c2-0000-7000-8000-000000000000", -1},
	}
	for _, tt := range tests {
		if got := CompareIDs(tt.a, tt.b); got != tt.want {
			t.Errorf("CompareIDs(%q, %q) = %d, want %d", tt.a, tt.b, got, tt.want)
		}
	}
}
