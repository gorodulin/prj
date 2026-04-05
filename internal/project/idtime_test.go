package project

import (
	"testing"
	"time"
)

func TestParseIDTime(t *testing.T) {
	tests := []struct {
		name   string
		id     string
		wantOK bool
		check  func(t *testing.T, got time.Time)
	}{
		{
			name:   "aYYYYMMDDb basic",
			id:     "p20260402a",
			wantOK: true,
			check: func(t *testing.T, got time.Time) {
				want := time.Date(2026, 4, 2, 0, 0, 0, 0, time.UTC)
				if !got.Equal(want) {
					t.Errorf("got %v, want %v", got, want)
				}
			},
		},
		{
			name:   "aYYYYMMDDb two-letter prefix",
			id:     "ab20231015a",
			wantOK: true,
			check: func(t *testing.T, got time.Time) {
				want := time.Date(2023, 10, 15, 0, 0, 0, 0, time.UTC)
				if !got.Equal(want) {
					t.Errorf("got %v, want %v", got, want)
				}
			},
		},
		{
			name:   "aYYYYMMDDb three-letter prefix",
			id:     "prj20260101b",
			wantOK: true,
			check: func(t *testing.T, got time.Time) {
				want := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
				if !got.Equal(want) {
					t.Errorf("got %v, want %v", got, want)
				}
			},
		},
		{
			name:   "aYYYYMMDDb prefix with dash separator",
			id:     "prj-20260101b",
			wantOK: true,
			check: func(t *testing.T, got time.Time) {
				want := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
				if !got.Equal(want) {
					t.Errorf("got %v, want %v", got, want)
				}
			},
		},
		{
			name:   "aYYYYMMDDb prefix with underscore separator",
			id:     "prj_20260101b",
			wantOK: true,
			check: func(t *testing.T, got time.Time) {
				want := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
				if !got.Equal(want) {
					t.Errorf("got %v, want %v", got, want)
				}
			},
		},
		{
			name:   "ULID round-trip",
			id:     "", // filled in init
			wantOK: true,
			check: func(t *testing.T, got time.Time) {
				// ULID has ms precision; check within 1 second of now.
				diff := time.Since(got)
				if diff < 0 || diff > time.Second {
					t.Errorf("ULID time %v is not close to now (diff %v)", got, diff)
				}
			},
		},
		{
			name:   "UUIDv7 round-trip",
			id:     "",
			wantOK: true,
			check: func(t *testing.T, got time.Time) {
				diff := time.Since(got)
				if diff < 0 || diff > time.Second {
					t.Errorf("UUIDv7 time %v is not close to now (diff %v)", got, diff)
				}
			},
		},
		{
			name:   "KSUID round-trip",
			id:     "",
			wantOK: true,
			check: func(t *testing.T, got time.Time) {
				// KSUID has second precision.
				diff := time.Since(got)
				if diff < 0 || diff > 2*time.Second {
					t.Errorf("KSUID time %v is not close to now (diff %v)", got, diff)
				}
			},
		},
		{
			name:   "unrecognized ID",
			id:     "not-a-project-id",
			wantOK: false,
		},
		{
			name:   "empty string",
			id:     "",
			wantOK: false,
		},
	}

	// Generate real IDs for round-trip tests.
	ulid, err := generateULID()
	if err != nil {
		t.Fatalf("generate ULID: %v", err)
	}
	tests[5].id = ulid

	uuid, err := generateUUIDv7()
	if err != nil {
		t.Fatalf("generate UUIDv7: %v", err)
	}
	tests[6].id = uuid

	ksuid, err := generateKSUID()
	if err != nil {
		t.Fatalf("generate KSUID: %v", err)
	}
	tests[7].id = ksuid

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := ParseIDTime(tt.id)
			if ok != tt.wantOK {
				t.Fatalf("ParseIDTime(%q) ok = %v, want %v", tt.id, ok, tt.wantOK)
			}
			if ok && tt.check != nil {
				tt.check(t, got)
			}
		})
	}
}

func TestParseIDTimeKnownULID(t *testing.T) {
	// ULID encoding: 2024-01-01T00:00:00.000Z = 1704067200000 ms
	// Encode as 10-char Crockford Base32 + 16 random chars.
	ms := uint64(1704067200000)
	const crockford = "0123456789ABCDEFGHJKMNPQRSTVWXYZ"
	var ts [10]byte
	for i := 9; i >= 0; i-- {
		ts[i] = crockford[ms&0x1F]
		ms >>= 5
	}
	id := string(ts[:]) + "0000000000000000" // 16 zero random chars

	got, ok := ParseIDTime(id)
	if !ok {
		t.Fatalf("ParseIDTime(%q) failed", id)
	}
	want := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	if !got.Equal(want) {
		t.Errorf("got %v, want %v", got, want)
	}
}
