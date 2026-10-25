package cmd

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"
	"github.com/gorodulin/prj/internal/config"
)

var lastCmd = &cobra.Command{
	Use:   "last [<N>]",
	Short: "List the N most recent projects",
	Long: `List the N most recent projects (default 1), oldest first.

Recency is the creation order encoded in project IDs, so "prj last 5" shows
the same projects as "prj list | tail -n 5" in any output format. Only local
projects are considered unless --all is given.`,
	Args: cobra.MaximumNArgs(1),
	RunE: runLast,
}

func init() {
	rootCmd.AddCommand(lastCmd)
	lastCmd.Flags().BoolP("all", "a", false, "include metadata-only projects (not present locally)")
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

	showAll, _ := cmd.Flags().GetBool("all")
	formatStr := resolveOutputFormat(cmd, cfg)

	if err := requireConfig(cfg, "projects_folder"); err != nil {
		return err
	}

	ids, err := collectIDs(cfg)
	if err != nil {
		return err
	}

	candidates := ids.localOnly()
	if showAll {
		candidates = ids.all
	}

	// Trim before resolving so only N projects' metadata is read.
	projects := resolveProjects(lastN(candidates, n), cfg, ids.localSet)
	if len(projects) == 0 {
		printNoProjects(ids, showAll, false)
		return nil
	}

	return writeProjectList(projects, formatStr, cfg)
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
