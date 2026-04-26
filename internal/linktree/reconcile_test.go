package linktree

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReconcile(t *testing.T) {
	t.Run("empty tree desired placements all creates", func(t *testing.T) {
		desired := map[string]DesiredLink{
			"/links/Work/ProjA":   {Target: "/projects/p20260101a", ID: "p20260101a"},
			"/links/Code/ProjB":   {Target: "/projects/p20260102b", ID: "p20260102b"},
		}
		actions := Reconcile(desired, nil, "symlink", "")

		creates := filterActions(actions, ActionCreate)
		if len(creates) != 2 {
			t.Errorf("got %d creates, want 2", len(creates))
		}
	})

	t.Run("all links correct all skips", func(t *testing.T) {
		desired := map[string]DesiredLink{
			"/links/Work/ProjA": {Target: "/projects/p20260101a", ID: "p20260101a"},
		}
		actual := []ManagedLink{
			{Path: "/links/Work/ProjA", ProjectID: "p20260101a", Kind: "symlink"},
		}
		actions := Reconcile(desired, actual, "symlink", "")

		skips := filterActions(actions, ActionSkip)
		if len(skips) != 1 {
			t.Errorf("got %d skips, want 1", len(skips))
		}
		if len(filterActions(actions, ActionCreate)) != 0 {
			t.Error("unexpected creates")
		}
	})

	t.Run("stale link removed", func(t *testing.T) {
		actual := []ManagedLink{
			{Path: "/links/Work/OldProject", ProjectID: "p20260101a", Kind: "symlink"},
		}
		actions := Reconcile(nil, actual, "symlink", "")

		removes := filterActions(actions, ActionRemove)
		if len(removes) != 1 {
			t.Errorf("got %d removes, want 1", len(removes))
		}
		if removes[0].ID != "p20260101a" {
			t.Errorf("removed ID = %q, want %q", removes[0].ID, "p20260101a")
		}
	})

	t.Run("link in wrong location remove old create new", func(t *testing.T) {
		desired := map[string]DesiredLink{
			"/links/Code/ProjA": {Target: "/projects/p20260101a", ID: "p20260101a"},
		}
		actual := []ManagedLink{
			{Path: "/links/Work/ProjA", ProjectID: "p20260101a", Kind: "symlink"},
		}
		actions := Reconcile(desired, actual, "symlink", "")

		creates := filterActions(actions, ActionCreate)
		removes := filterActions(actions, ActionRemove)
		if len(creates) != 1 {
			t.Errorf("got %d creates, want 1", len(creates))
		}
		if len(removes) != 1 {
			t.Errorf("got %d removes, want 1", len(removes))
		}
	})

	t.Run("wrong link kind triggers replace", func(t *testing.T) {
		desired := map[string]DesiredLink{
			"/links/Work/ProjA": {Target: "/projects/p20260101a", ID: "p20260101a"},
		}
		actual := []ManagedLink{
			{Path: "/links/Work/ProjA", ProjectID: "p20260101a", Kind: "finder-alias"},
		}
		actions := Reconcile(desired, actual, "symlink", "") // want symlink

		replaces := filterActions(actions, ActionReplace)
		if len(replaces) != 1 {
			t.Errorf("got %d replaces, want 1", len(replaces))
		}
	})

	t.Run("symlink to missing target accepted as finder-alias", func(t *testing.T) {
		// When target doesn't exist, symlink is the only viable kind.
		// Reconcile should skip (not replace) even if desired kind is finder-alias.
		desired := map[string]DesiredLink{
			"/links/Work/ProjA": {Target: "/nonexistent/p20260101a", ID: "p20260101a"},
		}
		actual := []ManagedLink{
			{Path: "/links/Work/ProjA", ProjectID: "p20260101a", Kind: "symlink"},
		}
		actions := Reconcile(desired, actual, "finder-alias", "")

		skips := filterActions(actions, ActionSkip)
		if len(skips) != 1 {
			t.Errorf("expected 1 skip, got actions:")
			for _, a := range actions {
				t.Logf("  kind=%d path=%q id=%s detail=%s", a.Kind, a.Path, a.ID, a.Detail)
			}
		}
	})

	t.Run("symlink upgraded to alias when target appears", func(t *testing.T) {
		base := t.TempDir()
		target := filepath.Join(base, "p20260101a")
		os.Mkdir(target, 0755)

		desired := map[string]DesiredLink{
			"/links/Work/ProjA": {Target: target, ID: "p20260101a"},
		}
		actual := []ManagedLink{
			{Path: "/links/Work/ProjA", ProjectID: "p20260101a", Kind: "symlink"},
		}
		actions := Reconcile(desired, actual, "finder-alias", "")

		replaces := filterActions(actions, ActionReplace)
		if len(replaces) != 1 {
			t.Errorf("expected 1 replace (upgrade to alias), got actions:")
			for _, a := range actions {
				t.Logf("  kind=%d path=%q id=%s detail=%s", a.Kind, a.Path, a.ID, a.Detail)
			}
		}
	})

	t.Run("regular file blocks desired path", func(t *testing.T) {
		base := t.TempDir()
		blocker := filepath.Join(base, "blocker")
		os.WriteFile(blocker, []byte("x"), 0644)

		desired := map[string]DesiredLink{
			blocker: {Target: "/projects/p20260101a", ID: "p20260101a"},
		}
		actions := Reconcile(desired, nil, "symlink", "")

		conflicts := filterActions(actions, ActionConflict)
		if len(conflicts) != 1 {
			t.Errorf("got %d conflicts, want 1", len(conflicts))
		}
		if conflicts[0].Detail != "blocked by regular file" {
			t.Errorf("detail = %q", conflicts[0].Detail)
		}
	})

	t.Run("foreign symlink blocks desired path", func(t *testing.T) {
		base := t.TempDir()
		foreign := filepath.Join(base, "foreign")
		os.Symlink("/tmp/somewhere", foreign)

		// Foreign link is NOT in actual (not managed), but exists on disk.
		desired := map[string]DesiredLink{
			foreign: {Target: "/projects/p20260101a", ID: "p20260101a"},
		}
		actions := Reconcile(desired, nil, "symlink", "")

		conflicts := filterActions(actions, ActionConflict)
		if len(conflicts) != 1 {
			t.Errorf("got %d conflicts, want 1", len(conflicts))
		}
	})

	t.Run("folder+ID match catches unicode normalization mismatch", func(t *testing.T) {
		// Simulate macOS NFD/NFC: actual path has decomposed й (0438+0306),
		// desired path has composed й (0439). Same folder, same project ID.
		nfd := "/links/work/p20260101a \xd0\xb8\xcc\x86" // й as NFD
		nfc := "/links/work/p20260101a \xd0\xb9"          // й as NFC

		desired := map[string]DesiredLink{
			nfc: {Target: "/projects/p20260101a", ID: "p20260101a"},
		}
		actual := []ManagedLink{
			{Path: nfd, ProjectID: "p20260101a", Kind: "symlink"},
		}

		actions := Reconcile(desired, actual, "symlink", "")

		skips := filterActions(actions, ActionSkip)
		if len(skips) != 1 {
			t.Errorf("expected 1 skip (same target, same kind), got actions: %v", actions)
			for _, a := range actions {
				t.Logf("  kind=%d path=%q id=%s", a.Kind, a.Path, a.ID)
			}
		}
	})

	t.Run("folder+ID match detects wrong kind across normalization", func(t *testing.T) {
		base := t.TempDir()
		target := filepath.Join(base, "p20260101a")
		os.Mkdir(target, 0755)

		nfd := "/links/work/p20260101a \xd0\xb8\xcc\x86"
		nfc := "/links/work/p20260101a \xd0\xb9"

		desired := map[string]DesiredLink{
			nfc: {Target: target, ID: "p20260101a"},
		}
		actual := []ManagedLink{
			{Path: nfd, ProjectID: "p20260101a", Kind: "symlink"}, // have symlink
		}

		actions := Reconcile(desired, actual, "finder-alias", "") // want alias

		replaces := filterActions(actions, ActionReplace)
		if len(replaces) != 1 {
			t.Errorf("expected 1 replace, got actions:")
			for _, a := range actions {
				t.Logf("  kind=%d path=%q id=%s detail=%s", a.Kind, a.Path, a.ID, a.Detail)
			}
		}
		// Replace should use the actual filesystem path (NFD) for removal.
		if len(replaces) == 1 && replaces[0].Path != nfd {
			t.Errorf("replace path = %q, want actual filesystem path %q", replaces[0].Path, nfd)
		}
	})

	t.Run("symlink stays put when junction wanted but target missing", func(t *testing.T) {
		// effectiveLinkKind falls back to symlink when target is missing,
		// regardless of whether the user wants junction or finder-alias.
		desired := map[string]DesiredLink{
			"/links/Work/ProjA": {Target: "/nonexistent/p20260101a", ID: "p20260101a"},
		}
		actual := []ManagedLink{
			{Path: "/links/Work/ProjA", ProjectID: "p20260101a", Kind: "symlink"},
		}
		actions := Reconcile(desired, actual, "junction", "")

		if len(filterActions(actions, ActionSkip)) != 1 {
			t.Errorf("expected 1 skip, got actions: %v", actions)
		}
	})

	t.Run("existing junction skipped when junction wanted (idempotent)", func(t *testing.T) {
		base := t.TempDir()
		target := filepath.Join(base, "p20260101a")
		os.Mkdir(target, 0755)

		desired := map[string]DesiredLink{
			"/links/Work/ProjA": {Target: target, ID: "p20260101a"},
		}
		actual := []ManagedLink{
			{Path: "/links/Work/ProjA", ProjectID: "p20260101a", Kind: "junction"},
		}
		actions := Reconcile(desired, actual, "junction", "")

		if len(filterActions(actions, ActionSkip)) != 1 {
			t.Errorf("expected 1 skip (idempotent), got actions: %v", actions)
		}
	})

	t.Run("symlink replaced with junction when wanted and feasible", func(t *testing.T) {
		base := t.TempDir()
		target := filepath.Join(base, "p20260101a")
		os.Mkdir(target, 0755)
		linkPath := filepath.Join(base, "link") // same volume as target

		desired := map[string]DesiredLink{
			linkPath: {Target: target, ID: "p20260101a"},
		}
		actual := []ManagedLink{
			{Path: linkPath, ProjectID: "p20260101a", Kind: "symlink"},
		}
		actions := Reconcile(desired, actual, "junction", "")

		if len(filterActions(actions, ActionReplace)) != 1 {
			t.Errorf("expected 1 replace (symlink → junction), got actions: %v", actions)
		}
	})

	t.Run("junction replaced with symlink when symlink wanted", func(t *testing.T) {
		base := t.TempDir()
		target := filepath.Join(base, "p20260101a")
		os.Mkdir(target, 0755)

		desired := map[string]DesiredLink{
			"/links/Work/ProjA": {Target: target, ID: "p20260101a"},
		}
		actual := []ManagedLink{
			{Path: "/links/Work/ProjA", ProjectID: "p20260101a", Kind: "junction"},
		}
		actions := Reconcile(desired, actual, "symlink", "")

		if len(filterActions(actions, ActionReplace)) != 1 {
			t.Errorf("expected 1 replace (junction → symlink), got actions: %v", actions)
		}
	})

	t.Run("stale link with NFD path removed", func(t *testing.T) {
		nfd := "/links/work/p20260101a \xd0\xb8\xcc\x86"

		actual := []ManagedLink{
			{Path: nfd, ProjectID: "p20260101a", Kind: "symlink"},
		}

		actions := Reconcile(nil, actual, "symlink", "")

		removes := filterActions(actions, ActionRemove)
		if len(removes) != 1 {
			t.Errorf("expected 1 remove, got %d", len(removes))
		}
	})
}

func TestReconcileForeignProjectLinks(t *testing.T) {
	t.Run("foreign project link auto-resolved with ID suffix", func(t *testing.T) {
		base := t.TempDir()
		projectsFolder := filepath.Join(base, "projects")
		os.MkdirAll(filepath.Join(projectsFolder, "h20260101a"), 0755)

		// Foreign link blocks desired path.
		linkDir := filepath.Join(base, "links")
		os.MkdirAll(linkDir, 0755)
		blocker := filepath.Join(linkDir, "2026-01-01 Test")
		os.Symlink(filepath.Join(projectsFolder, "h20260101a"), blocker)

		desired := map[string]DesiredLink{
			blocker: {Target: filepath.Join(projectsFolder, "w20260101a"), ID: "w20260101a"},
		}
		actions := Reconcile(desired, nil, "symlink", projectsFolder)

		creates := filterActions(actions, ActionCreate)
		if len(creates) != 1 {
			t.Fatalf("got %d creates, want 1; actions: %v", len(creates), actions)
		}
		wantPath := filepath.Join(linkDir, "2026-01-01 Test (w20260101a)")
		if creates[0].Path != wantPath {
			t.Errorf("create path = %q, want %q", creates[0].Path, wantPath)
		}
		if len(filterActions(actions, ActionConflict)) != 0 {
			t.Error("unexpected conflict")
		}
	})

	t.Run("suffixed path also blocked falls back to conflict", func(t *testing.T) {
		base := t.TempDir()
		projectsFolder := filepath.Join(base, "projects")
		os.MkdirAll(filepath.Join(projectsFolder, "h20260101a"), 0755)

		linkDir := filepath.Join(base, "links")
		os.MkdirAll(linkDir, 0755)
		blocker := filepath.Join(linkDir, "Test")
		os.Symlink(filepath.Join(projectsFolder, "h20260101a"), blocker)
		// Also block the suffixed path.
		suffixed := filepath.Join(linkDir, "Test (w20260101a)")
		os.WriteFile(suffixed, []byte("x"), 0644)

		desired := map[string]DesiredLink{
			blocker: {Target: filepath.Join(projectsFolder, "w20260101a"), ID: "w20260101a"},
		}
		actions := Reconcile(desired, nil, "symlink", projectsFolder)

		conflicts := filterActions(actions, ActionConflict)
		if len(conflicts) != 1 {
			t.Fatalf("got %d conflicts, want 1; actions: %v", len(conflicts), actions)
		}
	})

	t.Run("non-project symlink remains conflict", func(t *testing.T) {
		base := t.TempDir()
		projectsFolder := filepath.Join(base, "projects")
		os.MkdirAll(projectsFolder, 0755)

		linkDir := filepath.Join(base, "links")
		os.MkdirAll(linkDir, 0755)
		blocker := filepath.Join(linkDir, "Test")
		os.Symlink("/tmp/random", blocker)

		desired := map[string]DesiredLink{
			blocker: {Target: filepath.Join(projectsFolder, "w20260101a"), ID: "w20260101a"},
		}
		actions := Reconcile(desired, nil, "symlink", projectsFolder)

		conflicts := filterActions(actions, ActionConflict)
		if len(conflicts) != 1 {
			t.Fatalf("got %d conflicts, want 1", len(conflicts))
		}
		if conflicts[0].Detail != "blocked by existing file" {
			t.Errorf("detail = %q, want %q", conflicts[0].Detail, "blocked by existing file")
		}
	})

	t.Run("regular file remains conflict", func(t *testing.T) {
		base := t.TempDir()
		projectsFolder := filepath.Join(base, "projects")
		os.MkdirAll(projectsFolder, 0755)

		blocker := filepath.Join(base, "Test")
		os.WriteFile(blocker, []byte("x"), 0644)

		desired := map[string]DesiredLink{
			blocker: {Target: filepath.Join(projectsFolder, "w20260101a"), ID: "w20260101a"},
		}
		actions := Reconcile(desired, nil, "symlink", projectsFolder)

		conflicts := filterActions(actions, ActionConflict)
		if len(conflicts) != 1 {
			t.Fatalf("got %d conflicts, want 1", len(conflicts))
		}
		if conflicts[0].Detail != "blocked by regular file" {
			t.Errorf("detail = %q, want %q", conflicts[0].Detail, "blocked by regular file")
		}
	})

	t.Run("foreign ULID project link auto-resolved", func(t *testing.T) {
		base := t.TempDir()
		projectsFolder := filepath.Join(base, "projects")
		os.MkdirAll(filepath.Join(projectsFolder, "01ARYZ6S41TSV4RRFFQ69G5FAV"), 0755)

		linkDir := filepath.Join(base, "links")
		os.MkdirAll(linkDir, 0755)
		blocker := filepath.Join(linkDir, "Test")
		os.Symlink(filepath.Join(projectsFolder, "01ARYZ6S41TSV4RRFFQ69G5FAV"), blocker)

		desired := map[string]DesiredLink{
			blocker: {Target: filepath.Join(projectsFolder, "p20260101a"), ID: "p20260101a"},
		}
		actions := Reconcile(desired, nil, "symlink", projectsFolder)

		creates := filterActions(actions, ActionCreate)
		if len(creates) != 1 {
			t.Fatalf("got %d creates, want 1; actions: %v", len(creates), actions)
		}
		wantPath := filepath.Join(linkDir, "Test (p20260101a)")
		if creates[0].Path != wantPath {
			t.Errorf("create path = %q, want %q", creates[0].Path, wantPath)
		}
	})

	t.Run("empty projectsFolder disables retry", func(t *testing.T) {
		base := t.TempDir()
		projectsFolder := filepath.Join(base, "projects")
		os.MkdirAll(filepath.Join(projectsFolder, "h20260101a"), 0755)

		linkDir := filepath.Join(base, "links")
		os.MkdirAll(linkDir, 0755)
		blocker := filepath.Join(linkDir, "Test")
		os.Symlink(filepath.Join(projectsFolder, "h20260101a"), blocker)

		desired := map[string]DesiredLink{
			blocker: {Target: filepath.Join(projectsFolder, "w20260101a"), ID: "w20260101a"},
		}
		// Empty projectsFolder — should not retry.
		actions := Reconcile(desired, nil, "symlink", "")

		conflicts := filterActions(actions, ActionConflict)
		if len(conflicts) != 1 {
			t.Fatalf("got %d conflicts, want 1", len(conflicts))
		}
	})
}

func TestApply(t *testing.T) {
	t.Run("create symlink", func(t *testing.T) {
		base := t.TempDir()
		target := filepath.Join(base, "target")
		os.Mkdir(target, 0755)
		linkPath := filepath.Join(base, "link")

		actions := []Action{
			{Kind: ActionCreate, Path: linkPath, Target: target, ID: "p20260101a"},
		}

		if err := Apply(actions, "symlink"); err != nil {
			t.Fatal(err)
		}

		got, err := os.Readlink(linkPath)
		if err != nil {
			t.Fatalf("Readlink: %v", err)
		}
		if got != target {
			t.Errorf("symlink target = %q, want %q", got, target)
		}
	})

	t.Run("remove link", func(t *testing.T) {
		base := t.TempDir()
		linkPath := filepath.Join(base, "link")
		os.Symlink("/tmp/anything", linkPath)

		actions := []Action{
			{Kind: ActionRemove, Path: linkPath, ID: "p20260101a"},
		}

		if err := Apply(actions, "symlink"); err != nil {
			t.Fatal(err)
		}

		if _, err := os.Lstat(linkPath); !os.IsNotExist(err) {
			t.Error("expected link to be removed")
		}
	})

	t.Run("replace link", func(t *testing.T) {
		base := t.TempDir()
		target := filepath.Join(base, "target")
		os.Mkdir(target, 0755)
		linkPath := filepath.Join(base, "link")
		os.Symlink("/tmp/old", linkPath)

		actions := []Action{
			{Kind: ActionReplace, Path: linkPath, Target: target, ID: "p20260101a"},
		}

		if err := Apply(actions, "symlink"); err != nil {
			t.Fatal(err)
		}

		got, err := os.Readlink(linkPath)
		if err != nil {
			t.Fatalf("Readlink: %v", err)
		}
		if got != target {
			t.Errorf("symlink target = %q, want %q", got, target)
		}
	})

	t.Run("create link in nonexistent parent dir", func(t *testing.T) {
		base := t.TempDir()
		target := filepath.Join(base, "target")
		os.Mkdir(target, 0755)
		// Parent "deep/nested" doesn't exist yet.
		linkPath := filepath.Join(base, "deep", "nested", "link")

		actions := []Action{
			{Kind: ActionCreate, Path: linkPath, Target: target, ID: "p20260101a"},
		}

		if err := Apply(actions, "symlink"); err != nil {
			t.Fatal(err)
		}

		got, err := os.Readlink(linkPath)
		if err != nil {
			t.Fatalf("Readlink: %v", err)
		}
		if got != target {
			t.Errorf("symlink target = %q, want %q", got, target)
		}
	})

	t.Run("create finder-alias falls back to symlink for missing target", func(t *testing.T) {
		base := t.TempDir()
		missingTarget := filepath.Join(base, "nonexistent-project")
		linkPath := filepath.Join(base, "link")

		actions := []Action{
			{Kind: ActionCreate, Path: linkPath, Target: missingTarget, ID: "p20260101a"},
		}

		if err := Apply(actions, "finder-alias"); err != nil {
			t.Fatal(err)
		}

		// Should have created a symlink (fallback), not an alias.
		got, err := os.Readlink(linkPath)
		if err != nil {
			t.Fatalf("expected symlink fallback, Readlink failed: %v", err)
		}
		if got != missingTarget {
			t.Errorf("symlink target = %q, want %q", got, missingTarget)
		}
	})

	t.Run("skip and conflict are no-ops", func(t *testing.T) {
		actions := []Action{
			{Kind: ActionSkip, Path: "/nonexistent/skip"},
			{Kind: ActionConflict, Path: "/nonexistent/conflict"},
		}

		if err := Apply(actions, "symlink"); err != nil {
			t.Fatal(err)
		}
	})
}

func filterActions(actions []Action, kind ActionKind) []Action {
	var filtered []Action
	for _, a := range actions {
		if a.Kind == kind {
			filtered = append(filtered, a)
		}
	}
	return filtered
}
