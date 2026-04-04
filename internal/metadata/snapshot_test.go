package metadata

import (
	"os"
	"path/filepath"
	"testing"
)

func TestIsSnapshotFilename(t *testing.T) {
	tests := []struct {
		name string
		want bool
	}{
		{"20251028T232210Z.json", true},
		{"20260401T000000Z.json", true},
		{"20251028T232210Z.txt", false},
		{"snapshot.json", false},
		{"2025T232210Z.json", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsSnapshotFilename(tt.name); got != tt.want {
				t.Errorf("IsSnapshotFilename(%q) = %v, want %v", tt.name, got, tt.want)
			}
		})
	}
}

func TestReadSnapshots(t *testing.T) {
	dir := t.TempDir()

	writeJSON(t, dir, "20251028T232210Z.json", `{
		"based_on": [],
		"title_set": "First Title",
		"tags": ["infra"],
		"tags_added": ["infra"],
		"tags_removed": [],
		"machine_id": "newton",
		"machine_name": "Newton",
		"version": 1
	}`)
	writeJSON(t, dir, "20251104T232350Z.json", `{
		"based_on": ["20251028T232210Z.json"],
		"title_set": "Updated Title",
		"tags": ["infra", "server"],
		"tags_added": ["server"],
		"tags_removed": [],
		"machine_id": "newton",
		"machine_name": "Newton",
		"version": 1
	}`)
	// Non-snapshot file should be ignored
	writeJSON(t, dir, "notes.json", `{"foo": "bar"}`)

	snapshots, err := ReadSnapshots(dir)
	if err != nil {
		t.Fatalf("ReadSnapshots: %v", err)
	}

	if len(snapshots) != 2 {
		t.Fatalf("expected 2 snapshots, got %d", len(snapshots))
	}

	if snapshots[0].Filename != "20251028T232210Z.json" {
		t.Errorf("first snapshot = %q, want 20251028T232210Z.json", snapshots[0].Filename)
	}
	if snapshots[1].Filename != "20251104T232350Z.json" {
		t.Errorf("second snapshot = %q, want 20251104T232350Z.json", snapshots[1].Filename)
	}

	// Verify based_on normalization
	if len(snapshots[0].BasedOn) != 0 {
		t.Errorf("first snapshot based_on = %v, want empty", snapshots[0].BasedOn)
	}
	if len(snapshots[1].BasedOn) != 1 || snapshots[1].BasedOn[0] != "20251028T232210Z.json" {
		t.Errorf("second snapshot based_on = %v, want [20251028T232210Z.json]", snapshots[1].BasedOn)
	}
}

func TestReadSnapshotsNormalizeBasedOn(t *testing.T) {
	tests := []struct {
		name    string
		json    string
		wantLen int
	}{
		{"null", `{"based_on": null, "version": 1}`, 0},
		{"empty array", `{"based_on": [], "version": 1}`, 0},
		{"single string", `{"based_on": "parent.json", "version": 1}`, 1},
		{"array", `{"based_on": ["a.json", "b.json"], "version": 1}`, 2},
		{"missing field", `{"version": 1}`, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			writeJSON(t, dir, "20260101T000000Z.json", tt.json)

			snapshots, err := ReadSnapshots(dir)
			if err != nil {
				t.Fatalf("ReadSnapshots: %v", err)
			}
			if len(snapshots) != 1 {
				t.Fatalf("expected 1 snapshot, got %d", len(snapshots))
			}
			if len(snapshots[0].BasedOn) != tt.wantLen {
				t.Errorf("based_on length = %d, want %d", len(snapshots[0].BasedOn), tt.wantLen)
			}
		})
	}
}

func TestReadSnapshotsMissingDir(t *testing.T) {
	snapshots, err := ReadSnapshots(filepath.Join(t.TempDir(), "nonexistent"))
	if err != nil {
		t.Fatalf("expected nil error for missing dir, got: %v", err)
	}
	if snapshots != nil {
		t.Errorf("expected nil snapshots, got %v", snapshots)
	}
}

func TestReadSnapshotsTitleSetNull(t *testing.T) {
	dir := t.TempDir()
	writeJSON(t, dir, "20260101T000000Z.json", `{
		"based_on": [],
		"title_set": null,
		"tags": ["tag1"],
		"version": 1
	}`)

	snapshots, err := ReadSnapshots(dir)
	if err != nil {
		t.Fatalf("ReadSnapshots: %v", err)
	}
	if snapshots[0].TitleSet != nil {
		t.Errorf("expected nil TitleSet, got %q", *snapshots[0].TitleSet)
	}
}

func writeJSON(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}
