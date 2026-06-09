package projectref

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

// setupProject creates projects/<id> folders and returns the absolute projects
// folder plus the scanned project's directory.
func setupProject(t *testing.T, scanID string, otherIDs ...string) (projectsFolder, projDir string) {
	t.Helper()
	base := t.TempDir()
	projectsFolder = filepath.Join(base, "projects")
	for _, id := range append([]string{scanID}, otherIDs...) {
		if err := os.MkdirAll(filepath.Join(projectsFolder, id), 0755); err != nil {
			t.Fatal(err)
		}
	}
	return projectsFolder, filepath.Join(projectsFolder, scanID)
}

func mustSymlink(t *testing.T, target, link string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(link), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}
}

func scanAll(t *testing.T, projDir, projectsFolder string) []Candidate {
	t.Helper()
	cands, interrupted, err := Scan(context.Background(), projDir, projectsFolder, testFormat, testPrefix, nil)
	if err != nil {
		t.Fatal(err)
	}
	if interrupted {
		t.Fatal("unexpected interruption")
	}
	return cands
}

func TestScan(t *testing.T) {
	t.Run("absolute cross-project link is a changed candidate", func(t *testing.T) {
		projects, projDir := setupProject(t, "p20260402a", "p20260401b")
		mustSymlink(t, filepath.Join(projects, "p20260401b"), filepath.Join(projDir, "deps", "roles"))

		cands := scanAll(t, projDir, projects)
		if len(cands) != 1 {
			t.Fatalf("got %d candidates, want 1", len(cands))
		}
		c := cands[0]
		if c.TargetID != "p20260401b" {
			t.Errorf("TargetID = %q, want p20260401b", c.TargetID)
		}
		if !c.NeedsChange {
			t.Errorf("NeedsChange = false, want true")
		}
		if c.WantTarget != filepath.FromSlash("../../p20260401b") {
			t.Errorf("WantTarget = %q, want ../../p20260401b", c.WantTarget)
		}
	})

	t.Run("already-relative link is unchanged", func(t *testing.T) {
		projects, projDir := setupProject(t, "p20260402a", "p20260401b")
		mustSymlink(t, filepath.FromSlash("../../p20260401b"), filepath.Join(projDir, "deps", "roles"))

		cands := scanAll(t, projDir, projects)
		if len(cands) != 1 {
			t.Fatalf("got %d candidates, want 1", len(cands))
		}
		if cands[0].NeedsChange {
			t.Errorf("NeedsChange = true, want false")
		}
	})

	t.Run("non-project symlink and regular file ignored", func(t *testing.T) {
		projects, projDir := setupProject(t, "p20260402a", "p20260401b")
		mustSymlink(t, "/tmp/whatever/bin", filepath.Join(projDir, "node_modules", "x"))
		if err := os.WriteFile(filepath.Join(projDir, "README.md"), []byte("hi"), 0644); err != nil {
			t.Fatal(err)
		}

		if cands := scanAll(t, projDir, projects); len(cands) != 0 {
			t.Fatalf("got %d candidates, want 0", len(cands))
		}
	})

	t.Run("symlink to a directory is not descended into", func(t *testing.T) {
		projects, projDir := setupProject(t, "p20260402a", "p20260401b")
		// A symlink to the sibling project (a directory). If WalkDir followed
		// it, the sibling's contents would be walked. Put a decoy file in the
		// sibling that would (wrongly) be seen as inside projDir.
		if err := os.WriteFile(filepath.Join(projects, "p20260401b", "decoy"), []byte("x"), 0644); err != nil {
			t.Fatal(err)
		}
		mustSymlink(t, filepath.Join(projects, "p20260401b"), filepath.Join(projDir, "link-to-b"))

		cands := scanAll(t, projDir, projects)
		if len(cands) != 1 {
			t.Fatalf("got %d candidates, want 1 (only the symlink itself)", len(cands))
		}
	})

	t.Run("hidden directories are skipped", func(t *testing.T) {
		projects, projDir := setupProject(t, "p20260402a", "p20260401b")
		mustSymlink(t, filepath.Join(projects, "p20260401b"), filepath.Join(projDir, ".git", "ref"))

		if cands := scanAll(t, projDir, projects); len(cands) != 0 {
			t.Fatalf("got %d candidates, want 0", len(cands))
		}
	})

	t.Run("cancelled context interrupts the walk", func(t *testing.T) {
		projects, projDir := setupProject(t, "p20260402a", "p20260401b")
		mustSymlink(t, filepath.Join(projects, "p20260401b"), filepath.Join(projDir, "deps", "roles"))

		ctx, cancel := context.WithCancel(context.Background())
		cancel() // already cancelled before the walk starts

		_, interrupted, err := Scan(ctx, projDir, projects, testFormat, testPrefix, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !interrupted {
			t.Errorf("interrupted = false, want true")
		}
	})
}
