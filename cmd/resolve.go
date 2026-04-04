package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/gorodulin/prj/internal/config"
	"github.com/gorodulin/prj/internal/metadata"
	"github.com/gorodulin/prj/internal/project"
)

// resolveCurrentProjectID resolves the magic "current" ID to the actual
// project ID by checking if the cwd is inside projects_folder.
func resolveCurrentProjectID(cfg config.Config) (string, error) {
	if cfg.ProjectsFolder == "" {
		return "", fmt.Errorf("no projects folder configured")
	}

	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("get working directory: %w", err)
	}

	return projectIDFromPath(cwd, cfg.ProjectsFolder, cfg.ProjectIDType)
}

// projectIDFromPath extracts a valid project ID from cwd relative to
// projectsFolder. Returns an error if cwd is not inside a project folder.
func projectIDFromPath(cwd, projectsFolder, idFormat string) (string, error) {
	rel, err := filepath.Rel(projectsFolder, cwd)
	if err != nil {
		return "", fmt.Errorf("not inside a project folder (cwd is %s)", cwd)
	}

	// Must be inside projects_folder, not at it or outside.
	if rel == "." || strings.HasPrefix(rel, "..") || filepath.IsAbs(rel) {
		return "", fmt.Errorf("not inside a project folder (cwd is %s)", cwd)
	}

	// First path component is the project ID.
	id := strings.SplitN(rel, string(filepath.Separator), 2)[0]

	if idFormat != "" && !project.IsValidID(id, idFormat) {
		return "", fmt.Errorf("not inside a project folder (cwd is %s)", cwd)
	}

	return id, nil
}

// expandProjectID replaces "current" with the resolved project ID, or
// returns the input unchanged.
func expandProjectID(id string, cfg config.Config) (string, error) {
	if id != "current" {
		return id, nil
	}
	return resolveCurrentProjectID(cfg)
}

// resolveProject fills Title and Tags from metadata, falling back to
// README for title. Path is set if hasLocalFolder is true.
func resolveProject(id string, cfg config.Config, hasLocalFolder bool) project.Project {
	p := project.Project{ID: id, Local: hasLocalFolder}

	if hasLocalFolder {
		p.Path = filepath.Join(cfg.ProjectsFolder, id)
	}

	if cfg.MetadataFolder != "" {
		metaDir := cfg.MetadataDir(id)
		snapshots, err := metadata.ReadSnapshots(metaDir)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: read metadata for %s: %v\n", id, err)
		} else if len(snapshots) > 0 {
			m := metadata.LatestHead(snapshots)
			p.Title = m.Title
			p.Tags = m.Tags
		}
	}

	if p.Title == "" && p.Path != "" {
		p.Title = project.ReadmeTitle(p.Path)
	}

	return p
}
