package projectref

import "os"

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

// rewrite replaces the symlink at c.Path with one pointing at c.WantTarget,
// using the same remove-then-create idiom as linktree.Apply's replace path.
func rewrite(c Candidate) error {
	if err := os.Remove(c.Path); err != nil {
		return err
	}
	return os.Symlink(c.WantTarget, c.Path)
}
