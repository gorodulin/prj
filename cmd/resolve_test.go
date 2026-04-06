package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/gorodulin/prj/internal/config"
)

func TestProjectIDFromPath(t *testing.T) {
	projects := "/Users/dev/Projects"

	tests := []struct {
		name     string
		cwd      string
		idFormat string
		idPrefix string
		wantID   string
		wantErr  bool
	}{
		{
			name:   "directly inside project folder",
			cwd:    filepath.Join(projects, "p20260402a"),
			wantID: "p20260402a",
		},
		{
			name:   "deep inside project subfolder",
			cwd:    filepath.Join(projects, "p20260402a", "internal", "config"),
			wantID: "p20260402a",
		},
		{
			name:    "at projects folder itself",
			cwd:     projects,
			wantErr: true,
		},
		{
			name:    "outside projects folder",
			cwd:     "/tmp",
			wantErr: true,
		},
		{
			name:    "parent of projects folder",
			cwd:     filepath.Dir(projects),
			wantErr: true,
		},
		{
			name:     "valid ID format and prefix passes",
			cwd:      filepath.Join(projects, "p20260402a"),
			idFormat: "aYYYYMMDDb",
			idPrefix: "p",
			wantID:   "p20260402a",
		},
		{
			name:     "invalid ID format rejects",
			cwd:      filepath.Join(projects, "not-a-project"),
			idFormat: "aYYYYMMDDb",
			idPrefix: "p",
			wantErr:  true,
		},
		{
			name:     "wrong prefix rejects",
			cwd:      filepath.Join(projects, "prj20260402a"),
			idFormat: "aYYYYMMDDb",
			idPrefix: "p",
			wantErr:  true,
		},
		{
			name:   "no format validation accepts any folder name",
			cwd:    filepath.Join(projects, "my-experiment"),
			wantID: "my-experiment",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id, err := projectIDFromPath(tt.cwd, projects, tt.idFormat, tt.idPrefix)
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error, got id=%q", id)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if id != tt.wantID {
				t.Errorf("got %q, want %q", id, tt.wantID)
			}
		})
	}
}

func TestResolveCurrentProjectID_symlink(t *testing.T) {
	// Create a temporary projects folder with a project subfolder.
	tmp := t.TempDir()
	projFolder := filepath.Join(tmp, "Projects")
	projDir := filepath.Join(projFolder, "p20260402a")
	if err := os.MkdirAll(projDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Create a symlink pointing into the project folder.
	linkDir := filepath.Join(tmp, "Links")
	if err := os.MkdirAll(linkDir, 0o755); err != nil {
		t.Fatal(err)
	}
	symlink := filepath.Join(linkDir, "my-project")
	if err := os.Symlink(projDir, symlink); err != nil {
		t.Fatal(err)
	}

	// cd into the symlink.
	orig, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chdir(orig) })

	if err := os.Chdir(symlink); err != nil {
		t.Fatal(err)
	}

	cfg := config.Config{ProjectsFolder: projFolder}
	id, err := resolveCurrentProjectID(cfg)
	if err != nil {
		t.Fatalf("expected success from symlinked cwd, got: %v", err)
	}
	if id != "p20260402a" {
		t.Errorf("got %q, want %q", id, "p20260402a")
	}
}

func TestResolveCurrentProjectID_crossProjectSymlink(t *testing.T) {
	// projA/link → projB: should resolve to projB, not projA.
	tmp := t.TempDir()
	projFolder := filepath.Join(tmp, "Projects")
	projA := filepath.Join(projFolder, "projA")
	projB := filepath.Join(projFolder, "projB")
	if err := os.MkdirAll(projA, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(projB, 0o755); err != nil {
		t.Fatal(err)
	}

	// Symlink inside projA that points to projB.
	symlink := filepath.Join(projA, "link")
	if err := os.Symlink(projB, symlink); err != nil {
		t.Fatal(err)
	}

	orig, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chdir(orig) })

	if err := os.Chdir(symlink); err != nil {
		t.Fatal(err)
	}

	cfg := config.Config{ProjectsFolder: projFolder}
	id, err := resolveCurrentProjectID(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != "projB" {
		t.Errorf("got %q, want %q (should resolve to target, not containing project)", id, "projB")
	}
}

func TestExpandProjectID(t *testing.T) {
	cfg := config.Config{ProjectsFolder: "/Users/dev/Projects"}

	t.Run("non-current passes through", func(t *testing.T) {
		id, err := expandProjectID("p20260402a", cfg)
		if err != nil {
			t.Fatal(err)
		}
		if id != "p20260402a" {
			t.Errorf("got %q, want %q", id, "p20260402a")
		}
	})

	t.Run("empty string passes through", func(t *testing.T) {
		id, err := expandProjectID("", cfg)
		if err != nil {
			t.Fatal(err)
		}
		if id != "" {
			t.Errorf("got %q, want empty", id)
		}
	})
}
