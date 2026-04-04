package linktree

import (
	"path/filepath"
	"testing"
)

// Tree builder helpers for tests across the package.
func rootDir(children ...*Folder) *Folder {
	return &Folder{Name: "Links", Children: children}
}

func dir(name string, children ...*Folder) *Folder {
	return &Folder{
		Name:     name,
		Tags:     DeriveTags(name),
		Children: children,
	}
}

// sink creates a sink folder (no tags, just a name).
func sink(name string) *Folder {
	return &Folder{Name: name}
}

// assignPaths sets Path fields recursively. Call on root after building.
func assignPaths(f *Folder, prefix []string) {
	f.Path = prefix
	for _, c := range f.Children {
		childPath := append(append([]string{}, prefix...), c.Name)
		assignPaths(c, childPath)
	}
}

func TestDeriveTags(t *testing.T) {
	tests := []struct {
		name string
		want []string
	}{
		{"Programming", []string{"programming"}},
		{"Photo & Video", []string{"photo", "video"}},
		{"ML & AI", []string{"ml", "ai"}},
		{"C++", []string{"c++"}},
		{"ACME Inc", []string{"acme_inc"}},
		{"AT&T", []string{"at&t"}},
		{"  spaced  ", []string{"spaced"}},
		{"", nil},
		{"Data  Sets", []string{"data_sets"}},
		{"A & B & C", []string{"a", "b", "c"}},
		{"  leading & trailing  ", []string{"leading", "trailing"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DeriveTags(tt.name)
			if !strSliceEqual(got, tt.want) {
				t.Errorf("DeriveTags(%q) = %v, want %v", tt.name, got, tt.want)
			}
		})
	}
}

func TestFolderFullPath(t *testing.T) {
	f := &Folder{Path: []string{"Programming", "python"}}
	got := f.FullPath("/Users/dev/Links")
	want := filepath.Join("/Users/dev/Links", "Programming", "python")
	if got != want {
		t.Errorf("FullPath = %q, want %q", got, want)
	}
}

func TestFolderFullPathRoot(t *testing.T) {
	f := &Folder{Path: nil}
	got := f.FullPath("/Users/dev/Links")
	if got != "/Users/dev/Links" {
		t.Errorf("FullPath for root = %q, want %q", got, "/Users/dev/Links")
	}
}

func strSliceEqual(a, b []string) bool {
	if len(a) == 0 && len(b) == 0 {
		return true
	}
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
