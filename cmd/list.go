package cmd

import (
	"fmt"

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
before terms that start with a dash (e.g. prj list -- -foo).`,
	Args: cobra.ArbitraryArgs,
	RunE: runList,
}

func init() {
	rootCmd.AddCommand(listCmd)
	listCmd.Flags().BoolP("all", "a", false, "include metadata-only projects (not present locally)")
	listCmd.Flags().StringP("query", "q", "", "additional query term (same matching as positional terms)")
	listCmd.Flags().String("tags", "", "filter by tags (comma-separated, AND logic)")
	listCmd.Flags().String("tag", "", "deprecated alias for --tags")
	_ = listCmd.Flags().MarkDeprecated("tag", "use --tags instead")
	listCmd.Flags().Bool("missing", false, "show only metadata-only projects (not present locally)")
	addOutputFormatFlags(listCmd)
	listCmd.MarkFlagsMutuallyExclusive("all", "missing")
	listCmd.MarkFlagsMutuallyExclusive("tags", "tag")
}

func runList(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load(cfgFile)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	showAll, _ := cmd.Flags().GetBool("all")
	showMissing, _ := cmd.Flags().GetBool("missing")
	formatStr := resolveOutputFormat(cmd, cfg)

	if err := requireConfig(cfg, "projects_folder"); err != nil {
		return err
	}

	ids, err := collectIDs(cfg)
	if err != nil {
		return err
	}

	// Determine which IDs to display.
	var displayIDs []string
	switch {
	case showMissing:
		displayIDs = ids.missingOnly()
	case showAll:
		displayIDs = ids.all
	default:
		displayIDs = ids.localOnly()
	}

	projects := resolveProjects(displayIDs, cfg, ids.localSet)

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
	projects = filterProjects(projects, terms, filterTags)

	if len(projects) == 0 {
		if len(terms) > 0 || len(filterTags) > 0 {
			fmt.Println("No projects match the given filter.")
		} else {
			printNoProjects(ids, showAll, showMissing)
		}
		return nil
	}

	return writeProjectList(projects, formatStr, cfg)
}

// printNoProjects explains an empty, unfiltered project list.
func printNoProjects(ids idSets, showAll, showMissing bool) {
	switch {
	case showMissing:
		fmt.Println("No metadata-only projects found. All projects are available locally.")
	case !showAll && len(ids.all) > 0:
		fmt.Println("No local projects found. Use --all to include metadata-only projects.")
	default:
		fmt.Println("No projects found.")
	}
}
