package projectref

import (
	"os"
	"path/filepath"
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
