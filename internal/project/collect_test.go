package project

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCollectIDsFromFolder(t *testing.T) {
	dir := t.TempDir()

	os.Mkdir(filepath.Join(dir, "p20250101a"), 0755)
	os.Mkdir(filepath.Join(dir, "p20250102b"), 0755)
	os.Mkdir(filepath.Join(dir, "metadata"), 0755)
	os.Mkdir(filepath.Join(dir, ".hidden"), 0755)
	os.WriteFile(filepath.Join(dir, "notes.txt"), []byte("hello"), 0644)

	ids, err := CollectIDsFromFolder(dir, FormatAYMDb, "p", "")
	if err != nil {
		t.Fatalf("CollectIDsFromFolder: %v", err)
	}
	if len(ids) != 2 {
		t.Fatalf("expected 2 IDs, got %d: %v", len(ids), ids)
	}
}

func TestCollectIDsFromFolderWithSuffix(t *testing.T) {
	dir := t.TempDir()

	os.Mkdir(filepath.Join(dir, "p20250101a_meta"), 0755)
	os.Mkdir(filepath.Join(dir, "p20250102b_meta"), 0755)
	os.Mkdir(filepath.Join(dir, "p20250103c"), 0755) // no suffix, skipped
	os.Mkdir(filepath.Join(dir, "random_meta"), 0755) // suffix but invalid ID

	ids, err := CollectIDsFromFolder(dir, FormatAYMDb, "p", "_meta")
	if err != nil {
		t.Fatalf("CollectIDsFromFolder: %v", err)
	}
	if len(ids) != 2 {
		t.Fatalf("expected 2 IDs, got %d: %v", len(ids), ids)
	}
	if ids[0] != "p20250101a" || ids[1] != "p20250102b" {
		t.Errorf("unexpected IDs: %v", ids)
	}
}

func TestCollectIDsFromFolderNoFormat(t *testing.T) {
	dir := t.TempDir()

	os.Mkdir(filepath.Join(dir, "p20250101a"), 0755)
	os.Mkdir(filepath.Join(dir, "metadata"), 0755)

	ids, err := CollectIDsFromFolder(dir, "", "", "")
	if err != nil {
		t.Fatalf("CollectIDsFromFolder: %v", err)
	}
	if len(ids) != 2 {
		t.Fatalf("expected 2 when no format filter, got %d", len(ids))
	}
}

func TestCollectIDsFromFolderMissingDir(t *testing.T) {
	ids, err := CollectIDsFromFolder(filepath.Join(t.TempDir(), "nonexistent"), "", "", "")
	if err != nil {
		t.Fatalf("expected nil error for missing dir, got: %v", err)
	}
	if ids != nil {
		t.Errorf("expected nil ids, got %v", ids)
	}
}

func TestCollectIDsFromFolderPrefixFilter(t *testing.T) {
	dir := t.TempDir()

	os.Mkdir(filepath.Join(dir, "p20250101a"), 0755)
	os.Mkdir(filepath.Join(dir, "prj20250101a"), 0755)

	ids, err := CollectIDsFromFolder(dir, FormatAYMDb, "p", "")
	if err != nil {
		t.Fatalf("CollectIDsFromFolder: %v", err)
	}
	if len(ids) != 1 {
		t.Fatalf("expected 1 ID with prefix 'p', got %d: %v", len(ids), ids)
	}
	if ids[0] != "p20250101a" {
		t.Errorf("got %q, want %q", ids[0], "p20250101a")
	}
}

func TestReadmeTitle(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "README.md"), []byte("# My Project\n"), 0644)

	title := ReadmeTitle(dir)
	if title != "My Project" {
		t.Errorf("ReadmeTitle = %q, want %q", title, "My Project")
	}
}

func TestReadmeTitleMissing(t *testing.T) {
	title := ReadmeTitle(t.TempDir())
	if title != "" {
		t.Errorf("ReadmeTitle = %q, want empty", title)
	}
}
