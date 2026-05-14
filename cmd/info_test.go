package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/gorodulin/prj/internal/config"
	"github.com/gorodulin/prj/internal/project"
)

func newInfoTestCfg(t *testing.T) config.Config {
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
	return cfg
}

func TestInfoResult_JSONShape(t *testing.T) {
	// Empty tag slice round-trips as [] not null.
	r := infoResult{
		ID:   "p20260101a",
		Path: "/tmp/p20260101a",
		Tags: []string{},
	}
	b, err := json.Marshal(r)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var got map[string]any
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	wantKeys := []string{"id", "title", "path", "tags", "local", "has_readme"}
	for _, k := range wantKeys {
		if _, ok := got[k]; !ok {
			t.Errorf("missing key %q in %s", k, b)
		}
	}
	// date / datetime_utc are omitempty and should be absent when empty.
	if _, ok := got["date"]; ok {
		t.Errorf("date should be omitted when empty: %s", b)
	}
	if _, ok := got["datetime_utc"]; ok {
		t.Errorf("datetime_utc should be omitted when empty: %s", b)
	}
	// tags must marshal as [] not null.
	if tags, ok := got["tags"].([]any); !ok {
		t.Errorf("tags not an array: %T", got["tags"])
	} else if len(tags) != 0 {
		t.Errorf("tags length = %d, want 0", len(tags))
	}
}

func TestInfoResult_JSONShape_WithDate(t *testing.T) {
	r := infoResult{
		ID:          "p20260101a",
		Tags:        []string{},
		Date:        "2026-01-01",
		DateTimeUTC: "2026-01-01T00:00:00Z",
	}
	b, _ := json.Marshal(r)
	var got map[string]any
	_ = json.Unmarshal(b, &got)
	if got["date"] != "2026-01-01" {
		t.Errorf("date = %v, want 2026-01-01", got["date"])
	}
	if got["datetime_utc"] != "2026-01-01T00:00:00Z" {
		t.Errorf("datetime_utc = %v", got["datetime_utc"])
	}
}

func TestInfoProject_Found_Local(t *testing.T) {
	cfg := newInfoTestCfg(t)
	id := "p20260101a"
	mustMkProjectDir(t, cfg, id)

	res, perr := infoProject(id, cfg)
	if perr != nil {
		t.Fatalf("unexpected error: %+v", perr)
	}
	if res.ID != id {
		t.Errorf("ID = %q, want %q", res.ID, id)
	}
	if !res.Local {
		t.Errorf("Local = false, want true")
	}
	if res.Tags == nil {
		t.Errorf("Tags is nil, want empty slice")
	}
	if want := filepath.Join(cfg.ProjectsFolder, id); res.Path != want {
		t.Errorf("Path = %q, want %q", res.Path, want)
	}
}

func TestInfoProject_Found_MetadataOnly(t *testing.T) {
	cfg := newInfoTestCfg(t)
	id := "p20260101a"
	mustSeedSnapshot(t, cfg, id, []string{"foo"})

	res, perr := infoProject(id, cfg)
	if perr != nil {
		t.Fatalf("unexpected error: %+v", perr)
	}
	if res.Local {
		t.Errorf("Local = true, want false (no local folder)")
	}
	if !sliceEqual(res.Tags, []string{"foo"}) {
		t.Errorf("Tags = %v, want [foo]", res.Tags)
	}
}

func TestInfoProject_Unknown(t *testing.T) {
	cfg := newInfoTestCfg(t)
	id := "p20260101a"

	_, perr := infoProject(id, cfg)
	if perr == nil {
		t.Fatal("expected *IDError, got nil")
	}
	if perr.Code != errUnknownID {
		t.Errorf("code = %q, want %q", perr.Code, errUnknownID)
	}
}

func TestInfoProject_InvalidIDFormat(t *testing.T) {
	cfg := newInfoTestCfg(t)
	_, perr := infoProject("bogus", cfg)
	if perr == nil {
		t.Fatal("expected *IDError, got nil")
	}
	if perr.Code != errInvalidIDFormat {
		t.Errorf("code = %q, want %q", perr.Code, errInvalidIDFormat)
	}
}

func TestInfoEnvelope_AllValid_JSON(t *testing.T) {
	env := Envelope[infoResult]{
		Results: []infoResult{
			{ID: "p20260101a", Tags: []string{}},
			{ID: "p20260102b", Tags: []string{}},
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
	results, _ := got["results"].([]any)
	if len(results) != 2 {
		t.Errorf("results length = %d, want 2", len(results))
	}
	errs, _ := got["errors"].([]any)
	if len(errs) != 0 {
		t.Errorf("errors length = %d, want 0", len(errs))
	}
}

func TestInfoEnvelope_Mixed_JSON(t *testing.T) {
	env := Envelope[infoResult]{
		Results: []infoResult{{ID: "p20260101a", Tags: []string{}}},
		Errors: []IDError{{
			ID:     "missing",
			Code:   errUnknownID,
			Reason: `project "missing" not found`,
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
	if results, _ := got["results"].([]any); len(results) != 1 {
		t.Errorf("results length = %d, want 1", len(results))
	}
	errs, _ := got["errors"].([]any)
	if len(errs) != 1 {
		t.Errorf("errors length = %d, want 1", len(errs))
	}
}

func TestInfo_Variadic_Mixed_JSONL(t *testing.T) {
	var buf bytes.Buffer
	results := []infoResult{{ID: "p20260101a", Tags: []string{}}}
	errs := []IDError{{
		ID:     "missing",
		Code:   errUnknownID,
		Reason: `project "missing" not found`,
	}}
	if err := writeJSONLinesTo(&buf, results, errs); err != nil {
		t.Fatalf("writeJSONLinesTo: %v", err)
	}
	lines := splitNonEmpty(buf.String())
	if len(lines) != 2 {
		t.Fatalf("got %d lines, want 2: %q", len(lines), buf.String())
	}

	var ok infoResult
	if err := json.Unmarshal([]byte(lines[0]), &ok); err != nil {
		t.Fatalf("unmarshal success line: %v", err)
	}
	if ok.ID != "p20260101a" {
		t.Errorf("success line id = %q, want p20260101a", ok.ID)
	}

	var errLine struct {
		ID    string `json:"id"`
		Error struct {
			Code   string `json:"code"`
			Reason string `json:"reason"`
		} `json:"error"`
	}
	if err := json.Unmarshal([]byte(lines[1]), &errLine); err != nil {
		t.Fatalf("unmarshal error line: %v", err)
	}
	if errLine.Error.Code != errUnknownID {
		t.Errorf("error line code = %q, want %q", errLine.Error.Code, errUnknownID)
	}
}
