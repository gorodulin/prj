package cmd

import (
	"fmt"
	"slices"

	"github.com/gorodulin/prj/internal/config"
	"github.com/gorodulin/prj/internal/project"
)

// idSets holds the results of scanning both project and metadata folders.
type idSets struct {
	all      []string       // deduplicated union in creation order
	localSet map[string]bool // IDs with a local project folder
}

// collectIDs scans both project and metadata folders in one pass.
func collectIDs(cfg config.Config) (idSets, error) {
	projectIDs, err := project.CollectIDsFromFolder(cfg.ProjectsFolder, cfg.ProjectIDType, cfg.ProjectIDPrefix, "")
	if err != nil {
		return idSets{}, fmt.Errorf("scan project folders: %w", err)
	}

	var metaIDs []string
	if cfg.MetadataFolder != "" {
		metaIDs, err = project.CollectIDsFromFolder(cfg.MetadataFolder, cfg.ProjectIDType, cfg.ProjectIDPrefix, cfg.MetadataSuffix)
		if err != nil {
			return idSets{}, fmt.Errorf("scan metadata folders: %w", err)
		}
	}

	localSet := make(map[string]bool, len(projectIDs))
	for _, id := range projectIDs {
		localSet[id] = true
	}

	return idSets{
		all:      unionSorted(projectIDs, metaIDs),
		localSet: localSet,
	}, nil
}

// unionSorted merges two ID slices into a deduplicated result sorted in
// creation order (see project.CompareIDs).
func unionSorted(a, b []string) []string {
	seen := make(map[string]bool, len(a)+len(b))
	for _, s := range a {
		seen[s] = true
	}
	for _, s := range b {
		seen[s] = true
	}

	result := make([]string, 0, len(seen))
	for s := range seen {
		result = append(result, s)
	}
	slices.SortFunc(result, project.CompareIDs)
	return result
}
