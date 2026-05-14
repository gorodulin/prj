package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"

	"github.com/gorodulin/prj/internal/config"
	"github.com/gorodulin/prj/internal/metadata"
	"github.com/gorodulin/prj/internal/project"
)

func TestEditEnvelopeJSON_Empty(t *testing.T) {
	env := Envelope[editResult]{
		Results: []editResult{},
		Errors:  []IDError{},
	}
	got, err := json.Marshal(env)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	want := `{"error":null,"results":[],"errors":[]}`
	if string(got) != want {
		t.Errorf("marshal = %s, want %s", got, want)
	}
}

func TestEditEnvelopeJSON_RoundTrip(t *testing.T) {
	tests := []struct {
		name string
		env  Envelope[editResult]
	}{
		{
			name: "success with one result, empty tag slices",
			env: Envelope[editResult]{
				Results: []editResult{{
					ID:          "p20260101a",
					Status:      "updated",
					Title:       "hello",
					Path:        "/tmp/p20260101a",
					Local:       true,
					Tags:        []string{},
					TagsAdded:   []string{},
					TagsRemoved: []string{},
				}},
				Errors: []IDError{},
			},
		},
		{
			name: "success with populated tags",
			env: Envelope[editResult]{
				Results: []editResult{{
					ID:          "p20260101a",
					Status:      "created",
					Title:       "t",
					Path:        "/tmp/p",
					Local:       false,
					Tags:        []string{"a", "b"},
					TagsAdded:   []string{"a"},
					TagsRemoved: []string{"old"},
				}},
				Errors: []IDError{},
			},
		},
		{
			name: "top-level error envelope",
			env: Envelope[editResult]{
				Error:   &TopError{Code: errNoFlagsProvided, Reason: "nothing to edit"},
				Results: []editResult{},
				Errors:  []IDError{},
			},
		},
		{
			name: "per-id error envelope",
			env: Envelope[editResult]{
				Results: []editResult{},
				Errors: []IDError{{
					ID:     "p20260101a",
					Code:   errUnknownID,
					Reason: "unknown project p20260101a",
				}},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b, err := json.Marshal(tt.env)
			if err != nil {
				t.Fatalf("marshal: %v", err)
			}
			var got Envelope[editResult]
			if err := json.Unmarshal(b, &got); err != nil {
				t.Fatalf("unmarshal: %v", err)
			}

			// Empty tag slices must round-trip as [] not null.
			for i, r := range got.Results {
				if r.Tags == nil {
					t.Errorf("result[%d].Tags is nil after round-trip", i)
				}
				if r.TagsAdded == nil {
					t.Errorf("result[%d].TagsAdded is nil after round-trip", i)
				}
				if r.TagsRemoved == nil {
					t.Errorf("result[%d].TagsRemoved is nil after round-trip", i)
				}
			}

			// Re-marshal and compare for exact preservation.
			b2, err := json.Marshal(got)
			if err != nil {
				t.Fatalf("re-marshal: %v", err)
			}
			if string(b) != string(b2) {
				t.Errorf("round-trip mismatch:\n  before: %s\n  after:  %s", b, b2)
			}
		})
	}
}

func strPtr(s string) *string { return &s }

func tagsPtr(t ...string) *[]string {
	if t == nil {
		t = []string{}
	}
	return &t
}

func TestValidateIntent_NoFlags(t *testing.T) {
	code, _, ok := validateIntent(editIntent{}, false)
	if ok {
		t.Fatal("expected validation to fail for empty intent")
	}
	if code != errNoFlagsProvided {
		t.Errorf("code = %q, want %q", code, errNoFlagsProvided)
	}
}

func TestValidateIntent_TagsConflict(t *testing.T) {
	tests := []struct {
		name   string
		intent editIntent
	}{
		{
			name:   "tags + add-tags",
			intent: editIntent{tagsReplace: tagsPtr("a"), tagsAdd: []string{"b"}},
		},
		{
			name:   "tags + remove-tags",
			intent: editIntent{tagsReplace: tagsPtr("a"), tagsRemove: []string{"b"}},
		},
		{
			name:   "tags + add-tags + remove-tags",
			intent: editIntent{tagsReplace: tagsPtr("a"), tagsAdd: []string{"b"}, tagsRemove: []string{"c"}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code, _, ok := validateIntent(tt.intent, false)
			if ok {
				t.Fatal("expected validation to fail")
			}
			if code != errFlagsConflict {
				t.Errorf("code = %q, want %q", code, errFlagsConflict)
			}
		})
	}
}

func TestValidateIntent_Valid(t *testing.T) {
	tests := []struct {
		name   string
		intent editIntent
	}{
		{name: "title only", intent: editIntent{titleSet: strPtr("hello")}},
		{name: "title empty (clear)", intent: editIntent{titleSet: strPtr("")}},
		{name: "tags replace", intent: editIntent{tagsReplace: tagsPtr("a", "b")}},
		{name: "tags replace empty (clear)", intent: editIntent{tagsReplace: tagsPtr()}},
		{name: "add-tags only", intent: editIntent{tagsAdd: []string{"a"}}},
		{name: "remove-tags only", intent: editIntent{tagsRemove: []string{"a"}}},
		{name: "add + remove", intent: editIntent{tagsAdd: []string{"a"}, tagsRemove: []string{"b"}}},
		{name: "title + add-tags", intent: editIntent{titleSet: strPtr("t"), tagsAdd: []string{"a"}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code, reason, ok := validateIntent(tt.intent, false)
			if !ok {
				t.Errorf("expected valid, got code=%q reason=%q", code, reason)
			}
		})
	}
}

func TestAddToTags(t *testing.T) {
	tests := []struct {
		name     string
		existing []string
		add      []string
		want     []string
	}{
		{
			name:     "add new tags",
			existing: []string{"a", "b"},
			add:      []string{"c", "d"},
			want:     []string{"a", "b", "c", "d"},
		},
		{
			name:     "add overlapping tags",
			existing: []string{"a", "b"},
			add:      []string{"b", "c"},
			want:     []string{"a", "b", "c"},
		},
		{
			name:     "add to empty",
			existing: nil,
			add:      []string{"a", "b"},
			want:     []string{"a", "b"},
		},
		{
			name:     "add empty",
			existing: []string{"a", "b"},
			add:      nil,
			want:     []string{"a", "b"},
		},
		{
			name:     "add all duplicates",
			existing: []string{"a", "b"},
			add:      []string{"a", "b"},
			want:     []string{"a", "b"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := addToTags(tt.existing, tt.add)
			if !sliceEqual(got, tt.want) {
				t.Errorf("addToTags(%v, %v) = %v, want %v", tt.existing, tt.add, got, tt.want)
			}
		})
	}
}

func TestRemoveFromTags(t *testing.T) {
	tests := []struct {
		name     string
		existing []string
		remove   []string
		want     []string
	}{
		{
			name:     "remove existing tags",
			existing: []string{"a", "b", "c"},
			remove:   []string{"b"},
			want:     []string{"a", "c"},
		},
		{
			name:     "remove absent tags",
			existing: []string{"a", "b"},
			remove:   []string{"x", "y"},
			want:     []string{"a", "b"},
		},
		{
			name:     "remove all tags",
			existing: []string{"a", "b"},
			remove:   []string{"a", "b"},
			want:     []string{},
		},
		{
			name:     "remove from empty",
			existing: nil,
			remove:   []string{"a"},
			want:     []string{},
		},
		{
			name:     "remove empty",
			existing: []string{"a", "b"},
			remove:   nil,
			want:     []string{"a", "b"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := removeFromTags(tt.existing, tt.remove)
			if !sliceEqual(got, tt.want) {
				t.Errorf("removeFromTags(%v, %v) = %v, want %v", tt.existing, tt.remove, got, tt.want)
			}
		})
	}
}

func TestValidateIntent_TitleInBulk(t *testing.T) {
	intent := editIntent{titleSet: strPtr("hello")}

	if code, _, ok := validateIntent(intent, false); !ok {
		t.Errorf("single-ID --title rejected: code=%q", code)
	}

	code, _, ok := validateIntent(intent, true)
	if ok {
		t.Fatal("expected --title in bulk to fail validation")
	}
	if code != errTitleUnsupportedBulk {
		t.Errorf("code = %q, want %q", code, errTitleUnsupportedBulk)
	}
}

func newEditTestCfg(t *testing.T) (config.Config, string) {
	t.Helper()
	tmp := t.TempDir()
	cfg := config.Config{
		ProjectsFolder:  filepath.Join(tmp, "Projects"),
		MetadataFolder:  filepath.Join(tmp, "Metadata"),
		MetadataSuffix:  ".meta",
		MachineID:       "test",
		MachineName:     "test",
		RetentionDays:   30,
		ProjectIDType:   project.FormatAYMDb,
		ProjectIDPrefix: "p",
	}
	if err := os.MkdirAll(cfg.ProjectsFolder, 0o755); err != nil {
		t.Fatalf("mkdir Projects: %v", err)
	}
	if err := os.MkdirAll(cfg.MetadataFolder, 0o755); err != nil {
		t.Fatalf("mkdir Metadata: %v", err)
	}
	return cfg, tmp
}

func mustMkProjectDir(t *testing.T, cfg config.Config, id string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(cfg.ProjectsFolder, id), 0o755); err != nil {
		t.Fatalf("mkdir project %s: %v", id, err)
	}
}

// mustSeedSnapshot writes a snapshot with a fixed older timestamp so a
// subsequent same-second WriteSnapshot from editProject doesn't overwrite it.
func mustSeedSnapshot(t *testing.T, cfg config.Config, id string, tags []string) {
	t.Helper()
	dir := cfg.MetadataDir(id)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir metadata for %s: %v", id, err)
	}
	if tags == nil {
		tags = []string{}
	}
	payload := map[string]any{
		"based_on":     []string{},
		"title_set":    nil,
		"tags":         tags,
		"tags_added":   tags,
		"tags_removed": []string{},
		"machine_id":   cfg.MachineID,
		"machine_name": cfg.MachineName,
		"version":      1,
	}
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		t.Fatalf("marshal seed: %v", err)
	}
	path := filepath.Join(dir, "20200101T000000Z.json")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("write seed for %s: %v", id, err)
	}
}

func countSnapshotFiles(t *testing.T, dir string) int {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return 0
		}
		t.Fatalf("readdir %s: %v", dir, err)
	}
	n := 0
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if metadata.IsSnapshotFilename(e.Name()) {
			n++
		}
	}
	return n
}

func TestEditProject_FolderPlusMetadata_AddMissingTag(t *testing.T) {
	cfg, _ := newEditTestCfg(t)
	id := "p20260101a"
	mustMkProjectDir(t, cfg, id)
	mustSeedSnapshot(t, cfg, id, []string{"existing"})

	intent := editIntent{tagsAdd: []string{"foo"}}
	res, perr := editProject(id, intent, cfg)
	if perr != nil {
		t.Fatalf("unexpected error: %+v", perr)
	}
	if res.Status != "updated" {
		t.Errorf("status = %q, want updated", res.Status)
	}
	if !sliceEqual(res.TagsAdded, []string{"foo"}) {
		t.Errorf("TagsAdded = %v, want [foo]", res.TagsAdded)
	}
	if !res.Local {
		t.Errorf("Local = false, want true")
	}
	if want := filepath.Join(cfg.ProjectsFolder, id); res.Path != want {
		t.Errorf("Path = %q, want %q", res.Path, want)
	}
	if got := countSnapshotFiles(t, cfg.MetadataDir(id)); got != 2 {
		t.Errorf("snapshot files = %d, want 2 (seed + new)", got)
	}
}

func TestEditProject_FolderPlusMetadata_AddPresentTag(t *testing.T) {
	cfg, _ := newEditTestCfg(t)
	id := "p20260101a"
	mustMkProjectDir(t, cfg, id)
	mustSeedSnapshot(t, cfg, id, []string{"foo"})

	before := countSnapshotFiles(t, cfg.MetadataDir(id))
	intent := editIntent{tagsAdd: []string{"foo"}}
	res, perr := editProject(id, intent, cfg)
	if perr != nil {
		t.Fatalf("unexpected error: %+v", perr)
	}
	if res.Status != "unchanged" {
		t.Errorf("status = %q, want unchanged", res.Status)
	}
	if len(res.TagsAdded) != 0 {
		t.Errorf("TagsAdded = %v, want []", res.TagsAdded)
	}
	if len(res.TagsRemoved) != 0 {
		t.Errorf("TagsRemoved = %v, want []", res.TagsRemoved)
	}
	if got := countSnapshotFiles(t, cfg.MetadataDir(id)); got != before {
		t.Errorf("snapshot files = %d, want %d (no new snapshot)", got, before)
	}
}

func TestEditProject_FolderNoMetadata_NoForce(t *testing.T) {
	cfg, _ := newEditTestCfg(t)
	id := "p20260101a"
	mustMkProjectDir(t, cfg, id)

	intent := editIntent{tagsAdd: []string{"foo"}}
	res, perr := editProject(id, intent, cfg)
	if perr != nil {
		t.Fatalf("unexpected error: %+v", perr)
	}
	if res.Status != "updated" {
		t.Errorf("status = %q, want updated", res.Status)
	}
	if got := countSnapshotFiles(t, cfg.MetadataDir(id)); got != 1 {
		t.Errorf("snapshot files = %d, want 1", got)
	}
}

func TestEditProject_NoFolderNoMetadata_NoForce(t *testing.T) {
	cfg, _ := newEditTestCfg(t)
	id := "p20260101a"

	intent := editIntent{tagsAdd: []string{"foo"}}
	_, perr := editProject(id, intent, cfg)
	if perr == nil {
		t.Fatal("expected IDError, got nil")
	}
	if perr.Code != errUnknownID {
		t.Errorf("code = %q, want %q", perr.Code, errUnknownID)
	}
}

func TestEditProject_NoFolderNoMetadata_Force(t *testing.T) {
	cfg, _ := newEditTestCfg(t)
	id := "p20260101a"

	intent := editIntent{tagsAdd: []string{"foo"}, force: true}
	res, perr := editProject(id, intent, cfg)
	if perr != nil {
		t.Fatalf("unexpected error: %+v", perr)
	}
	if res.Status != "created" {
		t.Errorf("status = %q, want created", res.Status)
	}
	if res.Local {
		t.Errorf("Local = true, want false (project folder doesn't exist)")
	}
	if got := countSnapshotFiles(t, cfg.MetadataDir(id)); got != 1 {
		t.Errorf("snapshot files = %d, want 1", got)
	}
}

func TestEditProject_InvalidIDFormat(t *testing.T) {
	cfg, _ := newEditTestCfg(t)
	intent := editIntent{tagsAdd: []string{"foo"}, force: true}
	_, perr := editProject("bogus", intent, cfg)
	if perr == nil {
		t.Fatal("expected IDError, got nil")
	}
	if perr.Code != errInvalidIDFormat {
		t.Errorf("code = %q, want %q", perr.Code, errInvalidIDFormat)
	}
}

func TestEditProject_TagsReplacement(t *testing.T) {
	cfg, _ := newEditTestCfg(t)
	id := "p20260101a"
	mustMkProjectDir(t, cfg, id)
	mustSeedSnapshot(t, cfg, id, []string{"a", "b"})

	replace := []string{"c"}
	intent := editIntent{tagsReplace: &replace}
	res, perr := editProject(id, intent, cfg)
	if perr != nil {
		t.Fatalf("unexpected error: %+v", perr)
	}
	if !sliceEqual(res.Tags, []string{"c"}) {
		t.Errorf("Tags = %v, want [c]", res.Tags)
	}
	if !sliceEqual(res.TagsAdded, []string{"c"}) {
		t.Errorf("TagsAdded = %v, want [c]", res.TagsAdded)
	}
	if !sliceEqual(res.TagsRemoved, []string{"a", "b"}) {
		t.Errorf("TagsRemoved = %v, want [a b]", res.TagsRemoved)
	}
}

func TestEditProject_DryRun_NoSnapshotWritten(t *testing.T) {
	cfg, _ := newEditTestCfg(t)
	id := "p20260101a"
	mustMkProjectDir(t, cfg, id)
	mustSeedSnapshot(t, cfg, id, []string{"existing"})

	before := countSnapshotFiles(t, cfg.MetadataDir(id))
	intent := editIntent{tagsAdd: []string{"new"}, dryRun: true}
	res, perr := editProject(id, intent, cfg)
	if perr != nil {
		t.Fatalf("unexpected error: %+v", perr)
	}
	if res.Status != "updated" {
		t.Errorf("status = %q, want updated (status reflects intended effect)", res.Status)
	}
	if !sliceEqual(res.TagsAdded, []string{"new"}) {
		t.Errorf("TagsAdded = %v, want [new]", res.TagsAdded)
	}
	if got := countSnapshotFiles(t, cfg.MetadataDir(id)); got != before {
		t.Errorf("snapshot files = %d, want %d (dry-run must not write)", got, before)
	}
}

func TestEditProject_DryRun_Unchanged_NoWrite(t *testing.T) {
	// Dry-run on an unchanged edit must still not write (no-op anyway,
	// but verify symmetry with non-dry-run unchanged path).
	cfg, _ := newEditTestCfg(t)
	id := "p20260101a"
	mustMkProjectDir(t, cfg, id)
	mustSeedSnapshot(t, cfg, id, []string{"foo"})

	before := countSnapshotFiles(t, cfg.MetadataDir(id))
	intent := editIntent{tagsAdd: []string{"foo"}, dryRun: true}
	res, perr := editProject(id, intent, cfg)
	if perr != nil {
		t.Fatalf("unexpected error: %+v", perr)
	}
	if res.Status != "unchanged" {
		t.Errorf("status = %q, want unchanged", res.Status)
	}
	if got := countSnapshotFiles(t, cfg.MetadataDir(id)); got != before {
		t.Errorf("snapshot files = %d, want %d", got, before)
	}
}

func TestEditProject_DryRun_Force_NoMetadataDirCreated(t *testing.T) {
	// --force + --dry-run on an unknown ID: returns "created" status,
	// but no metadata file is materialized on disk.
	cfg, _ := newEditTestCfg(t)
	id := "p20260101a"

	intent := editIntent{tagsAdd: []string{"foo"}, force: true, dryRun: true}
	res, perr := editProject(id, intent, cfg)
	if perr != nil {
		t.Fatalf("unexpected error: %+v", perr)
	}
	if res.Status != "created" {
		t.Errorf("status = %q, want created", res.Status)
	}
	if got := countSnapshotFiles(t, cfg.MetadataDir(id)); got != 0 {
		t.Errorf("snapshot files = %d, want 0 (dry-run must not write)", got)
	}
}

func TestEditEnvelopeJSON_DryRun_True(t *testing.T) {
	env := Envelope[editResult]{
		DryRun:  true,
		Results: []editResult{},
		Errors:  []IDError{},
	}
	got, err := json.Marshal(env)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	want := `{"error":null,"dry_run":true,"results":[],"errors":[]}`
	if string(got) != want {
		t.Errorf("marshal = %s, want %s", got, want)
	}
}

func TestEditEnvelopeJSON_DryRun_FalseOmitted(t *testing.T) {
	// DryRun=false (zero value) must be omitted from JSON to keep the
	// non-dry-run envelope byte-identical to pre-dry-run releases.
	env := Envelope[editResult]{
		DryRun:  false,
		Results: []editResult{},
		Errors:  []IDError{},
	}
	got, err := json.Marshal(env)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if want := `{"error":null,"results":[],"errors":[]}`; string(got) != want {
		t.Errorf("marshal = %s, want %s (dry_run must be omitted)", got, want)
	}
}

func TestEditEnvelope_BulkAllValid(t *testing.T) {
	env := Envelope[editResult]{
		Results: []editResult{
			{
				ID:          "p20260101a",
				Status:      "updated",
				Title:       "alpha",
				Path:        "/tmp/p20260101a",
				Local:       true,
				Tags:        []string{"foo"},
				TagsAdded:   []string{"foo"},
				TagsRemoved: []string{},
			},
			{
				ID:          "p20260102b",
				Status:      "unchanged",
				Title:       "beta",
				Path:        "/tmp/p20260102b",
				Local:       true,
				Tags:        []string{"foo"},
				TagsAdded:   []string{},
				TagsRemoved: []string{},
			},
		},
		Errors: []IDError{},
	}
	b, err := json.Marshal(env)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var got map[string]any
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got["error"] != nil {
		t.Errorf("error = %v, want nil", got["error"])
	}
	results, ok := got["results"].([]any)
	if !ok {
		t.Fatalf("results not an array: %T", got["results"])
	}
	if len(results) != 2 {
		t.Errorf("results length = %d, want 2", len(results))
	}
	errs, ok := got["errors"].([]any)
	if !ok {
		t.Fatalf("errors not an array: %T", got["errors"])
	}
	if len(errs) != 0 {
		t.Errorf("errors length = %d, want 0", len(errs))
	}
}

func TestEditEnvelope_BulkMixed(t *testing.T) {
	env := Envelope[editResult]{
		Results: []editResult{{
			ID:          "p20260101a",
			Status:      "updated",
			Title:       "alpha",
			Path:        "/tmp/p20260101a",
			Local:       true,
			Tags:        []string{"foo"},
			TagsAdded:   []string{"foo"},
			TagsRemoved: []string{},
		}},
		Errors: []IDError{{
			ID:     "bogus",
			Code:   errInvalidIDFormat,
			Reason: `invalid project ID "bogus"`,
		}},
	}
	b, err := json.Marshal(env)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var got map[string]any
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got["error"] != nil {
		t.Errorf("error = %v, want nil", got["error"])
	}
	results, _ := got["results"].([]any)
	if len(results) != 1 {
		t.Errorf("results length = %d, want 1", len(results))
	}
	errs, _ := got["errors"].([]any)
	if len(errs) != 1 {
		t.Errorf("errors length = %d, want 1", len(errs))
	}
	if e0, ok := errs[0].(map[string]any); ok {
		if e0["code"] != errInvalidIDFormat {
			t.Errorf("errors[0].code = %v, want %q", e0["code"], errInvalidIDFormat)
		}
	} else {
		t.Errorf("errors[0] not an object: %T", errs[0])
	}
}

func TestEditEnvelope_TopError(t *testing.T) {
	env := Envelope[editResult]{
		Error:   &TopError{Code: errTitleUnsupportedBulk, Reason: "--title is not supported when editing multiple projects"},
		Results: []editResult{},
		Errors:  []IDError{},
	}
	b, err := json.Marshal(env)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var got map[string]any
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	topErr, ok := got["error"].(map[string]any)
	if !ok {
		t.Fatalf("error not an object: %T", got["error"])
	}
	if topErr["code"] != errTitleUnsupportedBulk {
		t.Errorf("error.code = %v, want %q", topErr["code"], errTitleUnsupportedBulk)
	}
	results, ok := got["results"].([]any)
	if !ok {
		t.Fatalf("results not an array: %T", got["results"])
	}
	if len(results) != 0 {
		t.Errorf("results length = %d, want 0", len(results))
	}
	errs, ok := got["errors"].([]any)
	if !ok {
		t.Fatalf("errors not an array: %T", got["errors"])
	}
	if len(errs) != 0 {
		t.Errorf("errors length = %d, want 0", len(errs))
	}
}

func TestEdit_JSONL(t *testing.T) {
	cfg, _ := newEditTestCfg(t)
	idOK := "p20260101a"
	mustMkProjectDir(t, cfg, idOK)
	mustSeedSnapshot(t, cfg, idOK, []string{})

	intent := editIntent{tagsAdd: []string{"jsonl"}}

	resOK, perr := editProject(idOK, intent, cfg)
	if perr != nil {
		t.Fatalf("unexpected error for %s: %+v", idOK, perr)
	}

	_, perrBad := editProject("p20260102b", intent, cfg)
	if perrBad == nil {
		t.Fatal("expected error for unknown ID, got nil")
	}

	var buf bytes.Buffer
	if err := writeJSONLinesTo(&buf, []editResult{resOK}, []IDError{*perrBad}); err != nil {
		t.Fatalf("writeJSONLinesTo: %v", err)
	}

	lines := splitNonEmpty(buf.String())
	if len(lines) != 2 {
		t.Fatalf("got %d lines, want 2: %q", len(lines), buf.String())
	}

	var first editResult
	if err := json.Unmarshal([]byte(lines[0]), &first); err != nil {
		t.Fatalf("unmarshal success line: %v", err)
	}
	if first.ID != idOK {
		t.Errorf("success line id = %q, want %q", first.ID, idOK)
	}
	if first.Status != "updated" {
		t.Errorf("success line status = %q, want updated", first.Status)
	}

	var second struct {
		ID    string `json:"id"`
		Error struct {
			Code   string `json:"code"`
			Reason string `json:"reason"`
		} `json:"error"`
	}
	if err := json.Unmarshal([]byte(lines[1]), &second); err != nil {
		t.Fatalf("unmarshal error line: %v", err)
	}
	if second.Error.Code != errUnknownID {
		t.Errorf("error line code = %q, want %q", second.Error.Code, errUnknownID)
	}
}

func TestEditIntent_TagAlias(t *testing.T) {
	// When --tag is provided (and --tags is not), buildIntent should
	// resolve tagsReplace from --tag.
	intent := buildIntent(makeEditCmdWith(t, map[string]string{"tag": "wip,cli"}))
	if intent.tagsReplace == nil {
		t.Fatalf("expected tagsReplace to be set when --tag is used; got nil")
	}
	want := []string{"cli", "wip"}
	if !sliceEqual(*intent.tagsReplace, want) {
		t.Errorf("tagsReplace = %v, want %v", *intent.tagsReplace, want)
	}
}

func TestEditIntent_TagsPreferredOverTag(t *testing.T) {
	// Both flags simultaneously is rejected by cobra's mutual exclusion
	// at the command level. At the buildIntent layer (called only after
	// flag parsing succeeds), only one of them can be Changed, but we
	// guard the read order: --tags wins if set, --tag otherwise.
	intent := buildIntent(makeEditCmdWith(t, map[string]string{"tags": "real"}))
	if intent.tagsReplace == nil {
		t.Fatalf("expected tagsReplace from --tags, got nil")
	}
	if !sliceEqual(*intent.tagsReplace, []string{"real"}) {
		t.Errorf("tagsReplace = %v, want [real]", *intent.tagsReplace)
	}
}

// makeEditCmdWith builds a throwaway *cobra.Command with the same flag
// definitions as editCmd, marks each named flag as Changed, and sets its
// value. Lets buildIntent be exercised without touching global cmd state.
func makeEditCmdWith(t *testing.T, set map[string]string) *cobra.Command {
	t.Helper()
	c := &cobra.Command{Use: "edit"}
	c.Flags().String("title", "", "")
	c.Flags().String("tags", "", "")
	c.Flags().String("tag", "", "")
	c.Flags().String("add-tags", "", "")
	c.Flags().String("remove-tags", "", "")
	c.Flags().Bool("force", false, "")
	c.Flags().Bool("dry-run", false, "")
	for name, value := range set {
		if err := c.Flags().Set(name, value); err != nil {
			t.Fatalf("set %s=%q: %v", name, value, err)
		}
	}
	return c
}
