package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/gorodulin/prj/internal/config"
	"github.com/gorodulin/prj/internal/linktree"
	"github.com/gorodulin/prj/internal/text"
)

// collectPlacements mirrors the project-iteration loop from runLink,
// returning the IDs that would receive link placements.
func collectPlacements(ids idSets, cfg config.Config, tree *linktree.Folder, showAll bool) []string {
	var placed []string
	for _, id := range ids.all {
		if !showAll && !ids.localSet[id] {
			continue
		}

		p := resolveProject(id, cfg, ids.localSet[id])
		tags := text.NormalizeTags(p.Tags)

		folders := linktree.FindPlacements(tree, tags, cfg.LinkSinkName)
		if len(folders) == 0 {
			continue
		}
		placed = append(placed, p.ID)
	}
	return placed
}

func TestCollectPlacements_ShowAll(t *testing.T) {
	base := t.TempDir()

	projectsDir := filepath.Join(base, "projects")
	metadataDir := filepath.Join(base, "metadata")
	linksDir := filepath.Join(base, "links")

	// Local project.
	os.MkdirAll(filepath.Join(projectsDir, "p20260101a"), 0755)
	// Metadata-only project (not present locally).
	os.MkdirAll(filepath.Join(metadataDir, "p20260102b_meta"), 0755)

	// Create a sink folder so all projects get placed.
	sinkName := "_unsorted"
	os.MkdirAll(filepath.Join(linksDir, sinkName), 0755)

	cfg := config.Config{
		ProjectsFolder:  projectsDir,
		MetadataFolder:  metadataDir,
		MetadataSuffix:  "_meta",
		ProjectIDType:   "aYYYYMMDDb",
		ProjectIDPrefix: "p",
		LinksFolder:     linksDir,
		LinkSinkName:    sinkName,
	}

	ids, err := collectIDs(cfg)
	if err != nil {
		t.Fatal(err)
	}

	tree, err := linktree.BuildTree(linksDir)
	if err != nil {
		t.Fatal(err)
	}

	t.Run("without --all skips metadata-only", func(t *testing.T) {
		placed := collectPlacements(ids, cfg, tree, false)
		if len(placed) != 1 {
			t.Fatalf("got %v, want [p20260101a]", placed)
		}
		if placed[0] != "p20260101a" {
			t.Errorf("got %s, want p20260101a", placed[0])
		}
	})

	t.Run("with --all includes metadata-only", func(t *testing.T) {
		placed := collectPlacements(ids, cfg, tree, true)
		if len(placed) != 2 {
			t.Fatalf("got %v, want [p20260101a p20260102b]", placed)
		}
		if placed[0] != "p20260101a" || placed[1] != "p20260102b" {
			t.Errorf("got %v, want [p20260101a p20260102b]", placed)
		}
	})
}
