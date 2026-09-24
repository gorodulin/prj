package projectref

import (
	"os"
	"path/filepath"
)

// Outcome is the per-candidate result of an Apply pass. It is returned (rather
// than a bare count) so callers can report per-link status — the data a future
// --json report would serialize.
type Outcome struct {
	Path    string // echoes Candidate.Path
	Applied bool   // rewritten successfully
	Err     error  // rewrite failure, if any
}

// Apply rewrites every candidate with NeedsChange (remove + re-create as a
// relative symlink) and returns one Outcome per such candidate, in order. It
// continues past individual failures. Candidates that need no change are
// skipped and produce no Outcome.
func Apply(cands []Candidate) []Outcome {
	var outcomes []Outcome
	for _, c := range cands {
		if !c.NeedsChange {
			continue
		}
		o := Outcome{Path: c.Path}
		if err := rewrite(c); err != nil {
			o.Err = err
		} else {
			o.Applied = true
		}
		outcomes = append(outcomes, o)
	}
	return outcomes
}

// rewrite replaces the symlink at c.Path with one pointing at c.WantTarget.
// The new link is created under a temporary name and renamed over the old one,
// so a failed create leaves the original link untouched.
func rewrite(c Candidate) error {
	tmp := filepath.Join(filepath.Dir(c.Path), "."+filepath.Base(c.Path)+".prj-tmp")
	if err := os.Symlink(c.WantTarget, tmp); err != nil {
		return err
	}
	if err := os.Rename(tmp, c.Path); err == nil {
		return nil
	}
	// Windows cannot rename over a directory symlink: remove, then move into
	// place, restoring the original target if that fails.
	if err := os.Remove(c.Path); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	if err := os.Rename(tmp, c.Path); err != nil {
		_ = os.Remove(tmp)
		_ = os.Symlink(c.RawTarget, c.Path)
		return err
	}
	return nil
}
