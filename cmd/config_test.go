package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gorodulin/prj/internal/config"
)

func TestConfigSetAndGet(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")

	// Set a value.
	cfg := config.Config{}
	f, _ := config.FieldByKey("projects_folder")
	if err := f.Set(&cfg, "/tmp/projects"); err != nil {
		t.Fatalf("Set: %v", err)
	}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("Validate: %v", err)
	}
	if err := config.Save(cfg, path); err != nil {
		t.Fatalf("Save: %v", err)
	}

	// Get it back.
	loaded, err := config.Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	got := f.Get(&loaded)
	if got != "/tmp/projects" {
		t.Errorf("Get = %q, want %q", got, "/tmp/projects")
	}
}

func TestConfigSetUnknownKey(t *testing.T) {
	_, ok := config.FieldByKey("nonexistent")
	if ok {
		t.Error("expected unknown key to return false")
	}
}

func TestConfigSetValidationError(t *testing.T) {
	cfg := config.Config{}
	f, _ := config.FieldByKey("projects_folder")
	f.Set(&cfg, "relative/path")
	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected validation error for relative path")
	}
	if !strings.Contains(err.Error(), "absolute path") {
		t.Errorf("error %q should mention absolute path", err.Error())
	}
}

func TestConfigSetRetentionDaysNonInteger(t *testing.T) {
	f, _ := config.FieldByKey("retention_days")
	var cfg config.Config
	err := f.Set(&cfg, "abc")
	if err == nil {
		t.Fatal("expected error for non-integer retention_days")
	}
}

func TestConfigListAllKeys(t *testing.T) {
	// On an empty config, all fields should still be represented.
	cfg := config.Config{}
	var empty, set int
	for _, f := range config.Fields {
		if f.IsEmpty(&cfg) {
			empty++
		} else {
			set++
		}
	}
	if empty != len(config.Fields) {
		t.Errorf("expected all %d fields empty on zero config, got %d empty and %d set", len(config.Fields), empty, set)
	}

	// After setting one field, it should no longer be empty.
	cfg.ProjectsFolder = "/tmp/p"
	pf, _ := config.FieldByKey("projects_folder")
	if pf.IsEmpty(&cfg) {
		t.Error("projects_folder should not be empty after setting")
	}
}

func TestConfigPathDefault(t *testing.T) {
	path, err := config.DefaultPath()
	if err != nil {
		t.Fatalf("DefaultPath: %v", err)
	}
	if path == "" {
		t.Error("DefaultPath returned empty string")
	}
	// Should end with the expected file name.
	if filepath.Base(path) != "config.json" {
		t.Errorf("DefaultPath = %q, want file named config.json", path)
	}
}

func TestConfigGetUnsetKey(t *testing.T) {
	cfg := config.Config{}
	f, _ := config.FieldByKey("projects_folder")
	got := f.Get(&cfg)
	if got != "" {
		t.Errorf("Get on unset string field = %q, want empty string", got)
	}

	rd, _ := config.FieldByKey("retention_days")
	got = rd.Get(&cfg)
	if got != "0" {
		t.Errorf("Get on unset retention_days = %q, want %q", got, "0")
	}
}

func TestConfigSetEmptyClears(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")

	// Set a value, then clear it.
	cfg := config.Config{ProjectsFolder: "/tmp/projects"}
	if err := config.Save(cfg, path); err != nil {
		t.Fatalf("Save: %v", err)
	}

	f, _ := config.FieldByKey("projects_folder")
	f.Set(&cfg, "")
	if err := config.Save(cfg, path); err != nil {
		t.Fatalf("Save after clear: %v", err)
	}

	loaded, err := config.Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if loaded.ProjectsFolder != "" {
		t.Errorf("ProjectsFolder = %q after clear, want empty", loaded.ProjectsFolder)
	}
}

func TestConfigSetOverwrite(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	f, _ := config.FieldByKey("projects_folder")

	// Set initial value.
	cfg := config.Config{}
	f.Set(&cfg, "/tmp/first")
	config.Save(cfg, path)

	// Overwrite.
	loaded, _ := config.Load(path)
	f.Set(&loaded, "/tmp/second")
	config.Save(loaded, path)

	// Verify latest value wins.
	final, err := config.Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	got := f.Get(&final)
	if got != "/tmp/second" {
		t.Errorf("Get after overwrite = %q, want %q", got, "/tmp/second")
	}
}

func TestConfigSetCreatesFileOnFreshInstall(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "subdir", "config.json")

	cfg := config.Config{}
	f, _ := config.FieldByKey("projects_folder")
	f.Set(&cfg, "/tmp/projects")
	if err := config.Save(cfg, path); err != nil {
		t.Fatalf("Save: %v", err)
	}

	if _, err := os.Stat(path); err != nil {
		t.Errorf("config file not created: %v", err)
	}

	loaded, err := config.Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if loaded.ProjectsFolder != "/tmp/projects" {
		t.Errorf("ProjectsFolder = %q, want %q", loaded.ProjectsFolder, "/tmp/projects")
	}
}
