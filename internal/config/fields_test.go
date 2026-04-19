package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestFieldsCount(t *testing.T) {
	// Fields exposes user-configurable fields only (excludes internal fields
	// like metadata_folder_suffix). Must be kept ≤ struct field count.
	numStructFields := reflect.TypeOf(Config{}).NumField()
	if len(Fields) > numStructFields {
		t.Errorf("Fields has %d entries, but Config only has %d fields", len(Fields), numStructFields)
	}
	if len(Fields) != 14 {
		t.Errorf("Fields has %d entries, want 14", len(Fields))
	}
}

func TestFieldByKeyFound(t *testing.T) {
	for _, f := range Fields {
		got, ok := FieldByKey(f.Key)
		if !ok {
			t.Errorf("FieldByKey(%q) not found", f.Key)
			continue
		}
		if got.Key != f.Key {
			t.Errorf("FieldByKey(%q).Key = %q", f.Key, got.Key)
		}
	}
}

func TestFieldByKeyNotFound(t *testing.T) {
	_, ok := FieldByKey("nonexistent_key")
	if ok {
		t.Error("FieldByKey(nonexistent_key) should return false")
	}
}

func TestFieldRoundTrip(t *testing.T) {
	tests := []struct {
		key   string
		value string
	}{
		{"projects_folder", "/tmp/projects"},
		{"metadata_folder", "/tmp/meta"},
		{"links_folder", "/tmp/links"},
		{"link_title_format", "{{.Title}}"},
		{"list_format", "json"},
		{"link_kind", "symlink"},
		{"link_sink_name", "unsorted"},
		{"link_comment_format", "{{.Tags}}"},
		{"project_id_type", "ULID"},
		{"project_id_prefix", "prj"},
		{"machine_name", "laptop"},
		{"machine_id", "abc123"},
		{"retention_days", "30"},
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			f, ok := FieldByKey(tt.key)
			if !ok {
				t.Fatalf("field %q not found", tt.key)
			}

			var cfg Config
			if err := f.Set(&cfg, tt.value); err != nil {
				t.Fatalf("Set: %v", err)
			}

			got := f.Get(&cfg)
			if got != tt.value {
				t.Errorf("Get = %q, want %q", got, tt.value)
			}
		})
	}
}

func TestRetentionDaysRejectsNonInteger(t *testing.T) {
	f, _ := FieldByKey("retention_days")
	var cfg Config
	if err := f.Set(&cfg, "abc"); err == nil {
		t.Error("expected error for non-integer retention_days")
	}
}

func TestSetFieldPreservesRawJSON(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	os.WriteFile(path, []byte(`{"projects_folder": "/tmp/p"}`), 0644)

	// Set a new key.
	if err := SetField(path, "machine_name", "laptop"); err != nil {
		t.Fatalf("SetField: %v", err)
	}

	// Read raw JSON and verify no defaults leaked.
	data, _ := os.ReadFile(path)
	var raw map[string]json.RawMessage
	json.Unmarshal(data, &raw)

	if _, ok := raw["metadata_folder_suffix"]; ok {
		t.Error("metadata_folder_suffix should not appear in saved file")
	}
	if _, ok := raw["project_id_type"]; ok {
		t.Error("project_id_type should not appear in saved file")
	}
	if _, ok := raw["machine_name"]; !ok {
		t.Error("machine_name should be in saved file")
	}
	if _, ok := raw["projects_folder"]; !ok {
		t.Error("projects_folder should be preserved in saved file")
	}
}

func TestSetFieldRemovesEmpty(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	os.WriteFile(path, []byte(`{"projects_folder": "/tmp/p", "machine_name": "old"}`), 0644)

	if err := SetField(path, "machine_name", ""); err != nil {
		t.Fatalf("SetField: %v", err)
	}

	data, _ := os.ReadFile(path)
	var raw map[string]json.RawMessage
	json.Unmarshal(data, &raw)

	if _, ok := raw["machine_name"]; ok {
		t.Error("machine_name should be removed when set to empty")
	}
}

func TestSetFieldValidates(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	os.WriteFile(path, []byte(`{}`), 0644)

	err := SetField(path, "projects_folder", "relative/path")
	if err == nil {
		t.Fatal("expected validation error for relative path")
	}
}

func TestSetFieldCreatesFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "subdir", "config.json")

	if err := SetField(path, "projects_folder", "/tmp/p"); err != nil {
		t.Fatalf("SetField: %v", err)
	}

	if _, err := os.Stat(path); err != nil {
		t.Errorf("config file not created: %v", err)
	}
}

func TestValidKeysHelp(t *testing.T) {
	help := ValidKeysHelp()
	for _, f := range Fields {
		if !strings.Contains(help, f.Key) {
			t.Errorf("ValidKeysHelp() missing key %q", f.Key)
		}
	}
}

func TestIsEmpty(t *testing.T) {
	var cfg Config

	for _, f := range Fields {
		if !f.IsEmpty(&cfg) {
			t.Errorf("field %q should be empty on zero Config", f.Key)
		}
	}

	cfg.ProjectsFolder = "/tmp"
	pf, _ := FieldByKey("projects_folder")
	if pf.IsEmpty(&cfg) {
		t.Error("projects_folder should not be empty after setting")
	}

	cfg.RetentionDays = 30
	rd, _ := FieldByKey("retention_days")
	if rd.IsEmpty(&cfg) {
		t.Error("retention_days should not be empty after setting to 30")
	}
}
