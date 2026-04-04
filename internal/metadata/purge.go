package metadata

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const minSurvivors = 2

// PurgeOldSnapshots deletes snapshots older than retentionDays in the given
// metadata directory. Safe by design:
//   - Each snapshot carries full state, so any survivor is a valid head.
//   - Always keeps at least 2 snapshots regardless of age.
//   - Skips purging entirely if any snapshot has a future timestamp (clock issue).
//   - retentionDays <= 0 disables purging (returns 0, nil).
func PurgeOldSnapshots(dir string, retentionDays int) (int, error) {
	if retentionDays <= 0 {
		return 0, nil
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, fmt.Errorf("read metadata dir %s: %w", dir, err)
	}

	// Collect valid snapshot filenames.
	var filenames []string
	for _, e := range entries {
		if !e.IsDir() && IsSnapshotFilename(e.Name()) {
			filenames = append(filenames, e.Name())
		}
	}

	if len(filenames) <= minSurvivors {
		return 0, nil
	}

	now := time.Now().UTC()
	cutoff := now.AddDate(0, 0, -retentionDays)

	// Safety: if any snapshot has a future timestamp, skip purging entirely.
	for _, name := range filenames {
		ts, err := parseSnapshotTimestamp(name)
		if err != nil {
			continue
		}
		if ts.After(now) {
			return 0, nil
		}
	}

	// Identify candidates for deletion (older than cutoff).
	var toDelete []string
	for _, name := range filenames {
		ts, err := parseSnapshotTimestamp(name)
		if err != nil {
			continue
		}
		if ts.Before(cutoff) {
			toDelete = append(toDelete, name)
		}
	}

	// Ensure at least minSurvivors remain.
	survivors := len(filenames) - len(toDelete)
	if survivors < minSurvivors {
		// Keep the newest candidates to meet the minimum.
		excess := minSurvivors - survivors
		toDelete = toDelete[:len(toDelete)-excess]
	}

	if len(toDelete) == 0 {
		return 0, nil
	}

	deleted := 0
	for _, name := range toDelete {
		if err := os.Remove(filepath.Join(dir, name)); err != nil {
			return deleted, fmt.Errorf("delete snapshot %s: %w", name, err)
		}
		deleted++
	}

	return deleted, nil
}

// parseSnapshotTimestamp extracts the UTC time from a snapshot filename.
func parseSnapshotTimestamp(name string) (time.Time, error) {
	// "20251028T232210Z.json" → "20251028T232210Z"
	if len(name) < 16 {
		return time.Time{}, fmt.Errorf("filename too short: %s", name)
	}
	return time.Parse("20060102T150405Z", name[:16])
}
