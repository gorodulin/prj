package projectref

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestApply(t *testing.T) {
	t.Run("rewrites changed link and resolves correctly", func(t *testing.T) {
		projects, projDir := setupProject(t, "p20260402a", "p20260401b")
		linkPath := filepath.Join(projDir, "deps", "roles")
		mustSymlink(t, filepath.Join(projects, "p20260401b"), linkPath)

		cands := scanAll(t, projDir, projects)
		outcomes := Apply(cands)

		if len(outcomes) != 1 {
			t.Fatalf("got %d outcomes, want 1", len(outcomes))
		}
		if !outcomes[0].Applied || outcomes[0].Err != nil {
			t.Fatalf("outcome = %+v, want Applied with no error", outcomes[0])
		}

		// On-disk target is now relative...
		raw, err := os.Readlink(linkPath)
		if err != nil {
			t.Fatal(err)
		}
		if raw != filepath.FromSlash("../../p20260401b") {
			t.Errorf("readlink = %q, want ../../p20260401b", raw)
		}
		// ...and still resolves to the sibling project.
		resolved, err := filepath.EvalSymlinks(linkPath)
		if err != nil {
			t.Fatal(err)
		}
		wantResolved, _ := filepath.EvalSymlinks(filepath.Join(projects, "p20260401b"))
		if resolved != wantResolved {
			t.Errorf("resolved = %q, want %q", resolved, wantResolved)
		}
	})

	t.Run("unchanged candidate produces no outcome", func(t *testing.T) {
		projects, projDir := setupProject(t, "p20260402a", "p20260401b")
		mustSymlink(t, filepath.FromSlash("../../p20260401b"), filepath.Join(projDir, "deps", "roles"))

		cands := scanAll(t, projDir, projects)
		if outcomes := Apply(cands); len(outcomes) != 0 {
			t.Fatalf("got %d outcomes, want 0", len(outcomes))
		}
	})

	t.Run("failed create keeps the original link", func(t *testing.T) {
		projects, projDir := setupProject(t, "p20260402a", "p20260401b")
		linkPath := filepath.Join(projDir, "deps", "roles")
		orig := filepath.Join(projects, "p20260401b")
		mustSymlink(t, orig, linkPath)

		// A target longer than PATH_MAX makes symlink creation fail.
		tooLong := strings.Repeat("a", 5000)
		outcomes := Apply([]Candidate{{Path: linkPath, RawTarget: orig, WantTarget: tooLong, NeedsChange: true}})
		if len(outcomes) != 1 || outcomes[0].Err == nil {
			t.Fatalf("outcomes = %+v, want one failure", outcomes)
		}
		raw, err := os.Readlink(linkPath)
		if err != nil {
			t.Fatalf("original link lost: %v", err)
		}
		if raw != orig {
			t.Errorf("readlink = %q, want original %q", raw, orig)
		}
		entries, _ := os.ReadDir(filepath.Dir(linkPath))
		if len(entries) != 1 {
			t.Errorf("expected only the original link in %s, found %d entries", filepath.Dir(linkPath), len(entries))
		}
	})

	t.Run("idempotent: second apply is a no-op", func(t *testing.T) {
		projects, projDir := setupProject(t, "p20260402a", "p20260401b")
		linkPath := filepath.Join(projDir, "deps", "roles")
		mustSymlink(t, filepath.Join(projects, "p20260401b"), linkPath)

		Apply(scanAll(t, projDir, projects))
		// Re-scan: nothing should need changing now.
		for _, c := range scanAll(t, projDir, projects) {
			if c.NeedsChange {
				t.Errorf("candidate still needs change after apply: %+v", c)
			}
		}
	})
}
