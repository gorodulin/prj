package cmd

import (
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
			name:     "valid ID format passes",
			cwd:      filepath.Join(projects, "p20260402a"),
			idFormat: "aYYYYMMDDb",
			wantID:   "p20260402a",
		},
		{
			name:     "invalid ID format rejects",
			cwd:      filepath.Join(projects, "not-a-project"),
			idFormat: "aYYYYMMDDb",
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
			id, err := projectIDFromPath(tt.cwd, projects, tt.idFormat)
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
