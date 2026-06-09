package projectref

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// errAborted is the sentinel returned from the walk callback on context
// cancellation. Go is pinned to 1.19, so fs.SkipAll (1.20+) is unavailable.
var errAborted = errors.New("scan aborted")

// Scan walks projectDir recursively and returns every symlink whose raw target
// references exactly one project (per RewriteTarget). progress, if non-nil, is
// called once per directory entered. It honors ctx cancellation, returning the
// candidates gathered so far with interrupted=true.
//
// filepath.WalkDir does not follow symlinks, so Scan never descends into a
// referenced project — no infinite recursion, no cross-project traversal.
// Hidden directories (names starting with ".") are skipped.
func Scan(ctx context.Context, projectDir, projectsFolder, idFormat, idPrefix string, progress func()) (cands []Candidate, interrupted bool, err error) {
	walkErr := filepath.WalkDir(projectDir, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return nil // skip unreadable entries
		}
		if ctx.Err() != nil {
			return errAborted
		}

		if d.IsDir() {
			if path != projectDir && strings.HasPrefix(d.Name(), ".") {
				return filepath.SkipDir
			}
			if progress != nil {
				progress()
			}
			return nil
		}

		if d.Type()&fs.ModeSymlink == 0 {
			return nil // only symlinks
		}

		raw, rerr := os.Readlink(path)
		if rerr != nil {
			return nil // unreadable link is not a candidate
		}

		want, id, ok := RewriteTarget(filepath.Dir(path), raw, projectsFolder, idFormat, idPrefix)
		if !ok {
			return nil
		}

		cands = append(cands, Candidate{
			Path:        path,
			RawTarget:   raw,
			WantTarget:  want,
			TargetID:    id,
			NeedsChange: raw != want,
		})
		return nil
	})

	if errors.Is(walkErr, errAborted) {
		return cands, true, nil
	}
	return cands, false, walkErr
}
