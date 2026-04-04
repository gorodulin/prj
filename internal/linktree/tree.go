// Package linktree manages a folder tree of links pointing to project folders.
// It handles tree building, tag-based placement, link naming, and reconciliation.
package linktree

import (
	"os"
	"path/filepath"
	"strings"
)

// Folder is a node in the link tree.
type Folder struct {
	Name     string
	Tags     []string   // derived from Name via DeriveTags
	Path     []string   // components from root (e.g. ["Programming", "python"])
	Children []*Folder
}

// FullPath joins linksRoot with the folder's path components.
func (f *Folder) FullPath(linksRoot string) string {
	parts := append([]string{linksRoot}, f.Path...)
	return filepath.Join(parts...)
}

// DeriveTags extracts tags from a folder name.
//
// Rules:
//   - Split on " & " (space-ampersand-space) into segments
//   - Each segment: lowercase, collapse whitespace runs to "_", strip outer "_"
//   - Special characters (+, #, $, etc.) preserved
//   - "AT&T" stays "at&t" (no spaces around &, no split)
func DeriveTags(name string) []string {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil
	}

	segments := strings.Split(name, " & ")
	var tags []string
	for _, seg := range segments {
		t := normalizeSegment(seg)
		if t != "" {
			tags = append(tags, t)
		}
	}
	return tags
}

// normalizeSegment lowercases and collapses internal whitespace to "_".
// Leading/trailing whitespace is trimmed before processing.
// Literal underscores in the input are preserved.
func normalizeSegment(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	if s == "" {
		return ""
	}

	// Collapse whitespace runs to single "_".
	var b strings.Builder
	inSpace := false
	for _, r := range s {
		if r == ' ' || r == '\t' {
			if !inSpace {
				b.WriteByte('_')
				inSpace = true
			}
			continue
		}
		inSpace = false
		b.WriteRune(r)
	}

	return b.String()
}

// BuildTree scans a directory into a Folder tree.
// Only directories are included; files and hidden dirs (prefix ".") are skipped.
func BuildTree(root string) (*Folder, error) {
	rootFolder := &Folder{Name: filepath.Base(root)}
	if err := buildChildren(root, rootFolder); err != nil {
		return nil, err
	}
	return rootFolder, nil
}

func buildChildren(dir string, parent *Folder) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}

	for _, e := range entries {
		if !e.IsDir() || strings.HasPrefix(e.Name(), ".") {
			continue
		}

		child := &Folder{
			Name: e.Name(),
			Tags: DeriveTags(e.Name()),
			Path: append(append([]string{}, parent.Path...), e.Name()),
		}
		parent.Children = append(parent.Children, child)

		if err := buildChildren(filepath.Join(dir, e.Name()), child); err != nil {
			return err
		}
	}

	return nil
}
