package cmd

import (
	"fmt"
	"os"
	"strconv"

	"github.com/spf13/cobra"
	"github.com/gorodulin/prj/internal/config"
	"github.com/gorodulin/prj/internal/project"
)

var lastCmd = &cobra.Command{
	Use:   "last [<N>]",
	Short: "List the N most recent projects",
	Long: `List the N most recent projects (default 1), oldest first.

Recency is the creation order encoded in project IDs, so "prj last 5" shows
the same projects as "prj list | tail -n 5" in any output format. Only local
projects are considered unless --missing or --all is given.

Projects tagged with any tag in the hide_tags config are skipped, so the N
most recent visible projects are shown; --no-hide or --all includes them.`,
	Args: cobra.MaximumNArgs(1),
	RunE: runLast,
}

func init() {
	rootCmd.AddCommand(lastCmd)
	addScopeFlags(lastCmd)
	addOutputFormatFlags(lastCmd)
}

func runLast(cmd *cobra.Command, args []string) error {
	n, err := parseLastCount(args)
	if err != nil {
		return err
	}

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

	candidates := ids.byPresence(scope.presence)
	hideTags := scope.activeHideTags(cfg.HideTags)

	var projects []project.Project
	var hiddenCount int
	if len(hideTags) == 0 {
		// Trim before resolving so only N projects' metadata is read.
		projects = resolveProjects(lastN(candidates, n), cfg, ids.localSet)
	} else {
		resolve := func(id string) project.Project { return resolveProject(id, cfg, ids.localSet[id]) }
		projects, hiddenCount = lastNVisible(candidates, n, resolve, hideTags)
	}
	if len(projects) == 0 {
		printNoProjects(ids, scope, hiddenCount)
		return nil
	}

	if err := writeProjectList(projects, formatStr, cfg); err != nil {
		return err
	}
	printHiddenHint(os.Stderr, hiddenCount)
	return nil
}

// parseLastCount returns the N argument of "prj last", defaulting to 1.
func parseLastCount(args []string) (int, error) {
	if len(args) == 0 {
		return 1, nil
	}
	n, err := strconv.Atoi(args[0])
	if err != nil || n <= 0 {
		return 0, fmt.Errorf("N must be a positive integer, got %q", args[0])
	}
	return n, nil
}

// lastN returns the final n IDs, or all of them if there are fewer.
func lastN(ids []string, n int) []string {
	if n >= len(ids) {
		return ids
	}
	return ids[len(ids)-n:]
}

// lastNVisible walks ids newest to oldest, resolving one at a time, until n
// projects not hidden by hideTags are collected. It returns them oldest
// first, plus how many hidden projects were skipped along the way. Hidden
// projects older than the n-th visible one are not resolved or counted.
func lastNVisible(ids []string, n int, resolve func(id string) project.Project, hideTags []string) ([]project.Project, int) {
	var newestFirst []project.Project
	hiddenCount := 0
	for i := len(ids) - 1; i >= 0 && len(newestFirst) < n; i-- {
		p := resolve(ids[i])
		if isHiddenByTags(p, hideTags, nil) {
			hiddenCount++
			continue
		}
		newestFirst = append(newestFirst, p)
	}

	oldestFirst := make([]project.Project, len(newestFirst))
	for i, p := range newestFirst {
		oldestFirst[len(newestFirst)-1-i] = p
	}
	return oldestFirst, hiddenCount
}
