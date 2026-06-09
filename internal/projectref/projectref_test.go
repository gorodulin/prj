package projectref

import (
	"path/filepath"
	"testing"
)

// Tests use the aYYYYMMDDb format with prefix "p" and an absolute projects
// folder so filepath.Rel produces stable relative paths.
const (
	testFormat = "aYYYYMMDDb"
	testPrefix = "p"
)

func TestRewriteTarget(t *testing.T) {
	projects := filepath.FromSlash("/home/alice/projects")
	// A symlink living at projects/p20260402a/deps/<name>.
	linkDir := filepath.Join(projects, "p20260402a", "deps")

	tests := []struct {
		name    string
		linkDir string
		raw     string
		wantNew string
		wantID  string
		wantOK  bool
	}{
		{
			name:    "absolute target, project root",
			linkDir: linkDir,
			raw:     filepath.FromSlash("/some/other/machine/projects/p20260401b"),
			wantNew: filepath.FromSlash("../../p20260401b"),
			wantID:  "p20260401b",
			wantOK:  true,
		},
		{
			name:    "absolute target, with suffix",
			linkDir: linkDir,
			raw:     filepath.FromSlash("/home/bob/sync/p20260401b/roles/common"),
			wantNew: filepath.FromSlash("../../p20260401b/roles/common"),
			wantID:  "p20260401b",
			wantOK:  true,
		},
		{
			name:    "already-correct relative target",
			linkDir: linkDir,
			raw:     filepath.FromSlash("../../p20260401b/roles/common"),
			wantNew: filepath.FromSlash("../../p20260401b/roles/common"),
			wantID:  "p20260401b",
			wantOK:  true,
		},
		{
			name:    "relative but wrong depth",
			linkDir: filepath.Join(projects, "p20260402a", "sub", "deps"),
			raw:     filepath.FromSlash("../../p20260401b/roles"),
			wantNew: filepath.FromSlash("../../../p20260401b/roles"),
			wantID:  "p20260401b",
			wantOK:  true,
		},
		{
			name:    "no project id component",
			linkDir: linkDir,
			raw:     filepath.FromSlash("../webpack/bin/webpack.js"),
			wantOK:  false,
		},
		{
			name:    "two project ids is ambiguous",
			linkDir: linkDir,
			raw:     filepath.FromSlash("/x/p20260401b/p20260101a/file"),
			wantOK:  false,
		},
		{
			name:    "foreign prefix is not a valid id",
			linkDir: linkDir,
			raw:     filepath.FromSlash("/x/q20260401b/file"), // prefix "q" != "p"
			wantOK:  false,
		},
		{
			name:    "backslash-separated target still parses",
			linkDir: linkDir,
			raw:     `C:\Users\bob\projects\p20260401b\lib`,
			wantNew: filepath.FromSlash("../../p20260401b/lib"),
			wantID:  "p20260401b",
			wantOK:  true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			gotNew, gotID, gotOK := RewriteTarget(tc.linkDir, tc.raw, projects, testFormat, testPrefix)
			if gotOK != tc.wantOK {
				t.Fatalf("ok = %v, want %v", gotOK, tc.wantOK)
			}
			if !tc.wantOK {
				return
			}
			if gotNew != tc.wantNew {
				t.Errorf("new = %q, want %q", gotNew, tc.wantNew)
			}
			if gotID != tc.wantID {
				t.Errorf("id = %q, want %q", gotID, tc.wantID)
			}
		})
	}
}
