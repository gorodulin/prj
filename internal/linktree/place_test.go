package linktree

import (
	"sort"
	"testing"
)

func buildTree(children ...*Folder) *Folder {
	r := rootDir(children...)
	assignPaths(r, nil)
	return r
}

// pathsOf extracts sorted path slices from placement results for comparison.
func pathsOf(folders []*Folder) []string {
	var paths []string
	for _, f := range folders {
		p := f.Name
		if len(f.Path) > 0 {
			p = ""
			for i, c := range f.Path {
				if i > 0 {
					p += "/"
				}
				p += c
			}
		}
		paths = append(paths, p)
	}
	sort.Strings(paths)
	return paths
}

func TestFindPlacements(t *testing.T) {
	tests := []struct {
		name     string
		root     *Folder
		tags     []string
		sinkName string
		want     []string
	}{
		// --- Basic placement ---
		{
			name: "single branch single match",
			root: buildTree(
				dir("Programming", dir("python"), dir("golang")),
			),
			tags: []string{"programming", "golang"},
			want: []string{"Programming/golang"},
		},
		{
			name: "single branch no deeper match stops at parent",
			root: buildTree(
				dir("Programming", dir("python"), dir("golang")),
			),
			tags: []string{"programming"},
			want: []string{"Programming"},
		},
		{
			name: "multi branch placement",
			root: buildTree(
				dir("Programming", dir("golang")),
				dir("Work", dir("cli")),
			),
			tags: []string{"programming", "golang", "work", "cli"},
			want: []string{"Programming/golang", "Work/cli"},
		},
		{
			name: "no match no sink returns empty",
			root: buildTree(
				dir("Programming", dir("python")),
			),
			tags: []string{"music"},
			want: nil,
		},
		{
			name: "tagless project no sink returns empty",
			root: buildTree(
				dir("Programming"),
			),
			tags: nil,
			want: nil,
		},

		// --- Deepest match ---
		{
			name: "three levels deep matches all goes deepest",
			root: buildTree(
				dir("Programming",
					dir("python",
						dir("ml"),
					),
				),
			),
			tags: []string{"programming", "python", "ml"},
			want: []string{"Programming/python/ml"},
		},
		{
			name: "matches parent but not child stops at parent",
			root: buildTree(
				dir("Programming", dir("python")),
			),
			tags: []string{"programming"},
			want: []string{"Programming"},
		},
		{
			name: "matches child tag but not parent is inaccessible",
			root: buildTree(
				dir("Programming", dir("python")),
			),
			tags: []string{"python"},
			want: nil,
		},
		{
			name: "nested folders with identical tags goes deepest",
			root: buildTree(
				dir("work",
					dir("work",
						dir("work"),
					),
				),
			),
			tags: []string{"work"},
			want: []string{"work/work/work"},
		},

		// --- Multi-tag / OR semantics ---
		{
			name: "folder with multiple tags project matches one",
			root: buildTree(
				dir("Photo & Video"),
			),
			tags: []string{"photo"},
			want: []string{"Photo & Video"},
		},
		{
			name: "folder with multiple tags project matches all same result",
			root: buildTree(
				dir("Photo & Video"),
			),
			tags: []string{"photo", "video"},
			want: []string{"Photo & Video"},
		},
		{
			name: "many project tags one matching branch",
			root: buildTree(
				dir("Programming", dir("python")),
			),
			tags: []string{"programming", "python", "ml", "data", "science"},
			want: []string{"Programming/python"},
		},

		// --- Sink enabled ---
		{
			name:     "branch sink redirects when no children match",
			sinkName: "_misc",
			root: buildTree(
				dir("Programming",
					dir("python"),
					sink("_misc"),
				),
			),
			tags: []string{"programming"},
			want: []string{"Programming/_misc"},
		},
		{
			name:     "branch sink ignored when children match",
			sinkName: "_misc",
			root: buildTree(
				dir("Programming",
					dir("python"),
					sink("_misc"),
				),
			),
			tags: []string{"programming", "python"},
			want: []string{"Programming/python"},
		},
		{
			name:     "root sink catches unmatched",
			sinkName: "_misc",
			root: buildTree(
				dir("Programming"),
				sink("_misc"),
			),
			tags: []string{"music"},
			want: []string{"_misc"},
		},
		{
			name:     "root sink catches tagless project",
			sinkName: "_misc",
			root: buildTree(
				dir("Programming"),
				sink("_misc"),
			),
			tags: nil,
			want: []string{"_misc"},
		},
		{
			name:     "sink tag stripped from project tags",
			sinkName: "_misc",
			root: buildTree(
				dir("_misc"),
				dir("Programming"),
			),
			tags: []string{"_misc", "programming"},
			want: []string{"Programming"},
		},
		{
			name:     "multiple sink levels redirect at each depth",
			sinkName: "_misc",
			root: buildTree(
				dir("Programming",
					dir("python",
						dir("web"),
						sink("_misc"),
					),
					sink("_misc"),
				),
			),
			tags: []string{"programming", "python"},
			want: []string{"Programming/python/_misc"},
		},

		// --- Sink disabled ---
		{
			name:     "sink disabled folder itself is target",
			sinkName: "",
			root: buildTree(
				dir("Programming",
					dir("python"),
					sink("_misc"),
				),
			),
			tags: []string{"programming"},
			want: []string{"Programming"},
		},
		{
			name:     "sink disabled no match returns empty",
			sinkName: "",
			root: buildTree(
				dir("Programming"),
				sink("_misc"),
			),
			tags: []string{"music"},
			want: nil,
		},
		{
			name:     "sink disabled sink-named folder has tags and matches",
			sinkName: "",
			root: buildTree(
				dir("_misc"),
			),
			tags: []string{"_misc"},
			want: []string{"_misc"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FindPlacements(tt.root, tt.tags, tt.sinkName)
			gotPaths := pathsOf(got)
			if !strSliceEqual(gotPaths, tt.want) {
				t.Errorf("FindPlacements = %v, want %v", gotPaths, tt.want)
			}
		})
	}
}
