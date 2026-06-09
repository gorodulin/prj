// Package projectref finds and fixes cross-project symlinks: links inside one
// project folder whose target references a sibling project. Such links are
// usually authored by hand with absolute, machine-specific paths and break when
// project folders sync to another machine. RewriteTarget computes the correct
// machine-independent relative target; Scan locates the links; Apply rewrites
// them.
//
// This is distinct from the linktree package, which manages the user-organized
// links *tree* (a projection of all projects). projectref operates on links
// found *inside* project folders.
package projectref

import (
	"path/filepath"
	"strings"

	"github.com/gorodulin/prj/internal/project"
)

// Candidate is a project-referencing symlink discovered by Scan.
type Candidate struct {
	Path        string // absolute path to the symlink
	RawTarget   string // current on-disk target (os.Readlink, unresolved)
	WantTarget  string // correct relative target
	TargetID    string // the single project ID referenced by RawTarget
	NeedsChange bool   // RawTarget != WantTarget
}

// RewriteTarget computes the correct relative target for a symlink located in
// linkDir whose raw target references a project. It splits rawTarget into path
// components, requires exactly one to be a valid project ID for the configured
// format, and rebuilds the path relative to linkDir. It returns ok=false when
// zero or two-or-more components are valid IDs (ambiguous → leave alone).
//
// Pure: no filesystem access, and the target need not exist. linkDir and
// projectsFolder should share the same absolute/relative basis (callers pass
// both absolute).
func RewriteTarget(linkDir, rawTarget, projectsFolder, idFormat, idPrefix string) (want, id string, ok bool) {
	comps := splitPathComponents(rawTarget)

	idIdx := -1
	for i, c := range comps {
		if !project.IsValidID(c, idFormat, idPrefix) {
			continue
		}
		if idIdx != -1 {
			return "", "", false // second match → ambiguous
		}
		idIdx = i
	}
	if idIdx == -1 {
		return "", "", false
	}

	id = comps[idIdx]
	suffix := comps[idIdx+1:]
	wantAbs := filepath.Join(append([]string{projectsFolder, id}, suffix...)...)

	rel, err := filepath.Rel(linkDir, wantAbs)
	if err != nil {
		return "", "", false
	}
	return rel, id, true
}

// splitPathComponents splits a raw symlink target on any path separator (both
// "/" and "\"), so a symlink authored on another OS still parses — prj syncs
// project folders across machines. Empty components (leading/duplicate
// separators) are dropped; "." and ".." are kept but harmless, since the path
// is rebuilt relative to projectsFolder.
func splitPathComponents(raw string) []string {
	return strings.FieldsFunc(raw, func(r rune) bool {
		return r == '/' || r == '\\'
	})
}
