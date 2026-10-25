package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/gorodulin/prj/internal/config"
	"github.com/gorodulin/prj/internal/project"
	"github.com/gorodulin/prj/internal/text"
)

var findCmd = &cobra.Command{
	Use:   "find [<query-term>...] [--tags <tags>]",
	Short: "Search all projects (local and metadata-only)",
	Long: `Search all known projects — local and metadata-only — by query terms.

Query terms are matched as case-insensitive substrings against project ID,
title, and tags; a project is shown if any term matches (OR logic). Use "--"
before terms that start with a dash (e.g. prj find -- -foo).

At least one query term or --tags is required; with --tags alone, every
project carrying all the given tags is shown.`,
	Args: requireQueryOrTags,
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

// requireQueryOrTags rejects a bare "prj find", which would list everything.
func requireQueryOrTags(cmd *cobra.Command, args []string) error {
	tagsRaw, _ := cmd.Flags().GetString("tags")
	if len(args) == 0 && len(text.ParseTags(tagsRaw)) == 0 {
		return fmt.Errorf("requires at least one query term or --tags")
	}
	return nil
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
