//go:build windows

package linktree

import (
	"os"
	"path/filepath"
	"testing"
)

// TestEffectiveLinkKindCrossVolume covers the Windows-only fallback rule:
// junction is impossible across volumes, so we degrade to symlink.
// filepath.VolumeName returns "" on Unix, so the volume comparison there
// never detects a mismatch — this case only exercises on Windows.
func TestEffectiveLinkKindCrossVolume(t *testing.T) {
	base := t.TempDir()
	target := filepath.Join(base, "exists")
	if err := os.Mkdir(target, 0755); err != nil {
		t.Fatal(err)
	}
	targetVol := filepath.VolumeName(target)
	if targetVol == "" {
		t.Skipf("VolumeName(%q) is empty; cannot construct cross-volume case", target)
	}

	// Pick any drive letter that's not the target's drive. We never write
	// to linkPath — effectiveLinkKind only reads its volume name.
	var linkVol string
	for _, c := range []string{"Z:", "Y:", "X:", "W:", "V:"} {
		if c != targetVol {
			linkVol = c
			break
		}
	}
	linkPath := linkVol + `\never\used`

	got := effectiveLinkKind("junction", target, linkPath)
	if got != "symlink" {
		t.Errorf("effectiveLinkKind(junction, target on %s, link on %s) = %q, want %q",
			targetVol, linkVol, got, "symlink")
	}
}
