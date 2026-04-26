package platform

import (
	"os"
	"path/filepath"
	"testing"
)

// realPath resolves symlinks in a path for reliable comparison.
// macOS /var → /private/var causes mismatches without this.
func realPath(t *testing.T, path string) string {
	t.Helper()
	real, err := filepath.EvalSymlinks(path)
	if err != nil {
		t.Fatalf("EvalSymlinks(%q): %v", path, err)
	}
	return real
}

func TestResolveLinkSymlink(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "target")
	link := filepath.Join(dir, "link")

	if err := os.Mkdir(target, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}

	got, kind, err := ResolveLink(link)
	if err != nil {
		t.Fatalf("ResolveLink: %v", err)
	}
	if got != target {
		t.Errorf("ResolveLink target = %q, want %q", got, target)
	}
	if kind != "symlink" {
		t.Errorf("ResolveLink kind = %q, want %q", kind, "symlink")
	}
}

func TestResolveLinkNotALink(t *testing.T) {
	dir := t.TempDir()
	regular := filepath.Join(dir, "regular")
	os.WriteFile(regular, []byte("hello"), 0644)

	_, _, err := ResolveLink(regular)
	if err == nil {
		t.Fatal("expected error for non-link, got nil")
	}
}

func TestCreateAndResolveAlias(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "target")
	alias := filepath.Join(dir, "alias")

	if err := os.Mkdir(target, 0755); err != nil {
		t.Fatal(err)
	}

	err := CreateAlias(alias, target)
	if err != nil {
		// On CGO_ENABLED=0 builds, alias creation returns an error. Skip.
		t.Skipf("CreateAlias not available: %v", err)
	}

	// Verify the alias file was created.
	if _, err := os.Stat(alias); err != nil {
		t.Fatalf("alias file not created: %v", err)
	}

	// Resolve it back.
	resolved, err := ResolveAlias(alias)
	if err != nil {
		t.Fatalf("ResolveAlias: %v", err)
	}
	if realPath(t, resolved) != realPath(t, target) {
		t.Errorf("ResolveAlias = %q, want %q", resolved, target)
	}
}

func TestResolveAliasNotAnAlias(t *testing.T) {
	dir := t.TempDir()
	regular := filepath.Join(dir, "regular")
	os.WriteFile(regular, []byte("hello"), 0644)

	_, err := ResolveAlias(regular)
	if err == nil {
		t.Fatal("expected error for non-alias, got nil")
	}
}

func TestResolveLinkFinderAlias(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "target")
	alias := filepath.Join(dir, "alias")

	if err := os.Mkdir(target, 0755); err != nil {
		t.Fatal(err)
	}

	if err := CreateAlias(alias, target); err != nil {
		t.Skipf("CreateAlias not available: %v", err)
	}

	// ResolveLink should fall back to alias resolution.
	resolved, kind, err := ResolveLink(alias)
	if err != nil {
		t.Fatalf("ResolveLink: %v", err)
	}
	if realPath(t, resolved) != realPath(t, target) {
		t.Errorf("ResolveLink target = %q, want %q", resolved, target)
	}
	if kind != "finder-alias" {
		t.Errorf("ResolveLink kind = %q, want %q", kind, "finder-alias")
	}
}

func TestSetAndGetFinderComment(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "target")
	link := filepath.Join(dir, "link")

	if err := os.Mkdir(target, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}

	comment := "#cli #golang"
	if err := SetFinderComment(link, comment); err != nil {
		t.Skipf("SetFinderComment not available: %v", err)
	}

	got, err := GetFinderComment(link)
	if err != nil {
		t.Fatalf("GetFinderComment: %v", err)
	}
	if got != comment {
		t.Errorf("GetFinderComment = %q, want %q", got, comment)
	}
}

func TestSetFinderCommentEmptyClears(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "target")
	link := filepath.Join(dir, "link")

	if err := os.Mkdir(target, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}

	// Set a comment, then clear it.
	if err := SetFinderComment(link, "#test"); err != nil {
		t.Skipf("SetFinderComment not available: %v", err)
	}
	if err := SetFinderComment(link, ""); err != nil {
		t.Fatalf("clear comment: %v", err)
	}

	got, err := GetFinderComment(link)
	if err != nil {
		t.Fatalf("GetFinderComment: %v", err)
	}
	if got != "" {
		t.Errorf("GetFinderComment after clear = %q, want empty", got)
	}
}

func TestFinderCommentViaSymlink(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "target")
	link := filepath.Join(dir, "link")

	if err := os.Mkdir(target, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}

	if err := SetFinderComment(link, "#on-link"); err != nil {
		t.Skipf("SetFinderComment not available: %v", err)
	}

	// Finder stores the comment on the target. Reading via the symlink
	// should return the same value (GetFinderCommentRaw follows symlinks).
	got, err := GetFinderComment(link)
	if err != nil {
		t.Fatalf("GetFinderComment via symlink: %v", err)
	}
	if got != "#on-link" {
		t.Errorf("GetFinderComment via symlink = %q, want %q", got, "#on-link")
	}

	gotTarget, err := GetFinderComment(target)
	if err != nil {
		t.Fatalf("GetFinderComment on target: %v", err)
	}
	if gotTarget != "#on-link" {
		t.Errorf("GetFinderComment on target = %q, want %q", gotTarget, "#on-link")
	}
}

func TestFinderCommentChanged(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "target")
	link := filepath.Join(dir, "link")

	if err := os.Mkdir(target, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}

	comment := "#cli #golang"
	encoded := EncodeBplistString(comment)

	if err := SetFinderComment(link, comment); err != nil {
		t.Skipf("SetFinderComment not available: %v", err)
	}

	// Same comment — should report no change.
	changed, err := FinderCommentChanged(link, encoded)
	if err != nil {
		t.Fatalf("FinderCommentChanged: %v", err)
	}
	if changed {
		t.Error("expected no change for identical comment")
	}

	// Different comment — should report changed.
	different := EncodeBplistString("#different")
	changed, err = FinderCommentChanged(link, different)
	if err != nil {
		t.Fatalf("FinderCommentChanged: %v", err)
	}
	if !changed {
		t.Error("expected change for different comment")
	}
}

func TestGetFinderCommentNoAttr(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "target")
	link := filepath.Join(dir, "link")

	if err := os.Mkdir(target, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}

	// No comment set — should return empty, no error.
	got, err := GetFinderComment(link)
	if err != nil {
		t.Fatalf("GetFinderComment: %v", err)
	}
	if got != "" {
		t.Errorf("GetFinderComment on fresh link = %q, want empty", got)
	}
}

func TestSupportedLinkTypes(t *testing.T) {
	types := SupportedLinkTypes()
	if len(types) == 0 {
		t.Fatal("expected at least one link type")
	}

	hasSymlink := false
	for _, lt := range types {
		if lt == "symlink" {
			hasSymlink = true
		}
	}
	if !hasSymlink {
		t.Error("expected 'symlink' in supported link types")
	}
}
