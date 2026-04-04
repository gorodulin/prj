package project

import (
	"testing"
	"time"
)

func TestBase26Suffix(t *testing.T) {
	tests := []struct {
		n    int
		want string
	}{
		{0, "a"}, {1, "b"}, {25, "z"},
		{26, "aa"}, {27, "ab"}, {51, "az"},
		{52, "ba"}, {701, "zz"},
		{702, "aaa"},
	}

	for _, tt := range tests {
		got := base26Suffix(tt.n)
		if got != tt.want {
			t.Errorf("base26Suffix(%d) = %q, want %q", tt.n, got, tt.want)
		}
	}
}

func TestGenerateID(t *testing.T) {
	today := time.Now().UTC().Format("20060102")

	t.Run("first ID for today", func(t *testing.T) {
		id, err := GenerateID(FormatAYMDb, nil)
		if err != nil {
			t.Fatal(err)
		}
		want := "p" + today + "a"
		if id != want {
			t.Errorf("got %q, want %q", id, want)
		}
	})

	t.Run("skips existing", func(t *testing.T) {
		existing := []string{
			"p" + today + "a",
			"p" + today + "b",
		}
		id, err := GenerateID(FormatAYMDb, existing)
		if err != nil {
			t.Fatal(err)
		}
		want := "p" + today + "c"
		if id != want {
			t.Errorf("got %q, want %q", id, want)
		}
	})

	t.Run("skips non-contiguous", func(t *testing.T) {
		existing := []string{
			"p" + today + "a",
			"p" + today + "c", // b is free
		}
		id, err := GenerateID(FormatAYMDb, existing)
		if err != nil {
			t.Fatal(err)
		}
		want := "p" + today + "b"
		if id != want {
			t.Errorf("got %q, want %q", id, want)
		}
	})

	t.Run("unsupported format", func(t *testing.T) {
		_, err := GenerateID("unknown", nil)
		if err == nil {
			t.Fatal("expected error for unsupported format")
		}
	})

	t.Run("UUIDv7 generates valid ID", func(t *testing.T) {
		for i := 0; i < 100; i++ {
			id, err := GenerateID(FormatUUIDv7, nil)
			if err != nil {
				t.Fatal(err)
			}
			if !IsValidID(id, FormatUUIDv7) {
				t.Fatalf("generated UUIDv7 %q does not pass validation", id)
			}
		}
	})

	t.Run("UUIDv7 has version 7 and variant bits", func(t *testing.T) {
		id, err := GenerateID(FormatUUIDv7, nil)
		if err != nil {
			t.Fatal(err)
		}
		// Format: xxxxxxxx-xxxx-7xxx-[89ab]xxx-xxxxxxxxxxxx
		if id[14] != '7' {
			t.Errorf("version nibble = %c, want '7'", id[14])
		}
		variant := id[19]
		if variant != '8' && variant != '9' && variant != 'a' && variant != 'b' {
			t.Errorf("variant nibble = %c, want [89ab]", variant)
		}
	})

	t.Run("ULID generates valid ID", func(t *testing.T) {
		for i := 0; i < 100; i++ {
			id, err := GenerateID(FormatULID, nil)
			if err != nil {
				t.Fatal(err)
			}
			if !IsValidID(id, FormatULID) {
				t.Fatalf("generated ULID %q does not pass validation", id)
			}
		}
	})

	t.Run("ULID is 26 chars Crockford base32", func(t *testing.T) {
		id, err := GenerateID(FormatULID, nil)
		if err != nil {
			t.Fatal(err)
		}
		if len(id) != 26 {
			t.Errorf("len = %d, want 26", len(id))
		}
		// First char should be 0-7 (timestamp fits in 48 bits)
		if id[0] < '0' || id[0] > '7' {
			t.Errorf("first char = %c, want 0-7", id[0])
		}
	})

	t.Run("KSUID generates valid ID", func(t *testing.T) {
		for i := 0; i < 100; i++ {
			id, err := GenerateID(FormatKSUID, nil)
			if err != nil {
				t.Fatal(err)
			}
			if !IsValidID(id, FormatKSUID) {
				t.Fatalf("generated KSUID %q does not pass validation", id)
			}
		}
	})

	t.Run("KSUID is 27 chars base62", func(t *testing.T) {
		id, err := GenerateID(FormatKSUID, nil)
		if err != nil {
			t.Fatal(err)
		}
		if len(id) != 27 {
			t.Errorf("len = %d, want 27", len(id))
		}
	})

	t.Run("generated IDs are unique", func(t *testing.T) {
		for _, format := range []string{FormatUUIDv7, FormatULID, FormatKSUID} {
			seen := make(map[string]bool)
			for i := 0; i < 50; i++ {
				id, err := GenerateID(format, nil)
				if err != nil {
					t.Fatal(err)
				}
				if seen[id] {
					t.Fatalf("%s: duplicate ID %q", format, id)
				}
				seen[id] = true
			}
		}
	})
}
