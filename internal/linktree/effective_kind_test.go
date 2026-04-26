package linktree

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEffectiveLinkKind(t *testing.T) {
	base := t.TempDir()
	existingTarget := filepath.Join(base, "exists")
	if err := os.Mkdir(existingTarget, 0755); err != nil {
		t.Fatal(err)
	}
	missingTarget := filepath.Join(base, "missing")
	linkPath := filepath.Join(base, "link") // same volume as targets

	tests := []struct {
		name   string
		want   string
		target string
		link   string
		expect string
	}{
		{
			name:   "symlink stays symlink",
			want:   "symlink",
			target: existingTarget,
			link:   linkPath,
			expect: "symlink",
		},
		{
			name:   "symlink stays symlink even when target missing",
			want:   "symlink",
			target: missingTarget,
			link:   linkPath,
			expect: "symlink",
		},
		{
			name:   "finder-alias stays alias when target exists",
			want:   "finder-alias",
			target: existingTarget,
			link:   linkPath,
			expect: "finder-alias",
		},
		{
			name:   "finder-alias falls back to symlink when target missing",
			want:   "finder-alias",
			target: missingTarget,
			link:   linkPath,
			expect: "symlink",
		},
		{
			name:   "junction stays junction when target exists same-volume",
			want:   "junction",
			target: existingTarget,
			link:   linkPath,
			expect: "junction",
		},
		{
			name:   "junction falls back to symlink when target missing",
			want:   "junction",
			target: missingTarget,
			link:   linkPath,
			expect: "symlink",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := effectiveLinkKind(tt.want, tt.target, tt.link)
			if got != tt.expect {
				t.Errorf("effectiveLinkKind(%q, %q, %q) = %q, want %q",
					tt.want, tt.target, tt.link, got, tt.expect)
			}
		})
	}
}
