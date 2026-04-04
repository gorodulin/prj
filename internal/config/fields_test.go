package config

import (
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
	if len(Fields) != 12 {
		t.Errorf("Fields has %d entries, want 12", len(Fields))
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
