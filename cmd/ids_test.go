package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/gorodulin/prj/internal/config"
)

func TestCollectIDs(t *testing.T) {
	base := t.TempDir()

	projectsDir := filepath.Join(base, "projects")
	metadataDir := filepath.Join(base, "metadata")
	os.MkdirAll(filepath.Join(projectsDir, "p20260101a"), 0755)
	os.MkdirAll(filepath.Join(projectsDir, "p20260102b"), 0755)
	os.MkdirAll(filepath.Join(metadataDir, "p20260101a_meta"), 0755) // also local
	os.MkdirAll(filepath.Join(metadataDir, "p20260103c_meta"), 0755) // metadata-only

	cfg := config.Config{
		ProjectsFolder:  projectsDir,
		MetadataFolder:  metadataDir,
		MetadataSuffix:  "_meta",
		ProjectIDType:   "aYYYYMMDDb",
		ProjectIDPrefix: "p",
	}

	ids, err := collectIDs(cfg)
	if err != nil {
		t.Fatal(err)
	}

	t.Run("all contains union of local and metadata-only", func(t *testing.T) {
		if len(ids.all) != 3 {
			t.Errorf("got %d IDs, want 3: %v", len(ids.all), ids.all)
		}
	})

	t.Run("localSet identifies projects with folders", func(t *testing.T) {
		if !ids.localSet["p20260101a"] {
			t.Error("p20260101a should be local")
		}
		if !ids.localSet["p20260102b"] {
			t.Error("p20260102b should be local")
		}
		if ids.localSet["p20260103c"] {
			t.Error("p20260103c should NOT be local (metadata-only)")
		}
	})

	t.Run("filtering by localSet excludes metadata-only", func(t *testing.T) {
		var local []string
		for _, id := range ids.all {
			if ids.localSet[id] {
				local = append(local, id)
			}
		}
		if len(local) != 2 {
			t.Errorf("got %d local IDs, want 2: %v", len(local), local)
		}
	})

	t.Run("showAll includes metadata-only", func(t *testing.T) {
		var all []string
		for _, id := range ids.all {
			all = append(all, id)
		}
		if len(all) != 3 {
			t.Errorf("got %d IDs with showAll, want 3: %v", len(all), all)
		}
	})
}
