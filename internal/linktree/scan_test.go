package linktree

import (
	"os"
	"path/filepath"
	"sort"
	"testing"
)

func setupScanTree(t *testing.T) (linksRoot, projectsFolder string) {
	t.Helper()
	base := t.TempDir()

	projectsFolder = filepath.Join(base, "projects")
	os.MkdirAll(filepath.Join(projectsFolder, "p20260101a"), 0755)
	os.MkdirAll(filepath.Join(projectsFolder, "p20260102b"), 0755)

	linksRoot = filepath.Join(base, "Links")
	os.MkdirAll(filepath.Join(linksRoot, "Work"), 0755)
	os.MkdirAll(filepath.Join(linksRoot, "Code", "python"), 0755)

	return linksRoot, projectsFolder
}

func TestScanManagedLinks(t *testing.T) {
	t.Run("symlink into projects folder detected", func(t *testing.T) {
		linksRoot, projectsFolder := setupScanTree(t)
		linkPath := filepath.Join(linksRoot, "Work", "MyProject")
		os.Symlink(filepath.Join(projectsFolder, "p20260101a"), linkPath)

		managed, err := ScanManagedLinks(linksRoot, projectsFolder, "aYYYYMMDDb", "p", "")
		if err != nil {
			t.Fatal(err)
		}
		if len(managed) != 1 {
			t.Fatalf("got %d managed links, want 1", len(managed))
		}
		if managed[0].ProjectID != "p20260101a" {
			t.Errorf("ProjectID = %q, want %q", managed[0].ProjectID, "p20260101a")
		}
		if !managed[0].IsSymlink {
			t.Error("expected IsSymlink = true")
		}
	})

	t.Run("symlink pointing elsewhere ignored", func(t *testing.T) {
		linksRoot, _ := setupScanTree(t)
		os.Symlink("/tmp/something", filepath.Join(linksRoot, "Work", "foreign"))

		managed, err := ScanManagedLinks(linksRoot, filepath.Join(t.TempDir(), "projects"), "aYYYYMMDDb", "p", "")
		if err != nil {
			t.Fatal(err)
		}
		if len(managed) != 0 {
			t.Errorf("got %d managed links, want 0", len(managed))
		}
	})

	t.Run("regular file ignored", func(t *testing.T) {
		linksRoot, projectsFolder := setupScanTree(t)
		os.WriteFile(filepath.Join(linksRoot, "Work", "notes.txt"), []byte("hi"), 0644)

		managed, err := ScanManagedLinks(linksRoot, projectsFolder, "aYYYYMMDDb", "p", "")
		if err != nil {
			t.Fatal(err)
		}
		if len(managed) != 0 {
			t.Errorf("got %d managed links, want 0", len(managed))
		}
	})

	t.Run("directory ignored", func(t *testing.T) {
		linksRoot, projectsFolder := setupScanTree(t)
		// "Work" dir already exists — should not be detected.
		managed, err := ScanManagedLinks(linksRoot, projectsFolder, "aYYYYMMDDb", "p", "")
		if err != nil {
			t.Fatal(err)
		}
		if len(managed) != 0 {
			t.Errorf("got %d managed links, want 0", len(managed))
		}
	})

	t.Run("broken symlink into projects folder detected", func(t *testing.T) {
		linksRoot, projectsFolder := setupScanTree(t)
		// Points to a valid-looking path that doesn't exist on disk.
		os.Symlink(filepath.Join(projectsFolder, "p20260103c"), filepath.Join(linksRoot, "Work", "Gone"))

		managed, err := ScanManagedLinks(linksRoot, projectsFolder, "aYYYYMMDDb", "p", "")
		if err != nil {
			t.Fatal(err)
		}
		if len(managed) != 1 {
			t.Fatalf("got %d managed links, want 1", len(managed))
		}
		if managed[0].ProjectID != "p20260103c" {
			t.Errorf("ProjectID = %q, want %q", managed[0].ProjectID, "p20260103c")
		}
	})

	t.Run("nested links at multiple depths", func(t *testing.T) {
		linksRoot, projectsFolder := setupScanTree(t)
		os.Symlink(filepath.Join(projectsFolder, "p20260101a"), filepath.Join(linksRoot, "Work", "ProjA"))
		os.Symlink(filepath.Join(projectsFolder, "p20260102b"), filepath.Join(linksRoot, "Code", "python", "ProjB"))

		managed, err := ScanManagedLinks(linksRoot, projectsFolder, "aYYYYMMDDb", "p", "")
		if err != nil {
			t.Fatal(err)
		}
		if len(managed) != 2 {
			t.Fatalf("got %d managed links, want 2", len(managed))
		}

		ids := []string{managed[0].ProjectID, managed[1].ProjectID}
		sort.Strings(ids)
		if ids[0] != "p20260101a" || ids[1] != "p20260102b" {
			t.Errorf("ProjectIDs = %v, want [p20260101a p20260102b]", ids)
		}
	})

	t.Run("filterID restricts results", func(t *testing.T) {
		linksRoot, projectsFolder := setupScanTree(t)
		os.Symlink(filepath.Join(projectsFolder, "p20260101a"), filepath.Join(linksRoot, "Work", "ProjA"))
		os.Symlink(filepath.Join(projectsFolder, "p20260102b"), filepath.Join(linksRoot, "Work", "ProjB"))

		managed, err := ScanManagedLinks(linksRoot, projectsFolder, "aYYYYMMDDb", "p", "p20260101a")
		if err != nil {
			t.Fatal(err)
		}
		if len(managed) != 1 {
			t.Fatalf("got %d managed links, want 1", len(managed))
		}
		if managed[0].ProjectID != "p20260101a" {
			t.Errorf("ProjectID = %q, want %q", managed[0].ProjectID, "p20260101a")
		}
	})
}
