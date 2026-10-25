package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/gorodulin/prj/internal/config"
	"github.com/gorodulin/prj/internal/text"
)

var findCmd = &cobra.Command{
	Use:   "find [<query-term>...] [--tags <tags>]",
	Short: "Search all projects (local and metadata-only), except hidden ones",
	Long: `Search all known projects — local and metadata-only — by query terms.

Query terms are matched as case-insensitive substrings against project ID,
title, and tags; a project is shown if any term matches (OR logic). Use "--"
before terms that start with a dash (e.g. prj find -- -foo).

At least one query term or --tags is required; with --tags alone, every
project carrying all the given tags is shown.

Use --local or --missing to search only projects with or without a local
folder. Projects tagged with any tag in the hide_tags config are hidden;
--no-hide or --all shows them, and --tags lifts hiding for the tags it
names. Query terms do not lift hiding.`,
	Args: requireQueryOrTags,
	RunE: runFind,
}

func init() {
	rootCmd.AddCommand(findCmd)
	addScopeFlags(findCmd)
	findCmd.Flags().String("tags", "", "filter by tags (comma-separated, AND logic)")
	addOutputFormatFlags(findCmd)
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

	scope := resolveScope(cmd, presenceAll)
	formatStr := resolveOutputFormat(cmd, cfg)

	if err := requireConfig(cfg, "projects_folder"); err != nil {
		return err
	}

	ids, err := collectIDs(cfg)
	if err != nil {
		return err
	}

	projects := resolveProjects(ids.byPresence(scope.presence), cfg, ids.localSet)

	tagsRaw, _ := cmd.Flags().GetString("tags")
	filterTags := text.ParseTags(tagsRaw)

	// Hide before filtering: explicit --tags lift hiding for the tags named.
	projects, hiddenCount := partitionHidden(projects, scope.activeHideTags(cfg.HideTags), filterTags)
	projects = filterProjects(projects, args, filterTags)

	if len(projects) == 0 {
		fmt.Println("No projects match the given query.")
		printHiddenHint(os.Stderr, hiddenCount)
		return nil
	}

	if err := writeProjectList(projects, formatStr, cfg); err != nil {
		return err
	}
	printHiddenHint(os.Stderr, hiddenCount)
	return nil
}
