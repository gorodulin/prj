package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/gorodulin/prj/internal/config"
	"github.com/gorodulin/prj/internal/project"
	"github.com/gorodulin/prj/internal/text"
)

var findCmd = &cobra.Command{
	Use:   "find <query-term> [<query-term>...]",
	Short: "Search all projects (local and metadata-only)",
	Long: `Search all known projects — local and metadata-only — by query terms.

Query terms are matched as case-insensitive substrings against project ID,
title, and tags; a project is shown if any term matches (OR logic). Use "--"
before terms that start with a dash (e.g. prj find -- -foo).

At least one query term is required. For tag-only filtering across all
projects, use: prj list -a --tags <tags>.`,
	Args: cobra.MinimumNArgs(1),
	RunE: runFind,
}

func init() {
	rootCmd.AddCommand(findCmd)
	findCmd.Flags().StringP("format", "f", "", `output format: json, jsonl, or Go template (e.g. "{{.ID}}\t{{.Title}}")`)
	findCmd.Flags().String("tags", "", "filter by tags (comma-separated, AND logic)")
	findCmd.Flags().Bool("json", false, "shorthand for --format json")
	findCmd.Flags().Bool("jsonl", false, "shorthand for --format jsonl")
	findCmd.MarkFlagsMutuallyExclusive("format", "json", "jsonl")
}

func runFind(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load(cfgFile)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	formatStr := resolveOutputFormat(cmd, cfg)

	if err := requireConfig(cfg, "projects_folder"); err != nil {
		return err
	}

	ids, err := collectIDs(cfg)
	if err != nil {
		return err
	}

	var projects []project.Project
	for _, id := range ids.all {
		projects = append(projects, resolveProject(id, cfg, ids.localSet[id]))
	}

	tagsRaw, _ := cmd.Flags().GetString("tags")
	filterTags := text.ParseTags(tagsRaw)
	projects = filterProjects(projects, args, filterTags)

	if len(projects) == 0 {
		fmt.Println("No projects match the given query.")
		return nil
	}

	return writeProjectList(projects, formatStr, cfg)
}
