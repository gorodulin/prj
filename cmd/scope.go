package cmd

import (
	"fmt"
	"io"
	"os"
	"slices"

	"github.com/spf13/cobra"
	"github.com/gorodulin/prj/internal/format"
	"github.com/gorodulin/prj/internal/project"
)

// presence selects projects by whether they have a local project folder.
type presence int

const (
	presenceLocal   presence = iota // only projects with a local folder
	presenceMissing                 // only metadata-only projects
	presenceAll                     // both
)

// listScope is the resolved visibility choice for a project-list command
// (list, find, last): which projects it shows before query/tag filtering.
type listScope struct {
	presence   presence
	showHidden bool // true when --no-hide or --all lifts hide_tags filtering
}

// addScopeFlags registers --local, --missing, --all and --no-hide on a
// project-list command; resolveScope reads them.
func addScopeFlags(cmd *cobra.Command) {
	cmd.Flags().Bool("local", false, "show only projects with a local folder")
	cmd.Flags().Bool("missing", false, "show only metadata-only projects (not present locally)")
	cmd.Flags().BoolP("all", "a", false, "show every project: local, metadata-only, and hidden (implies --no-hide)")
	cmd.Flags().Bool("no-hide", false, "show projects hidden by the hide_tags config")
	cmd.MarkFlagsMutuallyExclusive("local", "missing", "all")
}

// resolveScope returns the scope selected by the flags from addScopeFlags.
// defaultPresence applies when none of --local, --missing, --all is given.
// --all implies --no-hide; --local and --missing never affect hiding.
func resolveScope(cmd *cobra.Command, defaultPresence presence) listScope {
	local, _ := cmd.Flags().GetBool("local")
	missing, _ := cmd.Flags().GetBool("missing")
	all, _ := cmd.Flags().GetBool("all")
	noHide, _ := cmd.Flags().GetBool("no-hide")

	scope := listScope{presence: defaultPresence, showHidden: noHide || all}
	switch {
	case local:
		scope.presence = presenceLocal
	case missing:
		scope.presence = presenceMissing
	case all:
		scope.presence = presenceAll
	}
	return scope
}

// activeHideTags returns the hide tags that apply under this scope: none
// when hiding is lifted, otherwise the configured ones.
func (s listScope) activeHideTags(configured []string) []string {
	if s.showHidden {
		return nil
	}
	return configured
}

// byPresence returns the IDs matching p, in creation order.
func (ids idSets) byPresence(p presence) []string {
	switch p {
	case presenceMissing:
		return ids.missingOnly()
	case presenceAll:
		return ids.all
	default:
		return ids.localOnly()
	}
}

// isHiddenByTags reports whether the project carries a hide tag that the
// user did not ask for explicitly via --tags. Tags are compared exactly;
// both sides are already normalized.
func isHiddenByTags(p project.Project, hideTags, explicitTags []string) bool {
	for _, hideTag := range hideTags {
		if slices.Contains(p.Tags, hideTag) && !slices.Contains(explicitTags, hideTag) {
			return true
		}
	}
	return false
}

// partitionHidden drops projects hidden by hideTags (see isHiddenByTags)
// and returns the rest in order, plus how many were dropped.
func partitionHidden(projects []project.Project, hideTags, explicitTags []string) (visible []project.Project, hiddenCount int) {
	if len(hideTags) == 0 {
		return projects, 0
	}
	visible = make([]project.Project, 0, len(projects))
	for _, p := range projects {
		if isHiddenByTags(p, hideTags, explicitTags) {
			hiddenCount++
			continue
		}
		visible = append(visible, p)
	}
	return visible, hiddenCount
}

// printHiddenHint tells the user that hide_tags filtered out projects. It
// prints only when something was hidden and w is a terminal, so piped and
// machine-readable output is never affected. Callers pass os.Stderr.
func printHiddenHint(w io.Writer, hiddenCount int) {
	if hiddenCount == 0 {
		return
	}
	if f, ok := w.(*os.File); !ok || !format.IsTTY(f) {
		return
	}
	fmt.Fprintf(w, "%d project(s) hidden by hide_tags; use --no-hide to show them\n", hiddenCount)
}
