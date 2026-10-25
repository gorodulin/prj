package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/gorodulin/prj/internal/config"
	"github.com/gorodulin/prj/internal/text"
)

var listCmd = &cobra.Command{
	Use:   "list [<query-term>...]",
	Short: "List local projects",
	Long: `List local projects, optionally filtered by query terms.

Query terms are matched as case-insensitive substrings against project ID,
title, and tags; a project is shown if any term matches (OR logic). Use "--"
before terms that start with a dash (e.g. prj list -- -foo).

Projects tagged with any tag in the hide_tags config are hidden; --no-hide
or --all shows them, and --tags lifts hiding for the tags it names (e.g.
--tags archived shows archived projects).`,
	Args: cobra.ArbitraryArgs,
	RunE: runList,
}

func init() {
	rootCmd.AddCommand(listCmd)
	addScopeFlags(listCmd)
	listCmd.Flags().StringP("query", "q", "", "additional query term (same matching as positional terms)")
	listCmd.Flags().String("tags", "", "filter by tags (comma-separated, AND logic)")
	listCmd.Flags().String("tag", "", "deprecated alias for --tags")
	_ = listCmd.Flags().MarkDeprecated("tag", "use --tags instead")
	addOutputFormatFlags(listCmd)
	listCmd.MarkFlagsMutuallyExclusive("tags", "tag")
}

func runList(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load(cfgFile)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	scope := resolveScope(cmd, presenceLocal)
	formatStr := resolveOutputFormat(cmd, cfg)

	if err := requireConfig(cfg, "projects_folder"); err != nil {
		return err
	}

	ids, err := collectIDs(cfg)
	if err != nil {
		return err
	}

	projects := resolveProjects(ids.byPresence(scope.presence), cfg, ids.localSet)

	// Apply filters.
	terms := append([]string{}, args...)
	if query, _ := cmd.Flags().GetString("query"); query != "" {
		terms = append(terms, query)
	}
	tagsRaw, _ := cmd.Flags().GetString("tags")
	if !cmd.Flags().Changed("tags") {
		tagsRaw, _ = cmd.Flags().GetString("tag")
	}
	filterTags := text.ParseTags(tagsRaw)

	// Hide before filtering: explicit --tags lift hiding for the tags named.
	projects, hiddenCount := partitionHidden(projects, scope.activeHideTags(cfg.HideTags), filterTags)
	projects = filterProjects(projects, terms, filterTags)

	if len(projects) == 0 {
		if len(terms) > 0 || len(filterTags) > 0 {
			fmt.Println("No projects match the given filter.")
			printHiddenHint(os.Stderr, hiddenCount)
		} else {
			printNoProjects(ids, scope, hiddenCount)
		}
		return nil
	}

	if err := writeProjectList(projects, formatStr, cfg); err != nil {
		return err
	}
	printHiddenHint(os.Stderr, hiddenCount)
	return nil
}

// printNoProjects explains an empty, unfiltered project list.
func printNoProjects(ids idSets, scope listScope, hiddenCount int) {
	switch {
	case hiddenCount > 0:
		fmt.Printf("No visible projects. %d hidden by hide_tags; use --no-hide.\n", hiddenCount)
	case scope.presence == presenceMissing:
		fmt.Println("No metadata-only projects found. All projects are available locally.")
	case scope.presence == presenceLocal && len(ids.all) > 0:
		fmt.Println("No local projects found. Use --all to include metadata-only projects.")
	default:
		fmt.Println("No projects found.")
	}
}
