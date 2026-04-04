package metadata

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"time"
)

var snapshotFilenameRe = regexp.MustCompile(`^\d{8}T\d{6}Z\.json$`)

// Snapshot represents a single metadata snapshot file.
type Snapshot struct {
	Filename    string   // e.g. "20251028T232210Z.json"
	BasedOn     []string `json:"based_on"`
	TitleSet    *string  `json:"title_set"`
	Tags        []string `json:"tags"`
	TagsAdded   []string `json:"tags_added"`
	TagsRemoved []string `json:"tags_removed"`
	MachineID   string   `json:"machine_id"`
	MachineName string   `json:"machine_name"`
	Version     int      `json:"version"`
}

// IsSnapshotFilename reports whether name matches the snapshot naming convention.
func IsSnapshotFilename(name string) bool {
	return snapshotFilenameRe.MatchString(name)
}

// ReadSnapshots reads all valid snapshot files from a metadata directory.
// Returns them sorted by filename (chronological order).
// Returns nil, nil if the directory does not exist.
func ReadSnapshots(dir string) ([]Snapshot, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read metadata dir %s: %w", dir, err)
	}

	var snapshots []Snapshot
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !IsSnapshotFilename(name) {
			continue
		}

		s, err := readSnapshotFile(filepath.Join(dir, name))
		if err != nil {
			return nil, fmt.Errorf("read snapshot %s: %w", name, err)
		}
		s.Filename = name
		snapshots = append(snapshots, s)
	}

	sort.Slice(snapshots, func(i, j int) bool {
		return snapshots[i].Filename < snapshots[j].Filename
	})

	return snapshots, nil
}

func readSnapshotFile(path string) (Snapshot, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Snapshot{}, err
	}

	var raw struct {
		BasedOn     json.RawMessage `json:"based_on"`
		TitleSet    *string         `json:"title_set"`
		Tags        []string        `json:"tags"`
		TagsAdded   []string        `json:"tags_added"`
		TagsRemoved []string        `json:"tags_removed"`
		MachineID   string          `json:"machine_id"`
		MachineName string          `json:"machine_name"`
		Version     int             `json:"version"`
	}

	if err := json.Unmarshal(data, &raw); err != nil {
		return Snapshot{}, err
	}

	s := Snapshot{
		TitleSet:    raw.TitleSet,
		Tags:        raw.Tags,
		TagsAdded:   raw.TagsAdded,
		TagsRemoved: raw.TagsRemoved,
		MachineID:   raw.MachineID,
		MachineName: raw.MachineName,
		Version:     raw.Version,
	}

	// Normalize based_on: null, string, or []string → []string
	s.BasedOn = normalizeBasedOn(raw.BasedOn)

	return s, nil
}

// BuildSnapshotFilename returns a snapshot filename for the current UTC time.
func BuildSnapshotFilename() string {
	return time.Now().UTC().Format("20060102T150405Z") + ".json"
}

// WriteSnapshot creates the metadata directory if needed and writes a snapshot file.
func WriteSnapshot(dir string, s Snapshot) (string, error) {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("create metadata dir %s: %w", dir, err)
	}

	filename := BuildSnapshotFilename()
	path := filepath.Join(dir, filename)

	// Write with based_on always as array for consistency.
	out := struct {
		BasedOn     []string `json:"based_on"`
		TitleSet    *string  `json:"title_set"`
		Tags        []string `json:"tags"`
		TagsAdded   []string `json:"tags_added"`
		TagsRemoved []string `json:"tags_removed"`
		MachineID   string   `json:"machine_id"`
		MachineName string   `json:"machine_name"`
		Version     int      `json:"version"`
	}{
		BasedOn:     s.BasedOn,
		TitleSet:    s.TitleSet,
		Tags:        s.Tags,
		TagsAdded:   s.TagsAdded,
		TagsRemoved: s.TagsRemoved,
		MachineID:   s.MachineID,
		MachineName: s.MachineName,
		Version:     s.Version,
	}

	// Ensure nil slices become [] in JSON, not null.
	if out.BasedOn == nil {
		out.BasedOn = []string{}
	}
	if out.Tags == nil {
		out.Tags = []string{}
	}
	if out.TagsAdded == nil {
		out.TagsAdded = []string{}
	}
	if out.TagsRemoved == nil {
		out.TagsRemoved = []string{}
	}

	data, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return "", fmt.Errorf("marshal snapshot: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return "", fmt.Errorf("write snapshot %s: %w", path, err)
	}

	return filename, nil
}

// normalizeBasedOn converts the flexible based_on field to a uniform []string.
func normalizeBasedOn(raw json.RawMessage) []string {
	if raw == nil || string(raw) == "null" {
		return nil
	}

	// Try as array first (most common in real data)
	var arr []string
	if json.Unmarshal(raw, &arr) == nil {
		return arr
	}

	// Try as single string
	var single string
	if json.Unmarshal(raw, &single) == nil {
		return []string{single}
	}

	return nil
}
