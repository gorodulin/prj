package metadata

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func snapshotName(daysAgo int) string {
	t := time.Now().UTC().AddDate(0, 0, -daysAgo)
	return t.Format("20060102T150405Z") + ".json"
}

func touchSnapshot(t *testing.T, dir, name string) {
	t.Helper()
	os.WriteFile(filepath.Join(dir, name), []byte(`{"based_on":[],"version":1}`), 0644)
}

func listSnapshots(t *testing.T, dir string) []string {
	t.Helper()
	entries, _ := os.ReadDir(dir)
	var names []string
	for _, e := range entries {
		if IsSnapshotFilename(e.Name()) {
			names = append(names, e.Name())
		}
	}
	return names
}

func TestPurgeOldSnapshots(t *testing.T) {
	t.Run("disabled when retention is 0", func(t *testing.T) {
		dir := t.TempDir()
		touchSnapshot(t, dir, snapshotName(365))
		deleted, err := PurgeOldSnapshots(dir, 0)
		if err != nil {
			t.Fatal(err)
		}
		if deleted != 0 {
			t.Errorf("deleted %d, want 0", deleted)
		}
	})

	t.Run("deletes old snapshots", func(t *testing.T) {
		dir := t.TempDir()
		touchSnapshot(t, dir, snapshotName(200))
		touchSnapshot(t, dir, snapshotName(190))
		touchSnapshot(t, dir, snapshotName(100))
		touchSnapshot(t, dir, snapshotName(0))

		deleted, err := PurgeOldSnapshots(dir, 180)
		if err != nil {
			t.Fatal(err)
		}
		if deleted != 2 {
			t.Errorf("deleted %d, want 2", deleted)
		}
		remaining := listSnapshots(t, dir)
		if len(remaining) != 2 {
			t.Errorf("remaining %d, want 2", len(remaining))
		}
	})

	t.Run("keeps at least 2 survivors", func(t *testing.T) {
		dir := t.TempDir()
		touchSnapshot(t, dir, snapshotName(365))
		touchSnapshot(t, dir, snapshotName(300))
		touchSnapshot(t, dir, snapshotName(200))

		deleted, err := PurgeOldSnapshots(dir, 180)
		if err != nil {
			t.Fatal(err)
		}
		remaining := listSnapshots(t, dir)
		if len(remaining) < 2 {
			t.Errorf("remaining %d, want at least 2 (deleted %d)", len(remaining), deleted)
		}
	})

	t.Run("skips if only 2 snapshots", func(t *testing.T) {
		dir := t.TempDir()
		touchSnapshot(t, dir, snapshotName(365))
		touchSnapshot(t, dir, snapshotName(300))

		deleted, err := PurgeOldSnapshots(dir, 180)
		if err != nil {
			t.Fatal(err)
		}
		if deleted != 0 {
			t.Errorf("deleted %d, want 0 (should preserve minimum)", deleted)
		}
	})

	t.Run("skips if future timestamp found", func(t *testing.T) {
		dir := t.TempDir()
		touchSnapshot(t, dir, snapshotName(365))
		touchSnapshot(t, dir, snapshotName(200))
		touchSnapshot(t, dir, snapshotName(0))
		touchSnapshot(t, dir, snapshotName(-30)) // future

		deleted, err := PurgeOldSnapshots(dir, 180)
		if err != nil {
			t.Fatal(err)
		}
		if deleted != 0 {
			t.Errorf("deleted %d, want 0 (future timestamp = skip)", deleted)
		}
	})

	t.Run("missing dir is not an error", func(t *testing.T) {
		deleted, err := PurgeOldSnapshots(filepath.Join(t.TempDir(), "nope"), 180)
		if err != nil {
			t.Fatal(err)
		}
		if deleted != 0 {
			t.Errorf("deleted %d, want 0", deleted)
		}
	})

	t.Run("ignores non-snapshot files", func(t *testing.T) {
		dir := t.TempDir()
		touchSnapshot(t, dir, snapshotName(200))
		touchSnapshot(t, dir, snapshotName(100))
		touchSnapshot(t, dir, snapshotName(0))
		os.WriteFile(filepath.Join(dir, "notes.txt"), []byte("keep"), 0644)

		PurgeOldSnapshots(dir, 180)

		if _, err := os.Stat(filepath.Join(dir, "notes.txt")); err != nil {
			t.Error("non-snapshot file was deleted")
		}
	})
}

func TestParseSnapshotTimestamp(t *testing.T) {
	ts, err := parseSnapshotTimestamp("20251028T232210Z.json")
	if err != nil {
		t.Fatal(err)
	}
	if ts.Year() != 2025 || ts.Month() != 10 || ts.Day() != 28 {
		t.Errorf("unexpected date: %v", ts)
	}
}
