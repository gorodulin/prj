package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadMissingDefaultPath(t *testing.T) {
	// Empty path = default. Missing default config is OK (first run).
	// We can't easily test this without controlling UserConfigDir,
	// so we test Load("") doesn't panic. If the default file happens
	// to exist, it loads it; if not, returns zero config.
	_, err := Load("")
	if err != nil {
		t.Fatalf("Load with default path should not error: %v", err)
	}
}

func TestLoadExplicitMissingFileErrors(t *testing.T) {
	_, err := Load(filepath.Join(t.TempDir(), "nonexistent.json"))
	if err == nil {
		t.Fatal("expected error for explicit missing file, got nil")
	}
}

func TestLoadExplicitValidFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	os.WriteFile(path, []byte(`{"projects_folder": "/tmp/projects"}`), 0644)

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.ProjectsFolder != "/tmp/projects" {
		t.Errorf("ProjectsFolder = %q, want %q", cfg.ProjectsFolder, "/tmp/projects")
	}
}

func TestLoadAndSaveRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "prj", "config.json")

	original := Config{
		ProjectsFolder: "/home/user/projects",
		ProjectIDType:  "aYYYYMMDDb",
	}

	if err := Save(original, path); err != nil {
		t.Fatalf("save: %v", err)
	}

	loaded, err := Load(path)
	if err != nil {
		t.Fatalf("load: %v", err)
	}

	if loaded.ProjectsFolder != original.ProjectsFolder {
		t.Errorf("ProjectsFolder = %q, want %q", loaded.ProjectsFolder, original.ProjectsFolder)
	}
	if loaded.ProjectIDType != original.ProjectIDType {
		t.Errorf("ProjectIDType = %q, want %q", loaded.ProjectIDType, original.ProjectIDType)
	}
}

func TestLoadInvalidJSON(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bad.json")
	os.WriteFile(path, []byte("{not json}"), 0644)

	_, err := Load(path)
	if err == nil {
		t.Fatal("expected error for invalid JSON, got nil")
	}
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     Config
		wantErr bool
		errMsg  string
	}{
		// Enum: link_kind
		{
			name: "link_kind empty is ok",
			cfg:  Config{},
		},
		{
			name: "link_kind symlink is ok",
			cfg:  Config{LinkKind: "symlink"},
		},
		{
			name:    "link_kind finder-alias",
			cfg:     Config{LinkKind: "finder-alias"},
			wantErr: !finderAliasSupported,
		},
		{
			name:    "link_kind unknown rejected",
			cfg:     Config{LinkKind: "ssymlink"},
			wantErr: true,
			errMsg:  "link_kind",
		},

		// project_id_prefix
		{
			name: "project_id_prefix empty is ok",
			cfg:  Config{},
		},
		{
			name: "project_id_prefix single letter is ok",
			cfg:  Config{ProjectIDPrefix: "p"},
		},
		{
			name: "project_id_prefix three letters is ok",
			cfg:  Config{ProjectIDPrefix: "prj"},
		},
		{
			name: "project_id_prefix five letters is ok",
			cfg:  Config{ProjectIDPrefix: "abcde"},
		},
		{
			name:    "project_id_prefix uppercase rejected",
			cfg:     Config{ProjectIDPrefix: "PRJ"},
			wantErr: true,
			errMsg:  "project_id_prefix",
		},
		{
			name:    "project_id_prefix too long rejected",
			cfg:     Config{ProjectIDPrefix: "abcdef"},
			wantErr: true,
			errMsg:  "project_id_prefix",
		},
		{
			name:    "project_id_prefix digits rejected",
			cfg:     Config{ProjectIDPrefix: "p1"},
			wantErr: true,
			errMsg:  "project_id_prefix",
		},

		// Enum: project_id_type
		{
			name: "project_id_type empty is ok",
			cfg:  Config{},
		},
		{
			name: "project_id_type aYYYYMMDDb is ok",
			cfg:  Config{ProjectIDType: "aYYYYMMDDb"},
		},
		{
			name:    "project_id_type unknown rejected",
			cfg:     Config{ProjectIDType: "random"},
			wantErr: true,
			errMsg:  "project_id_type",
		},

		// Paths must be absolute
		{
			name:    "relative projects_folder rejected",
			cfg:     Config{ProjectsFolder: "relative/path"},
			wantErr: true,
			errMsg:  "projects_folder",
		},
		{
			name:    "relative metadata_folder rejected",
			cfg:     Config{MetadataFolder: "relative"},
			wantErr: true,
			errMsg:  "metadata_folder",
		},
		{
			name:    "relative links_folder rejected",
			cfg:     Config{LinksFolder: "relative"},
			wantErr: true,
			errMsg:  "links_folder",
		},
		{
			name: "absolute paths ok",
			cfg: Config{
				ProjectsFolder: "/home/user/projects",
				MetadataFolder: "/home/user/metadata",
				LinksFolder:    "/home/user/links",
			},
		},

		// Dangerous overlaps
		{
			name: "links_folder equals projects_folder rejected",
			cfg: Config{
				ProjectsFolder: "/home/user/stuff",
				LinksFolder:    "/home/user/stuff",
			},
			wantErr: true,
			errMsg:  "same path",
		},
		{
			name: "projects_folder inside links_folder rejected",
			cfg: Config{
				ProjectsFolder: "/home/user/links/projects",
				LinksFolder:    "/home/user/links",
			},
			wantErr: true,
			errMsg:  "inside links_folder",
		},
		{
			name: "links_folder inside projects_folder is ok",
			cfg: Config{
				ProjectsFolder: "/home/user/projects",
				LinksFolder:    "/home/user/projects/links",
			},
		},
		{
			name: "metadata_folder inside projects_folder is ok",
			cfg: Config{
				ProjectsFolder: "/home/user/projects",
				MetadataFolder: "/home/user/projects/metadata",
			},
		},
		{
			name: "links_folder equals metadata_folder rejected",
			cfg: Config{
				MetadataFolder: "/home/user/meta",
				LinksFolder:    "/home/user/meta",
			},
			wantErr: true,
			errMsg:  "same path",
		},
		{
			name: "disjoint paths ok",
			cfg: Config{
				ProjectsFolder: "/home/user/projects",
				MetadataFolder: "/home/user/projects/metadata",
				LinksFolder:    "/home/user/links",
			},
		},

		// machine_id
		{
			name: "machine_id empty is ok",
			cfg:  Config{},
		},
		{
			name: "machine_id alphanumeric is ok",
			cfg:  Config{MachineID: "abc123"},
		},
		{
			name: "machine_id with dot dash underscore is ok",
			cfg:  Config{MachineID: "host_a-1.local"},
		},
		{
			name: "machine_id uuid is ok",
			cfg:  Config{MachineID: "550e8400-e29b-41d4-a716-446655440000"},
		},
		{
			name:    "machine_id too long rejected",
			cfg:     Config{MachineID: "550e8400-e29b-41d4-a716-446655440000x"},
			wantErr: true,
			errMsg:  "machine_id",
		},
		{
			name:    "machine_id space rejected",
			cfg:     Config{MachineID: "host name"},
			wantErr: true,
			errMsg:  "machine_id",
		},
		{
			name:    "machine_id slash rejected",
			cfg:     Config{MachineID: "host/name"},
			wantErr: true,
			errMsg:  "machine_id",
		},
		{
			name:    "machine_id at-sign rejected",
			cfg:     Config{MachineID: "host@name"},
			wantErr: true,
			errMsg:  "machine_id",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if tt.errMsg != "" && !contains(err.Error(), tt.errMsg) {
					t.Errorf("error %q should contain %q", err.Error(), tt.errMsg)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchStr(s, substr)
}

func searchStr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
