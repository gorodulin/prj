package cmd

import (
	"strings"
	"testing"

	"github.com/gorodulin/prj/internal/config"
)

func TestRequireConfigUnknownKey(t *testing.T) {
	cfg := config.Config{}
	err := requireConfig(cfg, "nonexistent_key")
	if err == nil {
		t.Fatal("expected error for unknown key")
	}
	if !strings.Contains(err.Error(), "internal error") {
		t.Errorf("error %q should mention internal error", err.Error())
	}
}

func TestRequireConfig(t *testing.T) {
	tests := []struct {
		name    string
		cfg     config.Config
		keys    []string
		wantErr bool
		errContains []string
	}{
		{
			name:    "all set",
			cfg:     config.Config{ProjectsFolder: "/tmp/p"},
			keys:    []string{"projects_folder"},
			wantErr: false,
		},
		{
			name:    "one missing",
			cfg:     config.Config{},
			keys:    []string{"projects_folder"},
			wantErr: true,
			errContains: []string{
				"missing required configuration",
				"prj config set projects_folder",
			},
		},
		{
			name:    "multiple missing",
			cfg:     config.Config{},
			keys:    []string{"metadata_folder", "machine_name", "machine_id"},
			wantErr: true,
			errContains: []string{
				"metadata_folder",
				"machine_name",
				"machine_id",
				"prj config set",
			},
		},
		{
			name:    "partial missing",
			cfg:     config.Config{MetadataFolder: "/tmp/m"},
			keys:    []string{"metadata_folder", "machine_name"},
			wantErr: true,
			errContains: []string{"machine_name"},
		},
		{
			name:    "no keys required",
			cfg:     config.Config{},
			keys:    nil,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := requireConfig(tt.cfg, tt.keys...)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				for _, s := range tt.errContains {
					if !strings.Contains(err.Error(), s) {
						t.Errorf("error %q should contain %q", err.Error(), s)
					}
				}
				// Verify partial missing doesn't include set fields.
				if tt.name == "partial missing" {
					if strings.Contains(err.Error(), "metadata_folder") {
						t.Error("error should not mention metadata_folder (it was set)")
					}
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}
