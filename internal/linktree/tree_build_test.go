package linktree

import (
	"os"
	"path/filepath"
	"testing"
)

func TestBuildTree(t *testing.T) {
	base := t.TempDir()

	// Create a tree:
	// Links/
	//   Programming/
	//     python/
	//     golang/
	//   .hidden/
	//   Music/
	//   notes.txt (file, should be ignored)
	os.MkdirAll(filepath.Join(base, "Programming", "python"), 0755)
	os.MkdirAll(filepath.Join(base, "Programming", "golang"), 0755)
	os.MkdirAll(filepath.Join(base, ".hidden"), 0755)
	os.MkdirAll(filepath.Join(base, "Music"), 0755)
	os.WriteFile(filepath.Join(base, "notes.txt"), []byte("hi"), 0644)

	tree, err := BuildTree(base)
	if err != nil {
		t.Fatal(err)
	}

	t.Run("root has no tags or path", func(t *testing.T) {
		if len(tree.Tags) != 0 {
			t.Errorf("root tags = %v, want empty", tree.Tags)
		}
		if len(tree.Path) != 0 {
			t.Errorf("root path = %v, want empty", tree.Path)
		}
	})

	t.Run("hidden dirs skipped", func(t *testing.T) {
		for _, c := range tree.Children {
			if c.Name == ".hidden" {
				t.Error("hidden dir should be skipped")
			}
		}
	})

	t.Run("files ignored", func(t *testing.T) {
		for _, c := range tree.Children {
			if c.Name == "notes.txt" {
				t.Error("files should be ignored")
			}
		}
	})

	t.Run("top-level children correct", func(t *testing.T) {
		names := make(map[string]bool)
		for _, c := range tree.Children {
			names[c.Name] = true
		}
		if !names["Programming"] || !names["Music"] {
			t.Errorf("expected Programming and Music, got %v", names)
		}
		if len(tree.Children) != 2 {
			t.Errorf("got %d children, want 2", len(tree.Children))
		}
	})

	t.Run("nested children with tags and paths", func(t *testing.T) {
		var prog *Folder
		for _, c := range tree.Children {
			if c.Name == "Programming" {
				prog = c
				break
			}
		}
		if prog == nil {
			t.Fatal("Programming folder not found")
		}

		if !strSliceEqual(prog.Tags, []string{"programming"}) {
			t.Errorf("Programming tags = %v", prog.Tags)
		}
		if !strSliceEqual(prog.Path, []string{"Programming"}) {
			t.Errorf("Programming path = %v", prog.Path)
		}

		if len(prog.Children) != 2 {
			t.Fatalf("Programming has %d children, want 2", len(prog.Children))
		}

		childNames := make(map[string]bool)
		for _, c := range prog.Children {
			childNames[c.Name] = true
			// Check nested path is correct.
			wantPath := []string{"Programming", c.Name}
			if !strSliceEqual(c.Path, wantPath) {
				t.Errorf("%s path = %v, want %v", c.Name, c.Path, wantPath)
			}
		}
		if !childNames["python"] || !childNames["golang"] {
			t.Errorf("expected python and golang, got %v", childNames)
		}
	})
}
