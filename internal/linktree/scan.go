package linktree

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/gorodulin/prj/internal/platform"
	"github.com/gorodulin/prj/internal/project"
)

// ManagedLink represents an existing link in the tree owned by prj.
type ManagedLink struct {
	Path      string // full path to the link
	ProjectID string // extracted from resolved target
	IsSymlink bool   // true = symlink, false = alias/bookmark
}

// ScanManagedLinks walks the link tree and returns all links whose target
// resolves into projectsFolder/<valid-id>.
//
// filterID restricts results to links targeting that project (empty = all).
// Regular files, directories, and links pointing outside projectsFolder
// are silently ignored.
func ScanManagedLinks(linksRoot, projectsFolder, idFormat, idPrefix, filterID string) ([]ManagedLink, error) {
	var managed []ManagedLink

	err := filepath.WalkDir(linksRoot, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil // skip unreadable entries
		}

		// Skip the root itself and hidden directories.
		if path == linksRoot {
			return nil
		}
		if d.IsDir() && strings.HasPrefix(d.Name(), ".") {
			return filepath.SkipDir
		}

		// Only inspect non-directory entries (symlinks, files, aliases).
		if d.IsDir() {
			return nil
		}

		ml, ok := probeLink(path, projectsFolder, idFormat, idPrefix)
		if !ok {
			return nil
		}

		if filterID != "" && ml.ProjectID != filterID {
			return nil
		}

		managed = append(managed, ml)
		return nil
	})

	if err != nil {
		return nil, err
	}
	return managed, nil
}

// resolveTarget reads the target of a symlink or alias at path.
// Returns the resolved absolute path, whether it was a symlink, and success.
// Tries symlink first (fast); falls back to alias/bookmark resolution.
func resolveTarget(path string) (target string, isSymlink bool, ok bool) {
	target, err := os.Readlink(path)
	if err == nil {
		if !filepath.IsAbs(target) {
			target = filepath.Join(filepath.Dir(path), target)
		}
		return filepath.Clean(target), true, true
	}

	target, err = platform.ResolveAlias(path)
	if err != nil {
		return "", false, false
	}
	return target, false, true
}

// probeLink checks if path is a managed link (symlink or alias pointing
// into projectsFolder). Returns the ManagedLink and true if managed.
func probeLink(path, projectsFolder, idFormat, idPrefix string) (ManagedLink, bool) {
	target, isSymlink, ok := resolveTarget(path)
	if !ok {
		return ManagedLink{}, false
	}
	id, ok := extractProjectID(target, projectsFolder, idFormat, idPrefix)
	if !ok {
		return ManagedLink{}, false
	}
	return ManagedLink{Path: path, ProjectID: id, IsSymlink: isSymlink}, true
}

// extractProjectID checks whether target is directly inside projectsFolder
// and returns the valid project ID if so.
func extractProjectID(target, projectsFolder, idFormat, idPrefix string) (string, bool) {
	rel, err := filepath.Rel(projectsFolder, target)
	if err != nil {
		return "", false
	}

	// Must be a direct child, not nested (no path separator).
	if strings.Contains(rel, string(filepath.Separator)) || rel == "." || rel == ".." {
		return "", false
	}

	// Must start with a valid component (not ".." escape).
	if strings.HasPrefix(rel, "..") {
		return "", false
	}

	if idFormat != "" && !project.IsValidID(rel, idFormat, idPrefix) {
		return "", false
	}

	return rel, true
}
