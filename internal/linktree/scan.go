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
func ScanManagedLinks(linksRoot, projectsFolder, idFormat, filterID string) ([]ManagedLink, error) {
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

		ml, ok := probeLink(path, projectsFolder, idFormat)
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

// probeLink checks if path is a managed link (symlink or alias pointing
// into projectsFolder). Returns the ManagedLink and true if managed.
func probeLink(path, projectsFolder, idFormat string) (ManagedLink, bool) {
	// Try symlink first (fast).
	target, err := os.Readlink(path)
	if err == nil {
		// Make relative targets absolute.
		if !filepath.IsAbs(target) {
			target = filepath.Join(filepath.Dir(path), target)
		}
		target = filepath.Clean(target)

		if id, ok := extractProjectID(target, projectsFolder, idFormat); ok {
			return ManagedLink{Path: path, ProjectID: id, IsSymlink: true}, true
		}
		return ManagedLink{}, false
	}

	// Try alias/bookmark resolution.
	target, err = platform.ResolveAlias(path)
	if err != nil {
		return ManagedLink{}, false
	}

	if id, ok := extractProjectID(target, projectsFolder, idFormat); ok {
		return ManagedLink{Path: path, ProjectID: id, IsSymlink: false}, true
	}
	return ManagedLink{}, false
}

// extractProjectID checks whether target is directly inside projectsFolder
// and returns the valid project ID if so.
func extractProjectID(target, projectsFolder, idFormat string) (string, bool) {
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

	if idFormat != "" && !project.IsValidID(rel, idFormat) {
		return "", false
	}

	return rel, true
}
