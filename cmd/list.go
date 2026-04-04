package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/gorodulin/prj/internal/config"
	"github.com/gorodulin/prj/internal/format"
	"github.com/gorodulin/prj/internal/project"
	"github.com/gorodulin/prj/internal/text"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List local projects",
	RunE:  runList,
}

func init() {
	rootCmd.AddCommand(listCmd)
	listCmd.Flags().BoolP("all", "a", false, "include metadata-only projects (not present locally)")
	listCmd.Flags().StringP("format", "f", "table", `output format: table, json, jsonl, or Go template (e.g. "{{.ID}}\t{{.Title}}")`)
	listCmd.Flags().StringP("query", "q", "", "filter by substring (matches ID, title, tags)")
	listCmd.Flags().StringSlice("tag", nil, "filter by exact tag (repeatable, AND logic)")
	listCmd.Flags().Bool("missing", false, "show only metadata-only projects (not present locally)")
	listCmd.MarkFlagsMutuallyExclusive("all", "missing")
}

func runList(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load(cfgFile)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	showAll, _ := cmd.Flags().GetBool("all")
	showMissing, _ := cmd.Flags().GetBool("missing")

	// Priority: explicit --format flag > config list_format > default "table".
	formatStr, _ := cmd.Flags().GetString("format")
	if !cmd.Flags().Changed("format") && cfg.ListFormat != "" {
		formatStr = cfg.ListFormat
	}

	if cfg.ProjectsFolder == "" {
		fmt.Println("No projects folder configured. Set \"projects_folder\" in config.")
		return nil
	}

	ids, err := collectIDs(cfg)
	if err != nil {
		return err
	}

	// Determine which IDs to display.
	var displayIDs []string
	for _, id := range ids.all {
		switch {
		case showMissing:
			if !ids.localSet[id] {
				displayIDs = append(displayIDs, id)
			}
		case showAll:
			displayIDs = append(displayIDs, id)
		default:
			if ids.localSet[id] {
				displayIDs = append(displayIDs, id)
			}
		}
	}

	// Collect projects.
	var projects []project.Project
	for _, id := range displayIDs {
		projects = append(projects, resolveProject(id, cfg, ids.localSet[id]))
	}

	// Apply filters.
	query, _ := cmd.Flags().GetString("query")
	filterTags, _ := cmd.Flags().GetStringSlice("tag")
	filterTags = text.NormalizeTags(filterTags)
	projects = filterProjects(projects, strings.ToLower(query), filterTags)

	if len(projects) == 0 {
		if query != "" || len(filterTags) > 0 {
			fmt.Println("No projects match the given filter.")
		} else if showMissing {
			fmt.Println("No metadata-only projects found. All projects are available locally.")
		} else if !showAll && len(ids.all) > 0 {
			fmt.Println("No local projects found. Use --all to include metadata-only projects.")
		} else {
			fmt.Println("No projects found.")
		}
		return nil
	}

	opts := format.Options{
		Color: format.IsTTY(os.Stdout),
	}

	return format.Format(os.Stdout, projects, formatStr, opts)
}

func filterProjects(projects []project.Project, query string, tags []string) []project.Project {
	if query == "" && len(tags) == 0 {
		return projects
	}
	var result []project.Project
	for _, p := range projects {
		if matchesQuery(p, query) && matchesTags(p, tags) {
			result = append(result, p)
		}
	}
	return result
}

func matchesQuery(p project.Project, query string) bool {
	if query == "" {
		return true
	}
	q := strings.ToLower(query)
	if strings.Contains(strings.ToLower(p.ID), q) {
		return true
	}
	if strings.Contains(strings.ToLower(p.Title), q) {
		return true
	}
	for _, t := range p.Tags {
		if strings.Contains(strings.ToLower(t), q) {
			return true
		}
	}
	return false
}

func matchesTags(p project.Project, tags []string) bool {
	for _, required := range tags {
		found := false
		for _, t := range p.Tags {
			if t == required {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}
